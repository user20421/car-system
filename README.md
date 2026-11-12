# AURORA 车联网系统

本地运行的多用户、多车辆、四端车联网仿真。项目已按新架构重建，旧实现不再使用。

## 四端与基础设施

| 模块 | 目录 | 实现 |
|---|---|---|
| Go 后端 | `backend-go/` | HTTPS HTTP/2、WSS、内嵌 MQTTS Broker、JWT/mTLS、Outbox |
| Qt 车机 | `car-qt/` | Qt/QML 智能座舱、HAL 模拟器、多线程控制器 |
| Flutter 车主 App | `flutter_application/` | 多车车库、实时车况、远程控车、告警与趋势 |
| Vue 运营平台 | `admin-dashboard/` | 车队总览、地图、车辆详情、告警与远程控车 |
| 共享契约 | `contracts/` | Protobuf 与 OpenAPI |

Docker Compose 只有三个服务：MySQL 8.0、Redis 7.0、RabbitMQ 3。MySQL/Redis/RabbitMQ 分别承担持久化、实时状态和可靠业务事件；原始高频遥测不进入 RabbitMQ。

## 一键启动（Windows）

前置条件：Docker Desktop、Go 1.25+、Node.js、npm；车机构建另需 Qt 6/CMake/Ninja，App 开发另需 Flutter。

1. 首次运行前复制 `.env.example` 为 `.env`，替换全部 `CHANGE_ME`。仓库已有本地 `.env` 时无需重复操作。
2. 双击 `启动系统.bat`，或运行：

```powershell
.\start-all.ps1
```

脚本会生成开发证书、启动三个容器、等待健康检查、构建并启动后端、Vue 平台和已构建的 Qt 车机。Qt 车机使用真实 MQTTS/Protobuf；如跳过 Qt 或指定 `-UseGoDeviceSim`，则启动 Go 联调设备。

- 后端健康检查：`https://localhost:8443/health`（需信任 `deploy/certs/ca.crt`）
- 运营管理端：`http://127.0.0.1:5173`（业务 API/WSS 通过加密代理连接后端）
- MQTTS：`localhost:8883`，一机一证，无匿名/明文入口
- 同时启动 Flutter 演示 UI：`.\start-all.ps1 -IncludeFlutterDemo`

不要删除旧 Docker 卷。新系统使用 `car_refactor_*` 命名卷。

## 演示账号与车辆

| 账号 | 密码 | 权限 |
|---|---|---|
| `admin` | `Admin123!` | 管理全部三台车 |
| `owner` | `Owner123!` | 拥有并控制车 1、2 |
| `viewer` | `Viewer123!` | 只读共享车辆 1 |

车辆 VIN 为 `LTEST000000000001`～`LTEST000000000003`，对应 `deploy/certs/<VIN>.crt/.key`。可运行多个设备实例：

```powershell
Set-Location backend-go
go run ./cmd/devicesim -vin LTEST000000000001
go run ./cmd/devicesim -vin LTEST000000000002
```

`devicesim` 是真实 MQTTS + Protobuf 联调客户端，用于验证遥测和指令 ACK，不替代 Qt HAL。

## 协议摘要

- 车机实时遥测、控车、ACK、Presence：MQTTS + Protobuf。
- 车机配置、OTA 元数据、诊断和批量补传：HTTPS HTTP/2。
- App/管理端普通业务：HTTPS HTTP/2 + JSON。
- App/管理端实时事件：WSS + JSON，一次性短时效 ticket。
- 用户：ES256 短期 JWT + 旋转 Refresh Token；网页另支持 HttpOnly/Secure/SameSite=Strict Cookie 恢复会话。
- 设备：mTLS，证书 CN=VIN，Topic/API 按 VIN 授权。
- 控车：MySQL 事务 Outbox → RabbitMQ Publisher Confirm → MQTT QoS 1 → 车端应用 ACK。

详见 `docs/architecture.md`、`docs/protocol.md` 和 `contracts/`。

## 分手动启动

```powershell
# 基础设施与证书
go run .\scripts\generate-certs.go -out .\deploy\certs
docker compose up -d mysql redis rabbitmq

# 后端；先将根目录开发密码同步到 backend-go/.env
Set-Location backend-go
go run ./cmd/server

# 管理端
Set-Location ..\admin-dashboard
npm install
$env:NODE_EXTRA_CA_CERTS = '..\deploy\certs\ca.crt'
npm run dev

# Qt 车机
Set-Location ..\car-qt
.\启动车机.ps1 -Build -Vin LTEST000000000001

# Flutter 真实 API 模式
Set-Location ..\flutter_application
flutter pub get
flutter run --dart-define=AURORA_API_BASE_URL=https://localhost:8443
```

Flutter Web 演示模式使用 `AURORA_DEMO=true`，页面会明确标注演示数据。管理端演示模式使用 `VITE_USE_MOCK=true`，默认均为真实 API。

## 测试与验收

```powershell
# 四端构建、静态检查与单元测试
.\scripts\test-all.ps1

# 后端已运行时：真实 HTTP/2、多用户权限、Outbox/MQTT、Cookie/WSS ticket
.\scripts\test-all.ps1 -Smoke

# devicesim 已运行时：验证车辆执行终态和 MQTTS 遥测
go run .\scripts\smoke-api.go -expect-command succeeded
```

管理端另有 Playwright 界面脚本，使用本机 Chrome：`admin-dashboard/scripts/`。

## 当前边界

- QtMqtt 6.11.2 已在项目 `.tools/qtmqtt-install` 内从匹配源码安装，Qt Protobuf 和 `protoc` 已接入真实 `QMqttClient + MQTTS + Protobuf` 链路；Qt 遥测和车辆 ACK 已与后端联调成功。
- 内嵌 Broker 适合单 Go 实例、多车辆、多用户的本地仿真，不是生产高可用集群。
- 用户/设备目录、审计和车辆资料编辑的高级管理页面可演示，但真实后端目前只实现 OpenAPI 已声明的接口；管理端不会调用虚构端点。
- OTA 当前提供元数据查询，不执行程序自替换；Flutter 原生 HTTP/2 依赖 Android Cronet/iOS URLSession。
