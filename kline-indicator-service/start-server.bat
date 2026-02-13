@echo off
REM K线技术指标服务启动脚本

REM 设置Go环境变量
set GOROOT=d:\code\DevTools\go
set PATH=%GOROOT%\bin;%PATH%

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
echo   K线技术指标服务
echo ==========================================
echo 服务端口: %SERVER_PORT%
echo 外部API: %EXTERNAL_API_URL%
echo Redis: %REDIS_ADDR% (空则使用内存缓存)
echo 认证: %ENABLE_AUTH%
echo ==========================================

cd /d %~dp0
server.exe
