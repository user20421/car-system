<template>
  <div class="detail">
    <div class="head">
      <button class="back" @click="$router.push('/dashboard')">← 返回车队总览</button>
      <h2>车辆详情：<span class="vin">{{ vin }}</span></h2>
      <span :class="wsOnline ? 'green' : 'red'">{{ wsOnline ? '● 实时推送中' : '● 连接断开' }}</span>
    </div>

    <transition name="slide">
      <div v-if="fault" class="fault-banner">⚠ 检测到车辆故障码：{{ fault }}</div>
    </transition>

    <div class="cards">
      <div class="card"><span class="label">车速</span><span class="value">{{ latest.speed.toFixed(1) }} <small>km/h</small></span></div>
      <div class="card"><span class="label">转速</span><span class="value">{{ latest.rpm }} <small>rpm</small></span></div>
      <div class="card"><span class="label">电量</span><span class="value" :class="batteryClass">{{ latest.battery.toFixed(1) }} <small>%</small></span></div>
      <div class="card"><span class="label">位置</span><span class="value small">{{ latest.lat.toFixed(4) }}, {{ latest.lng.toFixed(4) }}</span></div>
    </div>

    <div class="charts">
      <div ref="speedChartEl" class="chart"></div>
      <div ref="rpmChartEl" class="chart"></div>
    </div>
    <div class="charts single">
      <div ref="batteryChartEl" class="chart"></div>
    </div>

    <!-- 远程控车 -->
    <div class="control-panel">
      <h3>🎮 远程控制</h3>
      <div class="cmd-buttons">
        <button :disabled="sending" @click="sendCommand('lock')">🔒 锁车</button>
        <button :disabled="sending" @click="sendCommand('unlock')">🔓 解锁</button>
        <button :disabled="sending" @click="sendCommand('ac_on')">❄️ 开空调</button>
        <button :disabled="sending" @click="sendCommand('ac_off')">💨 关空调</button>
        <button :disabled="sending" @click="sendCommand('honk')">📢 鸣笛</button>
      </div>
      <transition name="slide">
        <div v-if="cmdFeedback" class="cmd-feedback">{{ cmdFeedback }}</div>
      </transition>

      <h4>指令历史</h4>
      <table class="cmd-table">
        <thead>
          <tr><th>ID</th><th>指令</th><th>状态</th><th>下发时间</th><th>执行时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="c in commands" :key="c.id">
            <td>{{ c.id }}</td>
            <td>{{ cmdLabel(c.command) }}</td>
            <td><span class="cmd-status" :class="c.status">{{ statusLabel(c.status) }}</span></td>
            <td>{{ formatTime(c.created_at) }}</td>
            <td>{{ c.executed_at ? formatTime(c.executed_at) : '-' }}</td>
          </tr>
        </tbody>
      </table>
      <p v-if="!commands.length" class="dim-text">暂无指令记录</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import * as echarts from 'echarts'
import api from '../api'

const route = useRoute()
const vin = route.params.vin

const sending = ref(false)
const cmdFeedback = ref('')
const commands = ref([])
let feedbackTimer = null

function cmdLabel(c) {
  return { lock: '🔒 锁车', unlock: '🔓 解锁', ac_on: '❄️ 开空调', ac_off: '💨 关空调', honk: '📢 鸣笛' }[c] || c
}
function statusLabel(s) {
  return { pending: '待下发', sent: '已下发', executed: '已执行', failed: '失败' }[s] || s
}
function formatTime(t) { return new Date(t).toLocaleTimeString('zh-CN', { hour12: false }) }

async function sendCommand(cmd) {
  sending.value = true
  try {
    await api.post(`/api/v1/vehicle/${vin}/command`, { command: cmd })
    cmdFeedback.value = `指令「${cmdLabel(cmd)}」已下发，等待车机执行...`
    await loadCommands()
  } catch (e) {
    cmdFeedback.value = `下发失败：${e.msg || '网络错误'}`
  } finally {
    sending.value = false
    clearTimeout(feedbackTimer)
    feedbackTimer = setTimeout(() => (cmdFeedback.value = ''), 4000)
  }
}

async function loadCommands() {
  const res = await api.get(`/api/v1/vehicle/${vin}/commands?limit=10`)
  commands.value = res.data || []
}

const speedChartEl = ref(null)
const rpmChartEl = ref(null)
const batteryChartEl = ref(null)

const wsOnline = ref(false)
const fault = ref('')
const latest = reactive({ speed: 0, rpm: 0, battery: 0, lat: 0, lng: 0 })

const batteryClass = computed(() =>
  latest.battery > 50 ? 'green' : latest.battery > 20 ? 'yellow' : 'red',
)

const WINDOW = 60
const times = []
const speeds = []
const rpms = []
const batteries = []

let speedChart, rpmChart, batteryChart
let ws = null
let reconnectTimer = null
let faultTimer = null

function makeLineOption(title, color, unit) {
  return {
    title: { text: title, textStyle: { color: '#8b949e', fontSize: 14 } },
    grid: { left: 50, right: 20, top: 40, bottom: 30 },
    xAxis: {
      type: 'category', data: times,
      axisLine: { lineStyle: { color: '#30363d' } },
      axisLabel: { color: '#8b949e', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#8b949e' },
      splitLine: { lineStyle: { color: '#21262d' } },
      name: unit,
      nameTextStyle: { color: '#8b949e' },
    },
    series: [{
      type: 'line', data: [], showSymbol: false, smooth: true,
      lineStyle: { color, width: 2 }, areaStyle: { color: color + '22' },
    }],
    animation: false,
  }
}

function pushData(t) {
  times.push(new Date().toLocaleTimeString('zh-CN', { hour12: false }))
  speeds.push(t.speed); rpms.push(t.rpm); batteries.push(t.battery)
  if (times.length > WINDOW) { times.shift(); speeds.shift(); rpms.shift(); batteries.shift() }
  speedChart.setOption({ series: [{ data: speeds }] })
  rpmChart.setOption({ series: [{ data: rpms }] })
  batteryChart.setOption({ series: [{ data: batteries }] })
  if (t.fault_code) {
    fault.value = t.fault_code
    clearTimeout(faultTimer)
    faultTimer = setTimeout(() => (fault.value = ''), 5000)
  }
}

function connect() {
  ws = new WebSocket('ws://localhost:8080/ws')
  ws.onopen = () => { wsOnline.value = true }
  ws.onmessage = (e) => {
    try {
      const t = JSON.parse(e.data)
      if (t.vin !== vin) return // 只关注本车
      if (t.type === 'ack') {
        // 车机执行回执 → 刷新指令历史并提示
        cmdFeedback.value = `车机已执行「${cmdLabel(t.command)}」✅`
        clearTimeout(feedbackTimer)
        feedbackTimer = setTimeout(() => (cmdFeedback.value = ''), 4000)
        loadCommands()
        return
      }
      if (t.type === 'command') return
      Object.assign(latest, { speed: t.speed, rpm: t.rpm, battery: t.battery, lat: t.lat, lng: t.lng })
      pushData(t)
    } catch { /* 忽略 */ }
  }
  ws.onclose = () => {
    wsOnline.value = false
    reconnectTimer = setTimeout(connect, 3000)
  }
  ws.onerror = () => ws.close()
}

onMounted(() => {
  speedChart = echarts.init(speedChartEl.value)
  rpmChart = echarts.init(rpmChartEl.value)
  batteryChart = echarts.init(batteryChartEl.value)
  speedChart.setOption(makeLineOption('车速 (km/h)', '#4fc3f7', 'km/h'))
  rpmChart.setOption(makeLineOption('转速 (rpm)', '#d2a8ff', 'rpm'))
  batteryChart.setOption(makeLineOption('电量 (%)', '#3fb950', '%'))
  connect()
  loadCommands()
  window.addEventListener('resize', resizeAll)
})

function resizeAll() { speedChart?.resize(); rpmChart?.resize(); batteryChart?.resize() }

onUnmounted(() => {
  clearTimeout(reconnectTimer); clearTimeout(faultTimer)
  ws?.close()
  window.removeEventListener('resize', resizeAll)
  speedChart?.dispose(); rpmChart?.dispose(); batteryChart?.dispose()
})
</script>

<style scoped>
.detail { display: flex; flex-direction: column; gap: 16px; }
.head { display: flex; align-items: center; gap: 16px; }
.head h2 { font-size: 20px; }
.vin { font-family: Consolas, monospace; color: var(--blue); }
.back {
  background: none; border: 1px solid var(--border); color: var(--text-dim);
  border-radius: 8px; padding: 6px 14px; cursor: pointer; font-size: 13px;
}
.back:hover { color: var(--text); border-color: var(--blue); }
.green { color: var(--green); } .yellow { color: var(--yellow); } .red { color: var(--red); }

.fault-banner {
  background: #5c1a1a; border: 1px solid var(--red); color: #ff8a80;
  border-radius: 8px; padding: 12px 16px; font-weight: bold;
}
.slide-enter-active, .slide-leave-active { transition: all 0.3s; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-10px); }

.cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.card {
  background: var(--panel); border: 1px solid var(--border);
  border-radius: 10px; padding: 16px;
  display: flex; flex-direction: column; gap: 8px;
}
.label { color: var(--text-dim); font-size: 13px; }
.value { font-size: 24px; font-weight: bold; }
.value small { font-size: 13px; color: var(--text-dim); font-weight: normal; }
.value.small { font-size: 15px; }

.charts { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.charts.single { grid-template-columns: 1fr; }
.chart {
  height: 260px; background: var(--panel);
  border: 1px solid var(--border); border-radius: 10px; padding: 8px;
}

.control-panel {
  background: var(--panel); border: 1px solid var(--border);
  border-radius: 10px; padding: 20px;
}
.control-panel h3 { font-size: 16px; margin-bottom: 14px; }
.control-panel h4 { font-size: 14px; color: var(--text-dim); margin: 18px 0 10px; }
.cmd-buttons { display: flex; gap: 10px; flex-wrap: wrap; }
.cmd-buttons button {
  padding: 10px 20px; border: 1px solid var(--border); border-radius: 8px;
  background: #1c2128; color: var(--text); font-size: 14px; cursor: pointer;
  transition: border-color 0.2s;
}
.cmd-buttons button:hover:not(:disabled) { border-color: var(--blue); }
.cmd-buttons button:disabled { opacity: 0.4; cursor: not-allowed; }
.cmd-feedback {
  margin-top: 12px; padding: 10px 14px; border-radius: 8px;
  background: #1f6feb22; color: var(--blue); font-size: 14px;
}
.cmd-table { width: 100%; border-collapse: collapse; margin-top: 4px; }
.cmd-table th, .cmd-table td { padding: 8px 12px; text-align: left; font-size: 13px; }
.cmd-table th { color: var(--text-dim); font-weight: 600; }
.cmd-table tr + tr td { border-top: 1px solid var(--border); }
.cmd-status { padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.cmd-status.pending { background: #8b949e22; color: var(--text-dim); }
.cmd-status.sent { background: #4fc3f722; color: var(--blue); }
.cmd-status.executed { background: #3fb95022; color: var(--green); }
.cmd-status.failed { background: #ff525222; color: var(--red); }
.dim-text { color: var(--text-dim); font-size: 13px; margin-top: 10px; }
</style>
