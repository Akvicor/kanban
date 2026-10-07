package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// HeaderPrefix 是 Authorization 请求头中设备令牌的前缀。
const HeaderPrefix = "Bearer "

// New 生成一个新的设备令牌明文。令牌明文只在登录响应中返回一次，库中只保存哈希。
func New() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// Hash 返回令牌明文的 sha256 十六进制摘要，用于入库和查找。
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// FromHeader 从 Authorization 请求头取出令牌明文，格式不对时返回空串。
func FromHeader(header string) string {
	if !strings.HasPrefix(header, HeaderPrefix) {
		return ""
	}
	return strings.TrimSpace(header[len(HeaderPrefix):])
}
