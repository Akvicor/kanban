package api

import (
	"errors"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/service"

	"github.com/Akvicor/glog"
	"github.com/labstack/echo/v4"
)

// fail 把服务层错误转换为响应：只下发细粒度错误码，前端按码取文案；
// 编辑冲突另附对象当前数据和同步序号。非业务错误记日志，只返回「服务器错误」的码。
func fail(c echo.Context, err error) error {
	var businessError *service.Error
	if errors.As(err, &businessError) {
		if businessError.Data != nil {
			return resp.FailWithData(c, businessError.Code, "", businessError.Data, businessError.Revision)
		}
		return resp.Fail(c, businessError.Code)
	}
	glog.Error("%s %s: %v", c.Request().Method, c.Path(), err)
	return resp.Fail(c, resp.Failed)
}

// success 返回成功响应，并带上本次请求为当前用户产生的同步序号。需要登录的接口都用它返回。
func success(c echo.Context, data any) error {
	return resp.SuccessWithRevision(c, data, service.Revision(c.Request().Context()))
}

// bind 解析 JSON 请求体，格式错误时直接写出响应并返回 false。
func bind(c echo.Context, input any) bool {
	if err := c.Bind(input); err != nil {
		_ = resp.Fail(c, resp.MalformedRequest)
		return false
	}
	return true
}
