#!/bin/bash
# K线技术指标客户端 - Linux启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "  K线技术指标客户端"
echo "=========================================="

# 检查node是否可用
if ! command -v node &> /dev/null; then
    echo "[错误] Node.js未找到，请先安装Node.js 18+"
    exit 1
fi

echo "Node.js版本: $(node --version)"
echo "npm版本: $(npm --version)"
echo "=========================================="

# 安装依赖（如果node_modules不存在）
if [ ! -d "node_modules" ]; then
    echo "正在安装依赖..."
    npm install
    echo "依赖安装完成"
fi

echo "启动开发服务器 (端口5173)"
echo "请确保Go后端服务已在8080端口运行"
echo "=========================================="

npx vite --host
