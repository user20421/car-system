# AURORA · Qt/QML 智能座舱

原创深色宽屏车机，包含智能桌面、数字仪表、导航、媒体、双区空调、车辆中心、通知和设置。QML 只发起意图并渲染 DTO；锁车、空调、温度、除霜等成功状态只能来自 HAL 的终态 ACK，UI 不会伪造执行成功。

## 架构

- `IVehicleHardware`：纯硬件边界；`SimulatedVehicleHardware` 内部持有全部模拟状态与动力学。
- `VehicleController`：GUI 线程 ViewModel，DTO 暴露 QML 属性；指令去重、VIN/过期校验、逐条执行、RECEIVED/终态 ACK。
- 独立 `simulated-hal` QThread（100 ms 周期）、`device-network` QThread（网络事件循环）、`storage-worker` QThread（原子保存 latest 遥测）。
- `QtMqttTransport` 使用 `QMqttClient` 实现真实 MQTTS：MQTT 5、TLS 1.2+、CA 与 VIN mTLS、唯一 ClientID、持久会话、指数退避重连、retained ONLINE/LWT OFFLINE。遥测 QoS 0，指令订阅与 ACK QoS 1。
- `VehicleProtobuf` 使用 Qt 6 Protobuf 生成的 `vehicle_v1.qpb.*` 编解码 Envelope、Telemetry、Command、CommandAck、Presence；校验 topic/VIN/序列/时间并执行 QoS 1 去重、过期和未知指令终态 ACK。
- 网络与存储使用 latest-value 合并；指令不使用 latest 合并。跨线程传递可复制的值 DTO，并显式使用 `Qt::QueuedConnection`。
- `BoundedTaskQueue` 基于独立 `QThreadPool`，限制线程与待执行任务数，调用方立即获得背压结果。
- 停止顺序：停止接收指令 → 停 HAL 周期 → HAL 线程退出 → 网络停止 → 存储 flush/退出 → 任务池退出。
- `qml/theme`、`components` 提供统一主题、矢量车辆、Canvas 地图/路线、仪表、按钮和状态组件；窄屏切换为底部导航。

## 依赖

- Windows 10/11，CMake 3.24+（已用 4.4.3 验证），C++17 编译器，Ninja。
- Qt 6.5+：Core、Gui、Qml、Quick、QuickControls2、Network、Test、**Qt Protobuf**；本机使用 Qt 6.11.2 llvm-mingw。
- **QtMqtt 6.11.2**，本机源码安装于 `../.tools/qtmqtt-install`；生成代码需要 `../.tools/protoc/bin/protoc.exe`（36.2）。Qt Protobuf 直接支持本协议的 message envelope 和 `map<string,string>`，不需要也不链接额外 Google libprotobuf。

### MQTTS 安全传输

`proto/vehicle_v1.proto` 与 `../contracts/proto/vehicle_v1.proto` 保持一致，由 `qt_add_protobuf`/`qtprotobufgen` 在构建目录生成代码。车端只调用 `connectToHostEncrypted`，端口仅接受 8883；CA、`<VIN>.crt`、`<VIN>.key` 任一缺失、无效或证书 CN 与 VIN 不符都会 fail closed。**不存在明文 MQTT 或 JSON MQTT 回退，也不会在连接失败时伪造在线。**

HTTPS 客户端已经实现 config/OTA/diagnostic 请求，所有请求显式设置 `QNetworkRequest::Http2AllowedAttribute`，只接受 `https://`，配置 CA 与 VIN mTLS 证书。批量补传仍明确失败，不用其他格式冒充 Protobuf TelemetryBatch。HTTP/2 是否实际协商成功取决于服务端和 Qt TLS 后端。

## 配置与构建

通用 PowerShell 示例（将 Qt 路径换成本机套件）：

```powershell
$env:PATH = "D:\software\Qt\Tools\llvm-mingw1706_64\bin;..\.tools\qtmqtt-install\bin;D:\software\Qt\6.11.2\llvm-mingw_64\bin;..\.tools\protoc\bin;D:\software\Qt\Tools\Ninja;$env:PATH"
cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Debug `
  -DCMAKE_PREFIX_PATH="D:/code/car-system/.tools/qtmqtt-install;D:/software/Qt/6.11.2/llvm-mingw_64" `
  -DCMAKE_CXX_COMPILER="D:/software/Qt/Tools/llvm-mingw1706_64/bin/clang++.exe"
cmake --build build --parallel 4
ctest --test-dir build --output-on-failure
.\build\aurora.exe --vin LTEST000000000001 --debug
```

也可使用：

```powershell
.\启动车机.ps1 -Build -Vin LTEST000000000001 -DebugPanel
```

发布部署需运行与套件匹配的 `windeployqt --qmldir qml <aurora.exe 路径>`；开发脚本通过 Qt `bin` PATH 运行，不修改系统环境。

## 运行与多车机实例

命令行优先级高于 JSON 配置。参考 `config/aurora.example.json`：

```powershell
# 每个进程使用唯一 VIN、自己的模拟 HAL 和按 VIN 隔离的 latest 本地文件
.\build\aurora.exe --vin LTEST000000000001 --cert-dir ..\deploy\certs --https-base https://localhost:8443 --debug
.\build\aurora.exe --vin LTEST000000000002 --cert-dir ..\deploy\certs --https-base https://localhost:8443 --debug
.\build\aurora.exe --config config\aurora.example.json
```

参数：`--vin/-v`、`--config`、`--mqtt-host`、`--mqtt-port`（仅 8883 MQTTS 语义）、`--https-base`、`--cert-dir`、`--debug`、`--page 0..7`。证书目录只读取 `ca.crt`、`<VIN>.crt`、`<VIN>.key`；请使用开发 CA，生产密钥应进入安全 provisioning。

界面设置页可触发 HTTPS 配置、OTA、诊断和离线批次操作，加载/失败结果来自实际网络回调。Debug 构建或 `--debug` 下，左侧底部 `∆` 打开城市/高速/驻车/充电/故障场景；场景由模拟 HAL 执行。

## 验证

- `aurora_tests`：HAL 遥测、真实状态变更、过期/非法参数，控制器 ACK 与串行队列、VIN/重复指令，任务池背压，以及 Qt Protobuf 遥测/指令 map/未知类型/VIN 编解码校验。
- 界面离屏冒烟：

```powershell
.\build\aurora.exe -platform offscreen --screenshot build\aurora-smoke.png
```

- 静态检查可使用 Qt 套件 `qmllint`，导入路径指向构建目录；QML 同时经过 `qmlcachegen` 编译。

本机实测：Qt 6.11.2 + Clang 17 + Ninja 真实构建成功，CTest 聚合测试通过。已使用车辆 CA/证书连接 `localhost:8883`，界面显示“车辆在线”；随后通过 `scripts/smoke-api.go -expect-command succeeded` 完成 RabbitMQ → MQTTS → Qt HAL → Protobuf ACK 的真实闭环验证。未执行真实触摸屏、音频设备或 HTTPS 服务端集成测试。
