// 个人微信 iLink Bot：扫码登录与长轮询收消息等能力由 [github.com/openilink/openilink-sdk-go] 提供。
// 与本书签模块内的企业微信智能机器人 [WSClient]（wss://openws.work.weixin.qq.com）为两套体系，
// 凭据与协议互不通用：企业侧使用 BotID + Secret；个人侧登录成功后使用 BotToken 等 iLink 会话信息。
//
// [github.com/openilink/openilink-sdk-go]: https://github.com/openilink/openilink-sdk-go
package aibot

import (
	"context"

	ilink "github.com/openilink/openilink-sdk-go"
)

// iLink（个人微信）客户端与登录回调、选项类型别名，便于只依赖本模块即可完成扫码登录。
type (
	ILinkClient         = ilink.Client
	ILinkLoginCallbacks = ilink.LoginCallbacks
	ILinkLoginResult    = ilink.LoginResult
	ILinkOption         = ilink.Option
	ILinkWeixinMessage  = ilink.WeixinMessage
	ILinkMonitorOptions = ilink.MonitorOptions
)

// iLink 默认端点与扫码用 bot_type（与 openilink-sdk-go 一致）。
const (
	ILinkDefaultBaseURL    = ilink.DefaultBaseURL
	ILinkDefaultCDNBaseURL = ilink.DefaultCDNBaseURL
	ILinkDefaultBotType    = ilink.DefaultBotType
)

// NewILinkClient 创建个人微信 iLink API 客户端。token 为空表示未登录，可在 [ILinkClient.LoginWithQR] 成功后使用 [ILinkClient.Token]。
func NewILinkClient(token string, opts ...ILinkOption) *ILinkClient {
	return ilink.NewClient(token, opts...)
}

// ILinkWithBaseURL 覆盖 API Base URL（如自建网关或测试环境）。
func ILinkWithBaseURL(url string) ILinkOption { return ilink.WithBaseURL(url) }

// ILinkWithCDNBaseURL 覆盖 CDN Base URL（媒体上传下载）。
func ILinkWithCDNBaseURL(url string) ILinkOption { return ilink.WithCDNBaseURL(url) }

// ILinkWithHTTPClient 指定 HTTP 客户端。
func ILinkWithHTTPClient(doer ilink.HTTPDoer) ILinkOption { return ilink.WithHTTPClient(doer) }

// ILinkWithBotType 设置扫码登录时的 bot_type 查询参数（默认 [ILinkDefaultBotType]）。
func ILinkWithBotType(t string) ILinkOption { return ilink.WithBotType(t) }

// ILinkWithRouteTag 设置 SKRouteTag 请求头。
func ILinkWithRouteTag(tag string) ILinkOption { return ilink.WithRouteTag(tag) }

// ILinkWithVersion 设置 channel_version（base_info）。
func ILinkWithVersion(v string) ILinkOption { return ilink.WithVersion(v) }

// ILinkExtractText 从 iLink 消息体提取文本（含引用前缀、语音转文字后备），与 openilink-sdk-go 的 ExtractText 行为一致。
func ILinkExtractText(msg *ILinkWeixinMessage) string {
	return ilink.ExtractText(msg)
}

// PersonalWeChatLoginWithQR 使用个人微信扫码完成 iLink 登录。
// 成功后 result.Connected 为 true，且 client 已写入 Token；可继续 [ILinkClient.Monitor]、[ILinkClient.Push] 等。
// 若 result.Connected 为 false 且 err 为 nil，表示超时或用户侧未完成确认（见 result.Message）。
func PersonalWeChatLoginWithQR(ctx context.Context, client *ILinkClient, cb *ILinkLoginCallbacks) (*ILinkLoginResult, error) {
	return client.LoginWithQR(ctx, cb)
}
