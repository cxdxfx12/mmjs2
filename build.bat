@echo off
REM ============================================
REM 喵喵云结算 - 一键编译脚本
REM 用法: 双击运行或在 PowerShell 中执行 .\build.bat
REM ============================================
echo [喵喵云结算] 开始编译...

REM 设置 Go 环境
set GOTOOLCHAIN=local
set GOCACHE=%~dp0build\go-cache
set GOTMPDIR=%~dp0build\go-tmp

REM 创建缓存目录
if not exist "%~dp0build\go-cache" mkdir "%~dp0build\go-cache"
if not exist "%~dp0build\go-tmp" mkdir "%~dp0build\go-tmp"

REM 添加 Windows Defender 排除 (需管理员权限)
echo [喵喵云结算] 正在添加 Windows Defender 排除路径...
powershell -Command "Add-MpPreference -ExclusionPath '%~dp0' -ErrorAction SilentlyContinue" >nul 2>&1

REM 生成 exe 图标资源（优先 rsrc）
echo [喵喵云结算] 生成 exe 图标资源...
set "RSRC=%USERPROFILE%\go\bin\rsrc.exe"
if exist "%RSRC%" (
  "%RSRC%" -ico "%~dp0monkey.ico" -arch amd64 -o "%~dp0rsrc_windows_amd64.syso"
) else (
  python "%~dp0ico_to_syso.py" "%~dp0monkey.ico" "%~dp0rsrc_windows_amd64.syso" amd64
)
if %ERRORLEVEL% NEQ 0 (
    echo [警告] 图标资源生成失败，将尝试使用已有 rsrc_windows_amd64.syso
)

REM 编译前端
echo [喵喵云结算] 编译前端...
cd /d "%~dp0frontend"
call npm run build
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 前端编译失败！
    pause
    exit /b 1
)

REM Go 编译 (保留CMD窗口方便调试)
echo [喵喵云结算] Go 编译...
cd /d "%~dp0"
"C:\Program Files\Go\bin\go.exe" build -ldflags="-s -w -H windowsgui" -o _build_temp.exe .
if %ERRORLEVEL% NEQ 0 (
    echo [错误] Go 编译失败！
    pause
    exit /b 1
)

if exist "喵喵云结算.exe" del /f /q "喵喵云结算.exe"
move /y _build_temp.exe "喵喵云结算.exe" >nul
if exist "%~dp0yunfei.exe" del /f /q "%~dp0yunfei.exe"

echo [喵喵云结算] ✓ 编译完成！输出: 喵喵云结算.exe
for %%A in ("喵喵云结算.exe") do echo [喵喵云结算] 文件大小: %%~zA 字节

REM 清理缓存
echo [喵喵云结算] 清理构建缓存...
rmdir /s /q "%~dp0build\go-cache" 2>nul
rmdir /s /q "%~dp0build\go-tmp" 2>nul

pause
