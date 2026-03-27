// 企业微信 AI 机器人扫码创建：HTTP 调用 generate 与 query_result，成功后得到 BotID、Secret，
// 凭据用途与控制台创建的一致，交给 WSClient / SetCredentials，连上 openws 后发认证帧即完成登录。
package aibot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"time"
)

// 与企业微信扫码创建流程相关的 HTTP 接口（与 wecom-openclaw-cli 中 qrcode 工具逻辑一致）。
const (
	qrGenPagePath             = "https://work.weixin.qq.com/ai/qc/gen"
	qrGeneratePath            = "https://work.weixin.qq.com/ai/qc/generate"
	qrQueryPath               = "https://work.weixin.qq.com/ai/qc/query_result"
	defaultQRScanPollInterval = 3 * time.Second
	defaultQRScanTimeout      = 5 * time.Minute

	// QRScanDefaultSource 请求 source 参数默认值，与 CLI 保持一致以便服务端识别。
	QRScanDefaultSource = "wecom-cli"
)

var (
	// ErrQRGenerateBadResponse generate 接口返回缺少 scode 或 auth_url。
	ErrQRGenerateBadResponse = errors.New("aibot: qr generate response missing scode or auth_url")
	// ErrQRScanNoBotInfo 轮询状态为 success 但未返回完整 botid/secret。
	ErrQRScanNoBotInfo = errors.New("aibot: qr scan succeeded but bot_info is incomplete")
	// ErrQRScanTimeout 在超时时间内未完成扫码。
	ErrQRScanTimeout = errors.New("aibot: qr scan timed out")
)

// QRCodeSession 调用 generate 成功后得到的会话信息，用于展示二维码或引导用户扫码。
type QRCodeSession struct {
	Scode   string // 轮询 query_result 所需
	AuthURL string // 生成二维码的内容（通常为可扫码 URL）
}

// ScanBotCredentials 扫码创建成功后返回的机器人凭据，对应 WS 连接的 BotID / Secret。
type ScanBotCredentials struct {
	BotID  string
	Secret string
}

// QRScanHTTPConfig 扫码相关 HTTP 客户端与轮询行为（零值表示使用默认）。
type QRScanHTTPConfig struct {
	// Client 用于请求 generate / query_result；为 nil 时使用带超时的默认 Client。
	Client *http.Client
	// Source 请求参数 source，空则使用 QRScanDefaultSource。
	Source string
	// Plat 平台码：darwin=1, windows=2, linux=3, 其他=0；小于 0 时按 runtime.GOOS 自动推断。
	Plat int
	// PollInterval 轮询间隔；为零则 3s。
	PollInterval time.Duration
	// PollTimeout 从开始轮询起的最大等待时间；为零则 5 分钟。
	PollTimeout time.Duration
	// GenerateRequestURL 非空时，FetchQRCodeSession 直接向该完整 URL 发起 GET（用于测试或自定义网关）。
	GenerateRequestURL string
	// QueryURLBase 非空时，作为 query_result 的地址基底，并附加 scode 查询参数。
	QueryURLBase string
}

// QRPlatformFromGOOS 将 runtime.GOOS 映射为 generate 接口的 plat 参数（与 wecom-openclaw-cli 一致）。
func QRPlatformFromGOOS() int {
	switch runtime.GOOS {
	case "darwin":
		return 1
	case "windows":
		return 2
	case "linux":
		return 3
	default:
		return 0
	}
}

// QRGenerateURL 构造 generate 请求 URL（含 source 与 plat）。
func QRGenerateURL(source string, plat int) string {
	if source == "" {
		source = QRScanDefaultSource
	}
	u, err := url.Parse(qrGeneratePath)
	if err != nil {
		return qrGeneratePath
	}
	q := u.Query()
	q.Set("source", source)
	q.Set("plat", fmt.Sprintf("%d", plat))
	u.RawQuery = q.Encode()
	return u.String()
}

// QRGenPageURL 浏览器打开也可完成扫码的页面链接（与 CLI 打印的链接一致）。
func QRGenPageURL(scode string) string {
	u, err := url.Parse(qrGenPagePath)
	if err != nil {
		return qrGenPagePath
	}
	q := u.Query()
	q.Set("source", QRScanDefaultSource)
	q.Set("scode", scode)
	u.RawQuery = q.Encode()
	return u.String()
}

// qrGenPageURLWithSource 使用指定 source 构造 gen 页面 URL。
func qrGenPageURLWithSource(source, scode string) string {
	if source == "" {
		source = QRScanDefaultSource
	}
	u, err := url.Parse(qrGenPagePath)
	if err != nil {
		return qrGenPagePath
	}
	q := u.Query()
	q.Set("source", source)
	q.Set("scode", scode)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c QRScanHTTPConfig) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c QRScanHTTPConfig) source() string {
	if c.Source != "" {
		return c.Source
	}
	return QRScanDefaultSource
}

func (c QRScanHTTPConfig) plat() int {
	if c.Plat >= 0 {
		return c.Plat
	}
	return QRPlatformFromGOOS()
}

func (c QRScanHTTPConfig) pollInterval() time.Duration {
	if c.PollInterval > 0 {
		return c.PollInterval
	}
	return defaultQRScanPollInterval
}

func (c QRScanHTTPConfig) pollTimeout() time.Duration {
	if c.PollTimeout > 0 {
		return c.PollTimeout
	}
	return defaultQRScanTimeout
}

type qrGenerateResponse struct {
	Data *struct {
		Scode   string `json:"scode"`
		AuthURL string `json:"auth_url"`
	} `json:"data"`
}

type qrQueryResponse struct {
	Data *struct {
		Status  string `json:"status"`
		BotInfo *struct {
			Botid  string `json:"botid"`
			Secret string `json:"secret"`
		} `json:"bot_info"`
	} `json:"data"`
}

func httpGetBody(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aibot: http %s: %s", resp.Status, truncateForErr(body))
	}
	return body, nil
}

func truncateForErr(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

// FetchQRCodeSession 请求 generate 接口，获取用于扫码的 auth_url 与轮询用 scode。
func FetchQRCodeSession(ctx context.Context, cfg QRScanHTTPConfig) (*QRCodeSession, error) {
	client := cfg.httpClient()
	generateURL := cfg.GenerateRequestURL
	if generateURL == "" {
		generateURL = QRGenerateURL(cfg.source(), cfg.plat())
	}
	body, err := httpGetBody(ctx, client, generateURL)
	if err != nil {
		return nil, err
	}
	var parsed qrGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("aibot: decode generate response: %w", err)
	}
	if parsed.Data == nil || parsed.Data.Scode == "" || parsed.Data.AuthURL == "" {
		return nil, ErrQRGenerateBadResponse
	}
	return &QRCodeSession{
		Scode:   parsed.Data.Scode,
		AuthURL: parsed.Data.AuthURL,
	}, nil
}

// QRGenPageURLForSession 根据会话与当前 cfg 的 source 生成「浏览器扫码页」链接。
func QRGenPageURLForSession(cfg QRScanHTTPConfig, scode string) string {
	return qrGenPageURLWithSource(cfg.source(), scode)
}

// WaitForScanResult 轮询 query_result，直到 status=success 或超时 / context 取消。
func WaitForScanResult(ctx context.Context, cfg QRScanHTTPConfig, scode string) (*ScanBotCredentials, error) {
	if scode == "" {
		return nil, errors.New("aibot: empty scode")
	}
	client := cfg.httpClient()
	interval := cfg.pollInterval()
	deadline := time.Now().Add(cfg.pollTimeout())

	base := qrQueryPath
	if cfg.QueryURLBase != "" {
		base = cfg.QueryURLBase
	}
	queryURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	q := queryURL.Query()
	q.Set("scode", scode)
	queryURL.RawQuery = q.Encode()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, ErrQRScanTimeout
		}

		body, err := httpGetBody(ctx, client, queryURL.String())
		if err != nil {
			return nil, err
		}
		var parsed qrQueryResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("aibot: decode query_result response: %w", err)
		}
		if parsed.Data != nil && parsed.Data.Status == "success" {
			bi := parsed.Data.BotInfo
			if bi == nil || bi.Botid == "" || bi.Secret == "" {
				return nil, ErrQRScanNoBotInfo
			}
			return &ScanBotCredentials{BotID: bi.Botid, Secret: bi.Secret}, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// ScanQRCodeForBotInfo 完整流程：获取二维码会话 → 可选回调（展示二维码/链接）→ 轮询至扫码成功。
// 与 wecom-openclaw-cli 中 scanQRCodeForBotInfo 行为一致（不含终端 ASCII 二维码绘制）。
func ScanQRCodeForBotInfo(ctx context.Context, cfg QRScanHTTPConfig, onSession func(*QRCodeSession) error) (*ScanBotCredentials, error) {
	sess, err := FetchQRCodeSession(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if onSession != nil {
		if err := onSession(sess); err != nil {
			return nil, err
		}
	}
	return WaitForScanResult(ctx, cfg, sess.Scode)
}
