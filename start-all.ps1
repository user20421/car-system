param(
    [string]$QtRoot = "D:\software\Qt\6.11.2\llvm-mingw_64",   # Qt 安装目录（可用 -QtRoot 覆盖）
    [string]$QtToolsRoot = "D:\software\Qt\Tools"                 # Qt Tools 目录（CMake/Ninja）
)
# ============================================================
# Car-System 车联网系统 一键启动脚本
# 启动顺序：Docker中间件 → Go后端 → Qt车机 → Vue后台 → Flutter App
# 用法：右键"使用 PowerShell 运行"，或直接双击 启动系统.bat
# ============================================================

$root = $PSScriptRoot
$QtBin = "$QtRoot\bin"
$QtTools = "$QtToolsRoot\CMake_64\bin;$QtToolsRoot\Ninja;$QtToolsRoot\llvm-mingw1706_64\bin;$QtBin"

function Write-Step($text) {
    Write-Host "`n========== $text ==========" -ForegroundColor Cyan
}

# ---------- 1. Docker 中间件 ----------
Write-Step "1/5 启动中间件 (MySQL / Redis / RabbitMQ)"
Set-Location $root
docker compose up -d | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker 未运行，请先启动 Docker Desktop 再执行本脚本" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "等待 MySQL 就绪..."
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 2
    docker exec car-system-mysql-1 mysqladmin ping -uroot -proot 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { $ready = $true; break }
}
if (-not $ready) { Write-Host "❌ MySQL 启动超时" -ForegroundColor Red; pause; exit 1 }
Write-Host "✅ 中间件就绪" -ForegroundColor Green

# ---------- 2. Go 后端 ----------
Write-Step "2/5 编译并启动 Go 后端"
Set-Location "$root\backend-go"
go build -o car-server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) { Write-Host "❌ 后端编译失败" -ForegroundColor Red; pause; exit 1 }
Get-Process car-server -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$root\backend-go'; .\car-server.exe"

Write-Host "等待后端就绪..."
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 2
    try {
        $h = Invoke-RestMethod http://localhost:8080/health -TimeoutSec 2
        if ($h.status -eq "ok") { $ready = $true; break }
    } catch {}
}
if (-not $ready) { Write-Host "❌ 后端启动超时" -ForegroundColor Red; pause; exit 1 }
Write-Host "✅ 后端就绪 (http://localhost:8080)" -ForegroundColor Green

# ---------- 3. Qt 车机模拟器 ----------
Write-Step "3/5 启动 Qt 车机模拟器"
Get-Process car-qt -ErrorAction SilentlyContinue | Stop-Process -Force
if (-not (Test-Path "$root\car-qt\build\car-qt.exe")) {
    Write-Host "首次运行，正在编译 Qt 项目..."
    $env:Path = "$QtTools;" + $env:Path
    Set-Location "$root\car-qt"
    cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release `
        -DCMAKE_PREFIX_PATH="$($QtRoot -replace '\\','/')" `
        -DCMAKE_CXX_COMPILER="$($QtToolsRoot -replace '\\','/')/llvm-mingw1706_64/bin/clang++.exe" | Out-Null
    cmake --build build | Out-Null
}
$env:Path = "$QtBin;" + $env:Path
Start-Process "$root\car-qt\build\car-qt.exe"
Write-Host "✅ 车机仪表盘已启动" -ForegroundColor Green

# ---------- 4. Vue 管理后台 ----------
Write-Step "4/5 启动 Vue 管理后台"
if (-not (Test-Path "$root\admin-dashboard\node_modules")) {
    Write-Host "首次运行，正在安装依赖..."
    Set-Location "$root\admin-dashboard"
    npm install | Out-Null
}
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$root\admin-dashboard'; npm run dev"
Write-Host "✅ 管理后台启动中 (http://localhost:5173)" -ForegroundColor Green

# ---------- 5. Flutter 车主 App ----------
Write-Step "5/5 启动 Flutter 车主 App"
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$root\flutter_application'; flutter pub get; flutter run -d chrome --web-port=5000"
Write-Host "✅ 车主 App 启动中 (http://localhost:5000，首次编译约需1分钟)" -ForegroundColor Green

# ---------- 完成 ----------
Write-Host "`n============================================" -ForegroundColor Green
Write-Host "  🎉 Car-System 全部组件已启动！" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host "  📊 管理后台:   http://localhost:5173  (admin / admin123456)"
Write-Host "  📱 车主 App:   http://localhost:5000  (driver1 / driver123456)"
Write-Host "  📖 API 文档:   http://localhost:8080/swagger/index.html"
Write-Host "  🐰 MQ 控制台:  http://localhost:15672 (car-system / car123456)"
Write-Host "  🚗 Qt 车机:    桌面仪表盘窗口"
Write-Host "============================================`n"
pause

