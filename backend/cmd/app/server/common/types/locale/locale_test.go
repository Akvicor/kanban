package locale

import "testing"

// TestEffective 验证「跟随系统」时按 Accept-Language 解析默认文案语言，显式设置优先。
func TestEffective(t *testing.T) {
	cases := []struct {
		name           string
		setting        Type
		acceptLanguage string
		want           Type
	}{
		{"显式设置优先于浏览器", En, "zh-CN,zh;q=0.9", En},
		{"显式中文优先于浏览器", ZhCN, "en-US,en;q=0.9", ZhCN},
		{"跟随系统取中文浏览器", "", "zh-CN,zh;q=0.9,en;q=0.8", ZhCN},
		{"跟随系统取英文浏览器", "", "en-US,en;q=0.9", En},
		{"中文优先于英文", "", "en-US,zh-CN;q=0.9", ZhCN},
		{"无头信息回落默认语言", "", "", Default},
		{"无法识别的语言用英文", "", "ja-JP,ja;q=0.9", En},
		{"日文浏览器带英文回退用英文", "", "ja,en;q=0.9", En},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Effective(c.setting, c.acceptLanguage); got != c.want {
				t.Fatalf("Effective(%q, %q) = %q, want %q", c.setting, c.acceptLanguage, got, c.want)
			}
		})
	}
}
