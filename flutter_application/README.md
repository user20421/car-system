# AURORA 车主 App

原创的 AURORA 智能车主应用，覆盖多用户多车、实时车辆详情、安全控车、续航趋势、告警、车辆服务与个人设置。界面使用自绘 `CustomPainter` 车辆主视觉、蓝紫/青色品牌体系、响应式布局、浅色/深色主题与安全区适配；不是后台 CRUD 页面。

## 运行环境

- Flutter 3.47.5 / Dart 3.13.4
- 平台工程：Android、iOS、Web
- 默认使用真实 HTTPS JSON API，不会因为接口不可用而自动切换演示数据。

### 真实 API 模式

```powershell
flutter pub get
flutter run --dart-define=AURORA_API_BASE_URL=https://localhost:8443
```

Android 模拟器访问宿主机时通常使用：

```powershell
flutter run --dart-define=AURORA_API_BASE_URL=https://10.0.2.2:8443
```

本地开发 CA/证书必须由设备或浏览器信任；应用没有关闭 TLS 校验。生产环境应使用受信任证书和正确的 API 域名。

### 显式演示模式

```powershell
flutter run --dart-define=AURORA_DEMO=true -d chrome
```

- 账号：`demo`
- 密码：`aurora-demo`
- 演示数据只在本机生成，界面持续显示“演示模式”。
- 包含两辆可控制车辆、一辆仅查看共享车辆，以及告警和 24 小时/7 天趋势数据。
- 演示控车同样经过 `pending → sent → received → succeeded` 的异步回执过程，不会点击后立即显示成功。

### 构建与检查

```powershell
flutter analyze
flutter test
flutter build web --release
```

## API 与实时通道

实现依据根目录 `contracts/openapi/openapi.yaml` 和 `docs/protocol.md`：

- 登录、刷新、退出、当前用户、车辆列表、最新/历史遥测、控车、指令查询、告警确认和 WSS ticket。
- `ApiClient` 负责 Bearer 注入、单次并发的 refresh、401 重试、结构化 problem 错误映射。
- `RealtimeService` 每次连接/重连都重新获取短时一次性 ticket，通过 WSS 接收遥测与指令事件。
- WSS 只是提示通道；连接成功后会重新拉取 HTTPS 快照，断线采用指数退避重连。
- 页面仅使用服务端返回的可访问车辆；`viewer` 在 UI 禁用控车，服务端仍必须执行最终权限校验。

## 原生 HTTP/2 与安全会话

`native_dio_adapter` 通过条件导入只在原生端安装：

- Android：Cronet。
- iOS/macOS：`URLSession`。
- HTTPS 下由原生网络栈通过 ALPN 协商 HTTP/2。
- Web：使用 `BrowserHttpClientAdapter`，HTTP/2 或 HTTP/3 由浏览器和服务器协商，Dart 不能指定协议版本。

Android Cronet 依赖设备上可用的 Cronet provider。部分无 Google Play 服务的 AOSP 环境可能无法创建 Cronet Engine；本项目没有静默退回 Dart HTTP/1.1 适配器，以避免破坏原生协议保证。WebAssembly dry-run 会报告原生 `ffi` 依赖不兼容，因此当前 Web 发布目标是标准 JavaScript 构建，而不是 Wasm。

原生刷新令牌通过 `flutter_secure_storage` 写入 Android Keystore 加密存储 / iOS Keychain。Web 不向 LocalStorage 或 Secure Storage 写入令牌：

- 默认 bearer 响应仅保留在内存，并允许浏览器发送凭据。
- `SessionStore` 是可注入边界；Web 会话支持 HttpOnly、Secure、SameSite Cookie 部署，启动时通过 `/auth/me` 尝试恢复 Cookie 会话。
- 当前仓库服务端契约是 refresh-token JSON，并未声明 Cookie 登录端点。启用纯 Cookie 模式需要服务端增加对应策略，或由部署方替换 `SessionStore`/认证适配层；这不是客户端单方面可以完成的配置。

## 控车安全与状态

- 所有控车操作都有二次确认；解锁包含额外安全提示。
- 上一条指令未到达终态前阻止重复提交，每次新操作使用唯一幂等键。
- 状态严格展示 `pending / sent / received / succeeded / failed / expired`，终态不可逆。
- HTTPS 接受指令只代表创建/受理；只有车辆终态回执才显示成功或失败。
- 断网、无权限、提交中均禁用相应操作。离线退出会清除本地会话，但服务端 refresh token 撤销请求可能只能等下次联网完成。

## 数据边界

当前协议没有独立行程事件、充电枪/功率、充电控制或服务预约接口。因此：

- “行程与趋势”明确基于历史遥测，不虚构起终点、里程或行程事件。
- “续航与充电”不猜测充电状态，只展示电量、续航及历史趋势。
- 服务页提供官方服务入口提示，不伪造预约成功或 OTA 状态。

## 代码结构

```text
lib/
  models/       OpenAPI 模型与指令状态机
  services/     Dio API、真实/演示仓库、WSS、条件平台适配
  state/        Provider + ChangeNotifier，加载/离线/鉴权状态
  screens/      登录、车库、车辆、控车、趋势、告警、服务、我的
  widgets/      自绘车辆、电池环、趋势图、指令进度与通用状态
  theme/        AURORA 浅色/深色设计系统
test/           模型、权限、防重/回执和关键 Widget 测试
```
