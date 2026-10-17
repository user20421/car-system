# 车机模拟器启动脚本（本地开发用）
# 原理：把 Qt 运行时 bin 目录临时加入 PATH，再启动程序
param(
    [string]$QtRoot = "D:\software\Qt\6.11.2\llvm-mingw_64"   # Qt 安装目录（可用 -QtRoot 覆盖）
)

$ErrorActionPreference = "Stop"

$qtBin = "$QtRoot\bin"
if (-not (Test-Path "$qtBin\Qt6Core.dll")) {
    Write-Host "❌ Qt 运行时未找到：$qtBin（请用 -QtRoot 指定正确路径）" -ForegroundColor Red
    pause
    exit 1
}

$exe = "$PSScriptRoot\build\car-qt.exe"
if (-not (Test-Path $exe)) {
    Write-Host "❌ 车机程序不存在：$exe（请先执行 cmake --build build）" -ForegroundColor Red
    pause
    exit 1
}

$env:Path = "$qtBin;" + $env:Path
Start-Process $exe
Write-Host "车机模拟器已启动（仪表盘窗口应已弹出）" -ForegroundColor Green