package service

import (
	"context"
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/common/types/editormode"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/testutil/dbtest"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestUpdateSettings(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")

		user, err := Setting.Update(ctx, alice.ID, SettingChanges{
			Timezone:            ptr("America/New_York"),
			Palette:             ptr(palette.System),
			OpenMainBoardOnHome: ptr(true),
			RemindTemplate:      ptr("提醒：{{title}}"),
			EditorMode:          ptr(editormode.Markup),
		})
		if err != nil {
			t.Fatal(err)
		}
		if user.Timezone != "America/New_York" || user.Palette != palette.System || !user.OpenMainBoardOnHome ||
			user.RemindTemplate != "提醒：{{title}}" || user.EditorMode != editormode.Markup {
			t.Fatalf("保存后的设置 = %+v", user)
		}

		// 只修改一项时，其他设置保持不变；布尔值可以改回 false。
		user, err = Setting.Update(ctx, alice.ID, SettingChanges{OpenMainBoardOnHome: ptr(false)})
		if err != nil {
			t.Fatal(err)
		}
		if user.OpenMainBoardOnHome || user.Palette != palette.System {
			t.Fatalf("部分修改后的设置 = %+v", user)
		}

		invalid := []SettingChanges{
			{Timezone: ptr("Mars/Base")},
			{Timezone: ptr("Local")},
			{Palette: ptr(palette.Type("neon"))},
			{EditorMode: ptr(editormode.Type("rich"))},
			{DueTemplate: ptr(strings.Repeat("长", 4001))},
		}
		for _, changes := range invalid {
			if _, err = Setting.Update(ctx, alice.ID, changes); errorKind(err) != KindBadRequest {
				t.Fatalf("Update(%+v) err = %v", changes, err)
			}
		}
	})
}

func TestUpdateShortcuts(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")

		user, err := Setting.Update(ctx, alice.ID, SettingChanges{Shortcuts: shortcut.Bindings{shortcut.EditTitle: {"R"}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(user.Shortcuts) != 1 || user.Shortcuts[shortcut.EditTitle][0] != "R" {
			t.Fatalf("保存的差异 = %v", user.Shortcuts)
		}

		// 冲突时不保存，原来的绑定保持不变。
		if _, err = Setting.Update(ctx, alice.ID, SettingChanges{Shortcuts: shortcut.Bindings{shortcut.ArchiveCard: {"R"}}}); errorKind(err) != KindConflict {
			t.Fatalf("按键冲突 err = %v", err)
		}
		if _, err = Setting.Update(ctx, alice.ID, SettingChanges{Shortcuts: shortcut.Bindings{shortcut.ArchiveCard: {"ctrl+r"}}}); errorKind(err) != KindBadRequest {
			t.Fatalf("按键格式错误 err = %v", err)
		}
		saved, err := User.FindByID(ctx, alice.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got := shortcut.Effective(saved.Shortcuts); got[shortcut.EditTitle][0] != "R" || got[shortcut.ArchiveCard][0] != "V" {
			t.Fatalf("失败的修改改动了已保存的绑定: %v", got)
		}

		// 改回默认值后不再保存差异。
		user, err = Setting.Update(ctx, alice.ID, SettingChanges{Shortcuts: shortcut.Bindings{shortcut.EditTitle: {"T"}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(user.Shortcuts) != 0 {
			t.Fatalf("改回默认后差异 = %v", user.Shortcuts)
		}
	})
}
