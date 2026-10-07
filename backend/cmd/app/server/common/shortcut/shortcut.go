// Package shortcut 定义快捷键操作、默认绑定和按键格式，供个人设置校验和保存。
//
// 按键使用规范化的字符串：修饰键按 Mod、Alt、Shift 的顺序写在前面，用 + 连接，最后是主键，
// 例如 E、Enter、Mod+C、Mod+Shift+K。Mod 在 Windows/Linux 上是 Ctrl，在 macOS 上是 Cmd。
// 前端 src/shortcuts/keys.ts 按同一格式把键盘事件转换成按键字符串。
package shortcut

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Action 是可以绑定快捷键的操作。
type Action string

const (
	OpenCard    Action = "open_card"    // 打开卡片
	EditLabels  Action = "edit_labels"  // 编辑卡片标签
	EditTitle   Action = "edit_title"   // 修改卡片标题
	ArchiveCard Action = "archive_card" // 归档卡片
	CopyCard    Action = "copy_card"    // 复制卡片
	CutCard     Action = "cut_card"     // 剪切卡片
	PasteCard   Action = "paste_card"   // 粘贴到悬停的列
)

// ToggleLabel 返回「切换卡片上的第 n 个标签」操作，n 取 1 到 10。
func ToggleLabel(n int) Action {
	return Action(fmt.Sprintf("toggle_label_%d", n))
}

// MaxKeysPerAction 是一个操作最多绑定的按键数。
const MaxKeysPerAction = 3

// Bindings 是操作到按键列表的映射。
type Bindings map[Action][]string

// actions 是全部操作，顺序即设置界面中的展示顺序。
var actions = func() []Action {
	list := []Action{OpenCard, EditLabels, EditTitle, ArchiveCard}
	for n := 1; n <= 10; n++ {
		list = append(list, ToggleLabel(n))
	}
	return append(list, CopyCard, CutCard, PasteCard)
}()

// Actions 返回全部操作的副本。
func Actions() []Action {
	return slices.Clone(actions)
}

// Defaults 返回新用户的默认绑定，与参考项目一致并去掉成员相关操作。
func Defaults() Bindings {
	defaults := Bindings{
		OpenCard:    {"E", "Enter"},
		EditLabels:  {"L"},
		EditTitle:   {"T"},
		ArchiveCard: {"V"},
		CopyCard:    {"Mod+C"},
		CutCard:     {"Mod+X"},
		PasteCard:   {"Mod+V"},
	}
	for n := 1; n <= 9; n++ {
		defaults[ToggleLabel(n)] = []string{fmt.Sprint(n)}
	}
	defaults[ToggleLabel(10)] = []string{"0"}
	return defaults
}

// namedKeys 是除字母、数字和 F1–F12 之外允许作为主键的按键。
// Escape 和 Tab 用于关闭弹窗和切换焦点，不允许绑定。
var namedKeys = []string{"Enter", "Space", "Backspace", "Delete", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"}

// modifiers 是允许的修饰键，顺序即规范顺序。
var modifiers = []string{"Mod", "Alt", "Shift"}

// ValidKey 判断按键字符串是否符合规范格式。
func ValidKey(key string) bool {
	parts := strings.Split(key, "+")
	main := parts[len(parts)-1]
	// 修饰键必须按规范顺序出现，且不重复。
	next := 0
	for _, part := range parts[:len(parts)-1] {
		index := slices.Index(modifiers, part)
		if index < next {
			return false
		}
		next = index + 1
	}
	return validMainKey(main)
}

func validMainKey(key string) bool {
	if len(key) == 1 {
		c := key[0]
		return (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
	}
	if strings.HasPrefix(key, "F") {
		switch key[1:] {
		case "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12":
			return true
		}
	}
	return slices.Contains(namedKeys, key)
}

// Effective 把保存的差异合并到默认绑定上，得到实际生效的绑定。
func Effective(overrides Bindings) Bindings {
	effective := Defaults()
	for action, keys := range overrides {
		if _, ok := effective[action]; ok {
			effective[action] = slices.Clone(keys)
		}
	}
	return effective
}

// Apply 把 changes 中的操作改成新的按键，校验合并后的完整绑定，返回需要保存的差异。
// 差异只包含与默认值不同的操作；按键列表为空表示该操作不绑定任何按键。
func Apply(overrides, changes Bindings) (Bindings, error) {
	effective := Effective(overrides)
	for action, keys := range changes {
		if _, ok := effective[action]; !ok {
			return nil, fmt.Errorf("未知的快捷键操作: %s", action)
		}
		effective[action] = slices.Clone(keys)
	}
	if err := Validate(effective); err != nil {
		return nil, err
	}

	defaults := Defaults()
	result := Bindings{}
	for action, keys := range effective {
		if !sameKeys(keys, defaults[action]) {
			result[action] = keys
		}
	}
	return result, nil
}

// Validate 校验完整绑定：按键格式合法、每个操作的按键数不超限、同一按键不绑定两个操作。
func Validate(bindings Bindings) error {
	owner := map[string]Action{}
	for _, action := range actions {
		keys := bindings[action]
		if len(keys) > MaxKeysPerAction {
			return fmt.Errorf("每个操作最多绑定 %d 个按键", MaxKeysPerAction)
		}
		for _, key := range keys {
			if !ValidKey(key) {
				return fmt.Errorf("按键格式不正确: %s", key)
			}
			if other, ok := owner[key]; ok {
				if other == action {
					return fmt.Errorf("按键 %s 重复", key)
				}
				return &ConflictError{Key: key, First: other, Second: action}
			}
			owner[key] = action
		}
	}
	return nil
}

// ConflictError 表示同一按键被绑定到两个操作。
type ConflictError struct {
	Key           string
	First, Second Action
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("按键 %s 已绑定到其他操作", e.Key)
}

// IsConflict 判断错误是否为按键冲突。
func IsConflict(err error) bool {
	var conflict *ConflictError
	return errors.As(err, &conflict)
}

// sameKeys 判断两组按键是否相同，不考虑顺序。
func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA, sortedB := slices.Clone(a), slices.Clone(b)
	slices.Sort(sortedA)
	slices.Sort(sortedB)
	return slices.Equal(sortedA, sortedB)
}
