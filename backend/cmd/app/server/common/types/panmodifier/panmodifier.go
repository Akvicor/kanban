// Package panmodifier 定义拖动平移看板视图使用的修饰键。
package panmodifier

// Type 是修饰键标记，取值固定为小写。空值表示关闭「按住修饰键拖动平移」手势。
type Type string

const (
	// Ctrl 控制键。
	Ctrl Type = "ctrl"
	// Alt 选择键。
	Alt Type = "alt"
	// Shift 上档键。
	Shift Type = "shift"
)

// Default 是新用户使用的修饰键。
const Default = Ctrl

// Valid 判断修饰键是否受支持；空值表示关闭手势，允许保存。
func (t Type) Valid() bool {
	return t == "" || t == Ctrl || t == Alt || t == Shift
}
