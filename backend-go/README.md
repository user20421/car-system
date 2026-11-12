# Car System Go Backend

模块化单体，入口为 `cmd/server`。用户/API 流量仅暴露 HTTPS（HTTP/2，`:8443`），车辆实时流量仅暴露要求客户端证书的 MQTTS（`:8883`）。MySQL、Redis 与 RabbitMQ 客户端均强制 TLS；没有明文业务监听器。

## 模块

- `internal/config`：环境变量/`.env`、启动校验、ES256 私钥文件。
- `internal/auth`：Argon2id、ES256 access token（15 分钟）、不透明 refresh token 旋转与会话撤销。
- `internal/store`：显式内嵌 SQL migration、事务 outbox、权限过滤、幂等记录。
- `internal/telemetry`：Protobuf 校验、有界 channel、固定 batch worker、Redis 最新值；不经过 RabbitMQ。
- `internal/events` + `internal/commands`：AMQPS publisher confirm、quorum queue、manual ack、有限重试/DLX、按 VIN 串行分发与指令状态机。
- `internal/mqttbroker`：内嵌 Mochi MQTT、设备 mTLS CN=VIN、每车 topic ACL、telemetry/command/ack/presence。
- `internal/httpapi` + `internal/realtime`：OpenAPI 用户/设备接口、一次性 WSS ticket、按车辆权限过滤的 JSON 推送。
- `pkg/vehiclev1`：由权威 `contracts/proto/vehicle_v1.proto` 生成的 Go 代码。

## 启动

前置条件：Go 1.25+（本项目使用 Go 1.27.1 验证）、根目录开发证书，以及通过根目录 Compose 启动的 MySQL 8.0、Redis 7.0 和 RabbitMQ 3。后端不会绕过这些依赖启动。

```powershell
# 在仓库根目录，按根目录说明准备 .env 与 deploy/certs 后启动依赖
# docker compose up -d mysql redis rabbitmq

Set-Location backend-go
Copy-Item .env.example .env
# 编辑 .env，使三个 CHANGE_ME 密码与根目录 Compose 环境一致。
go run ./cmd/server
```

生产环境必须设置 `APP_ENV=production` 并预先安全 provision `JWT_PRIVATE_KEY_FILE`。开发环境缺少该文件时会生成 P-256 EC 私钥到 `.secrets/jwt-es256.key`（权限 0600）；私钥不会写入数据库。refresh token 仅以 SHA-256 哈希保存。

启动时自动执行 `internal/store/migrations/*.sql`，记录 checksum，并幂等创建开发种子数据：

| 账号 | 密码 | 权限 |
|---|---|---|
| `admin` | `Admin123!` | 管理全部三台车 |
| `owner` | `Owner123!` | 拥有车 1、2 |
| `viewer` | `Viewer123!` | 只读车 1 |

车辆为 `LTEST000000000001`～`LTEST000000000003`。设备证书使用根目录 `deploy/certs/<VIN>.crt` 和 `.key`，证书 CN 必须与 VIN 一致。

健康检查：`GET https://localhost:8443/health`。其余 REST 路径遵循 `contracts/openapi/openapi.yaml`；WSS 地址由 `POST /api/v1/ws-ticket` 返回，ticket 默认 60 秒且只能兑换一次。网页登录/刷新同时设置 HttpOnly、Secure、SameSite=Strict 的 refresh cookie，整页刷新可恢复会话；原生客户端继续使用 JSON refresh token。开发证书不受系统信任时，测试客户端需指定 `deploy/certs/ca.crt`，不要在生产关闭证书校验。

## 测试与构建

以下测试不需要 Docker，覆盖密码/JWT/refresh 旋转与撤销、RBAC+ABAC、状态机、遥测校验、Protobuf、MQTT ACL、WSS ticket 和 batch worker：

```powershell
Set-Location backend-go
go test ./...
go test -race ./internal/auth ./internal/domain ./internal/realtime ./internal/telemetry ./internal/mqttbroker
go vet ./...
go build -o car-server.exe ./cmd/server
```

真实 MySQL/Redis/RabbitMQ/MQTT 端到端测试需要上述基础设施；单元测试通过不等于集成测试通过。依赖和后端均已启动时，可运行：

```powershell
$env:CAR_INTEGRATION = "1"
go test -count=1 -v ./test
Remove-Item Env:CAR_INTEGRATION
```

该用例会真实验证 HTTP/2、登录与 RBAC、指令事务/outbox/AMQPS 分发、refresh 旋转与重用撤销、一次性 WSS、设备 mTLS config、HTTPS 遥测批量、Redis 最新值，以及 MQTTS 遥测和 QoS1 ACK 直至指令 `succeeded`。未设置 `CAR_INTEGRATION` 时自动跳过。

## 重新生成 Protobuf

仓库根目录提供 `.tools/bin/buf.exe`，无需系统 `protoc`。首次将插件安装到本目录：

```powershell
Set-Location backend-go
New-Item -ItemType Directory -Force .bin
$env:GOBIN = (Resolve-Path .bin).Path
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
../.tools/bin/buf.exe generate ../contracts/proto --template buf.gen.yaml
gofmt -w pkg/vehiclev1
```

`buf.gen.yaml` 的 managed mode 将 Go import path 重写到本模块，不修改权威 proto。

## 当前运维限制

- RabbitMQ 初始连接失败会拒绝启动；运行时长连接断开后会反映为 unhealthy，但自动重连/拓扑重建需由进程监督器重启完成。
- MQTT broker 使用内存会话；进程重启后持久业务指令仍由 MySQL outbox/RabbitMQ 恢复，但 MQTT 离线会话本身不跨进程保留。
- WSS 是实时分发而非可靠事件日志；车辆权限快照在连接建立时计算，权限变更后应重建连接。
- 设备 OTA 元数据由 `ota_releases` 表管理；本服务只提供最新版本查询，不代理固件文件下载。
