<template>
  <div class="login-page">
    <div class="login-card">
      <h1>🚗 Car-System</h1>
      <p class="subtitle">车联网运营管理平台</p>
      <form @submit.prevent="onSubmit">
        <input v-model="username" placeholder="用户名" autocomplete="username" />
        <input v-model="password" type="password" placeholder="密码" autocomplete="current-password" />
        <p v-if="error" class="error">{{ error }}</p>
        <button type="submit" :disabled="loading">
          {{ loading ? '登录中...' : '登 录' }}
        </button>
      </form>
      <p class="hint">默认管理员：admin / admin123456</p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    router.push('/dashboard')
  } catch (e) {
    error.value = e.msg || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-card {
  width: 360px;
  padding: 40px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  text-align: center;
}
h1 { font-size: 26px; margin-bottom: 4px; }
.subtitle { color: var(--text-dim); font-size: 13px; margin-bottom: 28px; }
input {
  width: 100%;
  padding: 12px 14px;
  margin-bottom: 14px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text);
  font-size: 14px;
  outline: none;
}
input:focus { border-color: var(--blue); }
button {
  width: 100%;
  padding: 12px;
  background: var(--blue);
  border: none;
  border-radius: 8px;
  color: #0d1117;
  font-size: 15px;
  font-weight: bold;
  cursor: pointer;
}
button:disabled { opacity: 0.6; cursor: not-allowed; }
.error { color: var(--red); font-size: 13px; margin-bottom: 10px; }
.hint { color: var(--text-dim); font-size: 12px; margin-top: 18px; }
</style>
