package aibot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// generateReqId 生成请求 ID
// 格式: {cmd}_{timestamp}_{random}
func generateReqId(cmd string) string {
	now := time.Now().UnixNano()
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	randHex := hex.EncodeToString(randBytes)
	return fmt.Sprintf("%s_%d_%s", cmd, now, randHex)
}

// GenerateReqId 生成请求 ID（公开方法）
func GenerateReqId(cmd string) string {
	return generateReqId(cmd)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// GenerateRandomString 生成随机字符串（公开方法）
func GenerateRandomString(length int) string {
	return generateRandomString(length)
}
