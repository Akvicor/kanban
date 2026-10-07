// Package listsort 定义列表中卡片的排序方式和方向，以及列首、列尾按钮的插入位置。
package listsort

// Mode 是列表的排序方式。Manual 按卡片的隐藏序号展示，序号来自新建时的插入位置和之后的拖动；
// 其余方式按对应字段排列，不改写隐藏序号。
type Mode string

const (
	Manual      Mode = "manual"
	Title       Mode = "title"
	Priority    Mode = "priority"
	RemindAt    Mode = "remind_at"
	DueAt       Mode = "due_at"
	CreatedAt   Mode = "created_at"
	StartedAt   Mode = "started_at"
	CompletedAt Mode = "completed_at"
)

// Valid 判断排序方式取值是否合法。
func (m Mode) Valid() bool {
	switch m {
	case Manual, Title, Priority, RemindAt, DueAt, CreatedAt, StartedAt, CompletedAt:
		return true
	}
	return false
}

// Direction 是排序方向。正序指标题从前往后、时间从早到晚、优先级从高到低，倒序相反。手动排序不使用方向。
type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

// Valid 判断排序方向取值是否合法。
func (d Direction) Valid() bool {
	return d == Asc || d == Desc
}

// End 是列表的一端，用于创建按钮的插入位置和恢复卡片的位置。
type End string

const (
	Head End = "head" // 列首
	Tail End = "tail" // 列尾
)

// Valid 判断取值是否合法。
func (e End) Valid() bool {
	return e == Head || e == Tail
}
