<script setup lang="ts">
import { computed } from 'vue'
import { Check, CircleDashed, Clock3, Radio, X } from 'lucide-vue-next'
import { useCommandStore } from '../../stores/commands'
import { commandLabel, formatTime } from '../../utils/format'
const props = defineProps<{ vin: string }>()
const store = useCommandStore()
const items = computed(() => store.history.filter(c => c.vin === props.vin))
const iconFor = (status: string) => status === 'succeeded' ? Check : status === 'failed' || status === 'expired' ? X : status === 'pending' ? Clock3 : status === 'sent' ? Radio : CircleDashed
</script>
<template>
  <div v-if="items.length" class="timeline"><div v-for="command in items" :key="command.command_id" class="timeline-item" :class="command.status"><span class="timeline-icon"><component :is="iconFor(command.status)" /></span><div class="timeline-content"><div class="row-between"><strong>{{ commandLabel(command.type) }}</strong><time>{{ formatTime(command.updated_at_ms) }}</time></div><p><StatusBadge :status="command.status" :pulse="!['succeeded','failed','expired'].includes(command.status)" /><span class="mono">{{ command.command_id.slice(0,8) }}</span></p><small v-if="command.error">{{ command.error.message }} · {{ command.error.code }}</small></div></div></div>
  <div v-else class="timeline-empty"><Radio /><strong>暂无控车指令</strong><span>发起指令后，可在此跟踪服务端投递与车辆执行状态。</span></div>
</template>
<script lang="ts">
import StatusBadge from '../common/StatusBadge.vue'
export default { components: { StatusBadge } }
</script>
<style scoped>
.timeline{display:flex;flex-direction:column}.timeline-item{display:flex;gap:12px;position:relative;padding-bottom:19px}.timeline-item:not(:last-child)::before{content:'';position:absolute;left:13px;top:29px;bottom:2px;width:1px;background:var(--line)}.timeline-icon{width:27px;height:27px;border-radius:50%;display:grid;place-items:center;background:var(--primary-soft);color:var(--primary);flex:none;z-index:1}.timeline-icon svg{width:13px;height:13px}.succeeded .timeline-icon{background:var(--green-soft);color:var(--green)}.failed .timeline-icon,.expired .timeline-icon{background:var(--red-soft);color:var(--red)}.timeline-content{flex:1;min-width:0}.timeline-content strong{font-size:11px}.timeline-content time{font-size:9px;color:var(--muted)}.timeline-content p{display:flex;align-items:center;gap:8px;margin-top:7px}.timeline-content p>span:last-child{font-size:8px;color:var(--muted)}.timeline-content small{display:block;color:var(--red);font-size:9px;margin-top:7px}.timeline-empty{min-height:170px;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;color:var(--muted);padding:20px}.timeline-empty>svg{width:23px;margin-bottom:12px}.timeline-empty strong{font-size:11px;color:var(--text-2)}.timeline-empty span{font-size:9px;line-height:1.6;max-width:240px;margin-top:6px}
</style>
