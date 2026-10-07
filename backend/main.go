package main

import (
	"context"
	"kanban/cmd/app"
	"log"
	"os"

	// 内嵌时区数据库，使用户时区的校验和换算不依赖运行环境是否安装 tzdata。
	_ "time/tzdata"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	if err := app.App.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
