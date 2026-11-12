<script setup lang="ts">
import { computed } from 'vue'
import { ArrowDownRight, ArrowUpRight, Minus } from 'lucide-vue-next'
const props = withDefaults(defineProps<{ label: string; value: string | number; unit?: string; trend?: number; hint?: string; tone?: 'violet' | 'cyan' | 'orange' | 'green'; points?: number[] }>(), { tone: 'violet', points: () => [] })
const line = computed(() => {
  if (!props.points.length) return ''
  const min = Math.min(...props.points), max = Math.max(...props.points), range = max - min || 1
  return props.points.map((p, i) => `${i * 100 / (props.points.length - 1)},${30 - (p - min) / range * 24}`).join(' ')
})
</script>
<template>
  <article class="metric card" :class="`metric-${tone}`">
    <div class="metric-top"><span>{{ label }}</span><span class="metric-icon"><slot name="icon" /></span></div>
    <div class="metric-value numeric">{{ value }}<small>{{ unit }}</small></div>
    <div class="metric-bottom">
      <span v-if="trend !== undefined" class="trend" :class="{ down: trend < 0, flat: trend === 0 }"><ArrowUpRight v-if="trend > 0" /><ArrowDownRight v-else-if="trend < 0" /><Minus v-else />{{ Math.abs(trend) }}%</span>
      <span class="metric-hint">{{ hint }}</span>
      <svg v-if="points.length" class="spark" viewBox="0 0 100 34" preserveAspectRatio="none"><polyline :points="line" fill="none" stroke="currentColor" stroke-width="2" vector-effect="non-scaling-stroke"/><polyline :points="`0,34 ${line} 100,34`" fill="currentColor" opacity=".06" stroke="none"/></svg>
    </div>
  </article>
</template>
<style scoped>
.metric{padding:20px 21px 17px;min-height:146px;position:relative;overflow:hidden}.metric::before{content:'';position:absolute;right:-30px;top:-55px;width:130px;height:130px;border-radius:50%;background:var(--primary);opacity:.035}.metric-top{display:flex;justify-content:space-between;align-items:center;color:var(--text-2);font-size:11px;font-weight:600}.metric-icon{width:30px;height:30px;border-radius:9px;background:var(--primary-soft);color:var(--primary);display:grid;place-items:center}.metric-icon :deep(svg){width:15px;height:15px}.metric-value{font-size:31px;letter-spacing:-.055em;font-weight:720;margin-top:12px;line-height:1}.metric-value small{font-size:12px;color:var(--muted);font-weight:550;letter-spacing:0;margin-left:6px}.metric-bottom{display:flex;align-items:center;margin-top:18px;gap:8px;position:relative}.trend{display:flex;align-items:center;color:var(--green);font-size:10px;font-weight:700}.trend svg{width:12px;height:12px}.trend.down{color:var(--red)}.trend.flat{color:var(--muted)}.metric-hint{font-size:10px;color:var(--muted)}.spark{position:absolute;right:0;bottom:-2px;width:82px;height:30px;color:var(--primary)}.metric-cyan .metric-icon{background:var(--cyan-soft);color:var(--cyan)}.metric-cyan .spark{color:var(--cyan)}.metric-orange .metric-icon{background:var(--orange-soft);color:var(--orange)}.metric-orange .spark{color:var(--orange)}.metric-green .metric-icon{background:var(--green-soft);color:var(--green)}.metric-green .spark{color:var(--green)}
</style>
