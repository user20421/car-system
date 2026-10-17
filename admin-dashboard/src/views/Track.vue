<template>
  <div class="track-page">
    <div class="head">
      <h2>🛣️ 轨迹回放 <span class="vin">{{ vin }}</span></h2>
      <button class="btn" @click="$router.push(`/vehicle/${vin}`)">← 返回详情</button>
    </div>

    <div class="toolbar">
      <button class="btn" :disabled="!points.length" @click="togglePlay">
        {{ playing ? '⏸ 暂停' : '▶ 回放' }}
      </button>
      <button class="btn-sm" :disabled="!points.length" @click="reset">⟲ 重置</button>
      <input
        type="range" min="0" :max="Math.max(points.length - 1, 0)" v-model.number="cursor"
        class="slider" :disabled="!points.length"
      />
      <span class="dim">{{ points.length ? `${cursor + 1} / ${points.length}` : '无数据' }}</span>
      <label class="dim">速度
        <select v-model.number="speedFactor">
          <option :value="1">1x</option>
          <option :value="4">4x</option>
          <option :value="16">16x</option>
        </select>
      </label>
    </div>

    <div class="info" v-if="current">
      <span>🕐 {{ fmtTime(current.created_at) }}</span>
      <span>🚗 {{ current.speed.toFixed(1) }} km/h</span>
      <span>🔋 {{ current.battery.toFixed(1) }}%</span>
      <span v-if="current.fault_code" class="red">⚠ {{ current.fault_code }}</span>
    </div>

    <div ref="mapEl" class="map"></div>
    <div v-if="!points.length && loaded" class="empty-tip">该车辆暂无含 GPS 的历史数据，请先让车机/模拟器跑一会儿</div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import api from '../api'

const route = useRoute()
const vin = route.params.vin

const mapEl = ref(null)
const points = ref([])
const loaded = ref(false)
const cursor = ref(0)
const playing = ref(false)
const speedFactor = ref(4)

const current = computed(() => points.value[cursor.value] || null)

let map = null
let trackLine = null     // 完整轨迹（暗色）
let passedLine = null    // 已走过的部分（亮色）
let marker = null
let timer = null

async function loadTrack() {
  try {
    const res = await api.get(`/api/v1/vehicle/${vin}/telemetry/history`, { params: { limit: 300 } })
    points.value = (res.data || []).filter((p) => p.lat && p.lng)
  } catch {
    points.value = [] // 失败也进入空态提示，不再卡在加载中
  } finally {
    loaded.value = true
  }
}

function initMap() {
  map = L.map(mapEl.value, { zoomControl: true })
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '© OpenStreetMap',
  }).addTo(map)

  const latlngs = points.value.map((p) => [p.lat, p.lng])
  trackLine = L.polyline(latlngs, { color: '#30363d', weight: 4 }).addTo(map)
  passedLine = L.polyline([], { color: '#4fc3f7', weight: 4 }).addTo(map)
  marker = L.circleMarker(latlngs[0], {
    radius: 8, color: '#fff', weight: 2, fillColor: '#4fc3f7', fillOpacity: 1,
  }).addTo(map)
  map.fitBounds(trackLine.getBounds(), { padding: [30, 30] })
}

function renderCursor() {
  if (!points.value.length || !map) return
  const passed = points.value.slice(0, cursor.value + 1).map((p) => [p.lat, p.lng])
  passedLine.setLatLngs(passed)
  const p = points.value[cursor.value]
  marker.setLatLng([p.lat, p.lng])
}

function togglePlay() {
  if (playing.value) {
    stopPlay()
    return
  }
  if (cursor.value >= points.value.length - 1) cursor.value = 0
  playing.value = true
  timer = setInterval(() => {
    if (cursor.value >= points.value.length - 1) {
      stopPlay()
      return
    }
    cursor.value++
  }, 1000 / speedFactor.value)
}
function stopPlay() {
  playing.value = false
  clearInterval(timer)
}
function reset() {
  stopPlay()
  cursor.value = 0
}
function fmtTime(t) { return t ? new Date(t).toLocaleTimeString() : '-' }

// 倍速切换时重启定时器
watch(speedFactor, () => { if (playing.value) { stopPlay(); togglePlay() } })
watch(cursor, renderCursor)

onMounted(async () => {
  await loadTrack()
  await nextTick()
  if (points.value.length) initMap()
})

onUnmounted(() => {
  stopPlay()
  map?.remove()
})
</script>

<style scoped>
.head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
h2 { font-size: 20px; }
.vin { font-family: Consolas, monospace; font-size: 14px; color: var(--text-dim); }
.toolbar {
  display: flex; align-items: center; gap: 12px; margin-bottom: 12px;
  background: var(--panel); border: 1px solid var(--border); border-radius: 10px; padding: 10px 14px;
}
.slider { flex: 1; accent-color: var(--blue); }
.btn { background: #1f6feb; color: #fff; border: none; border-radius: 6px; padding: 6px 14px; cursor: pointer; font-size: 13px; }
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-sm { background: #21262d; border: 1px solid var(--border); color: var(--text); border-radius: 6px; padding: 5px 10px; cursor: pointer; font-size: 12px; }
.dim { color: var(--text-dim); font-size: 13px; }
.red { color: var(--red); }
select { background: var(--bg); border: 1px solid var(--border); border-radius: 6px; color: var(--text); padding: 3px 6px; }
.info {
  display: flex; gap: 20px; margin-bottom: 12px; font-size: 14px;
  background: var(--panel); border: 1px solid var(--border); border-radius: 10px; padding: 10px 14px;
}
.map { height: calc(100vh - 240px); border-radius: 10px; border: 1px solid var(--border); }
.empty-tip { color: var(--text-dim); margin-top: 12px; font-size: 13px; }
</style>
