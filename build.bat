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

REM 生成 exe 图标资源（需要网络安装 go-winres: go install github.com/tc-hib/go-winres@latest）
REM 临时跳过图标，gen_syso.py 生成的 syso 会导致 PE 损坏
REM echo [喵喵云结算] 生成图标资源...
REM cd /d "%~dp0"
REM if not exist "rsrc_windows_amd64.syso" (
REM     if exist "monkey.ico" (
REM         go-winres make --product-version=1.0.0 --file-version=1.0.0 --icon=monkey.ico
REM     )
REM )

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
"C:\Program Files\Go\bin\go.exe" build -ldflags="-s -w -H windowsgui" -o yunfei.exe .
if %ERRORLEVEL% NEQ 0 (
    echo [错误] Go 编译失败！
    pause
    exit /b 1
)

echo [喵喵云结算] ✓ 编译完成！输出: yunfei.exe
for %%A in (yunfei.exe) do echo [喵喵云结算] 文件大小: %%~zA 字节

REM 清理缓存
echo [喵喵云结算] 清理构建缓存...
rmdir /s /q "%~dp0build\go-cache" 2>nul
rmdir /s /q "%~dp0build\go-tmp" 2>nul

pause
