package notifytemplate

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderPlaceholdersAndEscaping(t *testing.T) {
	values := map[string]string{"title": "发布 v1.2 (beta)", "list": "进行中", "due_at": "2026-10-03 18:00"}
	template := "{{title|markdown}} 在 {{ list }}\n截止：{{ due_at | markdown }} {{unknown}} {{list|html}} {{tasks}}"

	text := Render(template, values, false)
	if want := "发布 v1.2 (beta) 在 进行中\n截止：2026-10-03 18:00 {{unknown}} {{list|html}} "; text != want {
		t.Fatalf("文本渠道 = %q", text)
	}
	md := Render(template, values, true)
	if want := "发布 v1\\.2 \\(beta\\) 在 进行中\n截止：2026\\-10\\-03 18:00 {{unknown}} {{list|html}} "; md != want {
		t.Fatalf("Markdown 渠道 = %q", md)
	}
	// 不带属性和 |raw 的值在 Markdown 渠道中也原样插入。
	if got := Render("{{title}}|{{title|raw}}", values, true); got != "发布 v1.2 (beta)|发布 v1.2 (beta)" {
		t.Fatalf("原样插入 = %q", got)
	}
}

func TestRenderAddsTitleWhenMissing(t *testing.T) {
	values := map[string]string{"title": "买_牛奶", "remind_at": "09:00"}
	if got := Render("提醒时间 {{remind_at}}", values, true); got != "买\\_牛奶\n提醒时间 09:00" {
		t.Fatalf("Markdown 渠道补标题 = %q", got)
	}
	if got := Render("提醒时间 {{remind_at}}", values, false); got != "买_牛奶\n提醒时间 09:00" {
		t.Fatalf("文本渠道补标题 = %q", got)
	}
	// 写了不认识属性的 title 不算有标题。
	if got := Render("{{title|html}}", values, false); got != "买_牛奶\n{{title|html}}" {
		t.Fatalf("无效的 title = %q", got)
	}
}

func TestRenderTruncates(t *testing.T) {
	long := strings.Repeat("长", 3000)
	got := Render("{{title}}\n{{description}}", map[string]string{"title": "标题", "description": long}, false)
	// 「标题」两个字、一个换行，加上截断后的 1000 个字符。
	if utf8.RuneCountInString(got) != 3+MaxFieldLength || !strings.HasSuffix(got, "…") {
		t.Fatalf("单个字段截断后长度 = %d", utf8.RuneCountInString(got))
	}

	// 整条正文超长时截断到 4000 个字符；Markdown 渠道不留下落单的反斜杠。
	template := strings.Repeat("a", MaxMessageLength-2) + "{{title|markdown}}" + strings.Repeat("b", 100)
	md := Render(template, map[string]string{"title": "."}, true)
	if utf8.RuneCountInString(md) > MaxMessageLength || strings.HasSuffix(strings.TrimSuffix(md, "…"), `\`) {
		t.Fatalf("Markdown 正文截断 = %q", md[len(md)-10:])
	}
}
