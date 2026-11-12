[CmdletBinding()]
param([switch]$Smoke)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..')
Set-Location $root

docker compose config --quiet
if ($LASTEXITCODE -ne 0) { throw 'Compose 配置无效' }

Set-Location 'backend-go'
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go 测试失败' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet 失败' }
go build -o car-server.exe ./cmd/server
go build -o devicesim.exe ./cmd/devicesim
if ($LASTEXITCODE -ne 0) { throw 'Go 构建失败' }

Set-Location '..\car-qt'
if (Test-Path 'build\aurora_tests.exe') {
    $qtRoot = $(if ($env:QT6_ROOT) { $env:QT6_ROOT } else { 'D:\software\Qt\6.11.2\llvm-mingw_64' })
    $compilerRoot = $(if ($env:LLVM_MINGW_ROOT) { $env:LLVM_MINGW_ROOT } else { 'D:\software\Qt\Tools\llvm-mingw1706_64' })
    $mqttRoot = $(if ($env:QTMQTT_ROOT) { $env:QTMQTT_ROOT } else { "$root\.tools\qtmqtt-install" })
    $env:PATH = "$compilerRoot\bin;$qtRoot\bin;$mqttRoot\bin;$root\.tools\protoc\bin;$env:PATH"
    ctest --test-dir build --output-on-failure
    if ($LASTEXITCODE -ne 0) { throw 'Qt 测试失败' }
} else {
    Write-Warning 'Qt build 不存在，跳过本机 Qt 测试；请先执行构建。'
}

Set-Location '..\admin-dashboard'
npm run typecheck
if ($LASTEXITCODE -ne 0) { throw 'Vue 类型检查失败' }
npm test
if ($LASTEXITCODE -ne 0) { throw 'Vue 测试失败' }
npm run build
if ($LASTEXITCODE -ne 0) { throw 'Vue 构建失败' }
npm audit --omit=dev
if ($LASTEXITCODE -ne 0) { throw '管理端生产依赖存在已知漏洞' }

Set-Location '..\flutter_application'
flutter analyze
if ($LASTEXITCODE -ne 0) { throw 'Flutter analyze 失败' }
flutter test
if ($LASTEXITCODE -ne 0) { throw 'Flutter 测试失败' }
flutter build web --release --no-wasm-dry-run
if ($LASTEXITCODE -ne 0) { throw 'Flutter Web 构建失败' }

if ($Smoke) {
    Set-Location $root
    go run .\scripts\smoke-api.go
    if ($LASTEXITCODE -ne 0) { throw '真实 API 冒烟测试失败' }
}
Set-Location $root
Write-Host '全部可执行检查已完成。'
