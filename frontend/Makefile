PROJECT=kanban

SED=sed
GIT=git
SCC=/kayuki/vivc/bin/scc

# 编译信息
VERSION=$(shell $(GIT) describe --tags --always | $(SED) 's/^v//')

.PHONY: build dev count

# 编译项目
build:
	yarn install --immutable
	VITE_APP_VERSION="$(VERSION)" yarn build

# 调试运行，接口转发到 127.0.0.1:3000
dev:
	yarn start

# 统计代码行数
count:
	@$(SCC) ./
