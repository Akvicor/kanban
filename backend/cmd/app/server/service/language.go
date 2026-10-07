package service

import (
	"context"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/model"
)

// acceptLanguageKey 是请求的 Accept-Language 头在上下文中的键。
// mw.Language 把它放进请求上下文，服务层据此为「跟随系统」的用户解析默认文案语言。
type acceptLanguageKey struct{}

// WithAcceptLanguage 把请求的 Accept-Language 放进上下文。
func WithAcceptLanguage(ctx context.Context, header string) context.Context {
	return context.WithValue(ctx, acceptLanguageKey{}, header)
}

// uiLang 返回生成默认文案使用的语言：显式设置优先；未设置时按请求的浏览器语言解析，
// 与界面显示保持一致。后台任务等没有请求的场合取不到浏览器语言，见 locale.Type.Tag。
func uiLang(ctx context.Context, user *model.User) string {
	header, _ := ctx.Value(acceptLanguageKey{}).(string)
	return string(locale.Effective(user.Locale, header))
}
