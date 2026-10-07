# kanban 发布构建入口：供本地、GitHub Actions 与 Dockerfile 共同使用。

PROJECT ?= kanban
include supply-chain-tools.mk

# 系统工具
CP=cp
GIT=git
GO=go
GOFMT=gofmt
MKDIR=mkdir
RM=rm
SED=sed
DATE=date
CYCLONEDX=cyclonedx
CYCLONEDX_GOMOD=cyclonedx-gomod
CDXGEN=cdxgen
TRIVY=trivy
YARN=yarn

# 编译信息
BRANCH=$(shell $(GIT) rev-parse --abbrev-ref HEAD)
VERSION=$(shell $(GIT) describe --tags --always | $(SED) 's/^v//')
COMMIT=$(shell $(GIT) rev-parse --verify HEAD)
BUILD_TIME=$(shell $(DATE) +"%Y-%m-%d %H:%M:%S %z")
# Go 模块名固定为 kanban；PROJECT 只决定输出文件名，Docker 构建会把它改成 APP_NAME。
GO_FLAGS=-ldflags "-s -w -X 'kanban/cmd/def.Branch=$(BRANCH)' -X 'kanban/cmd/def.Version=$(VERSION)' -X 'kanban/cmd/def.Commit=$(COMMIT)' -X 'kanban/cmd/def.BuildTime=$(BUILD_TIME)'"

# AVIF 解码库默认会尝试加载系统中的 libavif 动态库；nodynamic 固定使用内置的 WebAssembly 版本，
# 使各平台、各部署环境的解码行为一致。所有 go 命令（构建、测试、vet、漏洞扫描）都带上这个标签。
export GOFLAGS=-tags=nodynamic

# GitHub Actions 与 Docker Buildx 通过 GOOS/GOARCH 指定目标平台。
TARGET_GOOS=$(if $(GOOS),$(GOOS),$(shell cd backend && $(GO) env GOOS))
TARGET_GOARCH=$(if $(GOARCH),$(GOARCH),$(shell cd backend && $(GO) env GOARCH))

FRONTEND_BUILD_DIR=frontend/build
EMBED_BUILD_DIR=backend/cmd/app/server/web/build
OUTPUT_DIR=build
REPORT_DIR=$(OUTPUT_DIR)/reports
SBOM_DIR=$(OUTPUT_DIR)/sbom
IMAGE ?=
BUILDER_IMAGE ?=
RUNTIME_IMAGE ?=
export IMAGE BUILDER_IMAGE RUNTIME_IMAGE

.PHONY: format verify backend-verify build pure-go-build \
	frontend-install frontend-test frontend-build frontend-audit frontend-audit-report \
	go-audit go-binary-audit audit secret-scan sbom sbom-source sbom-image \
	image-scan base-image-scan image-ref-check base-image-ref-check release-input-check \
	supply-chain-tool-versions

# 格式化后端源码并整理 Go 模块文件。
format:
	@cd backend && $(GO) mod tidy
	@cd backend && $(GOFMT) -s -w .
	@echo "Version: $(VERSION)-$(BRANCH)_$(COMMIT) ($(BUILD_TIME))"

frontend-install:
	@cd frontend && $(YARN) install --immutable

frontend-test: frontend-install
	@cd frontend && $(YARN) lint
	@cd frontend && CI=true $(YARN) test

frontend-build: frontend-install
	@cd frontend && $(RM) -rf build
	@cd frontend && VITE_APP_VERSION="$(VERSION)" $(YARN) build --assetsDir static
	@test -f "$(FRONTEND_BUILD_DIR)/index.html"
	@test -f "$(FRONTEND_BUILD_DIR)/.vite/manifest.json"
	@test -d "$(FRONTEND_BUILD_DIR)/static"

# CI 与发布流程的质量门禁只调用本入口；测试、构建检查和源码依赖审计均在此阻断。
verify: frontend-test backend-verify audit

# Go 命令在真实 Vite 产物复制到 embed 目录后运行，并始终清理复制内容。
backend-verify: frontend-build
	@set -eu; \
		trap '$(RM) -rf "$(EMBED_BUILD_DIR)"' 0 HUP INT TERM; \
		$(RM) -rf "$(EMBED_BUILD_DIR)"; \
		$(CP) -R "$(FRONTEND_BUILD_DIR)" "$(EMBED_BUILD_DIR)"; \
		unformatted="$$(cd backend && $(GOFMT) -s -l .)"; \
		if [ -n "$$unformatted" ]; then \
			printf '%s\n' "$$unformatted"; \
			exit 1; \
		fi; \
		(cd backend && $(GO) mod tidy -diff); \
		(cd backend && $(GO) test -count=1 ./...); \
		(cd backend && CGO_ENABLED=1 $(GO) test -race -count=1 ./...); \
		(cd backend && $(GO) vet ./...)

# 前端产物只在 Go embed 编译期间存在，结束或失败时都会清理。
build: frontend-build
	@set -eu; \
		trap '$(RM) -rf "$(EMBED_BUILD_DIR)"' 0 HUP INT TERM; \
		$(RM) -rf "$(EMBED_BUILD_DIR)"; \
		$(CP) -R "$(FRONTEND_BUILD_DIR)" "$(EMBED_BUILD_DIR)"; \
		$(MKDIR) -p "$(OUTPUT_DIR)"; \
		(cd backend && \
			CGO_ENABLED=0 GOOS="$(TARGET_GOOS)" GOARCH="$(TARGET_GOARCH)" \
			$(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/$(PROJECT)" main.go)

# 验证发布矩阵中的纯 Go SQLite 交叉编译，不运行目标平台二进制。
pure-go-build: frontend-build
	@set -eu; \
		trap '$(RM) -rf "$(EMBED_BUILD_DIR)"' 0 HUP INT TERM; \
		$(RM) -rf "$(EMBED_BUILD_DIR)"; \
		$(CP) -R "$(FRONTEND_BUILD_DIR)" "$(EMBED_BUILD_DIR)"; \
		$(MKDIR) -p "$(OUTPUT_DIR)/matrix"; \
		(cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/matrix/$(PROJECT)-linux-amd64" main.go); \
		(cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/matrix/$(PROJECT)-linux-arm64" main.go); \
		(cd backend && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/matrix/$(PROJECT)-windows-amd64.exe" main.go); \
		(cd backend && CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/matrix/$(PROJECT)-windows-arm64.exe" main.go); \
		(cd backend && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/matrix/$(PROJECT)-darwin-amd64" main.go); \
		(cd backend && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GO_FLAGS) -o "../$(OUTPUT_DIR)/matrix/$(PROJECT)-darwin-arm64" main.go)

# Go 模块完整性、更新/retract 报告和源码可达漏洞门禁。
go-audit:
	@$(MKDIR) -p "$(REPORT_DIR)"
	@cd backend && $(GO) mod verify > "../$(REPORT_DIR)/go-mod-verify.txt"
	@cd backend && $(GO) list -m -u -retracted all > "../$(REPORT_DIR)/go-modules.txt"
	@if grep -E '\[(retracted|deprecated):' "$(REPORT_DIR)/go-modules.txt" >/dev/null; then \
		printf '%s\n' '检测到 retracted 或 deprecated Go 模块，详见报告'; \
		exit 1; \
	fi
	@cd backend && $(GO) tool govulncheck -format json ./... > "../$(REPORT_DIR)/govuln-source.json"

# 最终纯 Go 二进制漏洞检查会补足源码 build tags 与实际链接结果。
go-binary-audit: build
	@$(MKDIR) -p "$(REPORT_DIR)"
	@cd backend && $(GO) tool govulncheck -mode binary "../$(OUTPUT_DIR)/$(PROJECT)" > "../$(REPORT_DIR)/govuln-binary.txt"

# 前端完整依赖树 critical/high 门禁。
frontend-audit: frontend-audit-report
	@$(MKDIR) -p "$(REPORT_DIR)"
	@cd frontend && $(YARN) explain peer-requirements > "../$(REPORT_DIR)/frontend-peers.txt"
	@cd frontend && $(YARN) npm audit --all --recursive --severity high --json > "../$(REPORT_DIR)/frontend-audit-high.ndjson"

# moderate/low 仅生成报告；Yarn 对任何发现都返回非零，因此本入口不作为门禁。
frontend-audit-report: frontend-install
	@$(MKDIR) -p "$(REPORT_DIR)"
	@cd frontend && $(YARN) npm audit --all --recursive --severity low --json > "../$(REPORT_DIR)/frontend-audit-all.ndjson" || \
		printf '%s\n' '完整前端审计报告包含已知问题，请按报告评估 moderate/low 项'

audit: go-audit frontend-audit

# 仓库 secret 扫描由 CI 提供的固定 Trivy 版本执行。
secret-scan:
	@$(MKDIR) -p "$(REPORT_DIR)"
	@$(TRIVY) filesystem --scanners secret --severity HIGH,CRITICAL --exit-code 1 \
		--format json --output "$(REPORT_DIR)/repository-secrets.json" .

# 生成 Go、前端和组合 CycloneDX 源码 SBOM。
sbom-source:
	@$(MKDIR) -p "$(SBOM_DIR)"
	@$(CYCLONEDX_GOMOD) mod -json -output "$(SBOM_DIR)/backend.cdx.json" backend
	@$(CDXGEN) -t js --spec-version 1.6 -o "$(SBOM_DIR)/frontend.cdx.json" frontend
	@$(CYCLONEDX) merge --input-files "$(SBOM_DIR)/backend.cdx.json" "$(SBOM_DIR)/frontend.cdx.json" \
		--output-file "$(SBOM_DIR)/source.cdx.json" --output-format json --output-version v1_6

# IMAGE 必须由发布流水线传入最终镜像的 sha256 摘要引用。
sbom-image: sbom-source image-ref-check
	@$(TRIVY) image --format cyclonedx --output "$(SBOM_DIR)/image.cdx.json" "$${IMAGE}"
	@$(CYCLONEDX) merge --input-files "$(SBOM_DIR)/source.cdx.json" "$(SBOM_DIR)/image.cdx.json" \
		--output-file "$(SBOM_DIR)/release.cdx.json" --output-format json --output-version v1_6

sbom: sbom-image

# 最终镜像同时检查 OS/语言包、文件和镜像配置中的 secret。
image-scan: image-ref-check
	@$(MKDIR) -p "$(REPORT_DIR)"
	@$(TRIVY) image --scanners vuln,secret --image-config-scanners secret \
		--severity HIGH,CRITICAL --exit-code 1 --format json \
		--output "$(REPORT_DIR)/release-image.json" "$${IMAGE}"

# 外部维护的 builder/runtime 基础镜像分别扫描，不在仓库内猜测镜像值。
base-image-scan: base-image-ref-check
	@$(MKDIR) -p "$(REPORT_DIR)"
	@$(TRIVY) image --scanners vuln,secret --image-config-scanners secret \
		--severity HIGH,CRITICAL --exit-code 1 --format json \
		--output "$(REPORT_DIR)/builder-image.json" "$${BUILDER_IMAGE}"
	@$(TRIVY) image --scanners vuln,secret --image-config-scanners secret \
		--severity HIGH,CRITICAL --exit-code 1 --format json \
		--output "$(REPORT_DIR)/runtime-image.json" "$${RUNTIME_IMAGE}"

# 发布镜像只接受 registry/repository@sha256:<64位小写十六进制>。
image-ref-check:
	@image="$${IMAGE:-}"; \
		if ! printf '%s\n' "$$image" | grep -Eq '^[^[:space:]@]+@sha256:[0-9a-f]{64}$$'; then \
			printf '%s\n' 'IMAGE 必须是包含完整 sha256 摘要的最终镜像引用' >&2; \
			exit 1; \
		fi

# Builder 和 runtime 镜像同样必须由外部流水线提供摘要引用。
base-image-ref-check:
	@for pair in "BUILDER_IMAGE=$${BUILDER_IMAGE:-}" "RUNTIME_IMAGE=$${RUNTIME_IMAGE:-}"; do \
		variable="$${pair%%=*}"; \
		value="$${pair#*=}"; \
		if ! printf '%s\n' "$$value" | grep -Eq '^[^[:space:]@]+@sha256:[0-9a-f]{64}$$'; then \
			printf '%s 必须是包含完整 sha256 摘要的镜像引用\n' "$$variable" >&2; \
			exit 1; \
		fi; \
	done

# Tag 发布在扫描开始前一次性验证全部外部镜像输入。
release-input-check: image-ref-check base-image-ref-check

# 校验扫描器实际版本，并把完整输出保存为流水线 artifact。
supply-chain-tool-versions:
	@$(MKDIR) -p "$(REPORT_DIR)"
	@set -eu; \
		report="$(REPORT_DIR)/supply-chain-tool-versions.txt"; \
		: > "$$report"; \
		check_version() { \
			name="$$1"; expected="$$2"; output="$$3"; \
			version="$${expected#v}"; \
			pattern="$$(printf '%s' "$$version" | $(SED) 's/\./\\./g')"; \
			printf '%s expected: %s\n%s\n' "$$name" "$$expected" "$$output" | tee -a "$$report"; \
			if ! printf '%s\n' "$$output" | grep -Eq "(^|[^[:alnum:].])v?$${pattern}([^[:alnum:].]|$$)"; then \
				printf '%s actual version does not match %s\n' "$$name" "$$expected" >&2; \
				exit 1; \
			fi; \
		}; \
		output="$$(cd backend && $(GO) tool govulncheck -version 2>&1)"; \
		check_version govulncheck "$(GOVULNCHECK_VERSION)" "$$output"; \
		output="$$( $(CYCLONEDX_GOMOD) version 2>&1)"; \
		check_version cyclonedx-gomod "$(CYCLONEDX_GOMOD_VERSION)" "$$output"; \
		output="$$( $(CDXGEN) --version 2>&1)"; \
		check_version cdxgen "$(CDXGEN_VERSION)" "$$output"; \
		output="$$( $(CYCLONEDX) --version 2>&1)"; \
		check_version cyclonedx-cli "$(CYCLONEDX_CLI_VERSION)" "$$output"; \
		output="$$( $(TRIVY) --version 2>&1)"; \
		check_version trivy "$(TRIVY_VERSION)" "$$output"
