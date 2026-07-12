@echo off
chcp 65001 >nul
title Pplx2Api - 本地服务
cd /d "%~dp0"

echo ========================================
echo   Pplx2Api 启动脚本
echo ========================================
echo.

:: 读取 .env 中的 SESSIONS 和 APIKEY
for /f "usebackq tokens=1,* delims==" %%a in (".env") do (
    if /i "%%a"=="SESSIONS" set "SESSIONS=%%b"
    if /i "%%a"=="APIKEY" set "APIKEY=%%b"
)

:: 设置其他环境变量
set ADDRESS=0.0.0.0:18080
set IS_INCOGNITO=true
set IGNORE_MODEL_MONITORING=true
set NO_ROLE_PREFIX=true
set MAX_CHAT_HISTORY_LENGTH=1000000

echo [1/2] 启动服务...
start "pplx2api" /B /MIN cmd /c "set SESSIONS=%SESSIONS% && set APIKEY=%APIKEY% && set ADDRESS=%ADDRESS% && set IS_INCOGNITO=%IS_INCOGNITO% && set IGNORE_MODEL_MONITORING=%IGNORE_MODEL_MONITORING% && set NO_ROLE_PREFIX=%NO_ROLE_PREFIX% && set MAX_CHAT_HISTORY_LENGTH=%MAX_CHAT_HISTORY_LENGTH% && start /B pplx2api.exe"

timeout /t 3 >nul

echo [2/2] 检查状态...
tasklist /fi "imagename eq pplx2api.exe" 2>nul | findstr pplx2api >nul
if %errorlevel%==0 (
    echo 服务已启动！
) else (
    echo 启动失败，请检查 .env 配置
)

echo.
echo ========================================
echo   访问地址: http://localhost:18080
echo   API Key:  %APIKEY%
echo ========================================
echo.
echo 按任意键关闭本窗口（服务继续在后台运行）
pause >nul
