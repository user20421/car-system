<template>
  <div class="deadletters">
    <div class="head">
      <h2>☠️ 死信管理</h2>
      <span class="dim">消费重试耗尽的消息在此隔离，可人工重投（5s 自动刷新）</span>
    </div>

    <table v-if="list.length">
      <thead>
        <tr><th>ID</th><th>来源队列</th><th>Routing Key</th><th>失败原因</th><th>消息体</th><th>时间</th><th>操作</th></tr>
      </thead>
      <tbody>
        <tr v-for="d in list" :key="d.id">
          <td>{{ d.id }}</td>
          <td class="mono">{{ d.queue }}</td>
          <td class="mono">{{ d.routing_key }}</td>
          <td class="red">{{ d.reason }}</td>
          <td class="mono body" :title="d.body">{{ truncate(d.body) }}</td>
          <td class="dim">{{ fmtTime(d.created_at) }}</td>
          <td>
            <button class="btn-sm" :disabled="requeuing === d.id" @click="requeue(d)">
              {{ requeuing === d.id ? '重投中…' : '↩ 重投' }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无死信 ✓（说明消息链路健康）</div>

    <transition name="slide">
      <div v-if="toast" class="toast" :class="{ err: toastErr }">{{ toast }}</div>
    </transition>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import api from '../api'

const list = ref([])
const requeuing = ref(null)
const toast = ref('')
const toastErr = ref(false)
let timer = null
let toastTimer = null

async function load() {
  try {
    const res = await api.get('/api/v1/system/dead-letters')
    list.value = res.data || []
  } catch { /* 下轮重试 */ }
}

async function requeue(d) {
  requeuing.value = d.id
  try {
    await api.post(`/api/v1/system/dead-letters/${d.id}/requeue`)
    showToast(`死信 #${d.id} 已重新投递`)
    load()
  } catch (e) {
    showToast(e?.msg || '重投失败', true)
  } finally {
    requeuing.value = null
  }
}

function showToast(msg, isErr = false) {
  toast.value = msg
  toastErr.value = isErr
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value = '' }, 3000)
}
function truncate(s) { return s && s.length > 60 ? s.slice(0, 60) + '…' : s }
function fmtTime(t) { return t ? new Date(t).toLocaleString('zh-CN') : '-' }

onMounted(() => {
  load()
  timer = setInterval(load, 5000)
})
onUnmounted(() => { clearInterval(timer); clearTimeout(toastTimer) })
</script>

<style scoped>
h2 { font-size: 20px; }
.head { display: flex; align-items: baseline; gap: 14px; margin-bottom: 16px; }
.dim { color: var(--text-dim); font-size: 13px; }
table {
  width: 100%; border-collapse: collapse; background: var(--panel);
  border-radius: 10px; overflow: hidden; border: 1px solid var(--border); font-size: 13px;
}
th, td { padding: 10px 14px; text-align: left; }
th { background: #1c2128; color: var(--text-dim); font-weight: 600; }
tr + tr td { border-top: 1px solid var(--border); }
.mono { font-family: Consolas, monospace; font-size: 12px; }
.body { max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.red { color: var(--red); }
.btn-sm {
  background: #21262d; border: 1px solid var(--border); color: var(--text);
  border-radius: 6px; padding: 4px 10px; cursor: pointer; font-size: 12px;
}
.btn-sm:hover:not(:disabled) { border-color: var(--blue); color: var(--blue); }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
.empty {
  color: var(--text-dim); background: var(--panel); border: 1px solid var(--border);
  border-radius: 10px; padding: 40px; text-align: center;
}
.toast {
  position: fixed; bottom: 24px; right: 24px; background: #238636; color: #fff;
  padding: 10px 18px; border-radius: 8px; font-size: 13px;
}
.toast.err { background: #da3633; }
.slide-enter-active, .slide-leave-active { transition: all 0.3s; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(10px); }
</style>
