# 重构架构

## 四端职责

- `backend-go`：HTTPS/WSS 服务、认证授权、车辆权限、设备接入、指令编排及数据持久化。
- `car-qt`：车机交互与硬件适配，通过 MQTTS/HTTPS 使用版本化车辆协议。
- `flutter_application`：个人用户移动端，管理本人可访问的多辆车并接收 WSS 更新。
- `admin-dashboard`：管理端，复用同一 API 契约并按管理员角色授权。

四端均已建立可运行实现，并通过 Go 单元测试、Qt 测试、Vue 测试/构建、Flutter 测试/Web 构建及真实 HTTP2/MQTTS/RabbitMQ 联调。Qt 原生 MQTTS 适配器仍受本机 QtMqtt/Protobuf 组件缺失限制，详见根 README。

## 多用户、多车辆

用户与车辆是多对多关系，绑定关系包含 `owner/driver/viewer` 访问角色；车辆另有唯一 `owner_user_id`。每次车辆、遥测、指令和告警访问都必须同时校验用户身份与绑定关系，不能只凭 VIN 授权。车辆设备身份独立于用户 JWT，由 mTLS 证书 CN 中的 VIN 标识。

## 数据边界

| 组件 | 负责 | 不负责 |
| --- | --- | --- |
| MySQL | 用户、车辆及绑定关系、历史遥测、指令状态、告警、OTA 元数据的事务性持久化 | 高频在线状态、一次性 ticket |
| Redis | 最新遥测及可重建的短期缓存 | 权威历史数据、可靠业务事件 |
| RabbitMQ | 指令状态、告警、OTA 等可靠业务事件的异步分发 | 实时遥测、历史遥测批量补传、文件分发 |

MySQL 是业务事实源；Redis 数据必须可失效或重建。RabbitMQ 消费者按事件 ID 幂等。遥测从 MQTT 或 HTTPS 批量接口直接进入后端写入路径，再更新 MySQL 历史与 Redis 最新快照，不经过 RabbitMQ。

## 链路

1. 用户端经 HTTPS 登录，使用短期 JWT 调用 API；WSS 使用单独的一次性 ticket。
2. 后端将控车指令持久化后通过 MQTTS 下发，ACK 驱动指令状态机并推送 WSS 更新。
3. 车端实时遥测走 MQTTS Protobuf；离线补传走 HTTPS Protobuf batch，按 `batch_id` 去重。
4. 后端间可靠业务副作用才发布到 AMQPS 的 `car` vhost。

## 本地基础设施

Compose 只启动 MySQL 8.0、Redis 7.0 与 RabbitMQ 3，全部端口仅绑定 `127.0.0.1`，数据使用新的 `car_refactor_*` 命名卷。MySQL/Redis/AMQP/管理界面均启用 TLS，车辆 MQTTS 由 Go 后端在 8883 提供；Compose 不包含四个应用端。证书仅用于本地开发，生产环境需独立 CA、密钥管理、证书轮换及最小权限 ACL。
