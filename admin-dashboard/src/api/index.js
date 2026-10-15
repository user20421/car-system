import axios from 'axios'
import { useAuthStore } from '../stores/auth'
import router from '../router'

// Axios 实例：统一 baseURL、超时、鉴权头注入、401 统一处理
const api = axios.create({
  baseURL: 'http://localhost:8080',
  timeout: 10000,
})

// 请求拦截器：自动携带 JWT
api.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

// 响应拦截器：统一错误处理
api.interceptors.response.use(
  (res) => res.data,
  (err) => {
    if (err.response?.status === 401) {
      const auth = useAuthStore()
      auth.logout()
      router.push('/login')
    }
    return Promise.reject(err.response?.data || { msg: err.message })
  },
)

export default api
