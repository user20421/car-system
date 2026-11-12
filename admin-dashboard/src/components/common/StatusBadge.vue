<script setup lang="ts">
import { computed } from 'vue'
import { statusLabel } from '../../utils/format'
const props = defineProps<{ status: string; label?: string; pulse?: boolean }>()
const tone = computed(() => {
  if (['online','succeeded','active','acknowledged','success'].includes(props.status)) return 'success'
  if (['pending','sent','received','info','provisioning'].includes(props.status)) return 'info'
  if (['warning','open','invited'].includes(props.status)) return 'warning'
  if (['critical','failed','denied','expired','suspended'].includes(props.status)) return 'danger'
  return 'neutral'
})
</script>
<template><span class="badge" :class="`badge-${tone}`"><i class="dot" :style="pulse ? 'animation:pulse 1.5s infinite' : ''" />{{ label || statusLabel(status) }}</span></template>
