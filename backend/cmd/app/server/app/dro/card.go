package dro

import (
	"encoding/json"
	"kanban/cmd/app/server/model"
	"time"
)

// Card 是卡片，连同它的标签。任务、操作记录和关联是单独的实体，通过 CardID 关联。
// ListID 和 Position 为空、ArchivedAt 不为空表示在卡片归档中。
type Card struct {
	ID              int64      `json:"id"`
	PanelID         int64      `json:"panel_id"`
	ListID          *int64     `json:"list_id"`
	Position        *int64     `json:"position"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	LabelIDs        []int64    `json:"label_ids"`
	PriorityLevelID *int64     `json:"priority_level_id"`
	RemindAt        *time.Time `json:"remind_at"`
	DueAt           *time.Time `json:"due_at"`
	RemindNotify    bool       `json:"remind_notify"`
	DueNotify       bool       `json:"due_notify"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	TimerSeconds    int64      `json:"timer_seconds"`
	TimerStartedAt  *time.Time `json:"timer_started_at"`
	ArchivedAt      *time.Time `json:"archived_at"`
	// CoverAttachmentID 是作为封面的图片附件，为空表示没有封面。
	CoverAttachmentID *int64 `json:"cover_attachment_id"`
	// NotifyChannelIDs 是卡片选择的通知渠道，提醒和截止共用。
	NotifyChannelIDs []int64 `json:"notify_channel_ids"`
}

// Task 是卡片中的任务。ParentID 为空表示第一层。
type Task struct {
	ID       int64  `json:"id"`
	CardID   int64  `json:"card_id"`
	ParentID *int64 `json:"parent_id"`
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	Position int64  `json:"position"`
}

// CardAction 是卡片的一条操作记录。
type CardAction struct {
	ID        int64           `json:"id"`
	CardID    int64           `json:"card_id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

// CardLink 是卡片上的关联，BoardID 和 PanelID 只有一个有值。
type CardLink struct {
	ID       int64  `json:"id"`
	CardID   int64  `json:"card_id"`
	BoardID  *int64 `json:"board_id"`
	PanelID  *int64 `json:"panel_id"`
	Position int64  `json:"position"`
}

// CardBundle 是一组卡片连同它们的任务、操作记录、关联、附件和通知发送记录，以及读取时的同步序号。
// 用于加载卡片归档、列表归档中的卡片。
type CardBundle struct {
	Revision    int64            `json:"revision"`
	Cards       []Card           `json:"cards"`
	Tasks       []Task           `json:"tasks"`
	Actions     []CardAction     `json:"card_actions"`
	Links       []CardLink       `json:"card_links"`
	Attachments []Attachment     `json:"attachments"`
	Deliveries  []NotifyDelivery `json:"notify_deliveries"`
}

// NewCard 从卡片模型、它的标签和所选通知渠道生成卡片信息。
func NewCard(card *model.Card, labelIDs, channelIDs []int64) Card {
	if labelIDs == nil {
		labelIDs = []int64{}
	}
	if channelIDs == nil {
		channelIDs = []int64{}
	}
	return Card{
		ID:              card.ID,
		PanelID:         card.PanelID,
		ListID:          card.ListID,
		Position:        card.Position,
		Title:           card.Title,
		Description:     card.Description,
		LabelIDs:        labelIDs,
		PriorityLevelID: card.PriorityLevelID,
		RemindAt:        card.RemindAt,
		DueAt:           card.DueAt,
		RemindNotify:    card.RemindNotify,
		DueNotify:       card.DueNotify,
		CreatedAt:       card.CreatedAt,
		StartedAt:       card.StartedAt,
		CompletedAt:     card.CompletedAt,
		TimerSeconds:    card.TimerSeconds,
		TimerStartedAt:  card.TimerStartedAt,
		ArchivedAt:      card.ArchivedAt,

		CoverAttachmentID: card.CoverAttachmentID,
		NotifyChannelIDs:  channelIDs,
	}
}

// NewTask 从任务模型生成任务信息。
func NewTask(task *model.Task) Task {
	return Task{ID: task.ID, CardID: task.CardID, ParentID: task.ParentID, Title: task.Title, Done: task.Done, Position: task.Position}
}

// NewCardAction 从操作记录模型生成操作记录信息。
func NewCardAction(action *model.CardAction) CardAction {
	return CardAction{ID: action.ID, CardID: action.CardID, Type: action.Type, Data: json.RawMessage(action.Data), CreatedAt: action.CreatedAt}
}

// NewCardLink 从关联模型生成关联信息。
func NewCardLink(link *model.CardLink) CardLink {
	return CardLink{ID: link.ID, CardID: link.CardID, BoardID: link.BoardID, PanelID: link.PanelID, Position: link.Position}
}
