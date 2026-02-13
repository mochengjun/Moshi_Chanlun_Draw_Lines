@echo off
taskkill /F /IM server.exe 2>nul
set "GOROOT=d:/code/DevTools/go"
set "PATH=d:/code/DevTools/go/bin;%PATH%"
cd /d "d:/code/Moshi_Chanlun_Draw_Lines/kline-indicator-service"
go build -o server.exe ./cmd/server/
if %errorlevel% equ 0 (
    echo BUILD_SUCCESS
) else (
    echo BUILD_FAILED errorlevel=%errorlevel%
)
