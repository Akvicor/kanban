// Package notifytemplate 定义提醒和截止通知的正文模板。
package notifytemplate

import "kanban/cmd/app/server/common/types/locale"

// 默认正文模板按语言各有一份：用户没有自定义模板时使用，随用户语言切换。
// 占位符带 |markdown，发到 Markdown 渠道时会被转义，因此在两种渠道中都能安全发送。
const (
	defaultRemindZhCN = "{{title|markdown}}\n提醒时间：{{remind_at|markdown}}"
	defaultDueZhCN    = "{{title|markdown}}\n截止时间：{{due_at|markdown}}"
	defaultRemindEn   = "{{title|markdown}}\nReminder: {{remind_at|markdown}}"
	defaultDueEn      = "{{title|markdown}}\nDue: {{due_at|markdown}}"
)

// MaxLength 是单个模板的最大字符数。
const MaxLength = 4000

// DefaultRemind 返回该语言的提醒正文默认模板，不受支持的语言回落默认语言。
func DefaultRemind(lang string) string {
	if locale.Type(lang) == locale.En {
		return defaultRemindEn
	}
	return defaultRemindZhCN
}

// DefaultDue 返回该语言的截止正文默认模板，不受支持的语言回落默认语言。
func DefaultDue(lang string) string {
	if locale.Type(lang) == locale.En {
		return defaultDueEn
	}
	return defaultDueZhCN
}
