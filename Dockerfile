# syntax=docker/dockerfile:1

ARG APP_NAME=app
ARG APP_PORT=3000

# 前端构建所需的 Node.js 与 Yarn 启动器来自官方 Node 镜像；
# Yarn 启动器会转交给仓库自带的 frontend/.yarn/releases 版本。
FROM --platform=$BUILDPLATFORM node:24-trixie-slim AS node

# 构建阶段固定在构建平台执行，目标平台的二进制由 Go 交叉编译生成。
# 官方 Go 镜像自带 git、make、gcc（make verify 中的 go test -race 需要 cgo）。
FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS builder

ARG APP_NAME
ARG TARGETOS
ARG TARGETARCH

COPY --from=node /usr/local/ /usr/local/
COPY --from=node /opt/ /opt/

WORKDIR /src
COPY . .

# 复用项目根 Makefile 的验证和跨平台构建接口。
RUN make verify
RUN GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" make build PROJECT="${APP_NAME}"

FROM debian:trixie-slim AS runtime

ARG APP_NAME
ARG APP_PORT

# curl 供健康检查使用，tzdata 供 TZ 时区使用。
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl tzdata && \
    rm -rf /var/lib/apt/lists/*

ENV APP_NAME="${APP_NAME}" \
    APP_PORT="${APP_PORT}" \
    DATA_DIR=/data \
    CONFIG_FILE=/data/config.yaml \
    TZ=Etc/GMT-8

WORKDIR /app

COPY --from=builder "/src/build/${APP_NAME}" "./${APP_NAME}"
COPY --chmod=755 entrypoint.sh /entrypoint.sh
COPY --chmod=755 healthcheck.sh /healthcheck.sh

VOLUME ["/data"]
EXPOSE ${APP_PORT}

HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
    CMD ["/healthcheck.sh"]

ENTRYPOINT ["/entrypoint.sh"]
