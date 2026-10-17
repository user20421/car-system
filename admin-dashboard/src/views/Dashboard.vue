<template>
  <div class="fleet">
    <h2>车队监控总览</h2>

    <!-- 实时告警横幅 -->
    <transition name="slide">
      <div v-if="alarmBanner" class="alarm-banner" :class="alarmBanner.severity">
        🚨 <b>{{ alarmBanner.title }}</b>：{{ alarmBanner.detail }}
        <button class="close" @click="alarmBanner = null">✕</button>
      </div>
    </transition>

    <!-- 统计卡片 -->
    <div class="stats">
      <div class="stat">
        <span class="label">车辆总数</span>
        <b>{{ vehicles.length }}</b>
      </div>
      <div class="stat">
        <span class="label">在线</span>
        <b class="green">{{ onlineCount }}</b>
      </div>
      <div class="stat">
        <span class="label">离线</span>
        <b class="dim">{{ vehicles.length - onlineCount }}</b>
      </div>
      <div class="stat">
        <span class="label">故障告警</span>
        <b :class="faultCount > 0 ? 'red' : 'dim'">{{ faultCount }}</b>
      </div>
      <div class="stat">
        <span class="label">实时通道</span>
        <b :class="wsOnline ? 'green' : 'red'">{{ wsOnline ? '● 已连接' : '● 断开' }}</b>
      </div>
    </div>

    <!-- 车辆网格 -->
    <div class="grid">
      <div
        v-for="v in vehicles"
        :key="v.vin"
        class="vehicle-card"
        :class="{ fault: v.fault_code }"
        @click="goDetail(v.vin)"
      >
        <div class="vc-head">
          <span class="vin">{{ v.vin }}</span>
          <span class="dot" :class="isOnline(v.vin) ? 'on' : 'off'"></span>
        </div>
        <div class="vc-body">
          <div class="vc-item"><span>车速</span><b>{{ fmt(v.speed, 1) }} km/h</b></div>
          <div class="vc-item"><span>电量</span><b :class="batteryClass(v.battery)">{{ fmt(v.battery, 1) }}%</b></div>
        </div>
        <div v-if="v.fault_code" class="vc-fault">⚠ {{ v.fault_code }}</div>
      </div>
      <div v-if="!vehicles.length" class="empty">
        暂无在线车辆数据——启动 Qt 车机或到「模拟器控制台」启动虚拟车队
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const wsOnline = ref(false)
const vehiclesMap = reactive({}) // vin -> 最新数据 + lastSeen
const alarmBanner = ref(null)    // 最新一条告警横幅
let alarmTimer = null

let ws = null
let reconnectTimer = null

// 最近 5 秒内有数据 = 在线
const now = ref(Date.now())
let clockTimer = null

const vehicles = computed(() =>
  Object.entries(vehiclesMap)
    .map(([vin, d]) => ({ vin, ...d }))
    .sort((a, b) => a.vin.localeCompare(b.vin)),
)
const onlineCount = computed(() =>
  Object.values(vehiclesMap).filter((d) => now.value - d.lastSeen < 5000).length,
)
const faultCount = computed(() =>
  Object.values(vehiclesMap).filter((d) => d.fault_code && now.value - d.lastSeen < 10000).length,
)

function isOnline(vin) {
  const d = vehiclesMap[vin]
  return d && now.value - d.lastSeen < 5000
}
function fmt(v, digits) { return typeof v === 'number' ? v.toFixed(digits) : '-' }
function batteryClass(b) { return b > 50 ? 'green' : b > 20 ? 'yellow' : 'red' }
function goDetail(vin) { router.push(`/vehicle/${vin}`) }

function connect() {
  const wsBase = import.meta.env.VITE_WS_BASE || 'ws://localhost:8080'
  ws = new WebSocket(`${wsBase}/ws?token=${encodeURIComponent(localStorage.getItem('token') || '')}`)
  ws.onopen = () => { wsOnline.value = true }
  ws.onmessage = (e) => {
    try {
      const t = JSON.parse(e.data)
      if (t.vin) vehiclesMap[t.vin] = { ...t, lastSeen: Date.now() }
      if (t.type === 'alarm' && t.data) {
        alarmBanner.value = t.data
        clearTimeout(alarmTimer)
        alarmTimer = setTimeout(() => { alarmBanner.value = null }, 10000)
      }
    } catch { /* 忽略 */ }
  }
  ws.onclose = () => {
    wsOnline.value = false
    reconnectTimer = setTimeout(connect, 3000)
  }
  ws.onerror = () => ws.close()
}

onMounted(() => {
  connect()
  clockTimer = setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  clearTimeout(reconnectTimer)
  clearTimeout(alarmTimer)
  clearInterval(clockTimer)
  if (ws) { ws.onclose = null; ws.close() } // 防止卸载后 onclose 再触发幽灵重连
})
</script>

<style scoped>
h2 { margin-bottom: 16px; font-size: 20px; }
.alarm-banner {
  display: flex; align-items: center; gap: 8px;
  border-radius: 10px; padding: 12px 16px; margin-bottom: 16px;
  font-size: 14px; border: 1px solid;
}
.alarm-banner.warning { background: #d2992222; border-color: var(--yellow); color: var(--yellow); }
.alarm-banner.critical { background: #ff525222; border-color: var(--red); color: var(--red); }
.alarm-banner.info { background: #1f6feb22; border-color: var(--blue); color: var(--blue); }
.alarm-banner .close {
  margin-left: auto; background: none; border: none; color: inherit;
  cursor: pointer; font-size: 14px; opacity: 0.7;
}
.slide-enter-active, .slide-leave-active { transition: all 0.3s; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-10px); }
.stats { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; margin-bottom: 16px; }
.stat {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
  display: flex; flex-direction: column; gap: 8px;
}
.label { color: var(--text-dim); font-size: 13px; }
.stat b { font-size: 26px; }
.green { color: var(--green); }
.yellow { color: var(--yellow); }
.red { color: var(--red); }
.dim { color: var(--text-dim); }

.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px; }
.vehicle-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 16px;
  cursor: pointer;
  transition: border-color 0.2s, transform 0.1s;
}
.vehicle-card:hover { border-color: var(--blue); transform: translateY(-2px); }
.vehicle-card.fault { border-color: var(--red); }
.vc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.vin { font-family: Consolas, monospace; font-size: 13px; color: var(--text); }
.dot { width: 10px; height: 10px; border-radius: 50%; }
.dot.on { background: var(--green); box-shadow: 0 0 6px var(--green); }
.dot.off { background: #555; }
.vc-body { display: flex; gap: 20px; }
.vc-item { display: flex; flex-direction: column; gap: 2px; }
.vc-item span { font-size: 12px; color: var(--text-dim); }
.vc-item b { font-size: 16px; }
.vc-fault {
  margin-top: 10px; padding: 4px 8px; border-radius: 6px;
  background: #5c1a1a; color: #ff8a80; font-size: 12px; font-weight: bold;
}
.empty { grid-column: 1 / -1; text-align: center; color: var(--text-dim); padding: 60px 0; }
</style>
