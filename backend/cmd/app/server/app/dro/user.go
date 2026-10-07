package dro

import (
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/common/types/editormode"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/common/types/panmodifier"
	"kanban/cmd/app/server/common/types/role"
	"kanban/cmd/app/server/model"
	"time"
)

// Account 是账号的公开信息，不含密码哈希。
type Account struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	Nickname string    `json:"nickname"` // 显示名，为空时与用户名相同
	Role     role.Type `json:"role"`
}

// Settings 是用户的个人设置。Shortcuts 是合并默认值后实际生效的完整绑定。
type Settings struct {
	Timezone            string            `json:"timezone"`
	Palette             palette.Type      `json:"palette"`
	Locale              locale.Type       `json:"locale"`       // 界面与通知语言；空值表示跟随系统
	PanModifier         panmodifier.Type  `json:"pan_modifier"` // 拖动平移看板的修饰键；空值表示关闭手势
	MainBoardID         *int64            `json:"main_board_id"`
	OpenMainBoardOnHome bool              `json:"open_main_board_on_home"`
	RemindTemplate      string            `json:"remind_template"`
	DueTemplate         string            `json:"due_template"`
	Shortcuts           shortcut.Bindings `json:"shortcuts"`
	EditorMode          editormode.Type   `json:"editor_mode"` // 卡片描述编辑器上次使用的模式
}

// Me 是当前登录用户的信息：账号、个人设置、当前设备，以及快捷键的默认绑定和展示顺序。
// 个人设置在登录时随 Me 返回，之后以同步推送为准。
type Me struct {
	Account          Account           `json:"account"`
	Settings         Settings          `json:"settings"`
	DeviceID         int64             `json:"device_id"`
	ShortcutDefaults shortcut.Bindings `json:"shortcut_defaults"`
	ShortcutActions  []shortcut.Action `json:"shortcut_actions"`
}

// LoginResult 是登录响应，Token 只在这里返回一次。
type LoginResult struct {
	Token string `json:"token"`
	Me    Me     `json:"me"`
}

// Device 是一台登录设备。客户端用 Me.DeviceID 判断哪一台是自己。
type Device struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// Snapshot 是同步用的全量数据：某个序号时该用户的全部可同步数据。
// 客户端没有本地数据，或落后超过变更记录的保留范围时，用它整体替换本地数据。
// 目录、看板归档和面板归档都由下面的列表得出，列表包含已归档的文件夹、看板和面板。
type Snapshot struct {
	Account        Account         `json:"account"`
	Settings       Settings        `json:"settings"`
	Devices        []Device        `json:"devices"`
	Folders        []Folder        `json:"folders"`
	Boards         []Board         `json:"boards"`
	Panels         []Panel         `json:"panels"`
	Labels         []Label         `json:"labels"`
	PriorityLevels []PriorityLevel `json:"priority_levels"`
	NotifyChannels []NotifyChannel `json:"notify_channels"`
}

// AdminUser 是管理员看到的账号信息。
type AdminUser struct {
	ID         int64      `json:"id"`
	Username   string     `json:"username"`
	Nickname   string     `json:"nickname"`
	Role       role.Type  `json:"role"`
	CreatedAt  time.Time  `json:"created_at"`
	DisabledAt *time.Time `json:"disabled_at"`
}

// NewAccount 从用户模型生成账号信息。
func NewAccount(user *model.User) Account {
	return Account{ID: user.ID, Username: user.Username, Nickname: user.DisplayName(), Role: user.Role}
}

// NewSettings 从用户模型生成个人设置。
func NewSettings(user *model.User) Settings {
	return Settings{
		Timezone:            user.Timezone,
		Palette:             user.Palette,
		Locale:              user.Locale,
		MainBoardID:         user.MainBoardID,
		OpenMainBoardOnHome: user.OpenMainBoardOnHome,
		RemindTemplate:      user.RemindTemplate,
		DueTemplate:         user.DueTemplate,
		Shortcuts:           shortcut.Effective(user.Shortcuts),
		EditorMode:          user.EditorMode,
		PanModifier:         user.PanModifier,
	}
}

// NewMe 生成当前登录用户的信息。
func NewMe(user *model.User, deviceID int64) Me {
	return Me{
		Account:          NewAccount(user),
		Settings:         NewSettings(user),
		DeviceID:         deviceID,
		ShortcutDefaults: shortcut.Defaults(),
		ShortcutActions:  shortcut.Actions(),
	}
}

// NewDevice 从设备模型生成设备信息，不含令牌哈希。
func NewDevice(device *model.Device) Device {
	return Device{
		ID:           device.ID,
		Name:         device.Name,
		CreatedAt:    device.CreatedAt,
		LastActiveAt: device.LastActiveAt,
	}
}

// NewDevices 生成设备列表。
func NewDevices(devices []*model.Device) []Device {
	list := make([]Device, 0, len(devices))
	for _, device := range devices {
		list = append(list, NewDevice(device))
	}
	return list
}

// NewAdminUsers 生成管理员看到的账号列表。
func NewAdminUsers(users []*model.User) []AdminUser {
	list := make([]AdminUser, 0, len(users))
	for _, user := range users {
		list = append(list, AdminUser{
			ID:         user.ID,
			Username:   user.Username,
			Nickname:   user.DisplayName(),
			Role:       user.Role,
			CreatedAt:  user.CreatedAt,
			DisabledAt: user.DisabledAt,
		})
	}
	return list
}
