import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as authApi from '@/api/auth'
import { TOKEN_KEY, REFRESH_KEY, safeRemoveStorage, safeSetStorage } from '@/api/http'

export const useUserStore = defineStore(
  'user',
  () => {
    const mockMode = import.meta.env.VITE_DEV_MOCK === '1'
    const accessToken = ref<string>(mockMode ? '' : (localStorage.getItem(TOKEN_KEY) || ''))
    const refreshToken = ref<string>(mockMode ? '' : (localStorage.getItem(REFRESH_KEY) || ''))
    const user = ref<authApi.UserInfo | null>(null)
    const permissions = ref<string[]>([])
    const role = ref<string>('')
    const menu = ref<authApi.MenuItem[]>([])

    const isLoggedIn = computed(() => !!accessToken.value)
    const isAdmin = computed(() => role.value === 'admin')

    function setTokens(tp: authApi.TokenPair) {
      accessToken.value = tp.access_token
      refreshToken.value = tp.refresh_token
      safeSetStorage(TOKEN_KEY, tp.access_token)
      safeSetStorage(REFRESH_KEY, tp.refresh_token)
    }

    async function login(email: string, password: string) {
      const data = await authApi.login({ email, password })
      setTokens(data.token)
      user.value = data.user
      // 登录后拉一次 me(得到 permissions),顺便拉 menu
      await fetchMe()
    }

    async function fetchMe() {
      const data = await authApi.getMe()
      user.value = data.user
      role.value = data.role
      permissions.value = data.permissions || []
    }

    async function fetchMenu() {
      const data = await authApi.getMenu()
      menu.value = data.menu || []
      role.value = data.role
      permissions.value = data.permissions || []
    }

    function hasPerm(perm: string | string[]): boolean {
      if (!perm) return true
      const arr = Array.isArray(perm) ? perm : [perm]
      if (arr.length === 0) return true
      return arr.some((p) => permissions.value.includes(p))
    }

    function clear() {
      accessToken.value = ''
      refreshToken.value = ''
      user.value = null
      permissions.value = []
      role.value = ''
      menu.value = []
      safeRemoveStorage(TOKEN_KEY)
      safeRemoveStorage(REFRESH_KEY)
    }

    async function logout() {
      clear()
    }

    return {
      accessToken,
      refreshToken,
      user,
      role,
      permissions,
      menu,
      isLoggedIn,
      isAdmin,
      setTokens,
      login,
      fetchMe,
      fetchMenu,
      hasPerm,
      clear,
      logout,
    }
  },
  {
    // 持久化 token 和 user,避免刷新后闪屏
    persist: import.meta.env.VITE_DEV_MOCK === '1' ? false : {
      key: 'gpt2api.user-store',
      paths: ['accessToken', 'refreshToken', 'user', 'role', 'permissions'],
    },
  },
)
