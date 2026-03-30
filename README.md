# wecom-aibot-go-sdk

企业微信智能机器人 Go SDK —— 基于 WebSocket 长连接通道，提供消息收发、流式回复、模板卡片、事件回调、文件下载解密等核心能力。

> **关于本项目**: 此项目是官方 [wecomteam/aibot-node-sdk](https://github.com/WecomTeam/aibot-node-sdk) 的 Go 语言复刻版本，会保持与官方 Node.js SDK 同步更新。

## 特性

- **WebSocket 长连接** — 基于 `wss://openws.work.weixin.qq.com` 内置默认地址，开箱即用
- **自动认证** — 连接建立后自动发送认证帧（botId + secret）
- **心跳保活** — 自动维护心跳，连续未收到 ack 时自动判定连接异常
- **断线重连** — 指数退避重连策略（1s → 2s → 4s → ... → 30s 上限），支持自定义最大重连次数
- **消息分发** — 自动解析消息类型并触发对应事件（text / image / mixed / voice / file）
- **流式回复** — 内置流式回复方法，支持 Markdown 和图文混排
- **模板卡片** — 支持回复模板卡片消息、流式+卡片组合回复、更新卡片
- **主动推送** — 支持向指定会话主动发送 Markdown 或模板卡片消息，无需依赖回调帧
- **事件回调** — 支持进入会话、模板卡片按钮点击、用户反馈等事件
- **串行回复队列** — 同一 req_id 的回复消息串行发送，自动等待回执
- **文件下载解密** — 内置 AES-256-CBC 文件解密，每个图片/文件消息自带独立的 aeskey
- **可插拔日志** — 支持自定义 Logger，内置带时间戳的 DefaultLogger
- **扫码创建机器人** — 调用企业微信 `generate` / `query_result` 接口，用扫码方式获取 `BotID` 与 `Secret`（与 [wecom-openclaw-cli](wecom-openclaw-cli/) 中二维码流程一致）
- **个人微信扫码登录（iLink）** — 集成 [openilink-sdk-go](https://github.com/openilink/openilink-sdk-go)，通过 `ilink` 协议完成个微扫码登录与长轮询收消息等能力（与本仓库 `WSClient` 企业通道独立）

## 安装

```bash
go get github.com/go-sphere/wecom-aibot-go-sdk
```

## 快速开始

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	aibot "github.com/go-sphere/wecom-aibot-go-sdk/aibot"
)

func main() {
	// 从环境变量获取配置
	botID := os.Getenv("WECOM_BOT_ID")
	botSecret := os.Getenv("WECOM_BOT_SECRET")

	if botID == "" || botSecret == "" {
		fmt.Println("请设置环境变量 WECOM_BOT_ID 和 WECOM_BOT_SECRET")
		return
	}

	// 创建客户端
	client := aibot.NewWSClient(aibot.WSClientOptions{
		BotID:  botID,
		Secret: botSecret,
	})

	// 连接客户端
	client.Connect()

	// 设置事件处理
	client.OnConnected(func() {
		fmt.Println("连接已建立")
	})

	client.OnAuthenticated(func() {
		fmt.Println("认证成功")
	})

	client.OnDisconnected(func(reason string) {
		fmt.Println("连接断开:", reason)
	})

	// 监听文本消息并进行流式回复
	client.OnMessageText(func(frame *aibot.WsFrame) {
		// 解析消息体
		var msg aibot.TextMessage
		if err := aibot.ParseMessageBody(frame, &msg); err != nil {
			fmt.Println("解析消息失败:", err.Error())
			return
		}

		fmt.Printf("收到文本: %s\n", msg.Text.Content)

		streamID := fmt.Sprintf("stream_%d", frame.Headers.ReqID)
		reply := "你好！你说的是: " + msg.Text.Content

		// 发送流式回复
		client.ReplyStream(frame, streamID, reply, true, nil, nil)
	})

	// 监听进入会话事件（发送欢迎语）
	client.OnEventEnterChat(func(frame *aibot.WsFrame) {
		welcomeBody := aibot.CreateTextReplyBody("您好！我是智能助手，有什么可以帮您的吗？")
		client.ReplyWelcome(frame, welcomeBody)
	})

	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	client.Disconnect()
}
```

## 配置选项

`WSClientOptions` 完整配置：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `BotID` | `string` | ✅ | — | 机器人 ID（企业微信后台获取） |
| `Secret` | `string` | ✅ | — | 机器人 Secret（企业微信后台获取） |
| `ReconnectInterval` | `int` | — | `1000` | 重连基础延迟（毫秒），实际延迟按指数退避递增（1s → 2s → 4s → ... → 30s 上限） |
| `MaxReconnectAttempts` | `int` | — | `10` | 最大重连次数（`-1` 表示无限重连） |
| `HeartbeatInterval` | `int` | — | `30000` | 心跳间隔（毫秒） |
| `RequestTimeout` | `int` | — | `10000` | HTTP 请求超时时间（毫秒） |
| `WSURL` | `string` | — | `wss://openws.work.weixin.qq.com` | 自定义 WebSocket 连接地址 |
| `Logger` | `Logger` | — | `DefaultLogger` | 自定义日志实例 |

## 扫码创建机器人（获取 BotID / Secret）

企业微信提供与企业微信客户端扫码绑定的创建流程：先请求 `https://work.weixin.qq.com/ai/qc/generate` 拿到 `auth_url`（用于生成二维码）和 `scode`，再轮询 `https://work.weixin.qq.com/ai/qc/query_result?scode=...`，直到 `data.status == "success"` 时从 `data.bot_info` 读取 `botid` 与 `secret`。默认 `source=wecom-cli`、`plat` 随操作系统映射（与 `wecom-openclaw-cli/dist/utils/qrcode.js` 一致）。

SDK 提供以下方法（均在包 `aibot` 内）：

| 函数 / 类型 | 说明 |
| --- | --- |
| `FetchQRCodeSession(ctx, cfg)` | 请求 generate，返回 `QRCodeSession`（`Scode`、`AuthURL`） |
| `WaitForScanResult(ctx, cfg, scode)` | 轮询 query_result，成功则返回 `ScanBotCredentials`（`BotID`、`Secret`） |
| `ScanQRCodeForBotInfo(ctx, cfg, onSession)` | 组合：拉取会话 → 可选回调展示二维码/链接 → 轮询至成功 |
| `QRGenPageURL(scode)` | 浏览器打开的扫码页 URL（等价于 CLI 中打印的 `.../ai/qc/gen?source=...&scode=...`） |
| `QRGenPageURLForSession(cfg, scode)` | 使用 `cfg.Source` 构造同上扫码页 |
| `QRGenerateURL(source, plat)` | 构造带 `source`、`plat` 的 generate 请求 URL |
| `QRPlatformFromGOOS()` | 将 `runtime.GOOS` 映射为接口要求的 `plat` |
| `QRScanHTTPConfig` | 可配置 `Client`、`Source`、`Plat`、轮询间隔（默认 3s）、超时（默认 5 分钟）；`GenerateRequestURL` / `QueryURLBase` 可用于测试或自定义网关 |

错误：`ErrQRGenerateBadResponse`、`ErrQRScanNoBotInfo`、`ErrQRScanTimeout`；`context` 取消时返回 `ctx.Err()`。

示例（两步：先展示链接或二维码内容，再等待扫码）：

```go
ctx := context.Background()
cfg := aibot.QRScanHTTPConfig{} // 可按需设置 Source、PollInterval、PollTimeout 等

sess, err := aibot.FetchQRCodeSession(ctx, cfg)
if err != nil {
	log.Fatal(err)
}
fmt.Println("请使用企业微信扫码，或浏览器打开:", aibot.QRGenPageURLForSession(cfg, sess.Scode))
fmt.Println("二维码内容 URL:", sess.AuthURL)

creds, err := aibot.WaitForScanResult(ctx, cfg, sess.Scode)
if err != nil {
	log.Fatal(err)
}
fmt.Println("BotID:", creds.BotID, "Secret:", creds.Secret)

client := aibot.NewWSClient(aibot.WSClientOptions{
	BotID:  creds.BotID,
	Secret: creds.Secret,
})
```

或使用一站式回调（在回调里把 `sess.AuthURL` 转成二维码图片展示等）：

```go
creds, err := aibot.ScanQRCodeForBotInfo(ctx, cfg, func(s *aibot.QRCodeSession) error {
	fmt.Println("打开链接扫码:", aibot.QRGenPageURLForSession(cfg, s.Scode))
	return nil
})
```

说明：本 SDK **不包含** 终端 ASCII 二维码绘制；若需在终端展示，可自行引入二维码库，以 `sess.AuthURL` 为内容生成图像或终端图案。

## 个人微信扫码登录（iLink / OpeniLink）

本仓库通过依赖 [openilink-sdk-go](https://github.com/openilink/openilink-sdk-go) 支持**个人微信**侧 iLink Bot 的扫码登录与后续 API（长轮询收消息、Push 等）。完整能力、超时与状态码说明见上游 [README](https://github.com/openilink/openilink-sdk-go/blob/main/README.md)。

与企业微信智能机器人 **不是同一条链路**：

| 场景 | 客户端 | 凭据 / 通道 |
| --- | --- | --- |
| 企业微信智能机器人 | `aibot.NewWSClient` | `BotID` + `Secret`，WebSocket `wss://openws.work.weixin.qq.com` |
| 个人微信（iLink） | `aibot.NewILinkClient` | 扫码成功后 `BotToken`、`BotID`（iLink）等，HTTPS `ilink/bot/...` |

`aibot` 内提供的封装如下（类型多为上游别名，便于单模块 import）：

| 符号 | 说明 |
| --- | --- |
| `NewILinkClient(token, opts...)` | 创建 iLink 客户端；未登录时 `token` 传空字符串 |
| `PersonalWeChatLoginWithQR(ctx, client, cb)` | 执行完整扫码流程，等价于 `client.LoginWithQR` |
| `ILinkLoginCallbacks` | `OnQRCode` / `OnScanned` / `OnExpired` 回调 |
| `ILinkLoginResult` | `Connected`、`BotID`、`BotToken`、`UserID`、`BaseURL`、`Message` |
| `ILinkWithBaseURL` / `ILinkWithHTTPClient` / `ILinkWithBotType` 等 | 与上游 `ilink.WithXxx` 一致 |
| `ILinkDefaultBaseURL` 等常量 | 默认指向 `https://ilinkai.weixin.qq.com` 等 |
| `ILinkExtractText` | 从 `ILinkWeixinMessage` 提取文本 |
| `ILinkClient` | 登录成功后调用 `Monitor`、`Push` 等方法（定义在上游包） |

示例：扫码登录并进入监听（与上游 [example/echo-bot](https://github.com/openilink/openilink-sdk-go/tree/main/example/echo-bot) 思路一致；`OnQRCode` 的字符串可按上游 README 渲染为二维码图片）：

```go
import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	aibot "github.com/go-sphere/wecom-aibot-go-sdk/aibot"
)

ctx := context.Background()
client := aibot.NewILinkClient("")

result, err := aibot.PersonalWeChatLoginWithQR(ctx, client, &aibot.ILinkLoginCallbacks{
	OnQRCode: func(imgContent string) {
		fmt.Println("请使用微信扫描（或将内容渲染为二维码）:", imgContent)
	},
	OnScanned: func() { fmt.Println("已扫码，请在手机上确认登录") },
	OnExpired: func(attempt, max int) { fmt.Printf("二维码过期，正在刷新 (%d/%d)\n", attempt, max) },
})
if err != nil {
	log.Fatal(err)
}
if !result.Connected {
	log.Fatalf("登录未完成: %s", result.Message)
}

listenCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer cancel()

_ = client.Monitor(listenCtx, func(msg aibot.ILinkWeixinMessage) {
	text := aibot.ILinkExtractText(&msg)
	if text == "" {
		return
	}
	if _, e := client.Push(listenCtx, msg.FromUserID, "echo: "+text); e != nil {
		log.Printf("回复失败: %v", e)
	}
}, &aibot.ILinkMonitorOptions{})
```

`Monitor`、`Push`、`ILinkMonitorOptions`（如 `InitialBuf`、`OnBufUpdate` 同步缓冲持久化）等行为以 [openilink-sdk-go](https://github.com/openilink/openilink-sdk-go) 为准；本仓库不重复实现 iLink 协议。

## API 文档

### WSClient

核心客户端类，提供连接管理、消息收发等功能。

#### 构造函数

```go
client := aibot.NewWSClient(options aibot.WSClientOptions)
```

#### 方法

| 方法 | 说明 | 返回值 |
| --- | --- | --- |
| `Connect()` | 建立 WebSocket 连接，连接后自动认证 | `*WSClient` |
| `Disconnect()` | 主动断开连接 | `void` |
| `IsConnected()` | 获取当前 WebSocket 连接状态 | `bool` |
| `Reply(frame, body, cmd)` | 通过 WebSocket 通道发送回复消息（通用方法） | `(*WsFrame, error)` |
| `ReplyStream(frame, streamID, content, finish, msgItem, feedback)` | 发送流式文本回复（便捷方法，支持 Markdown） | `(*WsFrame, error)` |
| `ReplyWelcome(frame, body)` | 发送欢迎语回复（支持文本或模板卡片），需在收到事件 5s 内调用 | `(*WsFrame, error)` |
| `ReplyTemplateCard(frame, templateCard, feedback)` | 回复模板卡片消息 | `(*WsFrame, error)` |
| `ReplyStreamWithCard(frame, streamID, content, finish, options)` | 发送流式消息 + 模板卡片组合回复 | `(*WsFrame, error)` |
| `UpdateTemplateCard(frame, templateCard, userIDs)` | 更新模板卡片（响应 template_card_event），需在收到事件 5s 内调用 | `(*WsFrame, error)` |
| `SendMessage(chatID, body)` | 主动发送消息（支持 Markdown 或模板卡片），无需依赖回调帧 | `(*WsFrame, error)` |
| `SendMarkdown(chatID, content)` | 主动发送 Markdown 消息 | `(*WsFrame, error)` |
| `SendTemplateCard(chatID, templateCard)` | 主动发送模板卡片消息 | `(*WsFrame, error)` |
| `DownloadFile(fileURL, aesKey)` | 下载文件并使用 AES 密钥解密，返回文件内容和文件名 | `([]byte, string, error)` |

### ReplyStream 详细说明

```go
client.ReplyStream(
	frame *aibot.WsFrame,      // 收到的原始 WebSocket 帧
	streamID string,            // 流式消息 ID（使用 aibot.GenerateReqId("stream") 生成）
	content string,             // 回复内容（支持 Markdown）
	finish bool,                // 是否结束流式消息，默认 false
	msgItem []aibot.ReplyMsgItem, // 图文混排项（仅 finish=true 时有效）
	feedback *aibot.ReplyFeedback, // 反馈信息（仅首次回复时设置）
)
```

### ReplyWelcome 详细说明

发送欢迎语回复，需在收到 `enter_chat` 事件 5 秒内调用，超时将无法发送。

```go
// 文本欢迎语
welcomeBody := aibot.CreateTextReplyBody("欢迎！")
client.ReplyWelcome(frame, welcomeBody)

// 模板卡片欢迎语
// 使用 aibot.TemplateCard 构建模板卡片
```

### SendMessage 详细说明

主动向指定会话推送消息，无需依赖收到的回调帧。

```go
// 发送 Markdown 消息
client.SendMarkdown("userid_or_chatid", "这是一条**主动推送**的消息")

// 发送模板卡片消息
templateCard := aibot.TemplateCard{
	CardType: "text_notice",
	MainTitle: &aibot.TemplateCardMainTitle{
		Title: "通知",
	},
}
client.SendTemplateCard("userid_or_chatid", templateCard)
```

### DownloadFile 使用示例

```go
// aesKey 取自消息体中的 Image.AesKey 或 File.AesKey
client.OnMessageImage(func(frame *aibot.WsFrame) {
	var msg aibot.ImageMessage
	aibot.ParseMessageBody(frame, &msg)

	data, filename, err := client.DownloadFile(msg.Image.URL, msg.Image.AesKey)
	if err != nil {
		fmt.Println("下载失败:", err)
		return
	}
	fmt.Printf("文件名: %s, 大小: %d bytes\n", filename, len(data))
})
```

## 事件列表

所有事件均通过 `client.OnXxx(handler)` 设置：

| 方法 | 回调参数 | 说明 |
| --- | --- | --- |
| `OnConnected` | — | WebSocket 连接建立 |
| `OnAuthenticated` | — | 认证成功 |
| `OnDisconnected` | `reason: string` | 连接断开 |
| `OnReconnecting` | `attempt: int` | 正在重连（第 N 次） |
| `OnError` | `err: error` | 发生错误 |
| `OnMessage` | `frame: *WsFrame` | 收到消息（所有类型） |
| `OnMessageText` | `frame: *WsFrame` | 收到文本消息 |
| `OnMessageImage` | `frame: *WsFrame` | 收到图片消息 |
| `OnMessageMixed` | `frame: *WsFrame` | 收到图文混排消息 |
| `OnMessageVoice` | `frame: *WsFrame` | 收到语音消息 |
| `OnMessageFile` | `frame: *WsFrame` | 收到文件消息 |
| `OnEvent` | `frame: *WsFrame` | 收到事件回调（所有事件类型） |
| `OnEventEnterChat` | `frame: *WsFrame` | 收到进入会话事件（用户当天首次进入单聊会话） |
| `OnEventTemplateCardEvent` | `frame: *WsFrame` | 收到模板卡片事件（用户点击卡片按钮） |
| `OnEventFeedbackEvent` | `frame: *WsFrame` | 收到用户反馈事件 |

## 消息类型

SDK 支持以下消息类型（`MessageType` 常量）：

| 类型 | 值 | 说明 |
| --- | --- | --- |
| `MessageTypeText` | `"text"` | 文本消息 |
| `MessageTypeImage` | `"image"` | 图片消息（URL 已加密，使用消息中的 `Image.AesKey` 解密） |
| `MessageTypeMixed` | `"mixed"` | 图文混排消息（包含 text / image 子项） |
| `MessageTypeVoice` | `"voice"` | 语音消息（已转文本） |
| `MessageTypeFile` | `"file"` | 文件消息（URL 已加密，使用消息中的 `File.AesKey` 解密） |

SDK 支持以下事件类型（`EventType` 常量）：

| 类型 | 值 | 说明 |
| --- | --- | --- |
| `EventTypeEnterChat` | `"enter_chat"` | 进入会话事件：用户当天首次进入机器人单聊会话 |
| `EventTypeTemplateCardEvent` | `"template_card_event"` | 模板卡片事件：用户点击模板卡片按钮 |
| `EventTypeFeedbackEvent` | `"feedback_event"` | 用户反馈事件：用户对机器人回复进行反馈 |

### 消息帧结构（`WsFrame`）

```go
type WsFrame struct {
	Cmd     string          // 命令类型
	Headers WsFrameHeaders // 请求头信息
	Body    json.RawMessage // 消息体
	ErrCode int            // 响应错误码
	ErrMsg  string         // 响应错误信息
}

type WsFrameHeaders struct {
	ReqID string // 请求 ID（回复时需透传）
}
```

## 自定义日志

实现 `Logger` 接口即可自定义日志输出：

```go
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}
```

使用示例：

```go
type CustomLogger struct{}

func (l *CustomLogger) Debug(format string, v ...interface{}) { /* ... */ }
func (l *CustomLogger) Info(format string, v ...interface{})  { /* ... */ }
func (l *CustomLogger) Warn(format string, v ...interface{})  { /* ... */ }
func (l *CustomLogger) Error(format string, v ...interface{}) { /* ... */ }

client := aibot.NewWSClient(aibot.WSClientOptions{
	BotID:  "your-bot-id",
	Secret: "your-bot-secret",
	Logger: &CustomLogger{},
})
```

## 工具函数

| 函数 | 说明 |
| --- | --- |
| `GetMsgID(frame)` | 从 frame 中提取 msgid |
| `GetReqID(frame)` | 从 frame 中提取 req_id |
| `GetMsgType(frame)` | 从 frame 中提取 msgtype |
| `GetEventType(frame)` | 从 frame 中提取 eventtype |
| `ParseMessageBody(frame, v)` | 解析消息体为指定类型 |
| `GenerateReqId(cmd)` | 生成请求 ID |
| `CreateTextReplyBody(content)` | 创建文本回复消息体 |
| `CreateMarkdownReplyBody(content)` | 创建 Markdown 回复消息体 |
| `CreateWelcomeReplyBody(content)` | 创建欢迎语回复消息体 |
| `CreateStreamReplyBody(streamID, content, finish, msgItem, feedback)` | 创建流式回复消息体 |

个人微信 iLink：`NewILinkClient`、`PersonalWeChatLoginWithQR`、`ILinkExtractText` 及选项函数见上文「个人微信扫码登录（iLink / OpeniLink）」。

## 项目结构

```
wecom-aibot-go-sdk/
├── aibot/
│   ├── client.go           # WSClient 核心客户端
│   ├── ws.go               # WebSocket 长连接管理器
│   ├── message_handler.go  # 消息解析与事件分发
│   ├── api.go              # HTTP API 客户端（文件下载）
│   ├── crypto.go           # AES-256-CBC 文件解密
│   ├── qrcode_scan.go      # 企业微信扫码创建机器人（generate / query_result）
│   ├── openilink.go        # 个人微信 iLink：封装 openilink-sdk-go（扫码登录等）
│   ├── logger.go           # 日志接口与默认实现
│   ├── utils.go            # 工具方法
│   ├── types.go            # 类型定义
├── examples/
│   └── basic/main.go       # 基础使用示例
├── go.mod
└── README.md
```

## 运行示例

```bash
# 克隆项目
git clone https://github.com/go-sphere/wecom-aibot-go-sdk.git
cd wecom-aibot-go-sdk

# 运行示例
WECOM_BOT_ID=your-bot-id WECOM_BOT_SECRET=your-bot-secret go run examples/basic/main.go
```

## License

MIT
