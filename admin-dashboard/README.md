# AURORA 车联网运营控制台

基于 Vue 3、TypeScript、Vite、Pinia、Vue Router、ECharts 和 Lucide 的运营后台。默认通过真实 HTTPS API / WSS 工作；本地 Mock 仅在显式配置后启用。

## 环境与启动

建议使用 Node.js 20+ 和 npm。

```powershell
Set-Location admin-dashboard
npm ci
Copy-Item .env.example .env.local
npm run dev
```

开发地址默认为 `http://localhost:5173`。请先启动 `../backend-go` 及其依赖；默认后端为 `https://localhost:8443`。本地证书不受信任时，需要按后端文档安装开发 CA，而不是在生产环境关闭证书校验。

Vite 将 `/api` 的 HTTP 与 WebSocket Upgrade 请求代理到 `VITE_PROXY_TARGET`，并将 `/health` 代理到同一地址。这样既避免开发环境跨域，也满足后端 WSS Origin 校验。

### 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `/api/v1` | 真实 API 基础路径，生产部署可改为绝对 URL |
| `VITE_PROXY_TARGET` | `https://localhost:8443` | 仅开发代理使用 |
| `VITE_PROXY_SECURE` | `true` | 是否校验后端证书；本地通过 NODE_EXTRA_CA_CERTS 指定开发 CA |
| `VITE_USE_MOCK` | `false` | 只有字符串 `true` 才启用演示适配器 |
| `VITE_APP_NAME` | `AURORA` | 预留的应用名称配置 |

`.env.development` 当前也显式设置 `VITE_USE_MOCK=false`。未配置变量、配置为其他值或生产构建均不会自动使用 Mock。

## 演示模式

演示模式用于离线查看界面，不代表连接了真实车辆：

```powershell
$env:VITE_USE_MOCK = 'true'
npm run dev
```

- 登录页会显示 `DEMO`，主界面顶部会显示演示横幅，连接状态显示“演示数据流”。
- 预填账号 `aurora.admin`、密码 `demo12345`；任意非空账号和至少 8 位密码均可登录。账号包含 `viewer` 时使用普通成员角色，可验证 403。
- 车辆、告警、指令、用户、设备和审计数据均在浏览器内生成或修改，刷新页面后重置。
- 演示实时流使用本地定时器，不申请 WSS ticket，也不连接真实车辆；指令终态仅用于界面验证。

## 页面

| 路由 | 功能 |
| --- | --- |
| `/login` | 登录、真实错误与演示状态提示 |
| `/overview` | 车队指标、车辆状态、分布和地图 |
| `/fleet` | 地图车辆、搜索筛选、在线状态与实时快照 |
| `/vehicles/:vin` | 实时仪表、历史遥测、轨迹、远程指令与时间线 |
| `/alerts` | 告警筛选、确认和 CSV 导出 |
| `/access` | 用户、车辆、设备目录与管理界面 |
| `/audit` | 审计事件筛选、详情和 CSV 导出 |
| `/settings` | 本地显示偏好、会话与连接信息 |
| `/403`、未知路由 | 无权限与页面不存在状态 |

前端权限只控制可见性，服务端始终为最终权限边界。普通 `owner` / `driver` 可执行车辆控制与告警确认，`viewer` 只读；管理员不受单车辆角色限制。

## 真实 API / WSS 契约

实现依据 `../contracts/openapi/openapi.yaml` 和 `../backend-go/internal/httpapi`，未自定义另一套字段或错误格式。

已接通的真实能力：

- `POST /auth/login`、`POST /auth/refresh`、`POST /auth/logout`、`GET /auth/me`
- 车辆列表与详情、最新遥测、历史遥测
- 指令创建与查询、告警列表与确认
- `POST /ws-ticket` 和返回地址对应的 `/ws`

HTTP 使用 Bearer access token、`X-Request-ID` 和 Problem Details；并发 401 共用一次刷新，成功后仅重试一次原请求。Access Token 仅保存在内存中；后端在登录/刷新时设置 HttpOnly、Secure、SameSite=Strict Cookie，浏览器整页刷新后以空 JSON body + `X-Request-ID` 恢复会话。原生 JSON refresh token 流程仍然兼容。

WSS ticket 短时且一次性。开发环境通过 Vite 同源代理连接，生产环境使用 ticket 返回的 `websocket_url`。服务端自动推送 ticket 用户获授权的 VIN，当前契约没有客户端 `subscribe` / `unsubscribe` 帧；前端只处理 `{ type, data }` 形式的 `telemetry`、`command`、`alert` 事件，断线后通过 HTTPS 快照恢复。

### 当前后端限制

当前 OpenAPI **没有** 用户目录/创建/状态管理、设备目录/预配置、车辆资料 PATCH、审计事件接口。为避免请求不存在的地址，真实适配器对这些操作返回明确的 `ENDPOINT_NOT_IN_CONTRACT`（501）错误，页面显示不可用状态；这些管理流程只能在显式演示模式中体验。

总览中的车队级历史趋势、健康分、今日里程等展示性聚合指标没有对应的后端聚合契约，不应视为真实运营统计。地图为本地矢量示意图，不依赖第三方地图服务。

## 验证与生产构建

```powershell
npm run typecheck   # 正确脚本名，不是 type-check
npm test
npm run build       # 先再次 typecheck，再生成 dist/
npm run preview     # 本地预览生产产物
```

测试覆盖认证会话、单航班刷新、Problem Details、权限、真实 REST 参数、非契约接口保护、WSS ticket/事件行为、指令和车队状态及部分公共组件。`scripts/capture-real.mjs` 另通过真实后端、WSS 遥测和远程控车终态完成浏览器联调。

生产部署需发布 `dist/`，为 Vue Router history 模式配置 SPA fallback，并确保 HTTPS/WSS、API 地址和受信任证书配置正确。`vite preview` 只用于本地验证，不是生产服务器。
