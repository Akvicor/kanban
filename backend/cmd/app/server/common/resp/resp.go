package resp

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Healthy 以 200 返回健康检查结果。
func Healthy(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, data)
}

// Unhealthy 以 503 返回健康检查结果，供 Docker 健康检查识别。
func Unhealthy(c echo.Context, data any) error {
	return c.JSON(http.StatusServiceUnavailable, data)
}

// Success 返回不带数据的成功响应。
func Success(c echo.Context) error {
	return c.JSON(http.StatusOK, NewModel(Succeeded, "", nil))
}

// SuccessWithData 返回带数据的成功响应。
func SuccessWithData(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, NewModel(Succeeded, "", data))
}

// SuccessWithRevision 返回带数据和同步序号的成功响应，revision 为 0 时省略。
func SuccessWithRevision(c echo.Context, data any, revision int64) error {
	model := NewModel(Succeeded, "", data)
	model.Revision = revision
	return c.JSON(http.StatusOK, model)
}

// FailWithData 返回带错误信息、数据和同步序号的失败响应，用于编辑冲突时附带对象的当前数据。
func FailWithData(c echo.Context, code Code, msg string, data any, revision int64) error {
	model := NewModel(code, msg, data)
	model.Revision = revision
	return c.JSON(http.StatusOK, model)
}

// Fail 返回只带错误码的失败响应。业务错误都用它返回，文案由前端按错误码取对应语言的版本。
func Fail(c echo.Context, code Code) error {
	return c.JSON(http.StatusOK, NewModel(code, "", nil))
}
