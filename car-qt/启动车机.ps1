# 车机模拟器启动脚本（本地开发用）
# 原理：把 Qt 运行时 bin 目录临时加入 PATH，再启动程序
$env:Path = "D:\software\Qt\6.11.2\llvm-mingw_64\bin;" + $env:Path
Start-Process "$PSScriptRoot\build\car-qt.exe"
Write-Host "车机模拟器已启动（仪表盘窗口应已弹出）"
