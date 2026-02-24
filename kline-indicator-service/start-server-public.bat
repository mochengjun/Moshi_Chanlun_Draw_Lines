@echo off
REM ============================================
REM K线技术指标服务 - 外网访问启动脚本
REM ============================================

REM 设置环境变量
set "GOROOT=d:/code/DevTools/go"
set "PATH=d:/code/DevTools/go/bin;%PATH%"

REM 服务配置
set "SERVER_PORT=8080"
set "SERVER_MODE=release"

REM 外部K线API配置
set "EXTERNAL_API_URL=http://101.201.37.86:8501"
set "EXTERNAL_API_TIMEOUT=5s"
set "MAX_RETRIES=3"

REM 缓存配置
set "MEMORY_CACHE_SIZE=1000"
set "MEMORY_CACHE_TTL=5m"

REM Redis配置 (留空使用内存缓存)
set "REDIS_ADDR="
set "REDIS_PASSWORD="
set "REDIS_DB=0"
set "REDIS_CACHE_TTL=30m"

REM 限流配置
set "RATE_LIMIT=100"
set "RATE_BURST=200"

REM 认证配置 (生产环境建议启用)
set "ENABLE_AUTH=false"
set "JWT_SECRET=your-secret-key-change-in-production"

REM C++ 计算引擎路径
set "CPP_CLI_PATH=%~dp0..\cpp-trading-system\build\calculator_cli.exe"

echo ============================================
echo   莫氏缠论画线指标服务
echo   模式: 外网访问 (监听 0.0.0.0:%SERVER_PORT%)
echo ============================================
echo 服务端口: %SERVER_PORT%
echo C++ CLI: %CPP_CLI_PATH%
echo 外部API: %EXTERNAL_API_URL%
echo Redis: %REDIS_ADDR% (空则使用内存缓存)
echo 认证: %ENABLE_AUTH%
echo ============================================
echo.
echo 启动服务后，可通过以下地址访问:
echo   本地: http://localhost:%SERVER_PORT%
echo   局域网: http://192.168.2.103:%SERVER_PORT%
echo   外网: http://[您的公网IP]:%SERVER_PORT%
echo.
echo 按 Ctrl+C 停止服务
echo ============================================

REM 检查C++ CLI是否存在
if not exist "%CPP_CLI_PATH%" (
    echo [错误] C++ CLI不存在: %CPP_CLI_PATH%
    echo [提示] 请先编译C++项目
    pause
    exit /b 1
)

cd /d %~dp0
server.exe
