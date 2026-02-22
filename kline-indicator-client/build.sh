#!/bin/bash
# K线技术指标客户端 - 构建脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "  K线技术指标客户端 - 构建"
echo "=========================================="

# 检查node是否可用
if ! command -v node &> /dev/null; then
    echo "[错误] Node.js未找到，请先安装Node.js 18+"
    exit 1
fi

echo "Node.js版本: $(node --version)"

# 安装依赖
echo "安装依赖..."
npm install

# TypeScript类型检查
echo "TypeScript类型检查..."
npx tsc --noEmit

# 构建生产版本
echo "构建生产版本..."
npx vite build

echo "=========================================="
echo "构建完成! 输出目录: dist/"
echo ""
echo "部署方式:"
echo "  1. 开发模式: ./start-client.sh"
echo "  2. 生产模式: 将 dist/ 目录部署到Nginx/Caddy等Web服务器"
echo "     示例Nginx配置:"
echo "       location / {"
echo "         root /path/to/dist;"
echo "         try_files \$uri \$uri/ /index.html;"
echo "       }"
echo "       location /api/ {"
echo "         proxy_pass http://localhost:8080;"
echo "       }"
echo "       location /ws/ {"
echo "         proxy_pass http://localhost:8080;"
echo "         proxy_http_version 1.1;"
echo "         proxy_set_header Upgrade \$http_upgrade;"
echo "         proxy_set_header Connection \"upgrade\";"
echo "       }"
echo "=========================================="
