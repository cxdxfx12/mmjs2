# 喵喵云结算 - 快速构建脚本 (PowerShell)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSCommandPath

# 设置环境变量
$env:GOTOOLCHAIN = "local"
$env:GOCACHE = "$root\build\go-cache"
$env:GOTMPDIR = "$root\build\go-tmp"

# 创建缓存目录
New-Item -ItemType Directory -Force -Path "$root\build\go-cache", "$root\build\go-tmp" | Out-Null

# 添加 Windows Defender 排除
Write-Host "[喵喵云结算] 添加 Windows Defender 排除..." -ForegroundColor Cyan
try { Add-MpPreference -ExclusionPath $root -ErrorAction SilentlyContinue } catch {}

# 编译前端
Write-Host "[喵喵云结算] 编译前端..." -ForegroundColor Cyan
Push-Location "$root\frontend"
npm run build
Pop-Location

# Go 编译 (无混淆，带图标+隐藏CMD窗口)
Write-Host "[喵喵云结算] Go 编译..." -ForegroundColor Cyan
Push-Location $root
go build -ldflags="-s -w -H windowsgui" -o yunfei.exe .
Pop-Location

$size = (Get-Item "$root\yunfei.exe").Length
Write-Host "[喵喵云结算] ✓ 编译完成！yunfei.exe ($([math]::Round($size/1MB,2)) MB)" -ForegroundColor Green

# 清理
Write-Host "[喵喵云结算] 清理缓存..." -ForegroundColor Cyan
Remove-Item -Recurse -Force "$root\build\go-cache", "$root\build\go-tmp" -ErrorAction SilentlyContinue
