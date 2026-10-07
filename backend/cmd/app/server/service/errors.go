package service

import (
	"kanban/cmd/app/server/common/passwd"
	"kanban/cmd/app/server/common/resp"
)

// Kind 是业务错误的类别，接口层按类别确定响应的协议级结果码（编辑冲突等）。
type Kind int

const (
	KindBadRequest   Kind = iota + 1 // 输入不合法
	KindUnauthorized                 // 未登录或登录已失效
	KindForbidden                    // 无权执行该操作
	KindNotFound                     // 对象不存在，或不属于当前用户
	KindConflict                     // 与现有数据冲突
)

// Error 是可以直接展示给用户的业务错误。其他错误都视为服务器内部错误，不向用户暴露细节。
// Code 是细粒度错误码（resp 中 1001 起），响应只下发码，前端按码取对应语言的文案；
// Msg 只用于日志和测试，不下发。Data 和 Revision 只在编辑冲突时设置：Data 是对象的当前数据，
// Revision 是读取它时用户的同步序号，客户端据此立即显示最新内容，并以该序号为基准重新提交。
type Error struct {
	Kind     Kind
	Code     resp.Code
	Msg      string
	Data     any
	Revision int64
}

func (e *Error) Error() string {
	return e.Msg
}

func newError(kind Kind, code resp.Code, msg string) error {
	return &Error{Kind: kind, Code: code, Msg: msg}
}

func badRequest(code resp.Code, msg string) error   { return newError(KindBadRequest, code, msg) }
func unauthorized(code resp.Code, msg string) error { return newError(KindUnauthorized, code, msg) }
func forbidden(code resp.Code, msg string) error    { return newError(KindForbidden, code, msg) }
func notFound(code resp.Code, msg string) error     { return newError(KindNotFound, code, msg) }
func conflict(code resp.Code, msg string) error     { return newError(KindConflict, code, msg) }

// editConflict 是带当前数据的编辑冲突，见 Error。响应码固定为 resp.Conflict。
func editConflict(msg string, data any, revision int64) error {
	return &Error{Kind: KindConflict, Code: resp.Conflict, Msg: msg, Data: data, Revision: revision}
}

// passwordInvalid 把密码校验错误转换为带错误码的业务错误。
func passwordInvalid(err error) error {
	switch err {
	case passwd.ErrCharset:
		return badRequest(resp.PasswordCharset, err.Error())
	case passwd.ErrTooShort:
		return badRequest(resp.PasswordTooShort, err.Error())
	default:
		return badRequest(resp.PasswordTooLong, err.Error())
	}
}
