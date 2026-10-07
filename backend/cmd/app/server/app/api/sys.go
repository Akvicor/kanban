package api

import (
	"context"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/db"
	"time"

	"github.com/Akvicor/glog"
	"github.com/labstack/echo/v4"
)

// Sys 是系统信息接口。
var Sys = new(sysApi)

type sysApi struct{}

type healthCheck func(context.Context) error

// healthCheckTimeout 是一次健康检查中全部依赖检查的总时限。
const healthCheckTimeout = 2 * time.Second

// Health 检查服务依赖是否就绪，全部就绪返回 200，否则返回 503。
func (a *sysApi) Health(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), healthCheckTimeout)
	defer cancel()
	health := runHealthChecks(ctx, map[string]healthCheck{
		"database": db.Ping,
	})
	if health.Status == "healthy" {
		return resp.Healthy(c, health)
	}
	return resp.Unhealthy(c, health)
}

// runHealthChecks 执行全部检查，任一失败或超时则整体为 unhealthy。
func runHealthChecks(ctx context.Context, checks map[string]healthCheck) *dro.SysHealth {
	health := &dro.SysHealth{Status: "healthy", Checks: make(map[string]string, len(checks))}
	for name, check := range checks {
		if err := check(ctx); err != nil || ctx.Err() != nil {
			health.Status = "unhealthy"
			health.Checks[name] = "unhealthy"
			if err == nil {
				err = ctx.Err()
			}
			glog.Warning("健康检查失败 [%s]: %v", name, err)
			continue
		}
		health.Checks[name] = "healthy"
	}
	return health
}
