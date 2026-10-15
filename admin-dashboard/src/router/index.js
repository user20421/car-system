import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('../layout/AdminLayout.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
      { path: 'vehicle/:vin', name: 'VehicleDetail', component: () => import('../views/VehicleDetail.vue') },
      { path: 'vehicles', name: 'Vehicles', component: () => import('../views/Vehicles.vue'), meta: { admin: true } },
      { path: 'simulator', name: 'Simulator', component: () => import('../views/Simulator.vue'), meta: { admin: true } },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：未登录 → 登录页；非 admin 访问 admin 页面 → 大屏
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) return '/login'
  if (to.meta.admin && !auth.isAdmin) return '/dashboard'
  return true
})

export default router
