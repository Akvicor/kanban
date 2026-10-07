package repository

import (
	"context"
	"fmt"
	"kanban/cmd/app/server/model"
	"time"
)

// Notify 是通知渠道、卡片的渠道选择和发送记录的仓储。按用户查询的函数都带用户条件；
// 后台发送任务按时间查询全部用户。
var Notify = new(notifyRepository)

type notifyRepository struct{}

// CreateChannel 新建通知渠道。
func (*notifyRepository) CreateChannel(ctx context.Context, channel *model.NotifyChannel) error {
	return conn(ctx).Create(channel).Error
}

// FindChannel 按 ID 查找用户的通知渠道。
func (*notifyRepository) FindChannel(ctx context.Context, userID, id int64) (*model.NotifyChannel, error) {
	channel := new(model.NotifyChannel)
	return channel, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(channel).Error
}

// ListChannels 返回用户的全部通知渠道，按创建顺序。
func (*notifyRepository) ListChannels(ctx context.Context, userID int64) ([]*model.NotifyChannel, error) {
	channels := make([]*model.NotifyChannel, 0)
	return channels, conn(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&channels).Error
}

// UpdateChannel 更新通知渠道的字段。
func (*notifyRepository) UpdateChannel(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.NotifyChannel{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// DeleteChannel 删除用户的通知渠道。
func (*notifyRepository) DeleteChannel(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.NotifyChannel{}))
}

// ChannelIDs 返回这些卡片选择的渠道，结果以卡片 ID 为键，渠道按 ID 排列。
func (*notifyRepository) ChannelIDs(ctx context.Context, userID int64, cardIDs []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(cardIDs))
	if len(cardIDs) == 0 {
		return result, nil
	}
	rows := make([]*model.CardNotifyChannel, 0)
	if err := conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("channel_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CardID] = append(result[row.CardID], row.ChannelID)
	}
	return result, nil
}

// AddCardChannel 为卡片选择一条渠道。
func (*notifyRepository) AddCardChannel(ctx context.Context, row *model.CardNotifyChannel) error {
	return conn(ctx).Create(row).Error
}

// RemoveCardChannel 取消卡片对一条渠道的选择。
func (*notifyRepository) RemoveCardChannel(ctx context.Context, userID, cardID, channelID int64) error {
	return conn(ctx).Where("user_id = ? AND card_id = ? AND channel_id = ?", userID, cardID, channelID).Delete(&model.CardNotifyChannel{}).Error
}

// CardIDsWithChannel 返回选择了这条渠道的卡片 ID。
func (*notifyRepository) CardIDsWithChannel(ctx context.Context, userID, channelID int64) ([]int64, error) {
	var ids []int64
	return ids, conn(ctx).Model(&model.CardNotifyChannel{}).Where("user_id = ? AND channel_id = ?", userID, channelID).Pluck("card_id", &ids).Error
}

// RemoveChannelFromCards 从所有卡片的选择中移除这条渠道。
func (*notifyRepository) RemoveChannelFromCards(ctx context.Context, userID, channelID int64) error {
	return conn(ctx).Where("user_id = ? AND channel_id = ?", userID, channelID).Delete(&model.CardNotifyChannel{}).Error
}

// ListDeliveries 返回这些卡片的发送记录。
func (*notifyRepository) ListDeliveries(ctx context.Context, userID int64, cardIDs []int64) ([]*model.NotifyDelivery, error) {
	deliveries := make([]*model.NotifyDelivery, 0)
	if len(cardIDs) == 0 {
		return deliveries, nil
	}
	return deliveries, conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("id ASC").Find(&deliveries).Error
}

// FindDelivery 查找一张卡片的一种通知在一条渠道上的发送记录。
func (*notifyRepository) FindDelivery(ctx context.Context, userID, cardID int64, kind string, channelID int64) (*model.NotifyDelivery, error) {
	delivery := new(model.NotifyDelivery)
	return delivery, conn(ctx).Where("user_id = ? AND card_id = ? AND kind = ? AND channel_id = ?", userID, cardID, kind, channelID).Take(delivery).Error
}

// SaveDelivery 新建或整体保存发送记录。
func (*notifyRepository) SaveDelivery(ctx context.Context, delivery *model.NotifyDelivery) error {
	return conn(ctx).Save(delivery).Error
}

// ListDeliveriesByChannel 返回一条渠道的全部发送记录。
func (*notifyRepository) ListDeliveriesByChannel(ctx context.Context, userID, channelID int64) ([]*model.NotifyDelivery, error) {
	deliveries := make([]*model.NotifyDelivery, 0)
	return deliveries, conn(ctx).Where("user_id = ? AND channel_id = ?", userID, channelID).Find(&deliveries).Error
}

// DeleteDeliveries 按 ID 删除发送记录。
func (*notifyRepository) DeleteDeliveries(ctx context.Context, userID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return conn(ctx).Where("user_id = ? AND id IN ?", userID, ids).Delete(&model.NotifyDelivery{}).Error
}

// DeleteByUser 删除用户的通知渠道、卡片的渠道选择和发送记录。
func (*notifyRepository) DeleteByUser(ctx context.Context, userID int64) error {
	for _, table := range []any{&model.NotifyDelivery{}, &model.CardNotifyChannel{}, &model.NotifyChannel{}} {
		if err := conn(ctx).Where("user_id = ?", userID).Delete(table).Error; err != nil {
			return err
		}
	}
	return nil
}

// notifyColumns 是一种通知在卡片和列表上对应的列：卡片的时间、卡片的通知开关、列表的关闭开关。
// 列名都是固定值，拼进 SQL 前不来自用户输入。
var notifyColumns = map[string]struct{ at, notify, off string }{
	model.NotifyRemind: {"remind_at", "remind_notify", "remind_off"},
	model.NotifyDue:    {"due_at", "due_notify", "due_off"},
}

// eligibleCondition 是卡片的一种通知满足发送条件（不含时间是否已到）的 SQL 条件：
// 卡片的通知开关打开、卡片在未归档的列表中且列表没有关闭这种通知、面板和看板未归档、用户未停用。
// 使用别名 c（card）、l（list）、p（panel）、b（board）、u（user）。参数依次为 true、false。
func eligibleCondition(kind string) string {
	columns := notifyColumns[kind]
	return fmt.Sprintf(`c.%s IS NOT NULL AND c.%s = ? AND c.archived_at IS NULL AND l.id IS NOT NULL AND l.archived_at IS NULL
		AND l.%s = ? AND p.archived_at IS NULL AND b.archived_at IS NULL AND u.disabled_at IS NULL`, columns.at, columns.notify, columns.off)
}

const notifyJoins = `LEFT JOIN lists l ON l.id = c.list_id
	LEFT JOIN panels p ON p.id = c.panel_id
	LEFT JOIN boards b ON b.id = p.board_id
	LEFT JOIN users u ON u.id = c.user_id`

// DueNotification 是一条到点需要发送（或重试）的通知：卡片、类型、渠道和这次针对的时间值。
type DueNotification struct {
	UserID    int64
	CardID    int64
	ChannelID int64
	TargetAt  time.Time
}

// ListDue 返回 now 时需要发送的某种通知：满足发送条件、时间已到，并且这条渠道还没有针对当前时间值的记录，
// 或者记录未发送且到了下次发送时间。
func (*notifyRepository) ListDue(ctx context.Context, kind string, now time.Time) ([]DueNotification, error) {
	at := "c." + notifyColumns[kind].at
	query := fmt.Sprintf(`SELECT c.user_id AS user_id, c.id AS card_id, cc.channel_id AS channel_id, %[1]s AS target_at
		FROM cards c
		%[2]s
		JOIN card_notify_channels cc ON cc.card_id = c.id
		LEFT JOIN notify_deliveries d ON d.card_id = c.id AND d.kind = ? AND d.channel_id = cc.channel_id
		WHERE %[3]s AND %[1]s <= ?
		AND (d.id IS NULL OR d.target_at <> %[1]s OR (d.sent_at IS NULL AND d.next_attempt_at <= ?))
		ORDER BY %[1]s ASC, c.id ASC, cc.channel_id ASC`, at, notifyJoins, eligibleCondition(kind))
	due := make([]DueNotification, 0)
	return due, conn(ctx).Raw(query, kind, true, false, now, now).Scan(&due).Error
}

// StaleDelivery 是需要删除的发送记录。
type StaleDelivery struct {
	ID     int64
	UserID int64
}

// ListStale 返回某种通知中需要删除的发送记录：卡片的时间已被修改或清空的记录；以及未发送、
// 但已不满足发送条件（关闭开关、列表关闭通知、归档、移除渠道、用户停用）的记录。
func (*notifyRepository) ListStale(ctx context.Context, kind string) ([]StaleDelivery, error) {
	at := "c." + notifyColumns[kind].at
	query := fmt.Sprintf(`SELECT d.id AS id, d.user_id AS user_id
		FROM notify_deliveries d
		LEFT JOIN cards c ON c.id = d.card_id
		%[2]s
		LEFT JOIN card_notify_channels cc ON cc.card_id = d.card_id AND cc.channel_id = d.channel_id
		WHERE d.kind = ? AND (c.id IS NULL OR %[1]s IS NULL OR %[1]s <> d.target_at
			OR (d.sent_at IS NULL AND (CASE WHEN cc.id IS NOT NULL AND %[3]s THEN 1 ELSE 0 END) = 0))`,
		at, notifyJoins, eligibleCondition(kind))
	stale := make([]StaleDelivery, 0)
	return stale, conn(ctx).Raw(query, kind, true, false).Scan(&stale).Error
}
