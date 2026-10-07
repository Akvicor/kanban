package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"time"

	"github.com/Akvicor/glog"
	"gorm.io/gorm"
)

// Device 是当前用户管理自己已登录设备的服务。
var Device = new(deviceService)

type deviceService struct{}

// Revoke 踢掉用户自己的某台设备。设备不属于该用户时按不存在处理。
func (s *deviceService) Revoke(ctx context.Context, userID, deviceID int64) error {
	return write(ctx, func(ctx context.Context) error {
		err := revokeDevice(ctx, userID, deviceID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound(resp.DeviceNotFound, "设备不存在")
		}
		return err
	})
}

// recordDevice 记录设备新增或变化。
func recordDevice(ctx context.Context, device *model.Device) error {
	return recordChange(ctx, device.UserID, hub.OpUpsert, EntityDevice, device.ID, dro.NewDevice(device))
}

// revokeDevice 删除设备记录并记录变更，提交后断开该设备的连接。必须在 write 中调用。
func revokeDevice(ctx context.Context, userID, deviceID int64) error {
	if err := repository.Device.Delete(ctx, userID, deviceID); err != nil {
		return err
	}
	if err := recordChange(ctx, userID, hub.OpDelete, EntityDevice, deviceID, nil); err != nil {
		return err
	}
	return afterCommit(ctx, func() { hub.Default.DisconnectDevice(userID, deviceID) })
}

// revokeDevicesExcept 吊销用户除 keepID 之外的全部设备；keepID 为 0 时吊销全部。必须在 write 中调用。
func revokeDevicesExcept(ctx context.Context, userID, keepID int64) error {
	devices, err := repository.Device.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if device.ID == keepID {
			continue
		}
		if err = revokeDevice(ctx, userID, device.ID); err != nil {
			return err
		}
	}
	return nil
}

// deviceIdleExpiry 是设备的闲置期限：最后活跃时间早于 now - deviceIdleExpiry 的设备令牌失效，设备被删除。
// 设备的每次请求和同步连接都会刷新最后活跃时间（见 authService.Authenticate），
// 因此只有长期不用、可能已被遗忘的设备会过期。
const deviceIdleExpiry = 90 * 24 * time.Hour

// deviceJobInterval 是后台清理闲置设备的间隔。
const deviceJobInterval = time.Hour

// deviceIdle 判断设备在 now 时是否已闲置超过期限。
func deviceIdle(device *model.Device, now time.Time) bool {
	return device.LastActiveAt.Before(now.Add(-deviceIdleExpiry))
}

// expireDevice 删除一台闲置设备并记录变更、断开连接。设备已被删除时不算错误。
func expireDevice(ctx context.Context, device *model.Device) error {
	return write(ctx, func(ctx context.Context) error {
		err := revokeDevice(ctx, device.UserID, device.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	})
}

// ExpireIdleDevices 删除所有用户中闲置超过期限的设备，设备列表通过同步移除这些设备。
func (s *deviceService) ExpireIdleDevices(ctx context.Context, now time.Time) error {
	devices, err := repository.Device.ListIdleBefore(ctx, now.Add(-deviceIdleExpiry))
	if err != nil {
		return err
	}
	for _, device := range devices {
		if err = expireDevice(ctx, device); err != nil {
			return err
		}
	}
	return nil
}

// StartDeviceJobs 启动设备相关的后台任务：启动时和之后每小时清理一次闲置设备。
func StartDeviceJobs(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(deviceJobInterval)
		defer ticker.Stop()
		for {
			if err := Device.ExpireIdleDevices(ctx, time.Now().UTC()); err != nil {
				glog.Error("清理闲置设备失败: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
