#!/bin/sh

set -eu

: "${APP_NAME:?APP_NAME is required}"

DATA_DIR=${DATA_DIR:-/data}
CONFIG_FILE=${CONFIG_FILE:-"$DATA_DIR/config.yaml"}
CONFIG_DIR=$(dirname "$CONFIG_FILE")
CONFIG_TEMP=

mkdir -p "$DATA_DIR" "$CONFIG_DIR"

# 配置生成失败或进程提前退出时，清理同目录中的临时文件。
cleanup_config_temp() {
    if [ -n "$CONFIG_TEMP" ]; then
        rm -f "$CONFIG_TEMP"
    fi
}

trap cleanup_config_temp 0
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

# 首次启动时完整生成临时配置，再通过同目录重命名原子发布。
if [ ! -f "$CONFIG_FILE" ]; then
    CONFIG_TEMP=$(mktemp "${CONFIG_FILE}.tmp.XXXXXX")
    if "./$APP_NAME" example -p "$DATA_DIR/" -c > "$CONFIG_TEMP"; then
        mv "$CONFIG_TEMP" "$CONFIG_FILE"
        CONFIG_TEMP=
    else
        example_status=$?
        exit "$example_status"
    fi
fi

# 迁移使用配置中的真实数据库路径，并负责创建首次启动所需的空库。
"./$APP_NAME" migrate -c "$CONFIG_FILE"

exec "./$APP_NAME" server -c "$CONFIG_FILE"
