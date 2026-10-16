<template>
  <div class="users">
    <h2>用户管理</h2>

    <!-- 创建用户 -->
    <form class="create-form" @submit.prevent="create">
      <b>创建用户</b>
      <input v-model.trim="form.username" placeholder="用户名（≥3位）" required minlength="3" />
      <input v-model="form.password" type="password" placeholder="密码（≥6位）" required minlength="6" />
      <select v-model="form.role">
        <option value="owner">车主（owner）</option>
        <option value="admin">管理员（admin）</option>
      </select>
      <button class="btn" type="submit">+ 创建</button>
      <span v-if="msg" class="msg" :class="{ err: msgErr }">{{ msg }}</span>
    </form>

    <table>
      <thead>
        <tr><th>ID</th><th>用户名</th><th>角色</th><th>名下车辆</th><th>注册时间</th></tr>
      </thead>
      <tbody>
        <tr v-for="u in users" :key="u.id">
          <td>{{ u.id }}</td>
          <td>{{ u.username }}</td>
          <td><span class="role" :class="u.role">{{ u.role === 'admin' ? '管理员' : '车主' }}</span></td>
          <td>{{ u.vehicle_count }}</td>
          <td class="dim">{{ fmtTime(u.created_at) }}</td>
        </tr>
      </tbody>
    </table>
    <p v-if="!users.length" class="empty">暂无用户</p>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../api'

const users = ref([])
const form = ref({ username: '', password: '', role: 'owner' })
const msg = ref('')
const msgErr = ref(false)

async function load() {
  try {
    const res = await api.get('/api/v1/users')
    users.value = res.data || []
  } catch (e) {
    console.error(e)
  }
}

async function create() {
  msg.value = ''
  try {
    await api.post('/api/v1/users', form.value)
    msgErr.value = false
    msg.value = `用户 ${form.value.username} 创建成功`
    form.value = { username: '', password: '', role: 'owner' }
    load()
  } catch (e) {
    msgErr.value = true
    msg.value = e?.msg || '创建失败'
  }
}

function fmtTime(t) { return new Date(t).toLocaleString('zh-CN') }

onMounted(load)
</script>

<style scoped>
h2 { margin-bottom: 16px; font-size: 20px; }
.create-form {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  background: var(--panel); border: 1px solid var(--border); border-radius: 10px;
  padding: 12px 14px; margin-bottom: 14px; font-size: 13px;
}
.create-form input, .create-form select {
  background: var(--bg); border: 1px solid var(--border); border-radius: 6px;
  color: var(--text); padding: 6px 10px; font-size: 13px;
}
.btn { background: #1f6feb; color: #fff; border: none; border-radius: 6px; padding: 6px 14px; cursor: pointer; font-size: 13px; }
.msg { font-size: 13px; color: var(--green); }
.msg.err { color: var(--red); }
table {
  width: 100%; border-collapse: collapse; background: var(--panel);
  border-radius: 10px; overflow: hidden; border: 1px solid var(--border); font-size: 14px;
}
th, td { padding: 12px 16px; text-align: left; }
th { background: #1c2128; color: var(--text-dim); font-weight: 600; }
tr + tr td { border-top: 1px solid var(--border); }
.role { padding: 3px 10px; border-radius: 12px; font-size: 12px; }
.role.admin { background: #d2992233; color: var(--yellow); }
.role.owner { background: #1f6feb33; color: var(--blue); }
.dim { color: var(--text-dim); }
.empty { color: var(--text-dim); margin-top: 20px; text-align: center; }
</style>
