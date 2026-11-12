<script setup lang="ts">
import { computed, ref } from 'vue'
import { AirVent, Flame, Lock, LockOpen, Snowflake, Volume2 } from 'lucide-vue-next'
import AppModal from '../common/AppModal.vue'
import { canControlVehicle } from '../../utils/permissions'
import { commandLabel } from '../../utils/format'
import type { CommandType, Telemetry, User, Vehicle } from '../../types/api'
import { useCommandStore } from '../../stores/commands'
import { useUiStore } from '../../stores/ui'
import { useOnline } from '../../composables/useOnline'
const props = defineProps<{ vehicle: Vehicle; telemetry?: Telemetry; user: User | null }>()
const store = useCommandStore(), ui = useUiStore()
const { online } = useOnline()
const confirmOpen = ref(false), confirmed = ref(false), target = ref<CommandType>('lock'), busy = ref(false)
const canControl = computed(() => canControlVehicle(props.user, props.vehicle))
const actions = computed(() => [
  { type: (props.telemetry?.locked ? 'unlock' : 'lock') as CommandType, label: props.telemetry?.locked ? '远程解锁' : '车辆闭锁', icon: props.telemetry?.locked ? LockOpen : Lock, danger: props.telemetry?.locked },
  { type: 'hvac_on' as CommandType, label: '开启空调', icon: AirVent },
  { type: (props.telemetry?.defrost ? 'defrost_off' : 'defrost_on') as CommandType, label: props.telemetry?.defrost ? '关闭除霜' : '前挡除霜', icon: Snowflake },
  { type: 'honk_flash' as CommandType, label: '鸣笛闪灯', icon: Volume2 },
])
function request(type: CommandType) { target.value = type; confirmed.value = false; confirmOpen.value = true }
async function execute() {
  busy.value = true
  const task = store.send(props.vehicle.vin, target.value, target.value === 'hvac_on' ? { target_temp_c: String(props.telemetry?.hvac?.target_temp_c || 22) } : {})
  confirmOpen.value = false
  try {
    const result = await task
    if (!result) return
    if (result.status === 'succeeded') ui.toast('车辆已确认执行成功', 'success', `${commandLabel(result.type)} · 服务端终态`)
    else if (result.status === 'failed') ui.toast('车辆执行失败', 'error', result.error?.message || '请查看指令时间线')
    else if (result.status === 'expired') ui.toast('指令已过期', 'error', '车辆未在有效期内执行，不会伪造成功状态')
  } catch (error) { ui.toast('指令未能受理', 'error', error instanceof Error ? error.message : '请稍后重试') }
  finally { busy.value = false }
}
</script>
<template>
  <div>
    <div v-if="!canControl" class="control-notice"><Lock />当前车辆角色为 {{ vehicle.access_role }}，服务端不会接受控车请求。</div>
    <div v-if="!online" class="control-notice"><Flame />离线时不可创建新指令。</div>
    <div class="control-grid"><button v-for="action in actions" :key="action.type" :disabled="!canControl || !online || busy || store.sending[`${vehicle.vin}:${action.type}`]" @click="request(action.type)"><span :class="{ danger: action.danger }"><component :is="action.icon" /></span><strong>{{ action.label }}</strong><small>{{ store.sending[`${vehicle.vin}:${action.type}`] ? '等待车辆状态…' : '二次确认后执行' }}</small></button></div>
    <AppModal :open="confirmOpen" :title="`确认${commandLabel(target)}？`" eyebrow="安全操作确认" :busy="busy" @close="confirmOpen=false">
      <div class="confirm-vehicle"><strong>{{ vehicle.display_name }}</strong><span class="mono">{{ vehicle.vin }}</span></div><p>此操作将通过加密通道下发至真实车辆。指令有效期为 60 秒，车辆执行结果以服务端返回的终态为准。</p><label class="confirm-check"><input v-model="confirmed" type="checkbox">我已核对车辆与操作，确认继续</label>
      <template #footer><button class="btn btn-secondary" @click="confirmOpen=false">取消</button><button class="btn btn-primary" :disabled="!confirmed" @click="execute"><Lock />确认并发送</button></template>
    </AppModal>
  </div>
</template>
<style scoped>
.control-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:9px}.control-grid button{min-height:88px;border:1px solid var(--line);border-radius:11px;background:var(--surface-solid);display:flex;flex-direction:column;align-items:flex-start;padding:12px;cursor:pointer;transition:all .18s;text-align:left}.control-grid button:hover:not(:disabled){border-color:var(--primary);transform:translateY(-1px);box-shadow:var(--shadow-sm)}.control-grid button:disabled{opacity:.45;cursor:not-allowed}.control-grid button>span{width:27px;height:27px;border-radius:8px;background:var(--primary-soft);color:var(--primary);display:grid;place-items:center;margin-bottom:8px}.control-grid button>span.danger{background:var(--red-soft);color:var(--red)}.control-grid svg{width:14px}.control-grid strong{font-size:10px}.control-grid small{font-size:8px;color:var(--muted);margin-top:3px}.control-notice{display:flex;gap:7px;align-items:center;background:var(--orange-soft);color:var(--orange);padding:9px 11px;border-radius:8px;font-size:9px;margin-bottom:10px}.control-notice svg{width:13px}.confirm-vehicle{display:flex;flex-direction:column;background:var(--surface-2);border-radius:10px;padding:12px;margin-bottom:12px;gap:5px}.confirm-vehicle strong{font-size:13px;color:var(--text)}.confirm-vehicle span{font-size:9px;color:var(--muted)}.confirm-check{display:flex;align-items:center;gap:8px;margin-top:15px;font-size:11px;color:var(--text);cursor:pointer}.confirm-check input{accent-color:var(--primary)}
</style>
