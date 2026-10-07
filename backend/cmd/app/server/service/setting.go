package service

import (
	"context"
	"kanban/cmd/app/server/common/notifytemplate"
	"kanban/cmd/app/server/common/passwd"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/common/timezone"
	"kanban/cmd/app/server/common/types/editormode"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/common/types/panmodifier"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"unicode/utf8"
)

// Setting 是用户修改自己个人设置和密码的服务。
var Setting = new(settingService)

type settingService struct{}

// SettingChanges 是一次个人设置修改。字段为 nil 表示不修改该项，因此各项可以分别保存。
type SettingChanges struct {
	Timezone            *string
	Palette             *palette.Type
	Locale              *locale.Type
	PanModifier         *panmodifier.Type
	OpenMainBoardOnHome *bool
	RemindTemplate      *string
	DueTemplate         *string
	Shortcuts           shortcut.Bindings // 只包含要修改的操作；为 nil 表示不修改快捷键
	EditorMode          *editormode.Type
}

// validateTemplate 检查正文模板长度和编码。
func validateTemplate(template string) error {
	if !utf8.ValidString(template) {
		return badRequest(resp.TemplateCharset, "正文模板包含无效字符")
	}
	if utf8.RuneCountInString(template) > notifytemplate.MaxLength {
		return badRequest(resp.TemplateTooLong, "正文模板不能超过 4000 个字符")
	}
	return nil
}

// Update 校验并保存个人设置，返回保存后的用户。
func (s *settingService) Update(ctx context.Context, userID int64, changes SettingChanges) (*model.User, error) {
	values := map[string]any{}
	if changes.Timezone != nil {
		if !timezone.Valid(*changes.Timezone) {
			return nil, badRequest(resp.TimezoneInvalid, "时区不正确")
		}
		values["timezone"] = *changes.Timezone
	}
	if changes.Palette != nil {
		if !changes.Palette.Valid() {
			return nil, badRequest(resp.PaletteInvalid, "配色不正确")
		}
		values["palette"] = *changes.Palette
	}
	if changes.Locale != nil {
		if !changes.Locale.Valid() {
			return nil, badRequest(resp.LocaleInvalid, "语言不正确")
		}
		values["locale"] = *changes.Locale
	}
	if changes.PanModifier != nil {
		if !changes.PanModifier.Valid() {
			return nil, badRequest(resp.PanModifierInvalid, "拖动平移修饰键不正确")
		}
		values["pan_modifier"] = *changes.PanModifier
	}
	if changes.EditorMode != nil {
		if !changes.EditorMode.Valid() {
			return nil, badRequest(resp.EditorModeInvalid, "编辑器模式不正确")
		}
		values["editor_mode"] = *changes.EditorMode
	}
	if changes.OpenMainBoardOnHome != nil {
		values["open_main_board_on_home"] = *changes.OpenMainBoardOnHome
	}
	if changes.RemindTemplate != nil {
		if err := validateTemplate(*changes.RemindTemplate); err != nil {
			return nil, err
		}
		values["remind_template"] = *changes.RemindTemplate
	}
	if changes.DueTemplate != nil {
		if err := validateTemplate(*changes.DueTemplate); err != nil {
			return nil, err
		}
		values["due_template"] = *changes.DueTemplate
	}

	var user *model.User
	err := write(ctx, func(ctx context.Context) error {
		current, err := User.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		if changes.Shortcuts != nil {
			overrides, err := shortcut.Apply(current.Shortcuts, changes.Shortcuts)
			if shortcut.IsConflict(err) {
				return conflict(resp.ShortcutConflict, err.Error())
			}
			if err != nil {
				return badRequest(resp.ShortcutInvalid, err.Error())
			}
			if err = repository.User.UpdateShortcuts(ctx, userID, overrides); err != nil {
				return err
			}
		}
		if len(values) > 0 {
			if err = repository.User.Update(ctx, userID, values); err != nil {
				return err
			}
		}
		user, err = recordSettings(ctx, userID)
		return err
	})
	return user, err
}

// ChangePassword 校验当前密码后修改密码，并吊销该用户除当前设备外的全部令牌。
func (s *settingService) ChangePassword(ctx context.Context, session *Session, currentPassword, newPassword string) error {
	if err := passwd.Validate(newPassword); err != nil {
		return passwordInvalid(err)
	}
	hash, err := passwd.Hash(newPassword)
	if err != nil {
		return err
	}
	return write(ctx, func(ctx context.Context) error {
		user, err := User.FindByID(ctx, session.User.ID)
		if err != nil {
			return err
		}
		if !passwd.Match(user.PasswordHash, currentPassword) {
			return badRequest(resp.CurrentPasswordWrong, "当前密码不正确")
		}
		if err = repository.User.Update(ctx, user.ID, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		return revokeDevicesExcept(ctx, user.ID, session.Device.ID)
	})
}
