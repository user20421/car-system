<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { Activity, Bell, CarFront, ChevronDown, CircleGauge, ClipboardList, CloudOff, Command, LogOut, Map, Menu, Moon, Search, Settings, ShieldCheck, Sun, UsersRound, X } from 'lucide-vue-next'
import AppLogo from '../common/AppLogo.vue'
import StatusBadge from '../common/StatusBadge.vue'
import { useAuthStore } from '../../stores/auth'
import { useFleetStore } from '../../stores/fleet'
import { useUiStore } from '../../stores/ui'
import { useOnline } from '../../composables/useOnline'
import { RealtimeClient } from '../../services/realtime'
import { useCommandStore } from '../../stores/commands'
import { useMockApi } from '../../api'

const auth = useAuthStore(), fleet = useFleetStore(), ui = useUiStore(), commands = useCommandStore()
const route = useRoute(), router = useRouter()
const { online } = useOnline()
const mobileOpen = ref(false), userOpen = ref(false), searchOpen = ref(false), search = ref('')
const realtime = new RealtimeClient()
const nav = computed(() => [
  { label: '车队总览', to: '/overview', icon: CircleGauge, show: auth.can('fleet:view') },
  { label: '实时车队', to: '/fleet', icon: Map, show: auth.can('fleet:view') },
  { label: '告警中心', to: '/alerts', icon: Bell, show: auth.can('fleet:view'), count: fleet.openAlerts.length },
  { label: '用户与资产', to: '/access', icon: UsersRound, show: auth.can('users:manage') },
  { label: '审计日志', to: '/audit', icon: ClipboardList, show: auth.can('audit:view') },
  { label: '平台设置', to: '/settings', icon: Settings, show: auth.can('settings:manage') },
])
const searchResults = computed(() => fleet.vehicles.filter(v => `${v.display_name}${v.vin}${v.model}`.toLowerCase().includes(search.value.toLowerCase())).slice(0, 5))
const realtimeLabel = computed(() => ({ live: useMockApi ? '演示数据流' : '实时连接', connecting: '正在连接', reconnecting: '重新连接', offline: '连接离线', closed: '已断开', idle: '待连接' }[fleet.realtimeState] || '实时连接'))

onMounted(async () => {
  realtime.on('state', state => { fleet.realtimeState = state })
  realtime.on('telemetry', fleet.updateTelemetry)
  realtime.on('alert', fleet.upsertAlert)
  realtime.on('command', update => {
    const existing = commands.commands[update.command_id]
    if (existing) commands.commands[update.command_id] = { ...existing, ...update }
  })
  realtime.on('snapshot', vins => { if (online.value) void fleet.refreshSnapshots(vins) })
  try {
    await fleet.load()
    realtime.subscribe(fleet.vehicles.map(v => v.vin))
    await realtime.start()
  } catch { /* Page-level state surfaces the API error. */ }
})
onBeforeUnmount(() => realtime.stop())
async function logout() { await auth.logout(); await router.push({ name: 'login' }) }
function goVehicle(vin: string) { searchOpen.value = false; search.value = ''; void router.push(`/vehicles/${vin}`) }
</script>

<template>
  <div class="shell">
    <div v-if="useMockApi" class="demo-banner">DEMO WORKSPACE · 当前为显式启用的演示数据，不连接真实车辆，指令状态仅用于界面验证</div>
    <div v-if="!online" class="offline-banner"><CloudOff :size="14" /> 网络连接已中断 · 页面保留最后已知数据，恢复后将自动获取 HTTPS 快照</div>
    <aside class="sidebar" :class="{ collapsed: ui.sidebarCollapsed, 'mobile-open': mobileOpen }">
      <div class="sidebar-head"><AppLogo :compact="ui.sidebarCollapsed" light /><button class="mobile-close" @click="mobileOpen=false"><X /></button></div>
      <div class="workspace" :class="{ collapsed: ui.sidebarCollapsed }"><span class="workspace-symbol">A</span><div v-if="!ui.sidebarCollapsed"><strong>Aurora 中国运营</strong><span>企业工作空间</span></div><ChevronDown v-if="!ui.sidebarCollapsed" /></div>
      <nav class="nav" aria-label="主导航">
        <span v-if="!ui.sidebarCollapsed" class="nav-label">OPERATIONS</span>
        <RouterLink v-for="item in nav.filter(n => n.show)" :key="item.to" :to="item.to" class="nav-item" active-class="active" :title="item.label" @click="mobileOpen=false"><component :is="item.icon" /><span v-if="!ui.sidebarCollapsed">{{ item.label }}</span><em v-if="item.count && !ui.sidebarCollapsed">{{ item.count }}</em></RouterLink>
      </nav>
      <div class="sidebar-bottom">
        <div v-if="!ui.sidebarCollapsed" class="system-card"><div class="system-orbit"><Activity /></div><strong>运营服务正常</strong><span>全部系统已安全连接</span><div class="system-bars"><i/><i/><i/><i/><i/></div></div>
        <button class="collapse-btn" @click="ui.sidebarCollapsed=!ui.sidebarCollapsed"><Menu /><span v-if="!ui.sidebarCollapsed">收起导航</span></button>
      </div>
    </aside>
    <div v-if="mobileOpen" class="sidebar-scrim" @click="mobileOpen=false" />
    <main class="main">
      <header class="topbar">
        <div class="topbar-left"><button class="btn btn-ghost btn-icon mobile-menu" @click="mobileOpen=true"><Menu /></button><div class="breadcrumb"><span>运营中心</span><b>/</b><strong>{{ route.meta.title }}</strong></div></div>
        <div class="topbar-actions">
          <button class="global-search" @click="searchOpen=true"><Search /><span>搜索车辆、VIN</span><kbd>⌘ K</kbd></button>
          <div class="connection"><span class="dot" :class="{ live: fleet.realtimeState==='live' }" />{{ realtimeLabel }}</div>
          <button class="btn btn-ghost btn-icon" :title="ui.theme==='light'?'切换深色':'切换浅色'" @click="ui.toggleTheme"><Moon v-if="ui.theme==='light'" /><Sun v-else /></button>
          <RouterLink to="/alerts" class="btn btn-ghost btn-icon notification" title="告警中心"><Bell /><i v-if="fleet.openAlerts.length">{{ fleet.openAlerts.length }}</i></RouterLink>
          <div class="user-wrap"><button class="user-button" @click="userOpen=!userOpen"><span class="avatar round">{{ auth.user?.display_name.slice(0,1) }}</span><span class="user-meta"><strong>{{ auth.user?.display_name }}</strong><small>{{ auth.isAdmin ? '运营管理员' : '车队成员' }}</small></span><ChevronDown /></button><Transition name="fade"><div v-if="userOpen" class="user-menu"><div class="user-menu-head"><strong>{{ auth.user?.display_name }}</strong><span>{{ auth.user?.email || auth.user?.username }}</span></div><RouterLink to="/settings" @click="userOpen=false"><Settings />账号与偏好</RouterLink><button @click="logout"><LogOut />退出登录</button></div></Transition></div>
        </div>
      </header>
      <RouterView v-slot="{ Component }"><Transition name="page" mode="out-in"><component :is="Component" /></Transition></RouterView>
      <footer class="app-footer"><span>AURORA Mobility Intelligence · v1.0</span><span><ShieldCheck /> TLS 安全连接 · 权限由服务端强制校验</span></footer>
    </main>
    <Teleport to="body"><Transition name="fade"><div v-if="searchOpen" class="search-overlay" @mousedown.self="searchOpen=false"><div class="command-palette"><div class="command-input"><Search /><input v-model="search" autofocus placeholder="输入车辆名称、型号或完整 VIN…" @keydown.esc="searchOpen=false"><kbd>ESC</kbd></div><div class="command-results"><span class="nav-label">车辆</span><button v-for="vehicle in searchResults" :key="vehicle.vin" @click="goVehicle(vehicle.vin)"><span class="result-icon"><CarFront /></span><span><strong>{{ vehicle.display_name }}</strong><small class="mono">{{ vehicle.vin }}</small></span><StatusBadge :status="fleet.isOnline(vehicle.vin)?'online':'offline'" /></button><div v-if="!searchResults.length" class="search-empty">未找到匹配车辆</div></div><div class="command-footer"><Command /> 选择车辆进入实时详情 <span>数据仅覆盖当前账号授权范围</span></div></div></div></Transition></Teleport>
  </div>
</template>

<style scoped>
.shell{min-height:100vh}.sidebar{position:fixed;left:0;top:0;bottom:0;width:232px;background:var(--sidebar);color:var(--sidebar-text);z-index:40;display:flex;flex-direction:column;padding:0 13px;transition:width .22s,transform .22s;border-right:1px solid rgba(255,255,255,.03)}.demo-banner~.sidebar{top:29px}.offline-banner~.sidebar{top:29px}.sidebar.collapsed{width:76px}.sidebar-head{height:75px;display:flex;align-items:center;padding:0 8px}.workspace{margin:2px 0 24px;padding:10px;display:flex;align-items:center;gap:9px;border:1px solid rgba(255,255,255,.07);background:rgba(255,255,255,.035);border-radius:11px}.workspace.collapsed{border:0;background:transparent;padding:8px}.workspace-symbol{display:grid;place-items:center;width:29px;height:29px;flex:none;border-radius:8px;background:linear-gradient(135deg,#6e61e9,#27b7b5);color:#fff;font-size:12px;font-weight:750}.workspace div{display:flex;flex-direction:column;min-width:0;flex:1}.workspace strong{font-size:10px;color:#e1e4ef;white-space:nowrap}.workspace span:not(.workspace-symbol){font-size:8px;color:#6f7690;margin-top:3px}.workspace>svg{width:12px}.nav{display:flex;flex-direction:column;gap:5px}.nav-label{font-size:8px;letter-spacing:.16em;color:#5d647d;font-weight:750;padding:0 12px;margin:0 0 8px}.nav-item{height:42px;display:flex;align-items:center;gap:12px;padding:0 12px;border-radius:9px;font-size:11px;font-weight:550;transition:all .18s;position:relative}.nav-item svg{width:17px;height:17px;stroke-width:1.7;flex:none}.nav-item:hover{background:rgba(255,255,255,.05);color:#fff}.nav-item.active{background:linear-gradient(90deg,rgba(116,100,239,.2),rgba(116,100,239,.08));color:#c8c0ff}.nav-item.active::before{content:'';position:absolute;left:-13px;top:10px;bottom:10px;width:3px;background:#8b7ffa;border-radius:0 4px 4px 0}.nav-item em{margin-left:auto;font-style:normal;background:#493e7e;color:#c8c0ff;padding:2px 6px;border-radius:6px;font-size:8px}.collapsed .nav-item{justify-content:center;padding:0}.sidebar-bottom{margin-top:auto;padding-bottom:15px}.system-card{margin:0 3px 14px;border:1px solid rgba(255,255,255,.07);background:radial-gradient(circle at 100% 0,rgba(70,211,195,.09),transparent 55%),rgba(255,255,255,.025);border-radius:12px;padding:14px;display:flex;flex-direction:column;position:relative;overflow:hidden}.system-orbit{position:absolute;right:12px;top:12px;color:#4abcb2}.system-orbit svg{width:16px}.system-card strong{font-size:10px;color:#d5d9e7}.system-card>span{font-size:8px;color:#6d748d;margin-top:5px}.system-bars{display:flex;gap:3px;margin-top:13px}.system-bars i{height:3px;flex:1;border-radius:3px;background:#3eb2a5}.system-bars i:last-child{background:#2c4050}.collapse-btn{border:0;background:transparent;color:#697089;width:100%;height:36px;display:flex;align-items:center;gap:12px;padding:0 13px;font-size:10px;cursor:pointer}.collapse-btn:hover{color:#fff}.collapse-btn svg{width:16px}.main{margin-left:232px;min-height:100vh;display:flex;flex-direction:column;transition:margin .22s}.collapsed~.main{margin-left:76px}.topbar{height:69px;position:sticky;top:0;z-index:30;background:color-mix(in srgb,var(--bg) 88%,transparent);backdrop-filter:blur(16px);border-bottom:1px solid var(--line);display:flex;align-items:center;justify-content:space-between;padding:0 32px;gap:20px}.topbar-left,.topbar-actions{display:flex;align-items:center;gap:8px}.breadcrumb{display:flex;align-items:center;gap:9px;font-size:11px}.breadcrumb span{color:var(--muted)}.breadcrumb b{color:var(--line-strong);font-weight:400}.breadcrumb strong{font-weight:650}.global-search{height:34px;width:205px;border:1px solid var(--line);border-radius:9px;background:var(--surface);display:flex;align-items:center;gap:8px;padding:0 10px;color:var(--muted);font-size:10px;cursor:pointer}.global-search svg{width:14px}.global-search kbd{margin-left:auto;border:1px solid var(--line);border-radius:4px;padding:1px 4px;font-size:8px;font-family:inherit}.connection{display:flex;align-items:center;gap:6px;color:var(--muted);font-size:9px;margin:0 6px;white-space:nowrap}.connection .dot{color:var(--orange)}.connection .dot.live{color:var(--green);box-shadow:0 0 0 4px color-mix(in srgb,var(--green) 10%,transparent)}.notification{position:relative}.notification i{position:absolute;right:1px;top:0;background:var(--red);color:#fff;font-style:normal;font-size:7px;min-width:14px;height:14px;border-radius:8px;display:grid;place-items:center;border:2px solid var(--bg)}.user-wrap{position:relative;margin-left:4px}.user-button{border:0;background:transparent;display:flex;align-items:center;gap:9px;padding:3px;cursor:pointer;text-align:left}.user-meta{display:flex;flex-direction:column;min-width:70px}.user-meta strong{font-size:10px}.user-meta small{font-size:8px;color:var(--muted);margin-top:2px}.user-button>svg{width:12px;color:var(--muted)}.user-menu{position:absolute;right:0;top:48px;width:210px;background:var(--surface-solid);border:1px solid var(--line);border-radius:12px;box-shadow:var(--shadow-md);padding:6px}.user-menu-head{padding:10px 11px 12px;border-bottom:1px solid var(--line);margin-bottom:5px;display:flex;flex-direction:column;gap:4px}.user-menu-head strong{font-size:12px}.user-menu-head span{font-size:9px;color:var(--muted)}.user-menu a,.user-menu button{display:flex;align-items:center;gap:9px;width:100%;padding:9px 10px;border:0;background:transparent;color:var(--text-2);font-size:11px;cursor:pointer;border-radius:7px}.user-menu a:hover,.user-menu button:hover{background:var(--surface-hover)}.user-menu svg{width:14px}.app-footer{margin-top:auto;padding:18px 34px;display:flex;justify-content:space-between;border-top:1px solid var(--line);font-size:9px;color:var(--muted)}.app-footer span{display:flex;align-items:center;gap:6px}.app-footer svg{width:12px}.mobile-menu,.mobile-close{display:none}.search-overlay{position:fixed;inset:0;background:rgba(8,10,20,.5);backdrop-filter:blur(5px);z-index:90;display:flex;justify-content:center;align-items:flex-start;padding-top:13vh}.command-palette{width:min(580px,calc(100vw - 30px));background:var(--surface-solid);border:1px solid var(--line);border-radius:17px;box-shadow:0 30px 90px rgba(0,0,0,.25);overflow:hidden}.command-input{height:60px;display:flex;align-items:center;padding:0 18px;gap:12px;border-bottom:1px solid var(--line)}.command-input>svg{width:19px;color:var(--muted)}.command-input input{flex:1;border:0;background:transparent;color:var(--text);outline:0;font-size:13px}.command-input kbd{font-size:9px;color:var(--muted);border:1px solid var(--line);padding:3px 6px;border-radius:5px}.command-results{padding:14px 10px;max-height:350px;overflow:auto}.command-results .nav-label{display:block;color:var(--muted);margin:4px 8px 8px}.command-results button{width:100%;border:0;background:transparent;display:flex;align-items:center;gap:12px;text-align:left;padding:10px;border-radius:9px;cursor:pointer}.command-results button:hover{background:var(--surface-hover)}.result-icon{width:36px;height:36px;border-radius:9px;background:var(--primary-soft);color:var(--primary);display:grid;place-items:center}.result-icon svg{width:17px}.command-results button>span:nth-child(2){display:flex;flex-direction:column;gap:4px;flex:1}.command-results strong{font-size:12px}.command-results small{font-size:9px;color:var(--muted)}.command-footer{display:flex;align-items:center;gap:7px;background:var(--bg-soft);border-top:1px solid var(--line);padding:11px 18px;font-size:9px;color:var(--muted)}.command-footer svg{width:12px}.command-footer span{margin-left:auto}.search-empty{text-align:center;color:var(--muted);padding:30px;font-size:12px}.sidebar-scrim{display:none}
@media(max-width:1150px){.connection{display:none}.global-search{width:160px}.global-search kbd{display:none}.topbar{padding-inline:22px}.user-meta{display:none}}
@media(max-width:800px){.sidebar{transform:translateX(-100%);width:232px!important;top:0!important}.sidebar.mobile-open{transform:translateX(0)}.main,.collapsed~.main{margin-left:0}.mobile-menu{display:inline-flex}.mobile-close{display:block;margin-left:auto;background:transparent;border:0;color:#fff}.mobile-close svg{width:18px}.sidebar-scrim{display:block;position:fixed;inset:0;background:rgba(0,0,0,.45);z-index:39}.global-search{width:38px;padding:0;justify-content:center}.global-search span{display:none}.topbar{padding-inline:14px}.breadcrumb span,.breadcrumb b{display:none}.app-footer{padding-inline:20px}.app-footer span:last-child{display:none}}
</style>
