import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { canAccessRoute, type Permission } from '../utils/permissions'

declare module 'vue-router' {
  interface RouteMeta { title?: string; permission?: Permission; public?: boolean }
}

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { public: true, title: '登录' } },
    {
      path: '/', component: () => import('../components/layout/AppShell.vue'),
      children: [
        { path: '', redirect: '/overview' },
        { path: 'overview', name: 'overview', component: () => import('../views/OverviewView.vue'), meta: { title: '车队总览', permission: 'fleet:view' } },
        { path: 'fleet', name: 'fleet', component: () => import('../views/FleetView.vue'), meta: { title: '实时车队', permission: 'fleet:view' } },
        { path: 'vehicles/:vin', name: 'vehicle', component: () => import('../views/VehicleDetailView.vue'), meta: { title: '车辆详情', permission: 'fleet:view' } },
        { path: 'alerts', name: 'alerts', component: () => import('../views/AlertsView.vue'), meta: { title: '告警中心', permission: 'fleet:view' } },
        { path: 'access', name: 'access', component: () => import('../views/AccessView.vue'), meta: { title: '用户与资产', permission: 'users:manage' } },
        { path: 'audit', name: 'audit', component: () => import('../views/AuditView.vue'), meta: { title: '审计日志', permission: 'audit:view' } },
        { path: 'settings', name: 'settings', component: () => import('../views/SettingsView.vue'), meta: { title: '平台设置', permission: 'settings:manage' } },
        { path: '403', name: 'forbidden', component: () => import('../views/ForbiddenView.vue'), meta: { title: '无权访问' } },
      ],
    },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('../views/NotFoundView.vue'), meta: { public: true, title: '页面未找到' } },
  ],
})

router.beforeEach(async to => {
  const auth = useAuthStore()
  await auth.initialize()
  document.title = `${to.meta.title || '运营平台'} · AURORA`
  if (to.meta.public) {
    if (to.name === 'login' && auth.isAuthenticated) return { name: 'overview' }
    return true
  }
  if (!auth.isAuthenticated) return { name: 'login', query: { redirect: to.fullPath } }
  if (!canAccessRoute(auth.user, to.meta.permission)) return { name: 'forbidden', query: { from: to.fullPath } }
  return true
})
