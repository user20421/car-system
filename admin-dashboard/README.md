# Car-System 管理后台

车联网平台的运营管理端（Vue 3 + Vite + Pinia + ECharts + Leaflet）。

## 页面

| 页面 | 路径 | 权限 |
|---|---|---|
| 监控大屏 | `/dashboard` | 登录 |
| 车辆详情（实时曲线+远程控车） | `/vehicle/:vin` | 登录（车主仅限名下车辆） |
| 轨迹回放 | `/vehicle/:vin/track` | 登录 |
| 车辆管理 | `/vehicles` | admin |
| 用户管理 | `/users` | admin |
| 告警中心（规则/围栏/记录） | `/alarms` | admin |
| 死信管理（人工重投） | `/dead-letters` | admin |
| 模拟器控制台 | `/simulator` | admin |

## 启动

```powershell
npm install
npm run dev   # http://localhost:5173
```

## 配置

后端地址通过环境变量注入，默认值即本机。需要覆盖时复制 `.env.example` 为 `.env.development`：

```
VITE_API_BASE=http://localhost:8080
VITE_WS_BASE=ws://localhost:8080
```
