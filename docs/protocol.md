# 通信协议（v1）

## 通道与格式

| 通道 | 用途 | 格式与安全 |
| --- | --- | --- |
| MQTTS `8883` | 车端实时遥测、在线状态、控车与 ACK | Protobuf；TLS 1.2+；车辆证书 CN 为 VIN |
| HTTPS（应用层应启用 HTTP/2） | 登录、车辆、历史遥测、批量补传、配置、OTA | 用户/配置使用 JSON，批量补传使用 Protobuf；用户 JWT Bearer，设备接口 mTLS |
| WSS | 向用户端推送遥测摘要、指令状态、告警 | JSON；先由 HTTPS 获取短时、一次性 ticket |
| AMQPS `5671` | 后端内部可靠业务事件 | 应用账号 `car_app`，独立 vhost `car` |

REST 与 WebSocket 的消息结构以 `contracts/openapi/openapi.yaml` 为准；MQTT 以 `contracts/proto/vehicle_v1.proto` 为准。时间戳字段 `_at_ms` 均为 UTC Unix 毫秒。

## MQTTS topic

| Topic | 发布者 → 订阅者 | QoS | Retain |
| --- | --- | --- | --- |
| `vehicles/{vin}/telemetry/v1` | 车辆 → 后端 | 0（关键快照可用 1） | 否 |
| `vehicles/{vin}/command/v1` | 后端 → 车辆 | 1 | 否 |
| `vehicles/{vin}/ack/v1` | 车辆 → 后端 | 1 | 否 |
| `vehicles/{vin}/status/v1` | 车辆/后端 → 后端/用户分发层 | 1 | 是 |

车辆离线消息由 Broker LWT 发布为 retained `Presence=OFFLINE`，正常上线后立即以 retained `Presence=ONLINE` 覆盖。QoS 1 只允许至少一次投递，消费者必须以 `message_id`、`command_id` 或 `batch_id` 去重，不能把它当成恰好一次。

每个 MQTT payload 的 `Envelope` 必须包含 `version/message_id/vin/device_boot_id/sequence/occurred_at_ms`。topic 中 VIN、Envelope VIN 与证书 CN 必须一致；Broker/后端 ACL 应限制每辆车只能访问自己的 topic。指令应使用 MQTT 5 Message Expiry，并同时校验 Protobuf `expires_at_ms`。

## 指令状态机

```text
pending → sent → received → succeeded
                    └────→ failed
pending/sent/received ────→ expired（均为终态）
```

1. 用户通过 HTTPS 创建指令，服务端校验车辆权限及 `idempotency_key`，持久化后以 QoS 1 下发。
2. 同一用户、车辆和幂等键重复请求返回原指令；不同参数复用同一键返回 `409`。
3. 车辆先回 `RECEIVED`，再回唯一的终态 `SUCCEEDED/FAILED/EXPIRED`。到期且尚未执行必须回 `EXPIRED`。
4. 终态不可逆转；重复 ACK 幂等处理。失败使用结构化 `CommandError`，不得仅依赖自由文本。

## RabbitMQ 边界

RabbitMQ **仅承载需要可靠投递的业务事件**，例如指令状态变更、告警确认、用户车辆关系变更和 OTA 任务。事件需有事件 ID、类型、版本和发生时间，由消费者幂等处理。

**实时遥测及遥测批量补传均不得进入 RabbitMQ。** 遥测从 MQTT/HTTPS 直接写入遥测存储与 Redis 最新值，避免高频数据压垮业务事件队列。WSS 是用户侧分发通道，不是可靠事件存储。
