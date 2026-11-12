param(
    [string]$QtRoot = $(if ($env:QT6_ROOT) { $env:QT6_ROOT } else { 'D:\software\Qt\6.11.2\llvm-mingw_64' }),
    [string]$CompilerRoot = $(if ($env:LLVM_MINGW_ROOT) { $env:LLVM_MINGW_ROOT } else { 'D:\software\Qt\Tools\llvm-mingw1706_64' }),
    [string]$MqttRoot = $(if ($env:QTMQTT_ROOT) { $env:QTMQTT_ROOT } else { "$PSScriptRoot\..\.tools\qtmqtt-install" }),
    [string]$ProtocRoot = $(if ($env:PROTOC_ROOT) { $env:PROTOC_ROOT } else { "$PSScriptRoot\..\.tools\protoc" }),
    [string]$Vin = 'LTEST000000000001',
    [string]$Config = '',
    [switch]$Build,
    [switch]$DebugPanel
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$buildDir = Join-Path $root 'build'
$qtBin = Join-Path $QtRoot 'bin'
$env:PATH = "$CompilerRoot\bin;$MqttRoot\bin;$qtBin;$ProtocRoot\bin;$env:PATH"

if ($Build -or -not (Test-Path (Join-Path $buildDir 'aurora.exe'))) {
    $ninja = Get-Command ninja -ErrorAction Stop
    cmake -S $root -B $buildDir -G Ninja -DCMAKE_BUILD_TYPE=Debug `
        "-DCMAKE_PREFIX_PATH=$($MqttRoot -replace '\\','/');$($QtRoot -replace '\\','/')" `
        "-DCMAKE_CXX_COMPILER=$($CompilerRoot -replace '\\','/')/bin/clang++.exe"
    if ($LASTEXITCODE -ne 0) { throw 'CMake configure failed' }
    cmake --build $buildDir --parallel 4
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
}

$arguments = @('--vin', $Vin, '--cert-dir', (Resolve-Path "$root\..\deploy\certs").Path)
if ($Config) { $arguments += @('--config', $Config) }
if ($DebugPanel) { $arguments += '--debug' }
& (Join-Path $buildDir 'aurora.exe') @arguments
exit $LASTEXITCODE
