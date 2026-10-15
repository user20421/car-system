<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="logo">🚗 Car-System</div>
      <nav>
        <router-link to="/dashboard" class="nav-item">📊 监控大屏</router-link>
        <router-link v-if="auth.isAdmin" to="/vehicles" class="nav-item">🚙 车辆管理</router-link>
        <router-link v-if="auth.isAdmin" to="/simulator" class="nav-item">🎛️ 模拟器控制台</router-link>
      </nav>
      <div class="user">
        <span>{{ auth.username }}（{{ auth.role }}）</span>
        <button @click="onLogout">退出</button>
      </div>
    </aside>
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
function onLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout { display: flex; height: 100vh; }
.sidebar {
  width: 200px;
  background: var(--panel);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  padding: 20px 12px;
}
.logo { font-size: 18px; font-weight: bold; margin-bottom: 32px; padding: 0 8px; }
nav { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.nav-item {
  padding: 10px 12px;
  border-radius: 8px;
  color: var(--text-dim);
  font-size: 14px;
}
.nav-item:hover { background: #21262d; color: var(--text); }
.nav-item.router-link-active { background: #1f6feb33; color: var(--blue); }
.user {
  border-top: 1px solid var(--border);
  padding-top: 14px;
  font-size: 13px;
  color: var(--text-dim);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.user button {
  background: none;
  border: 1px solid var(--border);
  color: var(--text-dim);
  border-radius: 6px;
  padding: 4px 10px;
  cursor: pointer;
  font-size: 12px;
}
.user button:hover { color: var(--red); border-color: var(--red); }
.content { flex: 1; overflow: auto; padding: 20px; }
</style>
