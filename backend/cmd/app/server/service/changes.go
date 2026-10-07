package service

import (
	"context"
	"encoding/json"
	"errors"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"time"
)

// 同步的实体类型，与前端 src/sync/store.ts 一致。
const (
	EntitySettings = "settings" // 个人设置，ID 为用户 ID
	EntityAccount  = "account"  // 账号的用户名和昵称，ID 为用户 ID
	EntityDevice   = "device"   // 登录设备
)

// change_log 的保留范围：每个用户保留最近 changeLogRetention 条记录。
// 序号每增加 changeLogTrimEvery 时清理一次，避免每次写入都执行删除。
// 客户端落后超过保留范围时，重连后收到全量快照。
const (
	changeLogRetention = 10000
	changeLogTrimEvery = 500
)

// changeSet 收集一次写入中产生的变更和提交后要执行的动作。
type changeSet struct {
	events map[int64][]hub.Event // 用户 ID → 按序号升序的变更
	hooks  []func()
}

type changeSetKey struct{}

// write 在事务中执行 f。f 中通过 recordChange 记录的变更在提交后推送，通过 afterCommit 登记的动作在推送后执行；
// 事务回滚时两者都不发生。嵌套调用时复用外层的事务和收集器，由最外层统一推送。
func write(ctx context.Context, f func(ctx context.Context) error) error {
	if _, ok := ctx.Value(changeSetKey{}).(*changeSet); ok {
		return f(ctx)
	}
	if repository.InTransaction(ctx) {
		return errors.New("write 不能在没有变更收集器的事务中调用，否则会在提交前推送")
	}
	set := &changeSet{events: map[int64][]hub.Event{}}
	if err := repository.Transaction(context.WithValue(ctx, changeSetKey{}, set), f); err != nil {
		return err
	}
	for userID, events := range set.events {
		hub.Default.Publish(userID, events)
		noteRevision(ctx, userID, events[len(events)-1].Revision)
	}
	for _, hook := range set.hooks {
		hook()
	}
	return nil
}

// afterCommit 登记一个在事务提交并推送之后执行的动作，例如断开已吊销设备的连接。
func afterCommit(ctx context.Context, hook func()) error {
	set, ok := ctx.Value(changeSetKey{}).(*changeSet)
	if !ok {
		return errors.New("afterCommit 必须在 write 中调用")
	}
	set.hooks = append(set.hooks, hook)
	return nil
}

// recordChange 为用户分配下一个同步序号，追加变更记录。data 是实体变更后的完整数据，删除时传 nil。
func recordChange(ctx context.Context, userID int64, op, entityType string, entityID int64, data any) error {
	set, ok := ctx.Value(changeSetKey{}).(*changeSet)
	if !ok {
		return errors.New("recordChange 必须在 write 中调用")
	}
	revision, err := repository.User.NextRevision(ctx, userID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = repository.ChangeLog.Create(ctx, &model.ChangeLog{
		UserID:    userID,
		Revision:  revision,
		Op:        op,
		Type:      entityType,
		EntityID:  entityID,
		Data:      string(payload),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	if err = trimChangeLog(ctx, userID, revision); err != nil {
		return err
	}
	set.events[userID] = append(set.events[userID], hub.Event{
		Revision: revision, Op: op, Type: entityType, ID: entityID, Data: payload,
	})
	return nil
}

// lastRecordedRevision 返回本次写入中为用户分配的最近一个序号，还没有记录变更时为 0。
func lastRecordedRevision(ctx context.Context, userID int64) int64 {
	set, ok := ctx.Value(changeSetKey{}).(*changeSet)
	if !ok || len(set.events[userID]) == 0 {
		return 0
	}
	events := set.events[userID]
	return events[len(events)-1].Revision
}

// trimChangeLog 在序号到达清理点时，删除保留范围之前的记录。
func trimChangeLog(ctx context.Context, userID, revision int64) error {
	if revision%changeLogTrimEvery != 0 || revision <= changeLogRetention {
		return nil
	}
	return repository.ChangeLog.DeleteUpTo(ctx, userID, revision-changeLogRetention)
}

// revisionNote 记录一次请求中当前用户产生的最大同步序号，由接口层放进响应。
type revisionNote struct {
	userID   int64
	revision int64
}

type revisionNoteKey struct{}

// WithRevisionNote 为 userID 的请求准备序号记录。接口层在认证后调用，
// 写入完成后用 Revision 取出序号放进响应，客户端据此忽略推送中同一序号的重复变更。
func WithRevisionNote(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, revisionNoteKey{}, &revisionNote{userID: userID})
}

// Revision 返回本次请求中当前用户产生的最大同步序号，没有变更时为 0。
func Revision(ctx context.Context) int64 {
	if note, ok := ctx.Value(revisionNoteKey{}).(*revisionNote); ok {
		return note.revision
	}
	return 0
}

func noteRevision(ctx context.Context, userID, revision int64) {
	if note, ok := ctx.Value(revisionNoteKey{}).(*revisionNote); ok && note.userID == userID && revision > note.revision {
		note.revision = revision
	}
}
