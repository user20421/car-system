# 🚗 Car-System 车联网分布式系统

一个架构完整的**企业级车联网平台**（本地开发与演示），覆盖四大终端 + 消息队列中间件，完整实现"车况采集 → 传输 → 存储 → 监控"全链路。

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
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                        ┌──────────┐   ┌───────────┐   ┌───────────┐
                        │  MySQL    │   │  RabbitMQ  │   │   Redis    │
                        │  持久化    │   │  异步削峰   │   │  实时缓存   │
                        └──────────┘   └───────────┘   └───────────┘
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
- ⚡ **Redis 实时缓存**：车辆最新状态 TTL 缓存，查询优先命中缓存、回源 MySQL
- 🔌 **WebSocket 广播**：车况数据实时推送到管理大屏
- 🔐 **JWT + RBAC**：bcrypt 密码哈希、角色权限中间件（admin/owner）
- 📊 **可观测性**：`/api/v1/system/metrics` 暴露 goroutines、堆内存、MySQL 连接池、MQ 吞吐、队列积压
- 📖 **Swagger 文档**：`/swagger/index.html`
- 🛡️ CORS、优雅启停、自动建库迁移（AutoMigrate）、连接池调优

### 车机端（Qt）
- 🧩 **HAL 硬件抽象层**：`IVehicleData` 接口，本地用 `VehicleSimulator` 模拟，换真车硬件只需替换实现类
- 📶 弱网韧性：断网离线缓存（上限 1000 条）+ 恢复后自动补传 + Token 过期自动重登
- 🎛️ QML 仪表盘：自绘圆形仪表（车速/转速）、电量条、GPS、故障告警横幅、在线呼吸灯
- 🎮 远程控车执行端：WebSocket 接收指令 → 模拟执行（锁车/空调/鸣笛）→ HTTP 回执

### 远程控车链路（指令闭环）
```
App/网页下发指令 → RabbitMQ(vehicle.control 交换机) → 分发消费者 → WebSocket 推送
→ 车机执行 → HTTP 回执 → 更新指令状态+车辆状态 → WebSocket 广播回执 → 三端实时同步
```
- 指令状态机：pending → sent → executed/failed，全程可追溯
- 幂等设计：重复回执不重复更新

### 管理后台（Vue）
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
- 🔐 Token 本地持久化（shared_preferences）、Dio 拦截器统一鉴权

## 快速开始

### 1. 启动中间件

```powershell
docker compose up -d
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

## 主要 API

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/health` | 健康检查 | - |
| POST | `/api/v1/auth/register` | 注册 | - |
| POST | `/api/v1/auth/login` | 登录 | - |
| POST | `/api/v1/vehicle/telemetry` | 车况上报 | JWT |
| GET | `/api/v1/vehicle/:vin/telemetry/latest` | 最新车况（Redis 优先） | JWT |
| POST | `/api/v1/vehicle/:vin/command` | 远程控车 | JWT |
| POST | `/api/v1/vehicle/:vin/command/:id/ack` | 车机执行回执 | JWT |
| GET | `/api/v1/vehicle/:vin/commands` | 指令历史 | JWT |
| GET | `/api/v1/my/vehicles` | 我的车辆（车主） | JWT |
| GET | `/api/v1/vehicle` | 车辆列表 | JWT + admin |
| POST | `/api/v1/simulator/start` | 启动虚拟车队 | JWT + admin |
| POST | `/api/v1/simulator/stop` | 停止虚拟车队 | JWT + admin |
| GET | `/api/v1/simulator/status` | 车队状态 | JWT + admin |
| GET | `/api/v1/system/metrics` | 系统运行时指标 | JWT + admin |
| GET | `/ws` | WebSocket 实时推送 | - |

## 开发笔记

- 所有中间件连接配置见 `backend-go/config/config.yaml`
- 后端单测：`go test ./internal/middleware/`
- 本项目为**本地开发/演示用途**，未包含部署运维相关内容
