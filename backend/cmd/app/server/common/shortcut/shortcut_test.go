package shortcut

import "testing"

func TestDefaultsAreValid(t *testing.T) {
	defaults := Defaults()
	if err := Validate(defaults); err != nil {
		t.Fatalf("默认绑定校验失败: %v", err)
	}
	for _, action := range Actions() {
		if _, ok := defaults[action]; !ok {
			t.Errorf("操作 %s 没有默认绑定", action)
		}
	}
}

func TestValidKey(t *testing.T) {
	for _, key := range []string{"E", "0", "Enter", "F12", "Mod+C", "Mod+Shift+K", "Alt+Shift+1", "ArrowUp"} {
		if !ValidKey(key) {
			t.Errorf("ValidKey(%q) = false", key)
		}
	}
	for _, key := range []string{"", "e", "Ctrl+C", "Shift+Mod+C", "Mod+Mod+C", "Escape", "Tab", "F13", "Mod+", "+C"} {
		if ValidKey(key) {
			t.Errorf("ValidKey(%q) = true", key)
		}
	}
}

func TestApplyStoresOnlyDifferences(t *testing.T) {
	overrides, err := Apply(nil, Bindings{EditTitle: {"R"}, OpenCard: {"Enter", "E"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(overrides) != 1 || overrides[EditTitle][0] != "R" {
		t.Fatalf("差异 = %v，只应包含修改过的操作", overrides)
	}

	// 改回默认值后不再保存差异。
	overrides, err = Apply(overrides, Bindings{EditTitle: {"T"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(overrides) != 0 {
		t.Fatalf("改回默认值后差异 = %v", overrides)
	}
}

func TestApplyRejectsConflictsAndUnknownActions(t *testing.T) {
	if _, err := Apply(nil, Bindings{EditTitle: {"L"}}); !IsConflict(err) {
		t.Fatalf("与「编辑卡片标签」的 L 冲突时 err = %v", err)
	}
	// 先把 L 让出来，再绑定给另一个操作，可以保存。
	overrides, err := Apply(nil, Bindings{EditLabels: {}, EditTitle: {"L"}})
	if err != nil {
		t.Fatalf("让出按键后仍报错: %v", err)
	}
	if got := Effective(overrides)[EditLabels]; len(got) != 0 {
		t.Fatalf("EditLabels = %v，应为未绑定", got)
	}
	if _, err = Apply(nil, Bindings{"unknown": {"Q"}}); err == nil {
		t.Fatal("未知操作未报错")
	}
	if _, err = Apply(nil, Bindings{EditTitle: {"Q", "W", "Y", "U"}}); err == nil {
		t.Fatal("超过按键数上限未报错")
	}
}
