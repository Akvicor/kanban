package timezone

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Fallback 是无法确定服务器时区时使用的时区。
const Fallback = "UTC"

// Valid 判断是否为可加载的 IANA 时区名。"Local" 依赖进程环境，不能作为用户时区保存。
func Valid(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

// Server 返回服务端进程当前的本地时区名，用作新用户的默认时区。
// 优先使用 TZ 环境变量对应的时区；未设置时从 /etc/localtime 的链接目标或 /etc/timezone 推断；都不可用时返回 UTC。
func Server() string {
	if name := time.Local.String(); name != "Local" && Valid(name) {
		return name
	}
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		if name := fromZoneinfoPath(target); Valid(name) {
			return name
		}
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		if name := strings.TrimSpace(string(data)); Valid(name) {
			return name
		}
	}
	return Fallback
}

// fromZoneinfoPath 从形如 /usr/share/zoneinfo/Asia/Shanghai 的路径取出时区名。
func fromZoneinfoPath(path string) string {
	path = filepath.ToSlash(path)
	const marker = "zoneinfo/"
	index := strings.LastIndex(path, marker)
	if index < 0 {
		return ""
	}
	return path[index+len(marker):]
}
