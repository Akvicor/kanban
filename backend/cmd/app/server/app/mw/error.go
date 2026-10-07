package mw

import (
	"errors"
	"kanban/cmd/app/server/common/resp"

	"github.com/labstack/echo/v4"
)

// Error 把处理函数返回的错误转换为统一响应。echo.HTTPError 的信息原样返回，其他错误只返回「服务器错误」。
func Error(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := next(c); err != nil {
			var he *echo.HTTPError
			if errors.As(err, &he) {
				return resp.Fail(c, resp.Failed)
			}
			return resp.Fail(c, resp.Failed)
		}
		return nil
	}
}
