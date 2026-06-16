#!/bin/bash
# MiMo WebUI 卸载脚本
# 用法: sudo bash uninstall.sh

set -e

APP_NAME="mimo-webui"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/mimo-webui"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"

echo "=== MiMo WebUI 卸载 ==="

# 检查 root
if [ "$(id -u)" -ne 0 ]; then
    echo "错误: 请使用 sudo 运行此脚本"
    exit 1
fi

# 1. 停止并禁用服务
echo "[1/4] 停止服务..."
systemctl stop "${APP_NAME}" 2>/dev/null || true
systemctl disable "${APP_NAME}" 2>/dev/null || true
echo "✓ 服务已停止"

# 2. 删除 systemd 服务文件
echo "[2/4] 删除 systemd 服务..."
rm -f "${SERVICE_FILE}"
systemctl daemon-reload
echo "✓ ${SERVICE_FILE} 已删除"

# 3. 删除二进制文件
echo "[3/4] 删除二进制文件..."
rm -f "${BIN_DIR}/${APP_NAME}"
echo "✓ ${BIN_DIR}/${APP_NAME} 已删除"

# 4. 询问是否删除配置和数据
echo "[4/4] 配置和数据..."
echo "  配置目录: ${CONFIG_DIR}"
echo "  数据库:   ${CONFIG_DIR}/mimo-webui.db"
read -p "  是否删除配置和数据？[y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "${CONFIG_DIR}"
    echo "✓ ${CONFIG_DIR} 已删除"
else
    echo "  跳过（配置和数据保留在 ${CONFIG_DIR}）"
fi

echo ""
echo "=== ✅ 卸载完成 ==="
