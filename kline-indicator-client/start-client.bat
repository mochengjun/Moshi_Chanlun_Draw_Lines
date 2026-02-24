@echo off
REM K线技术指标客户端启动脚本

REM 设置Node.js环境
set NODE_HOME=d:\tools\node-v20.18.0-win-x64
set PATH=%NODE_HOME%;%NODE_HOME%\node_modules\npm\bin;%PATH%

echo ==========================================
echo   K线技术指标客户端
echo ==========================================
echo Node.js版本:
node --version
if %ERRORLEVEL% NEQ 0 (
    echo [错误] Node.js未找到，请检查路径: %NODE_HOME%
    pause
    exit /b 1
)
echo ==========================================
echo 启动开发服务器 (端口5173)
echo 请确保Go后端服务已在8080端口运行
echo ==========================================

cd /d %~dp0
node node_modules\vite\bin\vite.js --host
