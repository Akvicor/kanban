package resp

// Code 是接口响应中的业务结果码。HTTP 状态码统一为 200，结果由 Code 区分。
// 前端 src/api/client.ts 按同样的取值处理。
type Code int

const (
	Succeeded       Code = 0 // 成功
	Failed          Code = 1 // 失败，通常是服务器内部错误
	NotFound        Code = 2 // 对象不存在，或不属于当前用户
	BadRequest      Code = 3 // 输入不合法
	UnAuthorized    Code = 4 // 未登录或登录已失效，前端收到后回到登录页
	Forbidden       Code = 5 // 无权执行该操作
	Conflict        Code = 6 // 与现有数据冲突
	TooManyRequests Code = 7 // 请求过于频繁
)

// Model 是接口的统一响应结构。
// Revision 是本次请求为当前用户产生的最大同步序号，没有产生变更时省略。
// 客户端用它把响应中的实体标记为已是该序号的状态，从而忽略推送中同一序号及更早的重复变更。
// 编辑冲突的失败响应中，Data 是对象的当前数据，Revision 是读取它时的同步序号。
type Model struct {
	Code     Code   `json:"code"`
	Msg      string `json:"msg,omitempty"`
	Data     any    `json:"data,omitempty"`
	Revision int64  `json:"revision,omitempty"`
}

// NewModel 创建统一响应结构。
func NewModel(code Code, msg string, data any) *Model {
	return &Model{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
