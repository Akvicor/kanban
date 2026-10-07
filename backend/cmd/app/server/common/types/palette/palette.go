package palette

// Type 是用户的配色偏好，与前端 src/theme/palette.ts 的取值一致。
// system 表示跟随系统：系统亮色用 clean，暗色用 dark；其余取值固定使用对应配色。
type Type string

const (
	System Type = "system"
	Clean  Type = "clean" // A 清爽，新用户默认
	Dark   Type = "dark"  // B 暗夜
	Paper  Type = "paper" // C 纸感
)

// Default 是新用户的配色偏好。
const Default = Clean

// Valid 判断配色偏好取值是否合法。
func (t Type) Valid() bool {
	switch t {
	case System, Clean, Dark, Paper:
		return true
	}
	return false
}
