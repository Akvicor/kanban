package role

// Type 是用户角色。管理员管理账号，普通用户使用看板；两者的数据都按用户隔离。
type Type string

const (
	Admin Type = "admin"
	User  Type = "user"
)

// Valid 判断角色取值是否合法。
func (t Type) Valid() bool {
	return t == Admin || t == User
}
