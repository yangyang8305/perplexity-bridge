@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ========================================
echo   Pplx2Api 本地启动脚本
echo ========================================
echo.

:: 加载 .env 文件中的环境变量
for /f "tokens=1,* delims==" %%a in (.env) do (
    if not "%%a"=="" if not "%%b"=="" (
        set %%a=%%b
    )
)

:: 设置默认值
if "%ADDRESS%"=="" set ADDRESS=0.0.0.0:18080
if "%APIKEY%"=="" set APIKEY=pplx2api2026
if "%IS_INCOGNITO%"=="" set IS_INCOGNITO=true

echo [1/2] 启动服务...
start /B "" pplx2api.exe

timeout /t 3 >nul

echo [2/2] 检查状态...
tasklist /fi "imagename eq pplx2api.exe" 2>nul

echo.
echo ========================================
echo   访问地址: http://localhost:18080
echo   API Key:  %APIKEY%
echo ========================================
echo.
echo 按任意键关闭本窗口（服务仍在后台运行）
pause >nul
