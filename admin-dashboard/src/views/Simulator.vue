<template>
  <div class="simulator">
    <h2>虚拟车队控制台</h2>

    <div class="panels">
      <!-- 控制面板 -->
      <div class="panel">
        <h3>⚙️ 模拟参数</h3>
        <div class="field">
          <label>车辆数量：<b>{{ params.vehicle_count }}</b> 辆</label>
          <input type="range" v-model.number="params.vehicle_count" min="1" max="50" />
        </div>
        <div class="field">
          <label>上报间隔：<b>{{ params.interval_ms }}</b> ms</label>
          <input type="range" v-model.number="params.interval_ms" min="100" max="5000" step="100" />
        </div>
        <div class="field">
          <label>故障概率：<b>{{ (params.fault_rate * 100).toFixed(1) }}</b> % / 帧</label>
          <input type="range" v-model.number="params.fault_rate" min="0" max="0.1" step="0.001" />
        </div>
        <div class="actions">
          <button class="start" :disabled="running" @click="start">▶ 启动车队</button>
          <button class="stop" :disabled="!running" @click="stop">■ 停止</button>
        </div>
        <p class="status-line">
          状态：
          <span :class="running ? 'green' : 'dim'">
            {{ running ? `运行中（${params.vehicle_count} 辆车 × ${params.interval_ms}ms）` : '未运行' }}
          </span>
        </p>
      </div>

      <!-- 实时指标卡片 -->
      <div class="panel">
        <h3>📊 后端实时指标</h3>
        <div class="metric-grid">
          <div class="metric"><span>Goroutines</span><b>{{ metrics.goroutines ?? '-' }}</b></div>
          <div class="metric"><span>堆内存</span><b>{{ fmtMb(metrics.heap_alloc_mb) }}</b></div>
          <div class="metric"><span>GC 次数</span><b>{{ metrics.gc_count ?? '-' }}</b></div>
          <div class="metric"><span>MySQL 连接</span><b>{{ metrics.mysql ? `${metrics.mysql.in_use}/${metrics.mysql.open}` : '-' }}</b></div>
          <div class="metric"><span>MQ 已发布</span><b>{{ metrics.mq_published ?? '-' }}</b></div>
          <div class="metric"><span>MQ 已落库</span><b>{{ metrics.mq_persisted ?? '-' }}</b></div>
          <div class="metric"><span>队列积压</span><b :class="{ red: metrics.mq_queue_ready > 0 }">{{ metrics.mq_queue_ready ?? '-' }}</b></div>
          <div class="metric"><span>运行时长</span><b>{{ fmtUptime(metrics.uptime_seconds) }}</b></div>
        </div>
      </div>
    </div>

    <!-- 性能曲线 -->
    <div class="charts">
      <div ref="heapChartEl" class="chart"></div>
      <div ref="rateChartEl" class="chart"></div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import * as echarts from 'echarts'
import api from '../api'

const running = ref(false)
const params = ref({ vehicle_count: 10, interval_ms: 1000, fault_rate: 0.005 })
const metrics = ref({})

const heapChartEl = ref(null)
const rateChartEl = ref(null)
let heapChart, rateChart, pollTimer

const times = []
const heapData = []
const rateData = []
let lastPublished = null
let lastTime = null

function fmtMb(v) { return v == null ? '-' : v.toFixed(1) + ' MB' }
function fmtUptime(s) {
  if (s == null) return '-'
  const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60)
  return h > 0 ? `${h}h${m}m` : `${m}m${s % 60}s`
}

async function start() {
  await api.post('/api/v1/simulator/start', params.value)
  running.value = true
}
async function stop() {
  await api.post('/api/v1/simulator/stop')
  running.value = false
}
async function refreshStatus() {
  const res = await api.get('/api/v1/simulator/status')
  running.value = res.data.running
  if (res.data.running) params.value = res.data.params
}

function chartOption(title, color, unit) {
  return {
    title: { text: title, textStyle: { color: '#8b949e', fontSize: 14 } },
    grid: { left: 60, right: 20, top: 40, bottom: 30 },
    xAxis: {
      type: 'category', data: times,
      axisLine: { lineStyle: { color: '#30363d' } },
      axisLabel: { color: '#8b949e', fontSize: 10 },
    },
    yAxis: {
      type: 'value', name: unit,
      axisLabel: { color: '#8b949e' },
      splitLine: { lineStyle: { color: '#21262d' } },
      nameTextStyle: { color: '#8b949e' },
    },
    series: [{ type: 'line', data: [], showSymbol: false, smooth: true,
      lineStyle: { color, width: 2 }, areaStyle: { color: color + '22' } }],
    animation: false,
  }
}

async function pollMetrics() {
  try {
    const res = await api.get('/api/v1/system/metrics')
    metrics.value = res.data
    const now = Date.now()
    const label = new Date().toLocaleTimeString('zh-CN', { hour12: false })
    times.push(label)
    heapData.push(res.data.heap_alloc_mb)
    // 吞吐 = 两次轮询间的发布增量 / 间隔秒数
    if (lastPublished != null && res.data.mq_published != null) {
      rateData.push(((res.data.mq_published - lastPublished) / ((now - lastTime) / 1000)).toFixed(1))
    } else {
      rateData.push(0)
    }
    lastPublished = res.data.mq_published
    lastTime = now
    if (times.length > 60) { times.shift(); heapData.shift(); rateData.shift() }
    heapChart.setOption({ series: [{ data: heapData }] })
    rateChart.setOption({ series: [{ data: rateData }] })
  } catch { /* 后端不可用时静默 */ }
}

onMounted(async () => {
  heapChart = echarts.init(heapChartEl.value)
  rateChart = echarts.init(rateChartEl.value)
  heapChart.setOption(chartOption('堆内存 (MB)', '#d29922', 'MB'))
  rateChart.setOption(chartOption('消息吞吐 (条/秒)', '#4fc3f7', 'msg/s'))
  await refreshStatus()
  pollMetrics()
  pollTimer = setInterval(pollMetrics, 2000)
})

onUnmounted(() => {
  clearInterval(pollTimer)
  heapChart?.dispose()
  rateChart?.dispose()
})
</script>

<style scoped>
h2 { margin-bottom: 16px; font-size: 20px; }
h3 { font-size: 15px; color: var(--text-dim); margin-bottom: 16px; }
.panels { display: grid; grid-template-columns: 1fr 1.4fr; gap: 14px; }
.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 20px;
}
.field { margin-bottom: 18px; }
.field label { display: block; font-size: 13px; color: var(--text-dim); margin-bottom: 8px; }
.field b { color: var(--blue); }
input[type="range"] { width: 100%; accent-color: var(--blue); }
.actions { display: flex; gap: 10px; margin-top: 8px; }
button {
  padding: 10px 22px; border: none; border-radius: 8px;
  font-size: 14px; font-weight: bold; cursor: pointer;
}
button:disabled { opacity: 0.35; cursor: not-allowed; }
.start { background: var(--green); color: #0d1117; }
.stop { background: var(--red); color: #fff; }
.status-line { margin-top: 14px; font-size: 13px; color: var(--text-dim); }
.green { color: var(--green); }
.dim { color: var(--text-dim); }
.red { color: var(--red); }

.metric-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.metric {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 12px;
  display: flex; flex-direction: column; gap: 4px;
}
.metric span { font-size: 12px; color: var(--text-dim); }
.metric b { font-size: 18px; }

.charts { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 14px; }
.chart {
  height: 240px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 8px;
}
</style>
