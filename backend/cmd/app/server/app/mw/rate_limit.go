package mw

import (
	"kanban/cmd/app/server/common/resp"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// 登录接口按客户端 IP 限流：平均每分钟 10 次，最多连续 10 次，用来减缓密码猜测。
const (
	loginRatePerMinute = 10
	loginBurst         = 10
	loginLimiterExpiry = 10 * time.Minute
)

// LoginRateLimit 返回登录接口的限流中间件。超过限制时返回「请求过于频繁」。
func LoginRateLimit() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      rate.Limit(loginRatePerMinute / 60.0),
			Burst:     loginBurst,
			ExpiresIn: loginLimiterExpiry,
		}),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			return resp.Fail(c, resp.TooManyLoginAttempt)
		},
		ErrorHandler: func(c echo.Context, _ error) error {
			return resp.Fail(c, resp.TooManyLoginAttempt)
		},
	})
}
