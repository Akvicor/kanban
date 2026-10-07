package web

import "embed"

// Resource 是嵌入二进制的前端构建产物。构建时由根目录 Makefile 把 frontend/build 复制到本目录的 build 下。
//
//go:embed *
var Resource embed.FS
