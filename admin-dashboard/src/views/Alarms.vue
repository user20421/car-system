<template>
  <div class="alarms">
    <h2>告警中心</h2>

    <!-- 活动告警 -->
    <section class="panel">
      <h3>🔴 活动告警 <span class="count">{{ activeAlarms.length }}</span></h3>
      <table v-if="activeAlarms.length">
        <thead>
          <tr><th>时间</th><th>级别</th><th>车辆</th><th>标题</th><th>详情</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="a in activeAlarms" :key="a.id">
            <td class="dim">{{ fmtTime(a.triggered_at) }}</td>
            <td><span class="sev" :class="a.severity">{{ severityText(a.severity) }}</span></td>
            <td class="mono">{{ a.vin }}</td>
            <td>{{ a.title }}</td>
            <td class="dim">{{ a.detail }}</td>
            <td><button class="btn-sm" @click="resolve(a.id)">恢复</button></td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无活动告警 ✓</div>
    </section>

    <!-- 告警规则 -->
    <section class="panel">
      <h3>📏 告警规则</h3>
      <table v-if="rules.length">
        <thead>
          <tr><th>名称</th><th>条件</th><th>级别</th><th>状态</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="r in rules" :key="r.id">
            <td>{{ r.name }}</td>
            <td class="mono">{{ metricText(r.metric) }} {{ opText(r.operator) }} {{ r.metric === 'fault_code' ? '非空' : r.threshold }}</td>
            <td><span class="sev" :class="r.severity">{{ severityText(r.severity) }}</span></td>
            <td>
              <button class="toggle" :class="{ on: r.enabled }" @click="toggleRule(r)">
                {{ r.enabled ? '已启用' : '已停用' }}
              </button>
            </td>
            <td><button class="btn-sm danger" @click="removeRule(r.id)">删除</button></td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无规则，可在下方新建</div>

      <!-- 新建规则 -->
      <form class="create-form" @submit.prevent="createRule">
        <input v-model.trim="newRule.name" placeholder="规则名称" required />
        <select v-model="newRule.metric">
          <option value="battery">电量</option>
          <option value="speed">车速</option>
          <option value="rpm">转速</option>
          <option value="fault_code">故障码</option>
        </select>
        <select v-model="newRule.operator" :disabled="newRule.metric === 'fault_code'">
          <option value="lt">&lt;</option>
          <option value="lte">≤</option>
          <option value="gt">&gt;</option>
          <option value="gte">≥</option>
          <option value="ne">≠</option>
        </select>
        <input v-model.number="newRule.threshold" type="number" step="0.1" :disabled="newRule.metric === 'fault_code'" placeholder="阈值" />
        <select v-model="newRule.severity">
          <option value="info">提示</option>
          <option value="warning">警告</option>
          <option value="critical">严重</option>
        </select>
        <button type="submit" class="btn">+ 新建规则</button>
      </form>
    </section>

    <!-- 电子围栏 -->
    <section class="panel">
      <h3>🗺️ 电子围栏</h3>
      <table v-if="fences.length">
        <thead>
          <tr><th>名称</th><th>圆心</th><th>半径</th><th>告警时机</th><th>车辆</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="f in fences" :key="f.id">
            <td>{{ f.name }}</td>
            <td class="mono">{{ Number(f.lat).toFixed(4) }}, {{ Number(f.lng).toFixed(4) }}</td>
            <td>{{ f.radius_m }} m</td>
            <td>{{ fenceOnText(f.alarm_on) }}</td>
            <td class="mono">{{ f.vin || '全部车辆' }}</td>
            <td><button class="btn-sm danger" @click="removeFence(f.id)">删除</button></td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无围栏</div>

      <form class="create-form" @submit.prevent="createFence">
        <input v-model.trim="newFence.name" placeholder="围栏名称" required />
        <input v-model.number="newFence.lat" type="number" step="0.0001" placeholder="纬度" required />
        <input v-model.number="newFence.lng" type="number" step="0.0001" placeholder="经度" required />
        <input v-model.number="newFence.radius_m" type="number" placeholder="半径(米)" required />
        <select v-model="newFence.alarm_on">
          <option value="enter">驶入告警</option>
          <option value="exit">驶出告警</option>
          <option value="both">进出都告警</option>
        </select>
        <input v-model.trim="newFence.vin" placeholder="VIN（空=全部）" />
        <button type="submit" class="btn">+ 新建围栏</button>
      </form>
    </section>

    <!-- 历史告警 -->
    <section class="panel">
      <h3>📜 历史告警</h3>
      <table v-if="resolvedAlarms.length">
        <thead>
          <tr><th>触发时间</th><th>恢复时间</th><th>级别</th><th>车辆</th><th>标题</th></tr>
        </thead>
        <tbody>
          <tr v-for="a in resolvedAlarms" :key="a.id" class="resolved">
            <td class="dim">{{ fmtTime(a.triggered_at) }}</td>
            <td class="dim">{{ a.resolved_at ? fmtTime(a.resolved_at) : '-' }}</td>
            <td><span class="sev" :class="a.severity">{{ severityText(a.severity) }}</span></td>
            <td class="mono">{{ a.vin }}</td>
            <td>{{ a.title }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无历史告警</div>
    </section>
    <!-- 操作反馈 toast -->
    <transition name="slide">
      <div v-if="toast" class="toast" :class="{ err: toastErr }">{{ toast }}</div>
    </transition>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import api from '../api'

const rules = ref([])
const fences = ref([])
const activeAlarms = ref([])
const resolvedAlarms = ref([])

const newRule = ref({ name: '', metric: 'battery', operator: 'lt', threshold: 20, severity: 'warning' })
const newFence = ref({ name: '', lat: 31.23, lng: 121.47, radius_m: 500, alarm_on: 'both', vin: '' })

let timer = null
let toastTimer = null
const toast = ref('')
const toastErr = ref(false)

function showToast(m, isErr = false) {
  toast.value = m
  toastErr.value = isErr
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value = '' }, 3000)
}

// act 统一包裹操作：失败弹 toast，成功自动刷新
async function act(fn, okMsg) {
  try {
    await fn()
    if (okMsg) showToast(okMsg)
    loadAll()
  } catch (e) {
    showToast(e?.msg || '操作失败', true)
  }
}

async function loadAll() {
  try {
    const [r, f, active, resolved] = await Promise.all([
      api.get('/api/v1/alarm/rules'),
      api.get('/api/v1/fences'),
      api.get('/api/v1/alarm/records', { params: { status: 'active' } }),
      api.get('/api/v1/alarm/records', { params: { status: 'resolved', limit: 20 } }),
    ])
    rules.value = r.data || []
    fences.value = f.data || []
    activeAlarms.value = active.data || []
    resolvedAlarms.value = resolved.data || []
  } catch { /* 静默，下个周期重试 */ }
}

async function toggleRule(r) {
  await act(() => api.put(`/api/v1/alarm/rules/${r.id}`, { ...r, enabled: !r.enabled }))
}
async function removeRule(id) {
  await act(() => api.delete(`/api/v1/alarm/rules/${id}`), '规则已删除')
}
async function createRule() {
  if (newRule.value.metric === 'fault_code') {
    newRule.value.operator = 'ne'
    newRule.value.threshold = 0
  }
  await act(() => api.post('/api/v1/alarm/rules', newRule.value), '规则已创建')
  newRule.value.name = ''
}
async function createFence() {
  await act(() => api.post('/api/v1/fences', newFence.value), '围栏已创建')
  newFence.value.name = ''
}
async function removeFence(id) {
  await act(() => api.delete(`/api/v1/fences/${id}`), '围栏已删除')
}
async function resolve(id) {
  await act(() => api.post(`/api/v1/alarm/records/${id}/resolve`), '已恢复')
}

function fmtTime(t) { return t ? new Date(t).toLocaleTimeString() : '-' }
function severityText(s) { return { info: '提示', warning: '警告', critical: '严重' }[s] || s }
function metricText(m) { return { speed: '车速', rpm: '转速', battery: '电量', fault_code: '故障码' }[m] || m }
function opText(o) { return { lt: '<', lte: '≤', gt: '>', gte: '≥', ne: '≠' }[o] || o }
function fenceOnText(v) { return { enter: '驶入', exit: '驶出', both: '进出' }[v] || v }

onMounted(() => {
  loadAll()
  timer = setInterval(loadAll, 5000)
})
onUnmounted(() => { clearInterval(timer); clearTimeout(toastTimer) })
</script>

<style scoped>
h2 { margin-bottom: 16px; font-size: 20px; }
.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
  margin-bottom: 16px;
}
.panel h3 { font-size: 15px; margin-bottom: 12px; }
.count { color: var(--red); }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th, td { text-align: left; padding: 8px 10px; border-bottom: 1px solid var(--border); }
th { color: var(--text-dim); font-weight: normal; }
.mono { font-family: Consolas, monospace; font-size: 12px; }
.dim { color: var(--text-dim); }
.sev { padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.sev.info { background: #1f6feb33; color: var(--blue); }
.sev.warning { background: #d2992233; color: var(--yellow); }
.sev.critical { background: #ff525233; color: var(--red); }
.btn, .btn-sm {
  background: #1f6feb; color: #fff; border: none; border-radius: 6px;
  padding: 6px 14px; cursor: pointer; font-size: 13px;
}
.btn-sm { padding: 3px 10px; font-size: 12px; background: #21262d; border: 1px solid var(--border); color: var(--text); }
.btn-sm.danger:hover { border-color: var(--red); color: var(--red); }
.toggle {
  padding: 3px 10px; font-size: 12px; border-radius: 10px; cursor: pointer;
  background: #21262d; border: 1px solid var(--border); color: var(--text-dim);
}
.toggle.on { background: #3fb95033; color: var(--green); border-color: var(--green); }
.empty { color: var(--text-dim); font-size: 13px; padding: 12px 0; }
.resolved td { opacity: 0.6; }
.create-form { display: flex; gap: 8px; margin-top: 12px; flex-wrap: wrap; }
.create-form input, .create-form select {
  background: var(--bg); border: 1px solid var(--border); border-radius: 6px;
  color: var(--text); padding: 6px 10px; font-size: 13px;
}
.create-form input { width: 130px; }
.toast {
  position: fixed; bottom: 24px; right: 24px; background: #238636; color: #fff;
  padding: 10px 18px; border-radius: 8px; font-size: 13px; z-index: 10;
}
.toast.err { background: #da3633; }
.slide-enter-active, .slide-leave-active { transition: all 0.3s; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(10px); }
</style>
