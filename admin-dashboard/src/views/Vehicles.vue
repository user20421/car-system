<template>
  <div class="vehicles">
    <h2>车辆管理</h2>
    <table>
      <thead>
        <tr>
          <th>ID</th><th>VIN</th><th>型号</th><th>车主</th><th>状态</th><th>登记时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="v in vehicles" :key="v.id">
          <td>{{ v.id }}</td>
          <td class="mono">{{ v.vin }}</td>
          <td>{{ v.model }}</td>
          <td>{{ v.owner?.username || '-' }}</td>
          <td>
            <span class="status" :class="v.status">{{ v.status === 'online' ? '在线' : '离线' }}</span>
          </td>
          <td>{{ formatTime(v.created_at) }}</td>
        </tr>
      </tbody>
    </table>
    <p v-if="!vehicles.length" class="empty">暂无车辆数据</p>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../api'

const vehicles = ref([])

function formatTime(t) {
  return new Date(t).toLocaleString('zh-CN')
}

onMounted(async () => {
  try {
    const res = await api.get('/api/v1/vehicle')
    vehicles.value = res.data || []
  } catch (e) {
    console.error(e)
  }
})
</script>

<style scoped>
h2 { margin-bottom: 16px; font-size: 20px; }
table {
  width: 100%;
  border-collapse: collapse;
  background: var(--panel);
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--border);
}
th, td { padding: 12px 16px; text-align: left; font-size: 14px; }
th { background: #1c2128; color: var(--text-dim); font-weight: 600; }
tr + tr td { border-top: 1px solid var(--border); }
.mono { font-family: Consolas, monospace; }
.status { padding: 3px 10px; border-radius: 12px; font-size: 12px; }
.status.online { background: #3fb95022; color: var(--green); }
.status.offline { background: #8b949e22; color: var(--text-dim); }
.empty { color: var(--text-dim); margin-top: 20px; text-align: center; }
</style>
