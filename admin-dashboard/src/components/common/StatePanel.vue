<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { AlertCircle, CloudOff, Inbox, LockKeyhole, RefreshCw } from 'lucide-vue-next'
import type { ApiError } from '../../api/errors'
const props = withDefaults(defineProps<{ loading?: boolean; empty?: boolean; offline?: boolean; error?: ApiError | null; title?: string; description?: string; compact?: boolean }>(), { compact: false })
const emit = defineEmits<{ retry: [] }>()
const router = useRouter()
const kind = computed(() => props.offline ? 'offline' : props.error?.status === 401 ? 'unauthorized' : props.error?.status === 403 ? 'forbidden' : props.error ? 'error' : props.empty ? 'empty' : 'loading')
const heading = computed(() => props.title || ({ offline: '当前处于离线状态', unauthorized: '登录状态已失效', forbidden: '没有访问权限', error: '暂时无法加载数据', empty: '这里还没有数据', loading: '正在连接运营服务' }[kind.value]))
const detail = computed(() => props.description || ({ offline: '数据可能不是最新。恢复网络后可重新获取云端快照。', unauthorized: '请重新登录以继续访问车队数据。', forbidden: '当前账号的角色无法查看此资源，如需访问请联系管理员。', error: props.error?.message || '服务暂时不可用，请稍后重试。', empty: '调整筛选条件，或等待新的运营数据接入。', loading: '正在安全获取最新数据，请稍候…' }[kind.value]))
const icon = computed(() => kind.value === 'offline' ? CloudOff : kind.value === 'empty' ? Inbox : kind.value === 'forbidden' || kind.value === 'unauthorized' ? LockKeyhole : AlertCircle)
function action() {
  if (kind.value === 'unauthorized') void router.push({ name: 'login' })
  else if (kind.value === 'forbidden') void router.push({ name: 'overview' })
  else emit('retry')
}
</script>
<template>
  <div v-if="loading || empty || offline || error" class="state-panel" :class="{ compact }" role="status">
    <div v-if="loading && !error" class="spinner" style="margin-bottom:20px" />
    <div v-else class="state-icon"><component :is="icon" /></div>
    <h3>{{ heading }}</h3><p>{{ detail }}</p>
    <div v-if="error?.requestId && error.requestId !== 'unavailable'" class="request-id mono">REQUEST ID · {{ error.requestId }}</div>
    <button v-if="!loading" class="btn btn-secondary" @click="action"><RefreshCw v-if="kind !== 'unauthorized'" :size="14" />{{ kind === 'unauthorized' ? '重新登录' : kind === 'forbidden' ? '返回总览' : '重新加载' }}</button>
  </div>
  <slot v-else />
</template>
<style scoped>.state-panel.compact{min-height:160px;padding:24px}.state-panel.compact .state-icon{width:44px;height:44px;margin-bottom:12px}</style>
