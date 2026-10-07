package model

import "time"

// Card 是面板中某个列表里的一条计划或待办。
//
// 卡片在列表中时 ListID 和 Position（隐藏序号）都有值；进入卡片归档时二者为空，ArchivedAt 记下归档时间，PanelID 保留。
// 随列表进入列表归档的卡片，自身的 ArchivedAt 为空，ListID 和 Position 保持不变。
type Card struct {
	ID              int64      `gorm:"column:id;primaryKey"`
	UserID          int64      `gorm:"column:user_id;not null;index"`
	PanelID         int64      `gorm:"column:panel_id;not null;index"`
	ListID          *int64     `gorm:"column:list_id;index"`
	Position        *int64     `gorm:"column:position"`
	Title           string     `gorm:"column:title;type:text;not null"`
	Description     string     `gorm:"column:description;type:text;not null"`
	PriorityLevelID *int64     `gorm:"column:priority_level_id;index"`
	RemindAt        *time.Time `gorm:"column:remind_at"`
	DueAt           *time.Time `gorm:"column:due_at"`
	RemindNotify    bool       `gorm:"column:remind_notify;not null"` // 到达提醒时间时是否通知
	DueNotify       bool       `gorm:"column:due_notify;not null"`    // 到达截止时间时是否通知
	CreatedAt       time.Time  `gorm:"column:created_at;not null"`    // 卡片真正创建的时刻，之后不变
	StartedAt       *time.Time `gorm:"column:started_at"`
	CompletedAt     *time.Time `gorm:"column:completed_at"`
	TimerSeconds    int64      `gorm:"column:timer_seconds;not null"` // 定时器累计秒数
	TimerStartedAt  *time.Time `gorm:"column:timer_started_at"`       // 定时器本次开始时间，不为空表示正在计时
	ArchivedAt      *time.Time `gorm:"column:archived_at"`
	// CoverAttachmentID 是作为封面的附件，必须是这张卡片上的图片文件附件；为空表示没有封面。
	CoverAttachmentID *int64 `gorm:"column:cover_attachment_id"`
	// TitleRevision 和 DescriptionRevision 是标题、描述最近一次修改时的同步序号，用于检测编辑冲突。
	TitleRevision       int64 `gorm:"column:title_revision;not null"`
	DescriptionRevision int64 `gorm:"column:description_revision;not null"`
}

// TableName 返回表名。
func (*Card) TableName() string {
	return "cards"
}

// Archived 判断卡片是否在卡片归档中。
func (c *Card) Archived() bool {
	return c.ArchivedAt != nil
}

// CardLabel 是卡片与标签的关联。标签必须属于卡片所在面板。
type CardLabel struct {
	ID      int64 `gorm:"column:id;primaryKey"`
	UserID  int64 `gorm:"column:user_id;not null;index"`
	CardID  int64 `gorm:"column:card_id;not null;uniqueIndex:idx_card_label"`
	LabelID int64 `gorm:"column:label_id;not null;uniqueIndex:idx_card_label;index"`
}

// TableName 返回表名。
func (*CardLabel) TableName() string {
	return "card_labels"
}

// Task 是卡片中的任务，最多三层。ParentID 为空表示第一层；Position 是在同一父任务下的顺序。
type Task struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	CardID   int64  `gorm:"column:card_id;not null;index"`
	ParentID *int64 `gorm:"column:parent_id"`
	Title    string `gorm:"column:title;type:text;not null"`
	Done     bool   `gorm:"column:done;not null"`
	Position int64  `gorm:"column:position;not null"`
}

// TableName 返回表名。
func (*Task) TableName() string {
	return "tasks"
}

// 卡片操作记录的类型。
const (
	ActionCreate         = "create"          // 创建卡片，复制时数据中带复制来源
	ActionMove           = "move"            // 跨列移动
	ActionTaskComplete   = "task_complete"   // 完成任务
	ActionTaskUncomplete = "task_uncomplete" // 取消完成任务
	ActionArchive        = "archive"         // 放入卡片归档
	ActionRestore        = "restore"         // 从卡片归档恢复
)

// CardAction 是卡片的一条操作记录，按时间展示。Data 是 JSON。
type CardAction struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id;not null;index"`
	CardID    int64     `gorm:"column:card_id;not null;index"`
	Type      string    `gorm:"column:type;size:32;not null"`
	Data      string    `gorm:"column:data;type:text;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*CardAction) TableName() string {
	return "card_actions"
}

// CardLink 是卡片上的关联：指向一个看板或一个面板（二者只填一个），只是快速跳转的入口。
// 同一卡片对同一目标只保留一条，按添加顺序展示。
type CardLink struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	CardID   int64  `gorm:"column:card_id;not null;index"`
	BoardID  *int64 `gorm:"column:board_id"`
	PanelID  *int64 `gorm:"column:panel_id"`
	Position int64  `gorm:"column:position;not null"`
}

// TableName 返回表名。
func (*CardLink) TableName() string {
	return "card_links"
}
