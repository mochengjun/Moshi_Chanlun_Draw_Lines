#!/bin/bash
# ============================================
# 莫氏缠论画线指标 - 前端部署脚本
# ============================================
# 使用前请填写以下配置项：
#   DEPLOY_SERVER - 服务器IP或域名
#   DEPLOY_USER   - SSH登录用户名
#   DEPLOY_PATH   - 服务器上的部署目录
# ============================================

# === 部署配置（需要填写） ===
DEPLOY_SERVER="TODO_SERVER_IP"
DEPLOY_USER="TODO_USERNAME"
DEPLOY_PATH="TODO_DEPLOY_PATH"  # 例如: /var/www/html 或 /usr/share/nginx/html

# === 本地配置 ===
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLIENT_DIR="$SCRIPT_DIR/kline-indicator-client"
DIST_DIR="$CLIENT_DIR/dist"
NODE_HOME="/d/tools/node-v20.18.0-win-x64"

# === 检查配置 ===
if [[ "$DEPLOY_SERVER" == "TODO_"* ]] || [[ "$DEPLOY_USER" == "TODO_"* ]] || [[ "$DEPLOY_PATH" == "TODO_"* ]]; then
    echo "[ERROR] 请先填写部署配置（DEPLOY_SERVER, DEPLOY_USER, DEPLOY_PATH）"
    exit 1
fi

# === Step 1: 构建前端 ===
echo "=== Step 1: 构建前端项目 ==="
export PATH="$NODE_HOME:$PATH"
cd "$CLIENT_DIR" || exit 1
npm run build
if [ $? -ne 0 ]; then
    echo "[ERROR] 前端构建失败"
    exit 1
fi
echo "[OK] 构建完成: $DIST_DIR"

# === Step 2: 部署到服务器 ===
echo "=== Step 2: 部署到 $DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH ==="
scp -r "$DIST_DIR"/* "$DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH/"
if [ $? -ne 0 ]; then
    echo "[ERROR] 部署失败"
    exit 1
fi

echo "=== 部署成功 ==="
echo "服务器: $DEPLOY_SERVER"
echo "路径:   $DEPLOY_PATH"
