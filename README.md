# 🚗 Car-System 车联网分布式系统

一个架构完整的**企业级车联网平台**（本地开发与演示），覆盖四大终端 + 消息队列中间件 + 可观测性全家桶，完整实现"车况采集 → 传输 → 存储 → 监控 → 告警"全链路。

**可靠性**：Outbox 发件箱 · Publisher Confirms 双轨发布 · 幂等落库 · DLX 死信 · 断线自动重连
**可观测性**：Prometheus 指标 · Grafana 预置大屏 · Jaeger 全链路追踪 · request-id 结构化日志
**业务深度**：告警规则引擎 · 电子围栏 · 轨迹回放 · 远程控车闭环 · 四端实时联动

## 系统架构

```
  ┌──────────────┐                    ┌──────────────────┐
  │  car-qt       │  HTTP+JWT 车况上报  │                  │
  │  Qt 车机端     │ ─────────────────► │                  │ ◄───── admin-dashboard
  │  (QML 仪表盘)  │                    │   backend-go      │  WebSocket   Vue 管理后台
  └──────┬───────┘  WebSocket 接收指令  │   Go 后端          │             (监控大屏/控制台)
         │ ◄────────────────────────── │   Gin+GORM+JWT     │
  ┌──────┴───────┐                    │                  │ ◄───── flutter_application
  │  虚拟车队引擎   │  内置数据管道        │                  │  REST+WS    Flutter 车主 App
  │  (后端内置)    │ ─────────────────► │                  │             (车况查看+远程控车)
  └──────────────┘                    └───────┬───────────┘
                                              │
                              ┌───────────────┼───────────────┬─────────────────────────┐
                              ▼               ▼               ▼                         ▼
                        ┌──────────┐   ┌───────────┐   ┌───────────┐        ┌──────────────────────┐
                        │  MySQL    │   │  RabbitMQ  │   │   Redis    │        │  可观测性（Docker）     │
                        │  持久化    │   │  异步削峰   │   │  实时缓存   │        │  Prometheus+Grafana   │
                        └──────────┘   └───────────┘   └───────────┘        │  +Jaeger              │
                                                                             └──────────────────────┘
```

## 四端说明

| 目录 | 角色 | 技术栈 |
|---|---|---|
| `backend-go/` | 后端中枢 | Go 1.27 + Gin + GORM + go-redis + amqp091 + zap + JWT + Swagger |
| `car-qt/` | 车机端（模拟硬件） | Qt 6.11 + QML 仪表盘 + C++ HAL 硬件抽象层 |
| `admin-dashboard/` | 运营管理后台 | Vue 3 + Vite + Pinia + Vue Router + ECharts + Axios |
| `flutter_application/` | 车主手机端 | Flutter 3.47 + Dio + WebSocket + shared_preferences |

## 核心特性（企业级看点）

### 后端
- 🏗️ 标准分层：`cmd / internal(handler/service/model/middleware) / pkg`
- 📨 **RabbitMQ 异步落库**：车况数据先进队列再持久化，削峰填谷，QoS 预取 + 手动 ACK
- 🔄 **MQ 断线自动重连**：连接级监控 + 指数退避重连，恢复后自动重建拓扑与消费者，Broker 重启全程无感
- ⚡ **发布 Channel 池化 + Publisher Confirms 双轨可靠性**：遥测走快路径不等回执（~60000 msg/s），控车指令走可靠路径等待 broker ack（~1900 msg/s），池化对比每消息建 channel 提升 2.4~75 倍（`go test ./internal/mq/ -bench .` 可复现）
- ☠️ **死信队列（DLX）**：消费失败应用层重试 3 次（指数退避），耗尽后转入 DLQ 并落 `dead_letters` 表，支持管理端查询与人工重投
- 📬 **Outbox 发件箱模式**：控车指令与待发消息同事务写入，relay 异步投递 + 失败自动补投，MQ 宕机指令零丢失
- 🧾 **遥测幂等落库**：消息级 message_id 唯一约束 + 冲突忽略，重复投递不产生重复数据
- ⚡ **Redis 实时缓存**：车辆最新状态 TTL 缓存，查询优先命中缓存、回源 MySQL
- 🔌 **WebSocket 广播**：车况数据实时推送到管理大屏；每客户端独立写协程 + 发送缓冲，慢客户端丢帧不拖累全局；广播非阻塞不反压上报链路
- 🔐 **JWT + RBAC**：bcrypt 密码哈希、角色权限中间件（admin/owner）
- 🚫 **车辆归属校验（防越权/IDOR）**：车主只能操作自己名下的车辆，admin 不受限
- 🔑 **WebSocket 握手鉴权**：`/ws?token=<jwt>`，未认证连接直接拒绝
- 📊 **可观测性**：`/api/v1/system/metrics` 暴露 goroutines、堆内存、MySQL 连接池、MQ 吞吐、队列积压
- 📈 **Prometheus + Grafana 监控大屏**：`/metrics` 指标端点（HTTP QPS/延迟分位、MQ 吞吐、WS 连接、队列积压、Outbox 积压、死信数），Grafana 预置大屏开箱即用
- 🔍 **Jaeger 全链路追踪**：OpenTelemetry 贯穿 HTTP → Outbox 投递 → MQ 消费，一条指令的完整旅程可视化
- 📇 **结构化访问日志**：zap 中间件记录每个请求，request-id 贯穿全链日志
- 🚨 **告警规则引擎**：遥测实时阈值判定（电量/超速/故障码），active → resolved 生命周期 + Redis 防抖，告警实时 WS 推送大屏
- 🗺️ **电子围栏**：圆形围栏进出判定（Haversine），驶入/驶出可配，触发即告警
- 🛣️ **轨迹回放**：历史 GPS 轨迹在 Leaflet 地图上动画回放，支持 1x/4x/16x 倍速与进度拖拽
- 📖 **Swagger 文档**：`/swagger/index.html`
- 🛡️ CORS、优雅启停（HTTP/MQ/WS Hub/模拟器/MySQL 统一有序退出）、自动建库迁移（AutoMigrate）、连接池调优
- ✅ **质量保障**：7 个集成测试覆盖核心闭环（登录/遥测链路/幂等/指令流转/越权/WS 鉴权/告警生命周期）+ `go test -race` 并发检测 + MQ 发布性能 benchmark

### 车机端（Qt）
- 🧩 **HAL 硬件抽象层**：`IVehicleData` 接口，本地用 `VehicleSimulator` 模拟，换真车硬件只需替换实现类
- 📶 弱网韧性：断网离线缓存（上限 1000 条）+ 恢复后自动补传 + Token 过期自动重登
- 🎛️ QML 仪表盘：自绘圆形仪表（车速/转速）、电量条、GPS、故障告警横幅、在线呼吸灯
- 🎮 远程控车执行端：WebSocket 接收指令 → 模拟执行（锁车/空调/鸣笛）→ HTTP 回执
- 🖥️ **中控屏本地控制**：驾驶员可在车机屏幕直接开关空调（锁车/鸣笛按真实权限逻辑仅限远程）
- 🛣️ **行程统计**：行驶时长、累计里程、平均速度实时累计
- ❄️ **空调能耗联动**：开启空调后电池掉电速度加快，模拟真实能耗
- 🚨 **告警联动**：服务端规则/围栏告警实时推送至车机屏幕（critical 红 / warning 黄 Toast）

### 远程控车链路（统一指令总线）
```
App/网页/车机中控屏下发指令 → RabbitMQ(vehicle.control 交换机) → 分发消费者 → WebSocket 推送
→ 车机执行 → HTTP 回执 → 更新指令状态+车辆状态 → WebSocket 广播回执 → 三端实时同步
```
- 指令状态机：pending → sent → executed/failed，全程可追溯
- 幂等设计：重复回执不重复更新
- **状态感知按钮**：已上锁则「锁车」置灰、空调已开则「开空调」置灰（App/Web/车机三端一致）
- 权限逻辑：锁车/鸣笛仅远程端可用；空调支持本地+远程双通道

### 管理后台（Vue）
管理端完整闭环：监控大屏 · 车辆详情 · 轨迹回放 · 车辆登记/编辑/删除 · 用户管理 · 告警中心 · 死信管理（人工重投） · 模拟器控制台
- 📊 **车队监控大屏**：多车总览（在线/离线/告警统计）+ 车辆卡片实时刷新 + 详情下钻
- 🎛️ **模拟器控制台**：可视化调节虚拟车队规模（1~50 辆）、上报频率、故障率，实时观察后端负载曲线
- 🛡️ Axios 拦截器自动注入 JWT、401 自动登出、路由守卫 RBAC
- 📈 ECharts 实时曲线（60 秒滑动窗口）+ WebSocket 断线自动重连

### 虚拟车队引擎（后端内置）
- 控制面/数据面分离：通过 API 热启动/停止/调参
- 每辆车独立 goroutine，模拟巡航车速、电量消耗/回充、GPS 漂移、随机故障码
- 虚拟车自动执行控车指令并回执（按 VIN 精确绑定队列）

### 车主 App（Flutter）
- 📱 车主视角：登录后仅见名下车辆（`/api/v1/my/vehicles`）
- ⚡ WebSocket 实时车况 + 远程控车 + 指令历史
- 📈 fl_chart 车况趋势曲线（车速/电量，历史数据 + 实时追加）
- ⚠️ 故障码实时告警横幅
- 🚨 **告警引擎联动**：规则/围栏告警实时推送（触发显示横幅，恢复自动消除）
- 🚘 **多车切换**：名下多辆车时顶部切换卡片，车况/曲线/指令/告警随车切换
- 🎮 状态感知控车按钮（与车辆当前状态相同的操作自动置灰）
- 🔐 Token 本地持久化（shared_preferences）、Dio 拦截器统一鉴权

## 快速开始

### 一键启动（推荐）

双击项目根目录的 **`启动系统.bat`**，自动按序拉起：Docker 中间件 → Go 后端 → Qt 车机 → Vue 后台 → Flutter App。

### 手动分步启动

### 1. 启动中间件

```powershell
docker compose up -d   # MySQL + Redis + RabbitMQ + Prometheus + Grafana + Jaeger
```

### 2. 启动后端

```powershell
cd backend-go
go run ./cmd/server        # 或 .\car-server.exe
```

- API 文档：http://localhost:8080/swagger/index.html
- 健康检查：http://localhost:8080/health

### 3. 启动管理后台

```powershell
cd admin-dashboard
npm install
npm run dev                # http://localhost:5173
```

### 4. 启动车机模拟器

```powershell
# 构建（首次）
cd car-qt
cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release `
  -DCMAKE_PREFIX_PATH="D:/software/Qt/6.11.2/llvm-mingw_64" `
  -DCMAKE_CXX_COMPILER="D:/software/Qt/Tools/llvm-mingw1706_64/bin/clang++.exe"
cmake --build build

# 运行（需 Qt 运行库在 PATH）
.\启动车机.ps1
```

### 5. 启动车主 App（Flutter）

```powershell
cd flutter_application
flutter pub get
flutter run -d chrome --web-port=5000   # http://localhost:5000
```

## 默认账号

| 账号 | 密码 | 角色 | 用途 |
|---|---|---|---|
| `admin` | `admin123456` | admin | 管理后台 / API |
| `driver1` | `driver123456` | owner | Qt 车机端 |
| `car-system` | `car123456` | - | RabbitMQ 管理后台 (localhost:15672) |
| `admin` | `admin` | - | Grafana 监控大屏 (localhost:3000) |

可观测性组件 UI：Prometheus `localhost:9090`，Jaeger `localhost:16686`

## 主要 API

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/health` | 健康检查 | - |
| POST | `/api/v1/auth/register` | 注册 | - |
| POST | `/api/v1/auth/login` | 登录 | - |
| POST | `/api/v1/vehicle/telemetry` | 车况上报 | JWT |
| GET | `/api/v1/vehicle/:vin/telemetry/latest` | 最新车况（Redis 优先） | JWT |
| GET | `/api/v1/vehicle/:vin` | 车辆详情（含锁/空调状态） | JWT |
| GET | `/api/v1/vehicle/:vin/telemetry/history` | 车况历史（曲线用） | JWT |
| POST | `/api/v1/vehicle/:vin/command` | 远程控车 | JWT |
| POST | `/api/v1/vehicle/:vin/command/:id/ack` | 车机执行回执 | JWT |
| GET | `/api/v1/vehicle/:vin/commands` | 指令历史 | JWT |
| GET | `/api/v1/my/vehicles` | 我的车辆（车主） | JWT |
| GET | `/api/v1/vehicle` | 车辆列表 | JWT + admin |
| POST | `/api/v1/simulator/start` | 启动虚拟车队 | JWT + admin |
| POST | `/api/v1/simulator/stop` | 停止虚拟车队 | JWT + admin |
| GET | `/api/v1/simulator/status` | 车队状态 | JWT + admin |
| GET | `/api/v1/system/metrics` | 系统运行时指标 | JWT + admin |
| GET | `/api/v1/system/dead-letters` | 死信消息列表 | JWT + admin |
| POST | `/api/v1/system/dead-letters/:id/requeue` | 死信人工重投 | JWT + admin |
| GET/POST | `/api/v1/users` | 用户列表 / 创建用户 | JWT + admin |
| POST/PUT/DELETE | `/api/v1/vehicle`、`/api/v1/vehicle/:vin` | 车辆登记 / 更新 / 删除 | JWT + admin |
| GET/POST/PUT/DELETE | `/api/v1/alarm/rules` | 告警规则管理 | JWT + admin |
| GET | `/api/v1/alarm/records` | 告警记录（可按状态过滤） | JWT + admin |
| POST | `/api/v1/alarm/records/:id/resolve` | 告警人工恢复 | JWT + admin |
| GET/POST/DELETE | `/api/v1/fences` | 电子围栏管理 | JWT + admin |
| GET | `/ws?token=<jwt>` | WebSocket 实时推送 | JWT（查询参数） |

## 开发笔记

- 所有中间件连接配置见 `backend-go/config/config.yaml`
- 地址与凭据均已配置化：
  - 管理后台：`admin-dashboard/.env.development`（`VITE_API_BASE` / `VITE_WS_BASE`）
  - Flutter：`flutter run --dart-define=API_BASE=http://10.0.2.2:8080 --dart-define=WS_URL=ws://10.0.2.2:8080/ws`（Android 模拟器用 `10.0.2.2` 映射宿主机）
  - Qt 车机：环境变量 `CAR_API_BASE` / `CAR_VIN` / `CAR_USERNAME` / `CAR_PASSWORD`
  - 一键脚本 Qt 路径可覆盖：`powershell -File start-all.ps1 -QtRoot "D:\Qt\6.x\llvm-mingw_64"`
- 后端单测：`go test ./internal/...`
- 后端集成测试（需中间件运行，不可用则自动跳过）：`cd backend-go && go test ./test/ -race -v`
- MQ 发布性能基准（需 RabbitMQ 运行）：`go test ./internal/mq/ -bench . -benchtime 2000x -run '^$'`
- 本项目为**本地开发/演示用途**，未包含部署运维相关内容
