package model

import (
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/common/types/editormode"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/common/types/panmodifier"
	"kanban/cmd/app/server/common/types/role"
	"time"
)

// User 是账号及其个人设置。用户是数据隔离边界，其他业务表都通过 user_id 归属到某个用户。
type User struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	Username string `gorm:"column:username;size:64;not null;uniqueIndex"`
	// Nickname 是显示名，不用于登录，不必唯一。为空时界面显示用户名。
	Nickname     string     `gorm:"column:nickname;size:64;not null;default:''"`
	PasswordHash string     `gorm:"column:password_hash;size:128;not null"`
	Role         role.Type  `gorm:"column:role;size:16;not null"`
	DisabledAt   *time.Time `gorm:"column:disabled_at"` // 不为空表示已停用，停用的账号不能登录
	CreatedAt    time.Time  `gorm:"column:created_at;not null"`
	// Revision 是该用户最近一次变更的同步序号。每次写入 change_log 前在同一事务中加一，
	// 更新这一行同时锁住它，因此同一用户的并发写入按提交顺序得到连续的序号。
	Revision int64 `gorm:"column:revision;not null;default:0"`

	// 个人设置
	Timezone string       `gorm:"column:timezone;size:64;not null"` // IANA 时区名，日期边界和时间显示按它换算
	Palette  palette.Type `gorm:"column:palette;size:16;not null"`
	// Locale 是界面与通知语言；空值表示跟随系统（前端按浏览器语言解析，后端文案回落默认语言）。
	// 列带默认值，已有用户迁移后为空串。
	Locale              locale.Type       `gorm:"column:locale;size:16;not null;default:''"`
	MainBoardID         *int64            `gorm:"column:main_board_id"`                       // 主看板，必须是该用户未归档的看板；看板归档时清空
	OpenMainBoardOnHome bool              `gorm:"column:open_main_board_on_home;not null"`    // 进入主页时是否打开主看板
	RemindTemplate      string            `gorm:"column:remind_template;type:text;not null"`  // 提醒通知正文模板
	DueTemplate         string            `gorm:"column:due_template;type:text;not null"`     // 截止通知正文模板
	Shortcuts           shortcut.Bindings `gorm:"column:shortcuts;type:text;serializer:json"` // 只保存与默认值不同的快捷键绑定
	// EditorMode 是卡片描述编辑器上次使用的模式。列带默认值，已有用户迁移后为所见即所得。
	EditorMode editormode.Type `gorm:"column:editor_mode;size:16;not null;default:wysiwyg"`
	// PanModifier 是「按住修饰键拖动平移看板」使用的修饰键；空值表示关闭该手势。
	PanModifier panmodifier.Type `gorm:"column:pan_modifier;size:16;not null;default:ctrl"`
}

// TableName 返回表名。
func (*User) TableName() string {
	return "users"
}

// Disabled 判断账号是否已停用。
func (u *User) Disabled() bool {
	return u.DisabledAt != nil
}

// DisplayName 返回界面上显示的名字：有昵称用昵称，否则用用户名。
func (u *User) DisplayName() string {
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}
