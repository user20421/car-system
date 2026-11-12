<script setup lang="ts">
import { computed } from 'vue'
import { BatteryCharging, Gauge, Thermometer } from 'lucide-vue-next'
import type { Telemetry } from '../../types/api'
const props = defineProps<{ telemetry: Telemetry }>()
const speedOffset = computed(() => 314 - Math.min(props.telemetry.speed_kph, 180) / 180 * 251)
const batteryOffset = computed(() => 314 - props.telemetry.battery_percent / 100 * 251)
</script>
<template>
  <div class="instruments">
    <div class="instrument speed"><svg viewBox="0 0 130 130"><circle cx="65" cy="65" r="50" class="dial-track"/><circle cx="65" cy="65" r="50" class="dial-value" :style="{ strokeDashoffset:speedOffset }"/><g class="ticks"><path v-for="n in 9" :key="n" d="M65 8v5" :transform="`rotate(${-125+(n-1)*31.25} 65 65)`"/></g></svg><div class="instrument-value"><Gauge /><strong class="numeric">{{ Math.round(telemetry.speed_kph) }}</strong><span>km/h</span><small>实时车速</small></div></div>
    <div class="instrument battery"><svg viewBox="0 0 130 130"><circle cx="65" cy="65" r="50" class="dial-track"/><circle cx="65" cy="65" r="50" class="dial-value" :style="{ strokeDashoffset:batteryOffset }"/></svg><div class="instrument-value"><BatteryCharging /><strong class="numeric">{{ telemetry.battery_percent.toFixed(0) }}<em>%</em></strong><span>{{ telemetry.range_km }} km</span><small>预计续航</small></div></div>
    <div class="climate"><Thermometer /><div><span>座舱温度</span><strong class="numeric">{{ telemetry.cabin_temp_c?.toFixed(1) || '—' }}<small>°C</small></strong></div><i/><div><span>空调设定</span><strong class="numeric">{{ telemetry.hvac?.target_temp_c || '—' }}<small>°C</small></strong></div></div>
  </div>
</template>
<style scoped>
.instruments{display:grid;grid-template-columns:1fr 1fr;gap:4px;position:relative}.instrument{position:relative;aspect-ratio:1.15;display:grid;place-items:center}.instrument svg{width:100%;height:100%;transform:rotate(125deg)}.dial-track,.dial-value{fill:none;stroke-width:5;stroke-dasharray:251 314;stroke-linecap:round}.dial-track{stroke:var(--surface-2)}.dial-value{stroke:var(--primary);transition:stroke-dashoffset .8s}.battery .dial-value{stroke:var(--cyan)}.ticks{fill:none;stroke:var(--line-strong);stroke-width:1}.instrument-value{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;padding-top:8px}.instrument-value>svg{width:13px;height:13px;color:var(--muted);margin-bottom:4px}.instrument-value strong{font-size:29px;line-height:1;letter-spacing:-.05em}.instrument-value strong em{font-size:13px;font-style:normal;color:var(--muted);margin-left:2px}.instrument-value>span{font-size:9px;color:var(--text-2);margin-top:5px}.instrument-value>small{font-size:8px;color:var(--muted);margin-top:8px}.climate{grid-column:1/-1;border-top:1px solid var(--line);padding:14px 10px 0;display:flex;align-items:center;gap:13px}.climate>svg{width:17px;color:var(--primary)}.climate>div{display:flex;flex-direction:column;gap:4px;flex:1}.climate span{font-size:8px;color:var(--muted)}.climate strong{font-size:17px}.climate small{font-size:9px;color:var(--muted);margin-left:2px}.climate i{width:1px;height:26px;background:var(--line)}
</style>
