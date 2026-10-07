package service

import (
	"context"
	"encoding/json"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"time"

	"gorm.io/gorm"
)

// 卡片相关的同步实体类型。卡片的数据包含它的标签；任务、操作记录和关联是单独的实体。
const (
	EntityCard       = "card"
	EntityTask       = "task"
	EntityCardAction = "card_action"
	EntityCardLink   = "card_link"
)

// nowUTC 返回当前时间，截断到微秒，使 SQLite 和 PostgreSQL 保存后读出的值一致。
// 一次操作只取一次，本次被设置或更新的时间字段都写这个值。
func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func findCard(ctx context.Context, userID, id int64) (*model.Card, error) {
	card, err := repository.Card.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.CardNotFound, "卡片不存在")
	}
	return card, err
}

// usableCard 查找可以修改的卡片：卡片在某个列表中，列表不在列表归档中，所在面板正常使用中。返回卡片和所在列表。
// 卡片归档和列表归档中的卡片只能查看。
func usableCard(ctx context.Context, userID, id int64) (*model.Card, *model.List, error) {
	card, err := findCard(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	if card.Archived() || card.ListID == nil {
		return nil, nil, badRequest(resp.CardArchived, "卡片已归档")
	}
	list, err := usableList(ctx, userID, *card.ListID)
	if err != nil {
		return nil, nil, err
	}
	return card, list, nil
}

// cardView 生成带标签和所选通知渠道的卡片信息。
func cardView(ctx context.Context, card *model.Card) (dro.Card, error) {
	labels, err := repository.Card.LabelIDs(ctx, card.UserID, []int64{card.ID})
	if err != nil {
		return dro.Card{}, err
	}
	channels, err := repository.Notify.ChannelIDs(ctx, card.UserID, []int64{card.ID})
	if err != nil {
		return dro.Card{}, err
	}
	return dro.NewCard(card, labels[card.ID], channels[card.ID]), nil
}

func recordCard(ctx context.Context, card *model.Card) error {
	view, err := cardView(ctx, card)
	if err != nil {
		return err
	}
	return recordChange(ctx, card.UserID, hub.OpUpsert, EntityCard, card.ID, view)
}

func recordTask(ctx context.Context, task *model.Task) error {
	return recordChange(ctx, task.UserID, hub.OpUpsert, EntityTask, task.ID, dro.NewTask(task))
}

func recordCardLink(ctx context.Context, link *model.CardLink) error {
	return recordChange(ctx, link.UserID, hub.OpUpsert, EntityCardLink, link.ID, dro.NewCardLink(link))
}

// addAction 给卡片追加一条操作记录并记录变更。data 会序列化为 JSON。
func addAction(ctx context.Context, card *model.Card, actionType string, data any, at time.Time) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	action := &model.CardAction{UserID: card.UserID, CardID: card.ID, Type: actionType, Data: string(payload), CreatedAt: at}
	if err = repository.CardAction.Create(ctx, action); err != nil {
		return err
	}
	return recordChange(ctx, card.UserID, hub.OpUpsert, EntityCardAction, action.ID, dro.NewCardAction(action))
}

// cardBundle 读取这些卡片连同标签、所选通知渠道、任务、操作记录、关联、附件和通知发送记录。
func cardBundle(ctx context.Context, userID int64, cards []*model.Card) (*dro.CardBundle, error) {
	ids := make([]int64, len(cards))
	for i, card := range cards {
		ids[i] = card.ID
	}
	labels, err := repository.Card.LabelIDs(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	tasks, err := repository.Task.ListByCards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	actions, err := repository.CardAction.ListByCards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	links, err := repository.CardLink.ListByCards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	attachments, err := repository.Attachment.ListByCards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	attachmentList, err := attachmentViews(ctx, userID, attachments)
	if err != nil {
		return nil, err
	}
	channels, err := repository.Notify.ChannelIDs(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	deliveries, err := repository.Notify.ListDeliveries(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	bundle := &dro.CardBundle{
		Cards:       make([]dro.Card, 0, len(cards)),
		Tasks:       mapItems(tasks, dro.NewTask),
		Actions:     mapItems(actions, dro.NewCardAction),
		Links:       mapItems(links, dro.NewCardLink),
		Attachments: attachmentList,
		Deliveries:  mapItems(deliveries, dro.NewNotifyDelivery),
	}
	for _, card := range cards {
		bundle.Cards = append(bundle.Cards, dro.NewCard(card, labels[card.ID], channels[card.ID]))
	}
	return bundle, nil
}

func mapItems[M any, D any](items []*M, convert func(*M) D) []D {
	list := make([]D, 0, len(items))
	for _, item := range items {
		list = append(list, convert(item))
	}
	return list
}

// recordCardBundle 记录卡片连同任务、操作记录、关联、附件和通知发送记录的当前状态。
// 卡片或列表从归档恢复时调用：此前没有加载归档内容的设备由此拿到完整的卡片。
func recordCardBundle(ctx context.Context, cards []*model.Card) error {
	if len(cards) == 0 {
		return nil
	}
	userID := cards[0].UserID
	bundle, err := cardBundle(ctx, userID, cards)
	if err != nil {
		return err
	}
	for _, card := range bundle.Cards {
		if err = recordChange(ctx, userID, hub.OpUpsert, EntityCard, card.ID, card); err != nil {
			return err
		}
	}
	for _, task := range bundle.Tasks {
		if err = recordChange(ctx, userID, hub.OpUpsert, EntityTask, task.ID, task); err != nil {
			return err
		}
	}
	for _, action := range bundle.Actions {
		if err = recordChange(ctx, userID, hub.OpUpsert, EntityCardAction, action.ID, action); err != nil {
			return err
		}
	}
	for _, link := range bundle.Links {
		if err = recordChange(ctx, userID, hub.OpUpsert, EntityCardLink, link.ID, link); err != nil {
			return err
		}
	}
	for _, attachment := range bundle.Attachments {
		if err = recordChange(ctx, userID, hub.OpUpsert, EntityAttachment, attachment.ID, attachment); err != nil {
			return err
		}
	}
	for _, delivery := range bundle.Deliveries {
		if err = recordChange(ctx, userID, hub.OpUpsert, EntityNotifyDelivery, delivery.ID, delivery); err != nil {
			return err
		}
	}
	return nil
}

// stopTimer 停止正在计时的定时器：把已经过的秒数加进累计秒数，并清空开始时间。不在计时时不做任何事。
func stopTimer(ctx context.Context, card *model.Card, now time.Time) error {
	if card.TimerStartedAt == nil {
		return nil
	}
	card.TimerSeconds += int64(now.Sub(*card.TimerStartedAt).Seconds())
	card.TimerStartedAt = nil
	if err := repository.Card.Update(ctx, card.UserID, card.ID, map[string]any{"timer_seconds": card.TimerSeconds, "timer_started_at": nil}); err != nil {
		return err
	}
	return recordCard(ctx, card)
}

// stopTimersInPanels 停止这些面板中所有正在计时的定时器。面板、看板或文件夹进入归档时调用。
func stopTimersInPanels(ctx context.Context, userID int64, panelIDs []int64, now time.Time) error {
	cards, err := repository.Card.ListRunningTimersInPanels(ctx, userID, panelIDs)
	if err != nil {
		return err
	}
	for _, card := range cards {
		if err = stopTimer(ctx, card, now); err != nil {
			return err
		}
	}
	return nil
}

// stopTimersInBoards 停止这些看板中未归档面板里正在计时的定时器。
func stopTimersInBoards(ctx context.Context, userID int64, boardIDs []int64, now time.Time) error {
	var panelIDs []int64
	for _, boardID := range boardIDs {
		panels, err := repository.Panel.ListActiveInBoard(ctx, userID, boardID)
		if err != nil {
			return err
		}
		for _, panel := range panels {
			panelIDs = append(panelIDs, panel.ID)
		}
	}
	return stopTimersInPanels(ctx, userID, panelIDs, now)
}
