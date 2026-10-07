package def

import (
	"fmt"
	"time"
)

const (
	AppName        = "kanban"
	AppUsage       = "Personal Kanban"
	AppDescription = "Kanban is a personal plan and todo service"
)

// 编译信息，由根目录 Makefile 通过 -ldflags 注入。
var (
	Branch    string
	Version   string
	Commit    string
	BuildTime string
)

// AppVersion 返回带分支、提交和编译时间的完整版本号。
func AppVersion() string {
	return fmt.Sprintf("%s-%s_%s (%s)", Version, Branch, Commit, BuildTime)
}

// Copyright 返回版权声明。
func Copyright() string {
	return fmt.Sprintf("Copyright © 2026-%d Akvicor, All Rights Reserved.", time.Now().Year())
}
