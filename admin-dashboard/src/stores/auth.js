import { defineStore } from 'pinia'
import api from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    username: localStorage.getItem('username') || '',
    role: localStorage.getItem('role') || '',
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
    isAdmin: (state) => state.role === 'admin',
  },
  actions: {
    async login(username, password) {
      const res = await api.post('/api/v1/auth/login', { username, password })
      this.token = res.data.token
      this.role = res.data.role
      this.username = username
      localStorage.setItem('token', this.token)
      localStorage.setItem('role', this.role)
      localStorage.setItem('username', this.username)
    },
    logout() {
      this.token = ''
      this.username = ''
      this.role = ''
      localStorage.clear()
    },
  },
})
