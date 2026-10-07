package service

import (
	"context"
	"errors"
	"fmt"
	"kanban/cmd/app/server/common/gmsgsend"
	"kanban/cmd/app/server/common/notifytemplate"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Akvicor/glog"
	"github.com/Akvicor/gmsg/gmodel"
	"gorm.io/gorm"
)

const (
	// notifyInterval 是后台发送任务的检查间隔。
	notifyInterval = 15 * time.Second
	// notifyConcurrency 是同一时间最多进行的 gmsg 请求数。
	notifyConcurrency = 4
	// retryBase、retryMax 是失败重试间隔的起点和上限：1、2、4……分钟，最长 1 小时。
	retryBase = time.Minute
	retryMax  = time.Hour
	// notifyTimeFormat 是正文中时间的格式，按用户时区显示。
	notifyTimeFormat = "2006-01-02 15:04"
)

// notifyClient 是发送 gmsg 请求的客户端，每次请求最多 30 秒，不跟随重定向。
var notifyClient = gmsgsend.NewClient(30 * time.Second)

// notifyTitles 是 gmsg 请求中的标题，按语言各有一份。
var notifyTitles = map[string]map[string]string{
	model.NotifyRemind: {string(locale.ZhCN): "提醒", string(locale.En): "Reminder"},
	model.NotifyDue:    {string(locale.ZhCN): "截止", string(locale.En): "Due"},
}

// notifyTitle 返回该语言的通知标题，未设置或不受支持的语言回落默认语言。
func notifyTitle(kind, lang string) string {
	titles := notifyTitles[kind]
	return titles[locale.Type(lang).Tag()]
}

// Notifier 是发送提醒和截止通知的后台任务。
//
// 每次检查先删除不再需要的发送记录（时间值已改变，或未发送但已不满足发送条件），
// 再找出满足发送条件、时间已到、这条渠道还没有发送或到了重试时间的通知，逐条发送并记录结果。
// 停机期间到达的通知在恢复后的第一次检查中补发；失败后按 1、2、4……分钟（最长 1 小时）的间隔一直重试。
var Notifier = &notifier{wake: make(chan struct{}, 1)}

type notifier struct {
	wake chan struct{}
	// running 保证同一时间只有一次检查在进行。
	running sync.Mutex
}

// Wake 让后台任务立即检查一次，不阻塞。
func (n *notifier) Wake() {
	select {
	case n.wake <- struct{}{}:
	default:
	}
}

// Start 启动后台任务：立即检查一次，之后每 15 秒或被唤醒时检查。
func (n *notifier) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(notifyInterval)
		defer ticker.Stop()
		for {
			if err := n.RunOnce(ctx, time.Now()); err != nil {
				glog.Error("检查通知失败: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-n.wake:
			}
		}
	}()
}

// RunOnce 按 now 检查并发送一次。
func (n *notifier) RunOnce(ctx context.Context, now time.Time) error {
	n.running.Lock()
	defer n.running.Unlock()
	now = now.UTC().Truncate(time.Microsecond)
	for _, kind := range []string{model.NotifyRemind, model.NotifyDue} {
		if err := cleanStaleDeliveries(ctx, kind); err != nil {
			return err
		}
	}
	semaphore := make(chan struct{}, notifyConcurrency)
	var wait sync.WaitGroup
	for _, kind := range []string{model.NotifyRemind, model.NotifyDue} {
		due, err := repository.Notify.ListDue(ctx, kind, now)
		if err != nil {
			return err
		}
		for _, item := range due {
			semaphore <- struct{}{}
			wait.Add(1)
			go func() {
				defer func() {
					<-semaphore
					wait.Done()
				}()
				if err := deliver(ctx, kind, item, now); err != nil {
					glog.Error("发送通知（卡片 %d，%s，渠道 %d）失败: %v", item.CardID, kind, item.ChannelID, err)
				}
			}()
		}
	}
	wait.Wait()
	return nil
}

// cleanStaleDeliveries 删除某种通知中不再需要的发送记录：时间值已改变或清空的，以及未发送但已不满足发送条件的。
func cleanStaleDeliveries(ctx context.Context, kind string) error {
	stale, err := repository.Notify.ListStale(ctx, kind)
	if err != nil || len(stale) == 0 {
		return err
	}
	byUser := map[int64][]*model.NotifyDelivery{}
	for _, item := range stale {
		byUser[item.UserID] = append(byUser[item.UserID], &model.NotifyDelivery{ID: item.ID})
	}
	for userID, deliveries := range byUser {
		if err = write(ctx, func(ctx context.Context) error { return deleteDeliveries(ctx, userID, deliveries) }); err != nil {
			return err
		}
	}
	return nil
}

// retryDelay 是第 attempts 次失败后的重试间隔。
func retryDelay(attempts int) time.Duration {
	delay := retryBase
	for i := 1; i < attempts && delay < retryMax; i++ {
		delay *= 2
	}
	return min(delay, retryMax)
}

// deliver 发送一条通知：建立或重置发送记录并生成正文，发送后记录结果。
func deliver(ctx context.Context, kind string, item repository.DueNotification, now time.Time) error {
	var deliveryID int64
	var channel gmsgsend.Channel
	var title, body string
	err := write(ctx, func(ctx context.Context) error {
		delivery, err := repository.Notify.FindDelivery(ctx, item.UserID, item.CardID, kind, item.ChannelID)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			delivery = &model.NotifyDelivery{UserID: item.UserID, CardID: item.CardID, Kind: kind, ChannelID: item.ChannelID}
		case err != nil:
			return err
		case delivery.TargetAt.Equal(item.TargetAt) && (delivery.SentAt != nil || delivery.NextAttemptAt.After(now)):
			return nil // 期间已被处理
		}
		if delivery.ID == 0 || !delivery.TargetAt.Equal(item.TargetAt) {
			// 新记录，或时间值已改变：按新的时间值从头开始。
			delivery.TargetAt, delivery.SentAt, delivery.Attempts, delivery.LastError = item.TargetAt, nil, 0, ""
			delivery.NextAttemptAt = now
			if err = repository.Notify.SaveDelivery(ctx, delivery); err != nil {
				return err
			}
			if err = recordDelivery(ctx, delivery); err != nil {
				return err
			}
		}
		source, err := findChannel(ctx, item.UserID, item.ChannelID)
		if err != nil {
			return err
		}
		if title, body, err = notifyMessage(ctx, item.UserID, item.CardID, kind, source.Format == gmodel.TypeMarkdown); err != nil {
			return err
		}
		deliveryID, channel = delivery.ID, channelOf(source)
		return nil
	})
	if err != nil || deliveryID == 0 {
		return err
	}

	sendErr := gmsgsend.Send(ctx, notifyClient, channel, gmsgsend.Message{Title: title, Body: body, At: now})

	return write(ctx, func(ctx context.Context) error {
		delivery, err := repository.Notify.FindDelivery(ctx, item.UserID, item.CardID, kind, item.ChannelID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && (delivery.ID != deliveryID || !delivery.TargetAt.Equal(item.TargetAt))) {
			return nil // 发送期间记录被删除或时间值被修改，结果不再适用
		}
		if err != nil {
			return err
		}
		if sendErr == nil {
			sentAt := nowUTC()
			delivery.SentAt, delivery.LastError = &sentAt, ""
		} else {
			delivery.Attempts++
			delivery.LastError = sendErr.Error()
			delivery.NextAttemptAt = now.Add(retryDelay(delivery.Attempts))
		}
		if err = repository.Notify.SaveDelivery(ctx, delivery); err != nil {
			return err
		}
		return recordDelivery(ctx, delivery)
	})
}

// notifyMessage 按用户的模板生成卡片某种通知的标题和正文，标题与默认模板跟随用户语言；
// 用户自定义了模板时用自定义模板，不受语言影响。
func notifyMessage(ctx context.Context, userID, cardID int64, kind string, markdown bool) (string, string, error) {
	user, err := User.FindByID(ctx, userID)
	if err != nil {
		return "", "", err
	}
	card, err := repository.Card.FindByID(ctx, userID, cardID)
	if err != nil {
		return "", "", err
	}
	location, err := time.LoadLocation(user.Timezone)
	if err != nil {
		location = time.UTC
	}
	formatTime := func(value *time.Time) string {
		if value == nil {
			return ""
		}
		return value.In(location).Format(notifyTimeFormat)
	}
	values := map[string]string{
		"title": card.Title, "description": card.Description,
		"remind_at": formatTime(card.RemindAt), "due_at": formatTime(card.DueAt), "created_at": formatTime(&card.CreatedAt),
		"started_at": formatTime(card.StartedAt), "completed_at": formatTime(card.CompletedAt),
	}
	if panel, err := repository.Panel.FindByID(ctx, userID, card.PanelID); err == nil {
		values["panel"] = panel.Name
		if board, err := repository.Board.FindByID(ctx, userID, panel.BoardID); err == nil {
			values["board"] = board.Name
		}
	}
	if card.ListID != nil {
		if list, err := repository.List.FindByID(ctx, userID, *card.ListID); err == nil {
			values["list"] = list.Name
		}
	}
	if card.PriorityLevelID != nil {
		if level, err := repository.PriorityLevel.FindByID(ctx, userID, *card.PriorityLevelID); err == nil {
			values["priority"] = level.Name
		}
	}
	labelIDs, err := repository.Card.LabelIDs(ctx, userID, []int64{cardID})
	if err != nil {
		return "", "", err
	}
	labels, err := repository.Label.ListByPanel(ctx, userID, card.PanelID)
	if err != nil {
		return "", "", err
	}
	var names []string
	for _, label := range labels { // 面板中的标签已按顺序排列
		if slices.Contains(labelIDs[cardID], label.ID) {
			names = append(names, label.Name)
		}
	}
	labelSeparator := "，"
	if locale.Type(user.Locale) == locale.En {
		labelSeparator = ", "
	}
	values["labels"] = strings.Join(names, labelSeparator)
	tasks, err := repository.Task.ListByCards(ctx, userID, []int64{cardID})
	if err != nil {
		return "", "", err
	}
	if len(tasks) > 0 {
		done := 0
		for _, task := range tasks {
			if task.Done {
				done++
			}
		}
		values["tasks"] = fmt.Sprintf("%d/%d", done, len(tasks))
	}
	// 空模板表示使用随语言的系统默认模板；自定义模板不随语言变化。
	template := user.RemindTemplate
	defaultTemplate := notifytemplate.DefaultRemind
	if kind == model.NotifyDue {
		template, defaultTemplate = user.DueTemplate, notifytemplate.DefaultDue
	}
	if template == "" {
		template = defaultTemplate(user.Locale.Tag()) // 后台发送没有请求信息，未设置语言时回落默认语言
	}
	return notifyTitle(kind, user.Locale.Tag()), notifytemplate.Render(template, values, markdown), nil
}
