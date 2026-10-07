package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/common/passwd"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/token"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// Auth 是登录、令牌校验和登出服务。
var Auth = new(authService)

type authService struct{}

// 设备名最多 64 个字符，为空时使用默认名称。
const deviceNameMaxLength = 64

// defaultDeviceName 返回设备名为空时使用的默认名称，按用户语言取用，不受支持的语言回落默认语言。
func defaultDeviceName(lang string) string {
	if locale.Type(lang) == locale.En {
		return "Unknown device"
	}
	return "未知设备"
}

// touchInterval 是更新设备最后活跃时间的最小间隔，避免每个请求都写库。
const touchInterval = time.Minute

// dummyHash 用于用户名不存在时仍做一次等价的密码比对，使响应时间不暴露用户名是否存在。
var dummyHash, _ = passwd.Hash("kanban-dummy-password")

// errLoginFailed 是用户名不存在、密码错误、账号停用时统一返回的错误。
var errLoginFailed = unauthorized(resp.LoginInvalid, "用户名或密码不正确，或账号已停用")

// Session 是一次已认证的访问：当前用户和发起请求的设备。
type Session struct {
	User   *model.User
	Device *model.Device
}

// normalizeDeviceName 去掉首尾空白并截断过长的设备名，空名使用该语言的默认名称。
func normalizeDeviceName(name, lang string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultDeviceName(lang)
	}
	if utf8.RuneCountInString(name) > deviceNameMaxLength {
		name = string([]rune(name)[:deviceNameMaxLength])
	}
	return name
}

// Login 校验用户名和密码，为这台设备签发新令牌。返回的令牌明文只在此处出现一次。
func (s *authService) Login(ctx context.Context, username, password, deviceName string) (string, *Session, error) {
	user, err := repository.User.FindByUsername(ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		passwd.Match(dummyHash, password)
		return "", nil, errLoginFailed
	}
	if err != nil {
		return "", nil, err
	}
	if !passwd.Match(user.PasswordHash, password) || user.Disabled() {
		return "", nil, errLoginFailed
	}

	plain := token.New()
	now := time.Now().UTC()
	device := &model.Device{
		UserID:       user.ID,
		Name:         normalizeDeviceName(deviceName, uiLang(ctx, user)),
		TokenHash:    token.Hash(plain),
		CreatedAt:    now,
		LastActiveAt: now,
	}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.Device.Create(ctx, device); err != nil {
			return err
		}
		return recordDevice(ctx, device)
	})
	if err != nil {
		return "", nil, err
	}
	return plain, &Session{User: user, Device: device}, nil
}

// Authenticate 按令牌明文找到设备和用户。令牌已吊销、设备闲置超过期限、账号已停用或不存在时返回未登录错误；
// 闲置超过期限的设备在这里被删除。
func (s *authService) Authenticate(ctx context.Context, plain string) (*Session, error) {
	if plain == "" {
		return nil, unauthorized(resp.NotLoggedIn, "请先登录")
	}
	device, err := repository.Device.FindByTokenHash(ctx, token.Hash(plain))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, unauthorized(resp.SessionExpired, "登录已失效，请重新登录")
	}
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if deviceIdle(device, now) {
		if err = expireDevice(ctx, device); err != nil {
			return nil, err
		}
		return nil, unauthorized(resp.SessionExpired, "登录已失效，请重新登录")
	}
	user, err := repository.User.FindByID(ctx, device.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, unauthorized(resp.SessionExpired, "登录已失效，请重新登录")
	}
	if err != nil {
		return nil, err
	}
	if user.Disabled() {
		return nil, unauthorized(resp.AccountDisabled, "账号已停用")
	}

	// 最后活跃时间在设备列表中参考，也用于判断设备是否闲置过期；更新它不记录变更、不推送，避免每分钟产生一次同步。
	if now.Sub(device.LastActiveAt) >= touchInterval {
		if err = repository.Device.Touch(ctx, device.ID, now); err != nil {
			return nil, err
		}
		device.LastActiveAt = now
	}
	return &Session{User: user, Device: device}, nil
}

// Logout 吊销当前设备的令牌。
func (s *authService) Logout(ctx context.Context, session *Session) error {
	return write(ctx, func(ctx context.Context) error {
		err := revokeDevice(ctx, session.User.ID, session.Device.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	})
}
