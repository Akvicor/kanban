package dto

import (
	"kanban/cmd/app/server/common/types/listsort"
	"time"
)

// CreateCard 是用列首或列尾的创建按钮新建卡片的请求。Button 是按下的按钮：head 或 tail。
type CreateCard struct {
	ListID int64        `json:"list_id"`
	Title  string       `json:"title"`
	Button listsort.End `json:"button"`
}

// UpdateCardText 是修改卡片标题或描述的请求。BaseRevision 是开始编辑时本地卡片数据的同步序号，用于检测冲突。
type UpdateCardText struct {
	ID           int64  `json:"id"`
	Text         string `json:"text"`
	BaseRevision int64  `json:"base_revision"`
}

// SetCardPriority 是设置卡片优先级的请求，PriorityLevelID 为 null 表示没有优先级。
type SetCardPriority struct {
	ID              int64  `json:"id"`
	PriorityLevelID *int64 `json:"priority_level_id"`
}

// SetCardDates 是修改提醒、截止时间和通知开关的请求，各项都需要提供，时间为 null 表示清空。
type SetCardDates struct {
	ID           int64      `json:"id"`
	RemindAt     *time.Time `json:"remind_at"`
	DueAt        *time.Time `json:"due_at"`
	RemindNotify bool       `json:"remind_notify"`
	DueNotify    bool       `json:"due_notify"`
}

// SetCardLabel 是给卡片加上（On 为 true）或去掉一个标签的请求。
type SetCardLabel struct {
	ID      int64 `json:"id"`
	LabelID int64 `json:"label_id"`
	On      bool  `json:"on"`
}

// CardTimer 是操作定时器的请求。Action 为 start、stop 或 set；set 时 Seconds 是新的累计秒数。
type CardTimer struct {
	ID      int64  `json:"id"`
	Action  string `json:"action"`
	Seconds int64  `json:"seconds"`
}

// MoveCard 是移动卡片的请求。Index 是在目标列表中按隐藏序号的位置（从 0 开始），为 null 时放到列尾。
type MoveCard struct {
	ID     int64 `json:"id"`
	ListID int64 `json:"list_id"`
	Index  *int  `json:"index"`
}

// RestoreCard 是把卡片从卡片归档恢复到某个列表一端的请求。
type RestoreCard struct {
	ID     int64        `json:"id"`
	ListID int64        `json:"list_id"`
	End    listsort.End `json:"end"`
}

// CopyCard 是复制卡片的请求。ListID 为 null 时副本插在原卡片下方，否则插到该列表的列首。
type CopyCard struct {
	ID     int64  `json:"id"`
	ListID *int64 `json:"list_id"`
}

// CreateTasks 是新建任务的请求。Titles 有多项时（粘贴多行文字）每项一个任务，都放在同一个父任务下。
type CreateTasks struct {
	CardID   int64    `json:"card_id"`
	ParentID *int64   `json:"parent_id"`
	Titles   []string `json:"titles"`
}

// RenameTask 是修改任务标题的请求。
type RenameTask struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// SetTaskDone 是勾选或取消勾选任务的请求。
type SetTaskDone struct {
	ID   int64 `json:"id"`
	Done bool  `json:"done"`
}

// MoveTask 是拖动任务的请求。ParentID 为 null 表示放到第一层；Index 为 null 时排在末尾。
type MoveTask struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id"`
	Index    *int   `json:"index"`
}

// AddCardLink 是给卡片加一条关联的请求，BoardID 和 PanelID 只提供一个。
type AddCardLink struct {
	CardID  int64  `json:"card_id"`
	BoardID *int64 `json:"board_id"`
	PanelID *int64 `json:"panel_id"`
}
