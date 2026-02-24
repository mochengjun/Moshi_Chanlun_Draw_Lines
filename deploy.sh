#!/bin/bash
# ============================================
# 莫氏缠论画线指标系统 - 完整部署脚本
# ============================================
# 架构：Go服务框架 + C++计算引擎 + React前端
# 
# 使用前请填写以下配置项：
#   DEPLOY_SERVER - 服务器IP或域名
#   DEPLOY_USER   - SSH登录用户名
#   DEPLOY_PATH   - 服务器上的部署目录
# ============================================

set -e  # 出错时退出

# === 部署配置（需要填写） ===
DEPLOY_SERVER="TODO_SERVER_IP"
DEPLOY_USER="TODO_USERNAME"
DEPLOY_PATH="TODO_DEPLOY_PATH"  # 例如: /opt/moshi-chanlun

# === 本地配置 ===
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLIENT_DIR="$SCRIPT_DIR/kline-indicator-client"
CPP_DIR="$SCRIPT_DIR/cpp-trading-system"
GO_DIR="$SCRIPT_DIR/kline-indicator-service"
DIST_DIR="$CLIENT_DIR/dist"
NODE_HOME="/d/tools/node-v20.18.0-win-x64"

# === 颜色输出 ===
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# === 检查配置 ===
check_config() {
    if [[ "$DEPLOY_SERVER" == "TODO_"* ]] || [[ "$DEPLOY_USER" == "TODO_"* ]] || [[ "$DEPLOY_PATH" == "TODO_"* ]]; then
        log_error "请先填写部署配置（DEPLOY_SERVER, DEPLOY_USER, DEPLOY_PATH）"
    fi
}

# === Step 1: 编译C++计算引擎 ===
build_cpp() {
    log_info "=== Step 1: 编译C++计算引擎 ==="
    
    cd "$CPP_DIR" || log_error "C++目录不存在: $CPP_DIR"
    
    # 创建build目录
    mkdir -p build
    cd build
    
    # CMake配置
    cmake .. -DCMAKE_BUILD_TYPE=Release
    if [ $? -ne 0 ]; then
        log_error "CMake配置失败"
    fi
    
    # 编译
    cmake --build . --config Release -j$(nproc 2>/dev/null || echo 4)
    if [ $? -ne 0 ]; then
        log_error "C++编译失败"
    fi
    
    # 验证CLI工具
    if [ ! -f "calculator_cli" ] && [ ! -f "calculator_cli.exe" ]; then
        log_error "calculator_cli 未生成"
    fi
    
    log_info "[OK] C++计算引擎编译完成"
}

# === Step 2: 构建Go服务 ===
build_go() {
    log_info "=== Step 2: 构建Go服务 ==="
    
    cd "$GO_DIR" || log_error "Go目录不存在: $GO_DIR"
    
    # 下载依赖
    go mod download
    if [ $? -ne 0 ]; then
        log_warn "go mod download 失败，尝试继续..."
    fi
    
    # 编译
    CGO_ENABLED=0 go build -o bin/kline-indicator-service ./cmd/server
    if [ $? -ne 0 ]; then
        log_error "Go服务编译失败"
    fi
    
    log_info "[OK] Go服务编译完成: $GO_DIR/bin/kline-indicator-service"
}

# === Step 3: 构建前端 ===
build_frontend() {
    log_info "=== Step 3: 构建前端项目 ==="
    
    export PATH="$NODE_HOME:$PATH"
    cd "$CLIENT_DIR" || log_error "前端目录不存在: $CLIENT_DIR"
    
    # 安装依赖
    npm install
    if [ $? -ne 0 ]; then
        log_warn "npm install 失败，尝试继续..."
    fi
    
    # 构建
    npm run build
    if [ $? -ne 0 ]; then
        log_error "前端构建失败"
    fi
    
    log_info "[OK] 前端构建完成: $DIST_DIR"
}

# === Step 4: 部署到服务器 ===
deploy_to_server() {
    log_info "=== Step 4: 部署到服务器 ==="
    check_config
    
    log_info "部署目标: $DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH"
    
    # 创建远程目录结构
    ssh "$DEPLOY_USER@$DEPLOY_SERVER" "mkdir -p $DEPLOY_PATH/{bin,config,static}"
    
    # 部署C++ CLI
    log_info "部署C++计算引擎..."
    scp "$CPP_DIR/build/calculator_cli"* "$DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH/bin/"
    
    # 部署Go服务
    log_info "部署Go服务..."
    scp "$GO_DIR/bin/kline-indicator-service" "$DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH/bin/"
    scp "$GO_DIR/config/config.yaml" "$DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH/config/"
    
    # 部署前端
    log_info "部署前端静态文件..."
    scp -r "$DIST_DIR"/* "$DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH/static/"
    
    log_info "[OK] 部署完成"
}

# === 本地运行（开发模式） ===
run_local() {
    log_info "=== 本地运行（开发模式）==="
    
    # 检查C++ CLI是否已编译
    if [ ! -f "$CPP_DIR/build/calculator_cli" ] && [ ! -f "$CPP_DIR/build/calculator_cli.exe" ]; then
        log_warn "C++ CLI未编译，正在编译..."
        build_cpp
    fi
    
    # 启动Go服务
    cd "$GO_DIR"
    log_info "启动Go服务 (端口8080)..."
    go run ./cmd/server
}

# === 帮助信息 ===
show_help() {
    echo "莫氏缠论画线指标系统 - 部署脚本"
    echo ""
    echo "用法: $0 <command>"
    echo ""
    echo "命令:"
    echo "  build-cpp      编译C++计算引擎"
    echo "  build-go       构建Go服务"
    echo "  build-frontend 构建前端项目"
    echo "  build-all      构建所有组件"
    echo "  deploy         部署到远程服务器"
    echo "  run            本地运行（开发模式）"
    echo "  help           显示此帮助信息"
    echo ""
    echo "架构说明:"
    echo "  - Go服务（kline-indicator-service）: API服务框架，端口8080"
    echo "  - C++计算引擎（cpp-trading-system）: 莫氏缠论核心算法"
    echo "  - React前端（kline-indicator-client）: K线图表可视化"
}

# === 主入口 ===
case "${1:-help}" in
    build-cpp)
        build_cpp
        ;;
    build-go)
        build_go
        ;;
    build-frontend)
        build_frontend
        ;;
    build-all)
        build_cpp
        build_go
        build_frontend
        ;;
    deploy)
        build_cpp
        build_go
        build_frontend
        deploy_to_server
        ;;
    run)
        run_local
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "未知命令: $1\n运行 '$0 help' 查看帮助"
        ;;
esac
