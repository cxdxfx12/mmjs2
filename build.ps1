# 喵喵云结算 - 快速构建脚本 (PowerShell)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSCommandPath
$exeName = (-join [char[]]@(0x55B5, 0x55B5, 0x4E91, 0x7ED3, 0x7B97)) + ".exe"

# 设置环境变量
$env:GOTOOLCHAIN = "local"
$env:GOCACHE = "$root\build\go-cache"
$env:GOTMPDIR = "$root\build\go-tmp"

# 创建缓存目录
New-Item -ItemType Directory -Force -Path "$root\build\go-cache", "$root\build\go-tmp" | Out-Null

# 添加 Windows Defender 排除
Write-Host "[喵喵云结算] 添加 Windows Defender 排除..." -ForegroundColor Cyan
try { Add-MpPreference -ExclusionPath $root -ErrorAction SilentlyContinue } catch {}

# 生成 exe 图标资源（优先 rsrc，备用 ico_to_syso.py）
Write-Host "[喵喵云结算] 生成 exe 图标资源..." -ForegroundColor Cyan
$icoPath = Join-Path $root "monkey.ico"
$sysoPath = Join-Path $root "rsrc_windows_amd64.syso"
$icoScript = Join-Path $root "ico_to_syso.py"
$rsrcExe = Join-Path $env:USERPROFILE "go\bin\rsrc.exe"
if ((Test-Path $icoPath) -and (Test-Path $rsrcExe)) {
  & $rsrcExe -ico $icoPath -arch amd64 -o $sysoPath
  if ($LASTEXITCODE -ne 0) { throw "rsrc 生成图标资源失败" }
} elseif ((Test-Path $icoPath) -and (Test-Path $icoScript)) {
  python $icoScript $icoPath $sysoPath amd64
} else {
  Write-Host "[喵喵云结算] 警告: 未找到图标工具，将使用已有 syso 资源" -ForegroundColor Yellow
}

# 编译前端
Write-Host "[喵喵云结算] 编译前端..." -ForegroundColor Cyan
Push-Location "$root\frontend"
npm run build
Pop-Location

# Go 编译 (带图标 + 隐藏 CMD 窗口)
Write-Host "[喵喵云结算] Go 编译..." -ForegroundColor Cyan
Push-Location $root
$tempExe = Join-Path $root "_build_temp.exe"
$exePath = Join-Path $root $exeName
Remove-Item -Force $tempExe, $exePath -ErrorAction SilentlyContinue
go build -ldflags="-s -w -H windowsgui" -o $tempExe .
if (Test-Path $exePath) { Remove-Item -Force $exePath }
[System.IO.File]::Move($tempExe, $exePath)
Pop-Location
$size = (Get-Item $exePath).Length
Write-Host "[喵喵云结算] ✓ 编译完成！$exeName ($([math]::Round($size/1MB,2)) MB)" -ForegroundColor Green

# 清理旧产物
Remove-Item -Force (Join-Path $root "yunfei.exe") -ErrorAction SilentlyContinue

# 清理
Write-Host "[喵喵云结算] 清理缓存..." -ForegroundColor Cyan
Remove-Item -Recurse -Force "$root\build\go-cache", "$root\build\go-tmp" -ErrorAction SilentlyContinue
