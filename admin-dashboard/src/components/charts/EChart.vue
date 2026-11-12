<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts/core'
import type { EChartsCoreOption } from 'echarts/core'
import { LineChart, PieChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { SVGRenderer } from 'echarts/renderers'
import { useUiStore } from '../../stores/ui'
echarts.use([LineChart, PieChart, GridComponent, LegendComponent, TooltipComponent, SVGRenderer])
const props = withDefaults(defineProps<{ option: EChartsCoreOption; height?: number; loading?: boolean }>(), { height: 280 })
const el = ref<HTMLDivElement>()
const ui = useUiStore()
let chart: ReturnType<typeof echarts.init> | undefined
let observer: ResizeObserver | undefined
function render() {
  if (!chart) return
  chart.setOption({ backgroundColor: 'transparent', textStyle: { fontFamily: 'Manrope Variable, sans-serif' }, ...props.option }, true)
  if (props.loading) chart.showLoading({ text: '', color: '#7668ed', maskColor: 'transparent' })
  else chart.hideLoading()
}
onMounted(() => {
  chart = echarts.init(el.value!, undefined, { renderer: 'svg' })
  render()
  observer = new ResizeObserver(() => chart?.resize())
  observer.observe(el.value!)
})
watch(() => [props.option, props.loading, ui.theme], render, { deep: true })
onBeforeUnmount(() => { observer?.disconnect(); chart?.dispose() })
</script>
<template><div ref="el" class="chart" :style="{ height: `${height}px` }" role="img" /></template>
<style scoped>.chart{width:100%;min-width:0}</style>
