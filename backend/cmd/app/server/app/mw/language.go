package mw

import (
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// Language 把请求的 Accept-Language 放进上下文，供服务层为「跟随系统」的用户解析默认文案语言。
func Language(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		request := c.Request().WithContext(service.WithAcceptLanguage(c.Request().Context(), c.Request().Header.Get("Accept-Language")))
		c.SetRequest(request)
		return next(c)
	}
}
