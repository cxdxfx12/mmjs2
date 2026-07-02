@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

if "%~1"=="" (
    set PORT=58080
) else (
    set PORT=%~1
)

echo 正在查找占用端口 !PORT! 的进程...

for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":!PORT! "') do (
    set PID=%%a
    if not "!PID!"=="0" (
        echo 发现进程 PID: !PID!
        taskkill /F /PID !PID! >nul 2>&1
        if !errorlevel!==0 (
            echo 已终止进程 !PID!
        ) else (
            echo 终止进程 !PID! 失败
        )
    )
)

echo 完成。
pause
