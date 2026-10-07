// Package gmsgsend 按 gmsg 的 SendBySign 协议，用一条渠道自己的 API、Token 和 Sign 发送一条消息。
//
// 协议（与 /kayuki/mvq/workspace/sync/milter 一致）：POST <API>/api/send，请求头 X-Access-Token 为 Token，
// 请求体 {"id": 0, "sign": Sign, "type": 内容格式, "at": 秒级时间戳, "title": 标题, "msg": 正文}；
// 响应 {"code": 0, "data": 消息 ID}，code 不为 0 或消息 ID 为 0 视为失败。
// gmsg 库自带的发送函数使用进程级全局的 API 和 Token，不能区分渠道，因此这里自行实现请求。
package gmsgsend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Akvicor/gmsg/gmodel"
)

const (
	// maxResponseBytes 是读取响应的上限。
	maxResponseBytes = 4 << 10
	// MaxErrorLength 是错误信息的最大字符数。
	MaxErrorLength = 500
)

// Channel 是发送使用的一条渠道的连接信息。
type Channel struct {
	API    string
	Token  string
	Sign   string
	Format gmodel.Type
}

// Message 是一条要发送的消息。
type Message struct {
	Title string
	Body  string
	At    time.Time
}

// ErrInvalidAPI 表示渠道 API 不是可用的 gmsg 服务地址。
var ErrInvalidAPI = errors.New("API 必须是 http:// 或 https:// 开头、不带查询参数和片段的地址")

// CheckAPI 校验渠道 API：必须是带主机的 http、https 地址，且不含查询参数和片段（? 与 #），
// 保证实际请求的路径固定为 <API>/api/send。地址可以是内网地址：gmsg 服务可以部署在内网。
func CheckAPI(api string) error {
	parsed, err := url.Parse(api)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(api, "?#") {
		return ErrInvalidAPI
	}
	return nil
}

// NewClient 返回发送 gmsg 请求的客户端，每次请求最多 timeout。
// 客户端不跟随重定向，3xx 按失败处理，使请求只发往渠道配置的地址。
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Send 发送一条消息。返回的错误信息已遮掉 Token 和 Sign，并且不超过 MaxErrorLength 个字符。
// 错误信息只包含状态码和 gmsg 响应中的错误码、msg 字段，对端响应正文不会出现在错误信息中。
func Send(ctx context.Context, client *http.Client, channel Channel, message Message) error {
	err := send(ctx, client, channel, message)
	if err == nil {
		return nil
	}
	return errors.New(clean(err.Error(), channel.Token, channel.Sign))
}

func send(ctx context.Context, client *http.Client, channel Channel, message Message) error {
	if err := CheckAPI(channel.API); err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"id": int64(0), "sign": channel.Sign, "type": channel.Format, "at": message.At.Unix(), "title": message.Title, "msg": message.Body,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(channel.API, "/")+"/api/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-Access-Token", channel.Token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求 gmsg 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gmsg 返回 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("读取 gmsg 响应失败: %w", err)
	}
	var result struct {
		Code gmodel.RespCode `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err = json.Unmarshal(data, &result); err != nil {
		return errors.New("gmsg 响应格式不正确")
	}
	if result.Code != gmodel.RespCodeSucceeded {
		return fmt.Errorf("gmsg 返回错误码 %d: %s", result.Code, result.Msg)
	}
	var id int64
	if err = json.Unmarshal(result.Data, &id); err != nil || id == 0 {
		return errors.New("gmsg 返回的消息 ID 无效")
	}
	return nil
}

// clean 遮掉错误信息中的 Token 和 Sign，替换无效的 UTF-8，并截断到 MaxErrorLength 个字符。
func clean(text string, secrets ...string) string {
	text = strings.ToValidUTF8(text, "?")
	for _, secret := range secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[已遮盖]")
		}
	}
	if utf8.RuneCountInString(text) > MaxErrorLength {
		text = string([]rune(text)[:MaxErrorLength-1]) + "…"
	}
	return text
}
