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
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import * as echarts from 'echarts'

const route = useRoute()
const vin = route.params.vin

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
</style>
