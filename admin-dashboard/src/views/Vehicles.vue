<template>
  <div class="vehicles">
    <div class="head">
      <h2>车辆管理</h2>
      <input v-model.trim="keyword" class="search" placeholder="🔍 搜索 VIN / 型号 / 车主" />
    </div>

    <!-- 登记 / 编辑表单 -->
    <form class="edit-panel" @submit.prevent="submit">
      <b>{{ editing ? '编辑车辆' : '登记新车' }}</b>
      <input v-model.trim="form.vin" placeholder="VIN（17 位）" maxlength="17"
             :disabled="!!editing" required style="text-transform: uppercase" />
      <input v-model.trim="form.model" placeholder="型号" required />
      <select v-model.number="form.owner_id" required>
        <option :value="0" disabled>选择车主</option>
        <option v-for="u in owners" :key="u.id" :value="u.id">{{ u.username }}（id:{{ u.id }}）</option>
      </select>
      <button class="btn" type="submit">{{ editing ? '保存' : '+ 登记' }}</button>
      <button v-if="editing" class="btn-sm" type="button" @click="cancelEdit">取消</button>
      <span v-if="formMsg" class="msg" :class="{ err: formErr }">{{ formMsg }}</span>
    </form>

    <table>
      <thead>
        <tr>
          <th>ID</th><th>VIN</th><th>型号</th><th>车主</th><th>状态</th><th>登记时间</th><th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="v in filtered" :key="v.id">
          <td>{{ v.id }}</td>
          <td class="mono">{{ v.vin }}</td>
          <td>{{ v.model }}</td>
          <td>{{ v.owner?.username || '-' }}</td>
          <td>
            <span class="status" :class="v.status">{{ v.status === 'online' ? '在线' : '离线' }}</span>
          </td>
          <td>{{ formatTime(v.created_at) }}</td>
          <td>
            <button class="btn-sm" @click="startEdit(v)">编辑</button>
            <button class="btn-sm danger" @click="remove(v)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="!filtered.length" class="empty">{{ vehicles.length ? '无匹配结果' : '暂无车辆数据' }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import api from '../api'

const vehicles = ref([])
const owners = ref([])
const keyword = ref('')
const editing = ref(null) // 正在编辑的车辆 vin
const form = ref({ vin: '', model: '', owner_id: 0 })
const formMsg = ref('')
const formErr = ref(false)

const filtered = computed(() => {
  const k = keyword.value.toLowerCase()
  if (!k) return vehicles.value
  return vehicles.value.filter((v) =>
    v.vin.toLowerCase().includes(k) ||
    (v.model || '').toLowerCase().includes(k) ||
    (v.owner?.username || '').toLowerCase().includes(k),
  )
})

async function load() {
  try {
    const [v, u] = await Promise.all([
      api.get('/api/v1/vehicle'),
      api.get('/api/v1/users'),
    ])
    vehicles.value = v.data || []
    owners.value = (u.data || []).filter((x) => x.role === 'owner' || x.role === 'admin')
  } catch (e) {
    console.error(e)
  }
}

async function submit() {
  formMsg.value = ''
  try {
    const payload = { vin: form.value.vin.toUpperCase(), model: form.value.model, owner_id: form.value.owner_id }
    if (editing.value) {
      await api.put(`/api/v1/vehicle/${editing.value}`, payload)
      showMsg('已保存')
    } else {
      await api.post('/api/v1/vehicle', payload)
      showMsg('登记成功')
    }
    cancelEdit()
    load()
  } catch (e) {
    formErr.value = true
    formMsg.value = e?.msg || '操作失败'
  }
}

function startEdit(v) {
  editing.value = v.vin
  form.value = { vin: v.vin, model: v.model, owner_id: v.owner_id || v.owner?.id || 0 }
  formMsg.value = ''
}
function cancelEdit() {
  editing.value = null
  form.value = { vin: '', model: '', owner_id: 0 }
}
async function remove(v) {
  if (!confirm(`确认删除车辆 ${v.vin}？`)) return
  try {
    await api.delete(`/api/v1/vehicle/${v.vin}`)
    if (editing.value === v.vin) cancelEdit() // 删除正在编辑的车辆时退出编辑态
    formErr.value = false
    formMsg.value = `已删除 ${v.vin}`
    load()
  } catch (e) {
    formErr.value = true
    formMsg.value = e?.msg || '删除失败'
  }
}
function showMsg(m) { formErr.value = false; formMsg.value = m }
function formatTime(t) { return new Date(t).toLocaleString('zh-CN') }

onMounted(load)
</script>

<style scoped>
h2 { font-size: 20px; }
.head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.search {
  background: var(--panel); border: 1px solid var(--border); border-radius: 8px;
  color: var(--text); padding: 8px 14px; width: 260px; font-size: 13px;
}
.edit-panel {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  background: var(--panel); border: 1px solid var(--border); border-radius: 10px;
  padding: 12px 14px; margin-bottom: 14px; font-size: 13px;
}
.edit-panel input, .edit-panel select {
  background: var(--bg); border: 1px solid var(--border); border-radius: 6px;
  color: var(--text); padding: 6px 10px; font-size: 13px;
}
.edit-panel input:first-of-type { width: 190px; font-family: Consolas, monospace; }
.btn { background: #1f6feb; color: #fff; border: none; border-radius: 6px; padding: 6px 14px; cursor: pointer; font-size: 13px; }
.btn-sm { background: #21262d; border: 1px solid var(--border); color: var(--text); border-radius: 6px; padding: 4px 10px; cursor: pointer; font-size: 12px; margin-right: 6px; }
.btn-sm.danger:hover { border-color: var(--red); color: var(--red); }
.msg { font-size: 13px; color: var(--green); }
.msg.err { color: var(--red); }
table {
  width: 100%; border-collapse: collapse; background: var(--panel);
  border-radius: 10px; overflow: hidden; border: 1px solid var(--border);
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
