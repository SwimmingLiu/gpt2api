import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import BasicLayout from '@/layouts/BasicLayout.vue'
import BlankLayout from '@/layouts/BlankLayout.vue'
import { useUserStore } from '@/stores/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: BlankLayout,
    meta: { public: true },
    children: [
      { path: '', redirect: '/admin/accounts' },
      { path: 'login', component: () => import('@/views/auth/Login.vue'), meta: { public: true, title: '登录' } },
    ],
  },
  {
    path: '/admin',
    component: BasicLayout,
    redirect: '/admin/accounts',
    children: [
      { path: 'accounts', component: () => import('@/views/admin/Accounts.vue'),
        meta: { title: 'GPT账号', perm: 'account:read' } },
      { path: 'proxies', component: () => import('@/views/admin/Proxies.vue'),
        meta: { title: '代理管理', perm: 'proxy:read' } },
      { path: 'account-pools', component: () => import('@/views/admin/AccountPools.vue'),
        meta: { title: '账号池', perm: 'account:read' } },
      { path: 'account-pool-routes', component: () => import('@/views/admin/AccountPoolRoutes.vue'),
        meta: { title: '池路由', perm: ['account:read', 'account:write'] } },
      { path: 'settings', component: () => import('@/views/admin/Settings.vue'),
        meta: { title: '系统设置', perm: 'system:setting' } },
    ],
  },
  {
    path: '/403',
    component: () => import('@/views/Error403.vue'),
    meta: { public: true, title: '403' },
  },
  {
    path: '/:pathMatch(.*)*',
    component: () => import('@/views/Error404.vue'),
    meta: { public: true, title: '404' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const store = useUserStore()
  const title = (to.meta.title as string) || 'GPT2API 控制台'
  document.title = title

  if (to.meta.public) return true

  if (!store.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (!store.user || store.permissions.length === 0) {
    try {
      await store.fetchMe()
    } catch {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }

  const perm = to.meta.perm as string | string[] | undefined
  if (perm && !store.hasPerm(perm)) {
    return { path: '/403' }
  }
  return true
})

export default router
