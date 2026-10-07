#!/bin/sh

set -eu

APP_PORT=${APP_PORT:-}
HEALTHCHECK_PATH=${HEALTHCHECK_PATH:-/api/sys/info/health}

if [ -z "$APP_PORT" ]; then
    printf 'APP_PORT is required\n' >&2
    exit 1
fi

# 将探测失败统一为 Docker 健康检查使用的退出码 1。
curl --fail --silent --show-error \
    "http://127.0.0.1:${APP_PORT}${HEALTHCHECK_PATH}" >/dev/null || exit 1
