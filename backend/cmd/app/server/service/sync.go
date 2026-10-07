package service

import (
	"context"
	"encoding/json"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/repository"
)

// Sync 是设备连接时补齐变更的服务。
var Sync = new(syncService)

type syncService struct{}

// CatchUp 是设备连接时需要发送的内容：全量快照，或 lastRevision 之后的变更，二者只有一个。
// Revision 是读取时该用户的最新序号，客户端发送完这些内容后就处在这个序号。
type CatchUp struct {
	Snapshot *dro.Snapshot
	Events   []hub.Event
	Revision int64
}

// CatchUp 返回设备从 lastRevision 继续所需的内容。以下情况发送全量快照：
//   - lastRevision 为 0：客户端没有本地数据。
//   - lastRevision 大于当前序号：服务端数据被恢复到更早的状态，客户端的数据不再可信。
//   - lastRevision 之后的第一条记录已被清理：客户端落后超过保留范围。
func (s *syncService) CatchUp(ctx context.Context, userID, lastRevision int64) (*CatchUp, error) {
	result := new(CatchUp)
	err := repository.ReadTransaction(ctx, func(ctx context.Context) error {
		user, err := User.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		result.Revision = user.Revision
		if lastRevision == user.Revision && lastRevision != 0 {
			return nil
		}
		if lastRevision > 0 && lastRevision < user.Revision {
			logs, err := repository.ChangeLog.ListAfter(ctx, userID, lastRevision)
			if err != nil {
				return err
			}
			if len(logs) > 0 && logs[0].Revision == lastRevision+1 {
				result.Events = make([]hub.Event, 0, len(logs))
				for _, log := range logs {
					result.Events = append(result.Events, hub.Event{
						Revision: log.Revision, Op: log.Op, Type: log.Type, ID: log.EntityID, Data: json.RawMessage(log.Data),
					})
				}
				return nil
			}
		}
		result.Snapshot, err = s.snapshot(ctx, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// snapshot 读取用户的全部可同步数据。必须与读取序号处在同一个只读事务中。
func (s *syncService) snapshot(ctx context.Context, userID int64) (*dro.Snapshot, error) {
	user, err := User.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	devices, err := repository.Device.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	folders, err := repository.Folder.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	boards, err := repository.Board.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	panels, err := repository.Panel.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	labels, err := repository.Label.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	levels, err := repository.PriorityLevel.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	channels, err := repository.Notify.ListChannels(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dro.Snapshot{
		NotifyChannels: dro.NewNotifyChannels(channels),
		Account:        dro.NewAccount(user),
		Settings:       dro.NewSettings(user),
		Devices:        dro.NewDevices(devices),
		Folders:        dro.NewFolders(folders),
		Boards:         dro.NewBoards(boards),
		Panels:         dro.NewPanels(panels),
		Labels:         dro.NewLabels(labels),
		PriorityLevels: dro.NewPriorityLevels(levels),
	}, nil
}
