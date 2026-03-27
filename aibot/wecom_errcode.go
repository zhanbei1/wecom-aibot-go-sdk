package aibot

import "strings"

// ErrCodeWeComDocOffsetLimit 企业微信《全局错误码》表格中的 600041 释义为「无效的 offset 或 limit 参数值」。
// AI 机器人 WebSocket 认证在 secret 无效时也可能返回 errcode=600041，此时应以响应体中的 errmsg 为准。
const ErrCodeWeComDocOffsetLimit = 600041

// AuthFailureUserHint 根据认证接口返回的 errcode 与 errmsg 生成中文排查说明。
// 当表格释义与 errmsg 不一致时（例如 600041），以服务端 errmsg 为权威描述。
func AuthFailureUserHint(errcode int, serverErrMsg string) string {
	msgLower := strings.ToLower(strings.TrimSpace(serverErrMsg))

	switch errcode {
	case ErrCodeWeComDocOffsetLimit:
		if strings.Contains(msgLower, "secret") {
			return "invalid secret 表示 bot_secret 与当前 bot_id 不匹配、已过期或已被轮换，请从企业微信管理端或与 bot_id 同一来源重新复制密钥，勿混用应用 Secret。" +
				"全局错误码文档中同编号条目为分页参数相关，与机器人长连接认证无关，请忽略该释义。"
		}
		return "全局错误码文档中 600041 为 offset/limit 参数问题；若当前为机器人 WebSocket 认证，请以 errmsg 为准排查参数与凭证。"
	default:
		if strings.Contains(msgLower, "secret") {
			return "请确认 bot_id 与 bot_secret 来自同一机器人配置，复制时不要含首尾空格或换行；若刚重置过密钥需更新本地配置。"
		}
	}
	return ""
}
