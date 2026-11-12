[CmdletBinding()]
param(
    [switch]$SkipInfrastructure,
    [switch]$SkipQt,
    [switch]$UseGoDeviceSim,
    [switch]$IncludeFlutterDemo
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root
$qtRoot = $(if ($env:QT6_ROOT) { $env:QT6_ROOT } else { 'D:\software\Qt\6.11.2\llvm-mingw_64' })
$compilerRoot = $(if ($env:LLVM_MINGW_ROOT) { $env:LLVM_MINGW_ROOT } else { 'D:\software\Qt\Tools\llvm-mingw1706_64' })
$env:PATH = "$compilerRoot\bin;$qtRoot\bin;$root\.tools\qtmqtt-install\bin;$root\.tools\protoc\bin;$env:PATH"

function Start-Terminal([string]$Title, [string]$WorkDir, [string]$Command) {
    $escapedDir = $WorkDir.Replace("'", "''")
    $escapedCommand = $Command.Replace("'", "''")
    Start-Process powershell.exe -ArgumentList @(
        '-NoExit', '-Command',
        "`$host.UI.RawUI.WindowTitle='$Title'; Set-Location '$escapedDir'; $escapedCommand"
    ) -WorkingDirectory $WorkDir | Out-Null
}

if (-not (Test-Path '.env')) {
    Copy-Item '.env.example' '.env'
    throw '已创建 .env。请先将所有 CHANGE_ME 改为本地开发密码，再重新运行本脚本。'
}
if ((Get-Content '.env' -Raw) -match 'CHANGE_ME') {
    throw '请先替换根目录 .env 中的所有 CHANGE_ME。'
}
if (-not (Test-Path 'deploy\certs\ca.crt')) {
    go run .\scripts\generate-certs.go -out .\deploy\certs
    if ($LASTEXITCODE -ne 0) { throw '证书生成失败' }
}

if (-not $SkipInfrastructure) {
    docker compose up -d mysql redis rabbitmq
    if ($LASTEXITCODE -ne 0) { throw '基础设施启动失败' }
    $deadline = (Get-Date).AddMinutes(3)
    do {
        $states = @('mysql', 'redis', 'rabbitmq') | ForEach-Object {
            docker inspect --format '{{.State.Health.Status}}' "car_refactor-$_-1" 2>$null
        }
        if (($states | Where-Object { $_ -ne 'healthy' }).Count -eq 0) { break }
        Start-Sleep -Seconds 3
    } while ((Get-Date) -lt $deadline)
    if (($states | Where-Object { $_ -ne 'healthy' }).Count -ne 0) {
        docker compose ps
        throw '基础设施未在 3 分钟内全部 healthy，请运行 docker compose logs 查看原因。'
    }
}

# 将根目录开发密码同步到后端，不在命令行或日志中输出密码。
$values = @{}
Get-Content '.env' | ForEach-Object {
    if ($_ -match '^([A-Z0-9_]+)=(.*)$') { $values[$matches[1]] = $matches[2] }
}
$backendEnv = Get-Content 'backend-go\.env.example' | ForEach-Object {
    $_ -replace 'CHANGE_ME_MYSQL_APP', $values['MYSQL_PASSWORD'] `
       -replace 'CHANGE_ME_REDIS_APP', $values['REDIS_PASSWORD'] `
       -replace 'CHANGE_ME_RABBITMQ_APP', $values['RABBITMQ_PASSWORD']
}
Set-Content 'backend-go\.env' $backendEnv

Set-Location 'backend-go'
go build -o car-server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) { throw '后端构建失败' }
go build -o devicesim.exe ./cmd/devicesim
if ($LASTEXITCODE -ne 0) { throw '设备联调客户端构建失败' }
Set-Location $root

Start-Terminal 'AURORA Backend' "$root\backend-go" '.\car-server.exe'
Start-Sleep -Seconds 4
if ($UseGoDeviceSim -or $SkipQt) {
    Start-Terminal 'AURORA Device MQTTS' "$root\backend-go" '.\devicesim.exe -vin LTEST000000000001'
}

if (-not (Test-Path 'admin-dashboard\node_modules')) {
    Set-Location 'admin-dashboard'
    npm install
    if ($LASTEXITCODE -ne 0) { throw '管理端依赖安装失败' }
    Set-Location $root
}
$adminCommand = "`$env:NODE_EXTRA_CA_CERTS='$root\deploy\certs\ca.crt'; `$env:VITE_USE_MOCK='false'; npm run dev -- --host 127.0.0.1 --port 5173"
Start-Terminal 'AURORA Operations' "$root\admin-dashboard" $adminCommand

if (-not $SkipQt) {
    if (Test-Path 'car-qt\build\aurora.exe') {
        Start-Process 'car-qt\build\aurora.exe' -ArgumentList '--vin','LTEST000000000001','--cert-dir','..\deploy\certs' -WorkingDirectory "$root\car-qt"
    } else {
        Write-Host 'Qt 尚未构建，正在调用本机构建脚本……'
        Start-Terminal 'AURORA Cockpit Build' "$root\car-qt" '.\启动车机.ps1 -Build -Vin LTEST000000000001'
    }
}

if ($IncludeFlutterDemo) {
    Start-Terminal 'AURORA Owner Demo' "$root\flutter_application" "flutter pub get; flutter run -d chrome --dart-define=AURORA_DEMO=true"
}

Write-Host ''
Write-Host 'AURORA 启动完成：'
Write-Host '  后端健康检查: https://localhost:8443/health'
Write-Host '  运营管理端:   http://127.0.0.1:5173 （业务 API/WSS 经 HTTPS 代理）'
Write-Host '  MQTTS:        localhost:8883（一机一证）'
Write-Host '  管理账号:     admin / Admin123!；车主: owner / Owner123!；只读: viewer / Viewer123!'
Write-Host '  Qt 使用真实 MQTTS/Protobuf；如需改用 Go devicesim，请添加 -UseGoDeviceSim。'
