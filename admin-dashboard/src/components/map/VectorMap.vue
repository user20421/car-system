<script setup lang="ts">
import { computed, ref } from 'vue'
import { LocateFixed, Minus, Navigation, Plus, Radio } from 'lucide-vue-next'
import type { Telemetry, Vehicle } from '../../types/api'
import { relativeTime } from '../../utils/format'

const props = withDefaults(defineProps<{ vehicles: Vehicle[]; telemetry: Record<string, Telemetry>; selectedVin?: string; history?: Telemetry[]; interactive?: boolean; height?: number }>(), { selectedVin: '', history: () => [], interactive: true, height: 520 })
const emit = defineEmits<{ select: [vin: string] }>()
const zoom = ref(1)
const bounds = computed(() => {
  const points = Object.values(props.telemetry).map(t => t.location).filter(Boolean)
  const lats = points.map(p => p!.latitude), lons = points.map(p => p!.longitude)
  const minLat = Math.min(...lats, 31.17), maxLat = Math.max(...lats, 31.28)
  const minLon = Math.min(...lons, 121.39), maxLon = Math.max(...lons, 121.56)
  return { minLat: minLat - .01, maxLat: maxLat + .01, minLon: minLon - .015, maxLon: maxLon + .015 }
})
function position(lat?: number, lon?: number) {
  if (lat === undefined || lon === undefined) return { left: '50%', top: '50%' }
  const b = bounds.value
  return { left: `${Math.max(3, Math.min(97, (lon - b.minLon) / (b.maxLon - b.minLon) * 100))}%`, top: `${Math.max(5, Math.min(95, (1 - (lat - b.minLat) / (b.maxLat - b.minLat)) * 100))}%` }
}
const trackPoints = computed(() => props.history.filter(t => t.location).map(t => {
  const p = position(t.location!.latitude, t.location!.longitude)
  return `${parseFloat(p.left) * 10},${parseFloat(p.top) * 5.6}`
}).join(' '))
const selected = computed(() => props.vehicles.find(v => v.vin === props.selectedVin))
const selectedTelemetry = computed(() => props.selectedVin ? props.telemetry[props.selectedVin] : undefined)
</script>
<template>
  <div class="vector-map" :style="{ height: `${height}px` }">
    <div class="map-world" :style="{ transform: `scale(${zoom})` }">
      <svg class="map-drawing" viewBox="0 0 1000 560" preserveAspectRatio="none" aria-label="上海车队本地矢量地图">
        <defs><pattern id="map-grid" width="36" height="36" patternUnits="userSpaceOnUse"><path d="M36 0H0V36" fill="none" stroke="currentColor" stroke-opacity=".045"/></pattern><linearGradient id="river" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#B6DDE4"/><stop offset="1" stop-color="#A6D3DF"/></linearGradient></defs>
        <rect width="1000" height="560" class="map-ground"/><rect width="1000" height="560" fill="url(#map-grid)"/>
        <g class="blocks">
          <path d="M45 40h175l32 83-48 62H54Z"/><path d="m280 28 194 5 28 117-184 35-58-73Z"/><path d="m560 25 175 20-7 128-170 9-35-78Z"/><path d="m790 25 170 13 12 150-152 11-64-78Z"/>
          <path d="m30 240 185-22 35 111-50 70H35Z"/><path d="m280 220 173-29 54 121-27 77-190-12-37-83Z"/><path d="m560 230 135-18 48 94-27 75-160 14-38-91Z"/><path d="m785 238 180-16 7 174-160-4-55-87Z"/>
          <path d="m65 440 175-17 46 112H54Z"/><path d="m335 432 169-13 28 119H309Z"/><path d="m596 435 140-24 57 126H580Z"/><path d="m825 434 137-7 12 111H813Z"/>
        </g>
        <path class="river" d="M713-30c-82 104-37 167-101 232-60 61-51 106-11 151 45 51 32 128-27 237h82c67-118 72-190 27-249-34-45-34-76 13-124 68-69 21-149 88-247Z"/>
        <g class="minor-roads"><path d="M0 201 1000 93M0 414l1000-89M232 0l46 560M520 0l-8 560M815 0l-42 560M0 75l1000 438M49 560 941 0"/></g>
        <g class="major-roads"><path d="M-20 315C203 269 356 341 548 279S810 154 1020 197"/><path d="M373-20c20 146 83 233 48 353s-20 172 9 247"/><path d="M-20 489c240-100 396-48 558-30s305-23 482-89"/></g>
        <g class="road-centers"><path d="M-20 315C203 269 356 341 548 279S810 154 1020 197"/><path d="M373-20c20 146 83 233 48 353s-20 172 9 247"/></g>
        <g class="map-labels"><text x="95" y="103">静安寺</text><text x="317" y="95">人民广场</text><text x="779" y="118">北外滩</text><text x="90" y="352">徐家汇</text><text x="325" y="292">淮海中路</text><text x="778" y="330">陆家嘴</text><text x="463" y="518">前 滩</text><text x="638" y="248" class="water">黄浦江</text></g>
        <polyline v-if="trackPoints" :points="trackPoints" class="track-shadow"/><polyline v-if="trackPoints" :points="trackPoints" class="track-line"/>
      </svg>
      <button v-for="vehicle in vehicles" :key="vehicle.vin" class="map-marker" :class="{ selected: selectedVin===vehicle.vin, offline: !telemetry[vehicle.vin] || Date.now()-telemetry[vehicle.vin].occurred_at_ms>120000 }" :style="position(telemetry[vehicle.vin]?.location?.latitude,telemetry[vehicle.vin]?.location?.longitude)" :title="vehicle.display_name" @click="emit('select',vehicle.vin)">
        <span class="marker-pulse"/><span class="marker-core"><Navigation :style="{ transform:`rotate(${telemetry[vehicle.vin]?.location?.heading_deg || 0}deg)` }" /></span><span v-if="selectedVin===vehicle.vin" class="marker-label">{{ vehicle.display_name }}</span>
      </button>
    </div>
    <div class="map-top-left"><span class="map-live"><Radio /> LIVE</span><span>上海 · 本地矢量地图</span></div>
    <div v-if="interactive" class="map-controls"><button title="放大" @click="zoom=Math.min(1.8,zoom+.15)"><Plus /></button><button title="缩小" @click="zoom=Math.max(.8,zoom-.15)"><Minus /></button><button title="复位" @click="zoom=1"><LocateFixed /></button></div>
    <Transition name="fade"><div v-if="selected && selectedTelemetry" class="map-popover"><div class="row-between"><strong>{{ selected.display_name }}</strong><span class="numeric">{{ Math.round(selectedTelemetry.speed_kph) }} <small>km/h</small></span></div><div class="map-popover-meta"><span>{{ selected.model }}</span><span>{{ relativeTime(selectedTelemetry.occurred_at_ms) }}</span></div><div class="battery-track"><i :style="{ width:`${selectedTelemetry.battery_percent}%` }" /></div><div class="map-popover-meta"><span>电量 {{ selectedTelemetry.battery_percent.toFixed(0) }}%</span><span>续航 {{ selectedTelemetry.range_km }} km</span></div></div></Transition>
    <div class="map-attribution">AURORA LOCAL VECTOR · 无外部地图密钥</div>
  </div>
</template>
<style scoped>
.vector-map{position:relative;overflow:hidden;background:#e9edf0;border-radius:14px;isolation:isolate;color:#75808d}.map-world{position:absolute;inset:0;transition:transform .3s;transform-origin:center}.map-drawing{width:100%;height:100%;display:block}.map-ground{fill:#e9edef}.blocks{fill:#e2e7e9;stroke:#dbe1e4;stroke-width:2}.river{fill:url(#river);opacity:.85}.minor-roads path{fill:none;stroke:#f8f9fa;stroke-width:7}.major-roads path{fill:none;stroke:#fff;stroke-width:14}.road-centers path{fill:none;stroke:#d5dce0;stroke-width:1;stroke-dasharray:7 8}.map-labels{font-family:var(--font);font-size:12px;font-weight:600;fill:#8b959f;letter-spacing:2px}.map-labels .water{fill:#72a9b6;font-size:13px;letter-spacing:5px}.track-shadow{fill:none;stroke:#fff;stroke-width:8;stroke-linecap:round;stroke-linejoin:round;opacity:.9}.track-line{fill:none;stroke:var(--primary);stroke-width:4;stroke-linecap:round;stroke-linejoin:round;stroke-dasharray:8 4}.map-marker{position:absolute;transform:translate(-50%,-50%);border:0;background:transparent;padding:0;width:34px;height:34px;cursor:pointer;z-index:2}.marker-core{position:absolute;inset:3px;border-radius:50% 50% 50% 10%;transform:rotate(-45deg);display:grid;place-items:center;background:var(--primary);color:#fff;border:2px solid #fff;box-shadow:0 3px 10px rgba(55,53,100,.28)}.marker-core svg{width:13px;height:13px;transform:rotate(45deg)!important;stroke-width:2.2}.marker-pulse{position:absolute;inset:-4px;border:1px solid var(--primary);border-radius:50%;opacity:0}.selected .marker-pulse{animation:map-pulse 2s infinite}.map-marker.selected{z-index:5}.map-marker.offline .marker-core{background:#8d96a3}.marker-label{position:absolute;bottom:38px;left:50%;transform:translateX(-50%);background:#20243a;color:#fff;font-size:9px;font-weight:650;padding:5px 8px;border-radius:6px;white-space:nowrap;box-shadow:0 3px 8px rgba(0,0,0,.18)}.map-top-left{position:absolute;left:14px;top:14px;display:flex;align-items:center;gap:9px;background:rgba(255,255,255,.86);backdrop-filter:blur(8px);padding:6px 9px;border-radius:8px;font-size:9px;box-shadow:0 2px 8px rgba(30,40,60,.06)}.map-live{display:flex;align-items:center;gap:4px;color:#159d82;font-weight:750;font-size:8px;letter-spacing:.08em}.map-live svg{width:11px}.map-controls{position:absolute;right:14px;top:14px;display:flex;flex-direction:column;background:rgba(255,255,255,.9);border-radius:9px;box-shadow:0 2px 10px rgba(30,40,60,.08);overflow:hidden}.map-controls button{width:32px;height:32px;border:0;background:transparent;color:#5e6875;display:grid;place-items:center;cursor:pointer}.map-controls button:hover{background:#f0f2f5}.map-controls svg{width:14px}.map-popover{position:absolute;left:14px;bottom:30px;width:235px;padding:13px 14px;background:rgba(255,255,255,.94);backdrop-filter:blur(10px);color:#232839;border-radius:11px;box-shadow:0 8px 30px rgba(40,50,70,.12)}.map-popover strong{font-size:12px}.map-popover .numeric{font-size:16px;font-weight:700}.map-popover small{font-size:8px;color:#8b939f}.map-popover-meta{display:flex;justify-content:space-between;color:#8a929e;font-size:9px;margin-top:6px}.battery-track{height:4px;background:#e6e9ed;border-radius:4px;margin-top:11px;overflow:hidden}.battery-track i{display:block;height:100%;background:linear-gradient(90deg,#7366e8,#36c6b8);border-radius:4px}.map-attribution{position:absolute;bottom:7px;right:10px;font-size:6px;letter-spacing:.1em;color:#98a1a9}.vector-map:deep(.badge){pointer-events:none}@keyframes map-pulse{0%{transform:scale(.6);opacity:.8}100%{transform:scale(1.7);opacity:0}}
:root[data-theme='dark'] .vector-map{filter:brightness(.86) saturate(.8)}
</style>
