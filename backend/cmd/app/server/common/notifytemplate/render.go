package notifytemplate

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Akvicor/gmsg"
)

// Fields 是正文模板中可用的占位符，与前端 src/pages/settings/TemplateSection.tsx 的说明一致。
var Fields = []string{
	"title", "description", "board", "panel", "list", "priority", "labels",
	"remind_at", "due_at", "created_at", "started_at", "completed_at", "tasks",
}

const (
	// MaxFieldLength 是单个占位符替换值的最大字符数，超过时截断并加省略号。
	MaxFieldLength = 1000
	// MaxMessageLength 是整条正文的最大字符数，超过时截断并加省略号。
	MaxMessageLength = 4000
	ellipsis         = "…"
)

// placeholder 匹配 {{字段}} 和 {{字段|属性}}，名字和属性两侧允许空格。
var placeholder = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*(?:\|\s*([a-z_]+)\s*)?\}\}`)

var known = func() map[string]bool {
	set := map[string]bool{}
	for _, name := range Fields {
		set[name] = true
	}
	return set
}()

// parse 判断一个占位符是否有效：字段名认识，属性为空、raw 或 markdown。escape 表示属性为 markdown。
func parse(match []string) (name string, escape bool, ok bool) {
	name, attribute := match[1], match[2]
	if !known[name] || (attribute != "" && attribute != "raw" && attribute != "markdown") {
		return "", false, false
	}
	return name, attribute == "markdown", true
}

// truncate 把文字截断为最多 limit 个字符，被截断时末尾加省略号（省略号计入长度）。
func truncate(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit-1]) + ellipsis
}

// Render 按模板生成正文。values 是各字段的替换值，缺少的字段替换为空字符串。
// markdown 为 true 表示发到 Markdown 渠道：带 |markdown 的值用 gmsg.Telegram.Escape 转义；
// 发到文本渠道时所有值原样插入。不认识的占位符和属性保持原样。
// 模板中没有 {{title}} 时，在开头补上标题，按 {{title|markdown}} 处理。
// 单个值超过 1000 个字符时在转义前截断；整条正文超过 4000 个字符时再截断，Markdown 渠道去掉末尾落单的反斜杠。
func Render(template string, values map[string]string, markdown bool) string {
	value := func(name string, escape bool) string {
		text := truncate(values[name], MaxFieldLength)
		if markdown && escape {
			text = gmsg.Telegram.Escape(text)
		}
		return text
	}

	hasTitle := false
	body := placeholder.ReplaceAllStringFunc(template, func(token string) string {
		name, escape, ok := parse(placeholder.FindStringSubmatch(token))
		if !ok {
			return token
		}
		if name == "title" {
			hasTitle = true
		}
		return value(name, escape)
	})
	if !hasTitle {
		body = value("title", true) + "\n" + body
	}

	if utf8.RuneCountInString(body) > MaxMessageLength {
		body = string([]rune(body)[:MaxMessageLength-1])
		if markdown {
			body = strings.TrimRight(body, `\`)
		}
		body += ellipsis
	}
	return body
}
