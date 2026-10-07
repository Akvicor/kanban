// Package locale 定义界面与通知使用的语言。
package locale

import "strings"

// Type 是语言标记，取值与前端语言包的目录名一致。
// 空值表示「跟随系统」：界面由前端按浏览器语言解析，后端生成文案（通知默认模板等）时回落 Default。
type Type string

const (
	// ZhCN 简体中文。
	ZhCN Type = "zh-CN"
	// En 英文。
	En Type = "en"
	// Default 是语言未设置或不受支持时后端使用的语言。
	Default = ZhCN
)

// Valid 判断语言是否受支持；空值表示跟随系统，允许保存。
func (t Type) Valid() bool {
	return t == "" || t == ZhCN || t == En
}

// Tag 返回用于取文案的语言标记，未设置或不受支持时回落默认语言。
// 后台任务等没有请求信息的场合用它；接口请求用 Effective 按浏览器语言解析。
func (t Type) Tag() string {
	if t == En {
		return string(En)
	}
	return string(Default)
}

// Effective 返回用于生成默认文案的语言：显式设置优先；未设置时按 Accept-Language 解析，
// 与前端 resolveLocale 的规则一致（先找 zh 再找 en，都不匹配用英文）；
// 请求没有 Accept-Language 时回落默认语言，保持既有行为。
func Effective(setting Type, acceptLanguage string) Type {
	if setting.Valid() && setting != "" {
		return setting
	}
	if acceptLanguage == "" {
		return Default
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		if strings.HasPrefix(tag, "zh") {
			return ZhCN
		}
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		if strings.HasPrefix(tag, "en") {
			return En
		}
	}
	return En
}
