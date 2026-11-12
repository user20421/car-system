<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, BatteryCharging, CheckCircle2, ChevronRight, CircleGauge, Clock3, Cpu, Download, KeyRound, MapPin, MoreHorizontal, Navigation, RefreshCw, Satellite, ShieldCheck, Signal, Thermometer, Wrench } from 'lucide-vue-next'
import PageHeader from '../components/common/PageHeader.vue'
import StatePanel from '../components/common/StatePanel.vue'
import StatusBadge from '../components/common/StatusBadge.vue'
import VehicleArt from '../components/vehicle/VehicleArt.vue'
import InstrumentCluster from '../components/vehicle/InstrumentCluster.vue'
import VehicleControls from '../components/vehicle/VehicleControls.vue'
import CommandTimeline from '../components/vehicle/CommandTimeline.vue'
import VectorMap from '../components/map/VectorMap.vue'
import EChart from '../components/charts/EChart.vue'
import { api } from '../api'
import { asApiError, type ApiError } from '../api/errors'
import type { Telemetry, Vehicle } from '../types/api'
import { useFleetStore } from '../stores/fleet'
import { useAuthStore } from '../stores/auth'
import { useUiStore } from '../stores/ui'
import { useOnline } from '../composables/useOnline'
import { formatTime, relativeTime } from '../utils/format'
const route = useRoute(), router = useRouter(), fleet = useFleetStore(), auth = useAuthStore(), ui = useUiStore()
const { online } = useOnline()
const vehicle = ref<Vehicle>(), telemetry = ref<Telemetry>(), history = ref<Telemetry[]>([])
const loading = ref(true), historyLoading = ref(false), error = ref<ApiError | null>(null), tab = ref<'telemetry'|'track'|'commands'>('telemetry'), range = ref('1h')
const vin = computed(() => String(route.params.vin))
const isOnline = computed(() => telemetry.value && Date.now() - telemetry.value.occurred_at_ms < 120_000)
const chartOption = computed(() => {
  const labels = history.value.map(t => formatTime(t.occurred_at_ms))
  const speed = history.value.map(t => Math.round(t.speed_kph * 10) / 10)
  const battery = history.value.map(t => t.battery_percent)
  return { animationDuration: 500, grid: { left: 42,right: 42,top: 40,bottom: 35 }, legend: { top: 3,right: 10,textStyle:{color:'#9298a9',fontSize:9},itemWidth:12,itemHeight:3 }, tooltip: { trigger:'axis' as const,backgroundColor:ui.theme==='dark'?'#242940':'#fff',borderColor:ui.theme==='dark'?'#343a55':'#e6e8ef',textStyle:{color:ui.theme==='dark'?'#eef0f8':'#25293a',fontSize:10} }, xAxis:{type:'category' as const,data:labels,axisLine:{lineStyle:{color:'#e1e4ec'}},axisTick:{show:false},axisLabel:{color:'#9298a9',fontSize:8,max:6} }, yAxis:[{type:'value' as const,name:'km/h',nameTextStyle:{color:'#9298a9',fontSize:8},axisLabel:{color:'#9298a9',fontSize:8},splitLine:{lineStyle:{color:ui.theme==='dark'?'#2b3048':'#eceef4',type:'dashed' as const}}},{type:'value' as const,name:'%',min:0,max:100,nameTextStyle:{color:'#9298a9',fontSize:8},axisLabel:{color:'#9298a9',fontSize:8},splitLine:{show:false}}], series:[{name:'车速',type:'line' as const,data:speed,smooth:true,showSymbol:false,lineStyle:{width:2,color:'#7567eb'},itemStyle:{color:'#7567eb'},areaStyle:{color:'rgba(117,103,235,.07)'}},{name:'电量',type:'line' as const,yAxisIndex:1,data:battery,smooth:true,showSymbol:false,lineStyle:{width:2,color:'#24b9b0'},itemStyle:{color:'#24b9b0'}}] }
})
const trackVehicles = computed(() => vehicle.value ? [vehicle.value] : [])
const telemetryMap = computed(() => telemetry.value ? { [telemetry.value.vin]: telemetry.value } : {})
async function load() {
  loading.value = true; error.value = null
  try {
    const [v,t] = await Promise.all([api.getVehicle(vin.value),api.latestTelemetry(vin.value)])
    vehicle.value = v; telemetry.value = t; fleet.selectedVin = v.vin; fleet.updateTelemetry(t)
    await loadHistory()
  } catch (cause) { error.value = asApiError(cause) }
  finally { loading.value = false }
}
async function loadHistory() {
  historyLoading.value = true
  const hours = range.value === '1h' ? 1 : range.value === '6h' ? 6 : 24
  try { history.value = (await api.telemetryHistory(vin.value,Date.now()-hours*3600_000,Date.now())).items }
  catch (cause) { ui.toast('历史数据加载失败','error',asApiError(cause).message) }
  finally { historyLoading.value = false }
}
function exportHistory() {
  if (!history.value.length) return
  const rows = ['time,vin,speed_kph,battery_percent,range_km,latitude,longitude',...history.value.map(t => `${new Date(t.occurred_at_ms).toISOString()},${t.vin},${t.speed_kph},${t.battery_percent},${t.range_km},${t.location?.latitude||''},${t.location?.longitude||''}`)]
  const url = URL.createObjectURL(new Blob([rows.join('\n')],{type:'text/csv;charset=utf-8'}))
  const a = document.createElement('a'); a.href=url;a.download=`${vin.value}-telemetry.csv`;a.click();URL.revokeObjectURL(url)
}
onMounted(() => void load())
watch(vin, () => void load())
watch(range, () => void loadHistory())
</script>
<template>
  <div class="page detail-page">
    <button class="back-link" @click="router.push('/fleet')"><ArrowLeft/>返回实时车队</button>
    <StatePanel :loading="loading" :error="error" :offline="!online&&!vehicle" @retry="load">
      <template v-if="vehicle && telemetry">
        <PageHeader :eyebrow="vehicle.model.toUpperCase()" :title="vehicle.display_name" :subtitle="`${vehicle.license_plate || '未登记号牌'} · ${vehicle.vin}`"><template #actions><StatusBadge :status="isOnline?'online':'offline'" :pulse="isOnline"/><button class="btn btn-secondary" @click="load"><RefreshCw/>刷新</button><button class="btn btn-secondary btn-icon"><MoreHorizontal/></button></template></PageHeader>
        <div class="vehicle-hero-grid">
          <article class="card vehicle-profile"><div class="profile-glow"/><div class="profile-top"><span class="badge badge-info">{{ vehicle.model_year }} MODEL</span><span class="profile-location"><MapPin/>{{ isOnline?'上海市 · 徐汇区':'最后已知位置' }}</span></div><div class="profile-art"><VehicleArt :color="vehicle.color"/></div><div class="profile-meta"><div><span>车辆识别代码</span><strong class="mono">{{ vehicle.vin }}</strong></div><div><span>当前固件</span><strong>AURORA OS {{ vehicle.firmware_version }}</strong></div><div><span>我的权限</span><strong>{{ vehicle.access_role==='owner'?'车主':vehicle.access_role==='driver'?'驾驶员':'观察者' }}</strong></div></div></article>
          <article class="card instrument-card"><div class="card-header"><div><h3 class="card-title">实时仪表</h3><p class="card-caption">{{ relativeTime(telemetry.occurred_at_ms) }}更新 · 序列 #{{ telemetry.sequence }}</p></div><Signal/></div><div class="card-body"><InstrumentCluster :telemetry="telemetry"/></div></article>
          <article class="card quick-stats"><div class="quick-stat"><span class="stat-icon violet"><Navigation/></span><div><span>电机转速</span><strong class="numeric">{{ telemetry.rpm.toLocaleString() }} <small>rpm</small></strong></div></div><div class="quick-stat"><span class="stat-icon cyan"><KeyRound/></span><div><span>车辆锁止</span><strong>{{ telemetry.locked?'已闭锁':'已解锁' }}</strong></div></div><div class="quick-stat"><span class="stat-icon green"><ShieldCheck/></span><div><span>活跃故障</span><strong>{{ telemetry.fault.filter(f=>f.active).length }} <small>项</small></strong></div></div><div class="quick-stat"><span class="stat-icon orange"><Satellite/></span><div><span>定位精度</span><strong>±{{ telemetry.location?.horizontal_accuracy_m||'—' }} <small>m</small></strong></div></div><div class="device-boot"><Cpu/><span>设备启动标识</span><strong class="mono">{{ telemetry.device_boot_id }}</strong></div></article>
        </div>
        <div class="detail-content-grid">
          <section class="card detail-main"><div class="detail-tabs"><div><button :class="{active:tab==='telemetry'}" @click="tab='telemetry'"><CircleGauge/>历史遥测</button><button :class="{active:tab==='track'}" @click="tab='track'"><MapPin/>行驶轨迹</button><button :class="{active:tab==='commands'}" @click="tab='commands'"><Clock3/>指令时间线</button></div><div v-if="tab!=='commands'" class="segmented"><button v-for="r in ['1h','6h','24h']" :key="r" :class="{active:range===r}" @click="range=r">{{ r==='1h'?'1 小时':r==='6h'?'6 小时':'24 小时' }}</button></div></div>
            <div v-if="tab==='telemetry'" class="tab-content"><div class="chart-toolbar"><div><strong>车速与电量</strong><span>采样点 {{ history.length }} · 按发生时间升序</span></div><button class="btn btn-ghost btn-sm" :disabled="!history.length" @click="exportHistory"><Download/>导出 CSV</button></div><StatePanel :loading="historyLoading" :empty="!historyLoading&&!history.length" compact @retry="loadHistory"><EChart :option="chartOption" :height="315"/></StatePanel><div class="chart-insights"><div><span>平均车速</span><strong>{{ history.length?(history.reduce((s,t)=>s+t.speed_kph,0)/history.length).toFixed(1):'—' }} <small>km/h</small></strong></div><div><span>最高车速</span><strong>{{ history.length?Math.max(...history.map(t=>t.speed_kph)).toFixed(0):'—' }} <small>km/h</small></strong></div><div><span>电量变化</span><strong>{{ history.length>1?(history.at(-1)!.battery_percent-history[0].battery_percent).toFixed(1):'—' }} <small>%</small></strong></div></div></div>
            <div v-else-if="tab==='track'" class="tab-content"><div class="chart-toolbar"><div><strong>车辆轨迹回放</strong><span>本地矢量渲染 · {{ history.filter(t=>t.location).length }} 个定位点</span></div><span class="badge badge-info">{{ range.toUpperCase() }}</span></div><VectorMap :vehicles="trackVehicles" :telemetry="telemetryMap" :selected-vin="vehicle.vin" :history="history" :height="390"/><div class="track-note"><MapPin/>轨迹由所选时间范围内的历史遥测生成，仅展示服务端已保存的数据。</div></div>
            <div v-else class="tab-content"><div class="chart-toolbar"><div><strong>指令投递与执行</strong><span>终态不可逆转，失败与过期不会被标记为成功</span></div></div><CommandTimeline :vin="vehicle.vin"/></div>
          </section>
          <aside class="stack"><article class="card"><div class="card-header"><div><h3 class="card-title">远程控制</h3><p class="card-caption">所有指令均需二次确认</p></div><KeyRound/></div><div class="card-body"><VehicleControls :vehicle="vehicle" :telemetry="telemetry" :user="auth.user"/></div></article><article class="card health-detail"><div class="card-header"><div><h3 class="card-title">车辆健康</h3><p class="card-caption">最新自检与设备信息</p></div><span class="badge badge-success">正常</span></div><div class="card-body"><div class="health-row"><span><BatteryCharging/>动力电池</span><strong>{{ telemetry.battery_percent.toFixed(0) }}%<CheckCircle2/></strong></div><div class="health-row"><span><Thermometer/>座舱与空调</span><strong>{{ telemetry.hvac?.mode==='off'?'已关闭':'运行中' }}<CheckCircle2/></strong></div><div class="health-row"><span><Wrench/>故障诊断</span><strong>{{ telemetry.fault.filter(f=>f.active).length }} 项活跃<ChevronRight/></strong></div><div v-for="fault in telemetry.fault.filter(f=>f.active)" :key="fault.code" class="fault-item"><strong>{{ fault.code }}</strong><span>{{ fault.message }}</span></div></div></article></aside>
        </div>
      </template>
    </StatePanel>
  </div>
</template>
<style scoped>
.back-link{display:flex;align-items:center;gap:6px;border:0;background:transparent;color:var(--muted);font-size:10px;padding:0;margin-bottom:16px;cursor:pointer}.back-link:hover{color:var(--primary)}.back-link svg{width:13px}.vehicle-hero-grid{display:grid;grid-template-columns:minmax(0,1.5fr) minmax(280px,1fr) minmax(210px,.75fr);gap:18px}.vehicle-profile{position:relative;overflow:hidden;padding:20px 22px}.profile-glow{position:absolute;inset:0;background:radial-gradient(circle at 60% 45%,color-mix(in srgb,var(--primary) 10%,transparent),transparent 55%);pointer-events:none}.profile-top{display:flex;justify-content:space-between;position:relative}.profile-location{display:flex;align-items:center;gap:5px;color:var(--muted);font-size:8px}.profile-location svg{width:11px}.profile-art{height:185px;margin-top:-2px}.profile-art :deep(svg){height:100%}.profile-meta{display:grid;grid-template-columns:1.4fr 1fr .7fr;border-top:1px solid var(--line);padding-top:14px;gap:12px;position:relative}.profile-meta>div{display:flex;flex-direction:column;gap:5px}.profile-meta span{font-size:8px;color:var(--muted)}.profile-meta strong{font-size:9px;font-weight:600}.instrument-card>.card-header>svg{width:15px;color:var(--green)}.instrument-card .card-body{padding-top:2px}.quick-stats{padding:8px 18px;display:flex;flex-direction:column}.quick-stat{display:flex;align-items:center;gap:11px;padding:13px 0;border-bottom:1px solid var(--line)}.stat-icon{width:32px;height:32px;border-radius:9px;background:var(--primary-soft);color:var(--primary);display:grid;place-items:center}.stat-icon svg{width:15px}.stat-icon.cyan{background:var(--cyan-soft);color:var(--cyan)}.stat-icon.green{background:var(--green-soft);color:var(--green)}.stat-icon.orange{background:var(--orange-soft);color:var(--orange)}.quick-stat>div{display:flex;flex-direction:column;gap:4px}.quick-stat span{font-size:8px;color:var(--muted)}.quick-stat strong{font-size:14px}.quick-stat small{font-size:8px;color:var(--muted);font-weight:500}.device-boot{display:flex;align-items:center;gap:6px;padding:13px 0 9px;font-size:7px;color:var(--muted)}.device-boot svg{width:11px}.device-boot strong{margin-left:auto;font-size:7px}.detail-content-grid{display:grid;grid-template-columns:minmax(0,1.65fr) minmax(280px,.7fr);gap:18px;margin-top:18px}.detail-main{min-width:0}.detail-tabs{padding:0 20px;height:60px;border-bottom:1px solid var(--line);display:flex;justify-content:space-between;align-items:center}.detail-tabs>div:first-child{display:flex;height:100%;gap:20px}.detail-tabs>div>button{border:0;background:transparent;color:var(--muted);font-size:10px;display:flex;align-items:center;gap:6px;cursor:pointer;position:relative}.detail-tabs>div:first-child>button.active{color:var(--primary);font-weight:650}.detail-tabs>div:first-child>button.active::after{content:'';position:absolute;bottom:-1px;left:0;right:0;height:2px;background:var(--primary)}.detail-tabs svg{width:13px}.tab-content{padding:18px 20px}.chart-toolbar{display:flex;justify-content:space-between;align-items:center;margin-bottom:8px}.chart-toolbar>div{display:flex;flex-direction:column;gap:5px}.chart-toolbar strong{font-size:12px}.chart-toolbar span{font-size:8px;color:var(--muted)}.chart-insights{display:grid;grid-template-columns:repeat(3,1fr);background:var(--bg-soft);border-radius:10px;padding:12px}.chart-insights>div{display:flex;flex-direction:column;align-items:center;gap:6px;border-right:1px solid var(--line)}.chart-insights>div:last-child{border:0}.chart-insights span{font-size:8px;color:var(--muted)}.chart-insights strong{font-size:14px}.chart-insights small{font-size:8px;color:var(--muted)}.track-note{display:flex;align-items:center;gap:6px;font-size:8px;color:var(--muted);margin-top:10px}.track-note svg{width:12px}.health-detail .card-body{padding-top:12px}.health-row{display:flex;justify-content:space-between;align-items:center;padding:10px 0;border-bottom:1px solid var(--line);font-size:9px}.health-row>span{display:flex;align-items:center;gap:7px;color:var(--text-2)}.health-row svg{width:13px;color:var(--muted)}.health-row strong{display:flex;align-items:center;gap:5px;font-size:9px}.health-row strong svg{color:var(--green)}.fault-item{display:flex;flex-direction:column;gap:4px;background:var(--orange-soft);color:var(--orange);padding:9px;border-radius:8px;margin-top:10px}.fault-item strong{font-size:9px}.fault-item span{font-size:8px}
@media(max-width:1300px){.vehicle-hero-grid{grid-template-columns:1.4fr 1fr}.quick-stats{grid-column:1/-1;display:grid;grid-template-columns:repeat(4,1fr);gap:15px}.device-boot{display:none}.quick-stat{border:0}}
@media(max-width:1000px){.detail-content-grid{grid-template-columns:1fr}.detail-content-grid>aside{display:grid;grid-template-columns:1fr 1fr}.profile-art{height:160px}}
@media(max-width:700px){.vehicle-hero-grid{grid-template-columns:1fr}.quick-stats{grid-template-columns:1fr 1fr}.detail-content-grid>aside{display:flex}.detail-tabs{height:auto;min-height:60px;gap:10px;flex-wrap:wrap;padding-block:10px}.detail-tabs>div:first-child{height:40px}.profile-meta{grid-template-columns:1fr 1fr}.profile-meta>div:last-child{display:none}}
</style>
