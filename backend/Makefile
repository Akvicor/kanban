PROJECT=kanban

# 系统工具
MKDIR=mkdir
SED=sed
DATE=date
# 其他工具
GIT=git
GO=go
GOFMT=gofmt
SCC=/kayuki/vivc/bin/scc

# 编译信息
BRANCH=$(shell $(GIT) rev-parse --abbrev-ref HEAD)
VERSION=$(shell $(GIT) describe --tags --always | $(SED) 's/^v//')
COMMIT=$(shell $(GIT) rev-parse --verify HEAD)
BUILD_TIME=$(shell $(DATE) +"%Y-%m-%d %H:%M:%S %z")
GO_FLAGS=-ldflags "-s -w -X '$(PROJECT)/cmd/def.Branch=$(BRANCH)' -X '$(PROJECT)/cmd/def.Version=$(VERSION)' -X '$(PROJECT)/cmd/def.Commit=$(COMMIT)' -X '$(PROJECT)/cmd/def.BuildTime=$(BUILD_TIME)'"

.PHONY: format build example migrate dev count

# 格式化项目
format:
	@$(GO) mod tidy
	@$(GOFMT) -s -w .
	@echo "Version: $(VERSION)-$(BRANCH)_$(COMMIT) ($(BUILD_TIME))"

# 编译项目（使用 web 目录中现有的前端产物）
build:
	$(MAKE) format
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o bin/$(PROJECT) main.go

# 生成默认配置文件
example:
	$(MAKE) build
	- $(MKDIR) bin/data
	cd bin && ./$(PROJECT) example -p ./data/ -c > ./data/config.yaml

# 创建或升级数据库
migrate:
	$(MAKE) build
	cd bin && ./$(PROJECT) migrate -c ./data/config.yaml

# 调试运行：配置中 debug = true 时从 web-path 实时读取前端文件
dev:
	$(MAKE) build
	cd bin && ./$(PROJECT) migrate -c ./data/config.yaml && ./$(PROJECT) server -c ./data/config.yaml

# 统计代码行数
count:
	@$(SCC) ./
