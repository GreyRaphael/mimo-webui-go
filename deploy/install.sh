#!/bin/bash
# MiMo WebUI 安装脚本
# 用法: sudo bash install.sh
# 从 release tar.gz 解压后，在解压目录中运行

set -e

APP_NAME="mimo-webui"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/mimo-webui"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== MiMo WebUI 安装 ==="

# 检查 root
if [ "$(id -u)" -ne 0 ]; then
    echo "错误: 请使用 sudo 运行此脚本"
    exit 1
fi

# 检查二进制文件
if [ ! -f "${SCRIPT_DIR}/${APP_NAME}" ]; then
    echo "错误: 未找到 ${APP_NAME} 二进制文件，请在解压目录中运行此脚本"
    exit 1
fi

# 1. 安装二进制文件
echo "[1/5] 安装二进制文件到 ${BIN_DIR}..."
cp "${SCRIPT_DIR}/${APP_NAME}" "${BIN_DIR}/${APP_NAME}"
chmod +x "${BIN_DIR}/${APP_NAME}"
echo "✓ ${BIN_DIR}/${APP_NAME}"

# 2. 创建配置目录
echo "[2/5] 创建配置目录 ${CONFIG_DIR}..."
mkdir -p "${CONFIG_DIR}"
if [ ! -f "${CONFIG_DIR}/config.toml" ]; then
    cp "${SCRIPT_DIR}/config.toml.example" "${CONFIG_DIR}/config.toml"
    echo "✓ 配置文件已生成: ${CONFIG_DIR}/config.toml"
    echo "  ⚠️  请编辑 config.toml 填入你的 API Key 和 admin 密码"
else
    echo "✓ 配置文件已存在，跳过"
fi

# 3. 创建 systemd 服务
echo "[3/5] 创建 systemd 服务..."
cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=MiMo WebUI
After=network.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/${APP_NAME} -config ${CONFIG_DIR}/config.toml
WorkingDirectory=${CONFIG_DIR}
Restart=on-failure
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF
echo "✓ ${SERVICE_FILE}"

# 4. 重载 systemd
echo "[4/5] 重载 systemd..."
systemctl daemon-reload

# 5. 启用并启动
echo "[5/5] 启用并启动服务..."
systemctl enable "${APP_NAME}" 2>/dev/null
systemctl restart "${APP_NAME}"
sleep 2

if systemctl is-active --quiet "${APP_NAME}"; then
    IP=$(hostname -I | awk '{print $1}')
    echo ""
    echo "=== ✅ 安装完成 ==="
    echo "服务状态: $(systemctl is-active ${APP_NAME})"
    echo "访问地址: http://${IP}:3000"
    echo "配置文件: ${CONFIG_DIR}/config.toml"
    echo ""
    echo "常用命令:"
    echo "  查看日志:  journalctl -u ${APP_NAME} -f"
    echo "  重启服务:  systemctl restart ${APP_NAME}"
    echo "  停止服务:  systemctl stop ${APP_NAME}"
    echo "  卸载:      sudo bash ${SCRIPT_DIR}/uninstall.sh"
else
    echo ""
    echo "=== ⚠️ 服务启动失败 ==="
    echo "请检查配置文件: ${CONFIG_DIR}/config.toml"
    echo "查看日志: journalctl -u ${APP_NAME} -n 30"
    exit 1
fi
