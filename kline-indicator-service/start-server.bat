@echo off
REM K线技术指标服务启动脚本
REM 架构：Go服务框架 + C++计算引擎

REM 设置Go环境变量
set GOROOT=d:\code\DevTools\go
set PATH=%GOROOT%\bin;%PATH%

REM C++ 计算引擎路径配置
set CPP_CLI_PATH=%~dp0..\cpp-trading-system\build\calculator_cli.exe

REM 服务配置
set SERVER_PORT=8080
set SERVER_MODE=debug

REM 外部K线API配置 (需要修改为实际的API地址)
set EXTERNAL_API_URL=http://101.201.37.86:8501/
set EXTERNAL_API_TIMEOUT=5s
set MAX_RETRIES=3

REM 缓存配置
set MEMORY_CACHE_SIZE=1000
set MEMORY_CACHE_TTL=5m

REM Redis配置 (可选,留空则仅使用内存缓存)
set REDIS_ADDR=
set REDIS_PASSWORD=
set REDIS_DB=0
set REDIS_CACHE_TTL=30m

REM 限流配置
set RATE_LIMIT=100
set RATE_BURST=200

REM 认证配置 (设置为false禁用JWT认证)
set ENABLE_AUTH=false
set JWT_SECRET=your-secret-key-change-in-production

echo ==========================================
echo   莫氏缠论画线指标服务
echo   架构: Go服务 + C++计算引擎
echo ==========================================
echo 服务端口: %SERVER_PORT%
echo C++ CLI: %CPP_CLI_PATH%
echo 外部API: %EXTERNAL_API_URL%
echo Redis: %REDIS_ADDR% (空则使用内存缓存)
echo 认证: %ENABLE_AUTH%
echo ==========================================

REM 检查C++ CLI是否存在
if not exist "%CPP_CLI_PATH%" (
    echo [警告] C++ CLI不存在: %CPP_CLI_PATH%
    echo [提示] 请先编译C++项目: cd ..\cpp-trading-system ^&^& mkdir build ^&^& cd build ^&^& cmake .. ^&^& cmake --build .
)

cd /d %~dp0
server.exe
