package service

import (
	"kanban/cmd/app/server/common/resp"
	"regexp"
	"strings"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateColor 检查颜色为 #RRGGBB 格式，并统一为大写保存。
// 标签、优先级挡位和列表的颜色都由取色器任意选择，不使用固定色板。
func validateColor(color string) (string, error) {
	color = strings.TrimSpace(color)
	if !hexColorPattern.MatchString(color) {
		return "", badRequest(resp.ColorInvalid, "颜色必须是 #RRGGBB 格式")
	}
	return strings.ToUpper(color), nil
}
