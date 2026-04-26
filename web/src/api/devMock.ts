import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios'

const mockPools = [
  {
    id: 1,
    code: 'image-main',
    name: '图片主池',
    pool_type: 'image',
    description: '主力图片账号池',
    enabled: true,
    dispatch_strategy: 'least_recently_used',
    sticky_ttl_sec: 300,
  },
  {
    id: 2,
    code: 'image-fallback',
    name: '图片回退池',
    pool_type: 'image',
    description: 'fallback 图片账号池',
    enabled: true,
    dispatch_strategy: 'least_recently_used',
    sticky_ttl_sec: 300,
  },
]

const mockMembers: Record<number, any[]> = {
  1: [
    { id: 11, pool_id: 1, account_id: 101, enabled: true, weight: 150, priority: 100, max_parallel: 1, note: '高质量图像号' },
    { id: 12, pool_id: 1, account_id: 102, enabled: true, weight: 100, priority: 120, max_parallel: 1, note: '备用图像号' },
  ],
  2: [
    { id: 21, pool_id: 2, account_id: 102, enabled: true, weight: 100, priority: 100, max_parallel: 1, note: '回退池成员' },
  ],
}

const mockAccounts = [
  { id: 101, email: 'image1@example.com', account_type: 'chatgpt', status: 'healthy', notes: '图片号', has_rt: true, has_st: false, client_id: '', chatgpt_account_id: '', oai_session_id: '', oai_device_id: '', plan_type: 'plus', daily_image_quota: 100, today_used_count: 3, last_refresh_source: 'rt', refresh_error: '', image_quota_remaining: 47, image_quota_total: 50, created_at: '', updated_at: '' },
  { id: 102, email: 'image2@example.com', account_type: 'chatgpt', status: 'warned', notes: '备用图像号', has_rt: true, has_st: true, client_id: '', chatgpt_account_id: '', oai_session_id: '', oai_device_id: '', plan_type: 'plus', daily_image_quota: 100, today_used_count: 9, last_refresh_source: 'st', refresh_error: '', image_quota_remaining: 20, image_quota_total: 50, created_at: '', updated_at: '' },
]

const mockModels = [
  { id: 1, slug: 'gpt-image-2', type: 'image', upstream_model_slug: 'auto', input_price_per_1m: 0, output_price_per_1m: 0, cache_read_price_per_1m: 0, image_price_per_call: 500000, description: '图片模型', enabled: true, created_at: '', updated_at: '' },
]

const mockProxies = [
  { id: 1, name: 'Proxy-A', host: '127.0.0.1', port: 7890, enabled: true, remark: '本地代理', scheme: 'http', country: 'CN', isp: 'local', health_score: 100, created_at: '', updated_at: '' },
  { id: 2, name: 'Proxy-B', host: '127.0.0.1', port: 7891, enabled: true, remark: '备用代理', scheme: 'http', country: 'US', isp: 'local', health_score: 90, created_at: '', updated_at: '' },
]

const mockRoutes = [
  { id: 1, model_id: 1, pool_id: 1, fallback_pool_id: 2, enabled: true },
]

const mockUser = {
  id: 1,
  email: 'admin@example.com',
  nickname: 'admin',
  role: 'admin',
  status: 'active',
  group_id: 1,
  credit_balance: 999999,
  credit_frozen: 0,
}

const mockPermissions = [
  'account:read', 'account:write',
  'proxy:read', 'proxy:write',
  'model:read', 'model:write',
  'system:setting',
]

const mockMenu = [
  {
    key: 'admin',
    title: '后台管理',
    icon: 'Setting',
    path: '/admin',
    children: [
      { key: 'admin.accounts', title: 'GPT账号', icon: 'Connection', path: '/admin/accounts' },
      { key: 'admin.account-pools', title: '账号池', icon: 'CollectionTag', path: '/admin/account-pools' },
      { key: 'admin.account-pool-routes', title: '池路由', icon: 'Share', path: '/admin/account-pool-routes' },
      { key: 'admin.proxies', title: '代理管理', icon: 'Guide', path: '/admin/proxies' },
      { key: 'admin.settings', title: '系统设置', icon: 'Tools', path: '/admin/settings' },
    ],
  },
]

function ok(data: any, config?: InternalAxiosRequestConfig): AxiosResponse<any> {
  return {
    data: { code: 0, message: 'ok', data },
    status: 200,
    statusText: 'OK',
    headers: {},
    config: (config || {}) as any,
  }
}

export function maybeMockResponse(config: InternalAxiosRequestConfig): AxiosResponse<any> | null {
  const enabled = import.meta.env.VITE_DEV_MOCK === '1'
  if (!enabled) return null

  const url = config.url || ''
  const method = (config.method || 'get').toLowerCase()

  if (method === 'get' && url === '/api/admin/account-pools') {
    return ok({ items: mockPools, total: mockPools.length }, config)
  }
  if (method === 'get' && url.match(/^\/api\/admin\/account-pools\/\d+\/members$/)) {
    const poolID = Number(url.split('/')[4])
    const items = mockMembers[poolID] || []
    return ok({ items, total: items.length }, config)
  }
  if (method === 'get' && url === '/api/admin/account-pool-routes') {
    return ok({ items: mockRoutes, total: mockRoutes.length }, config)
  }
  if (method === 'get' && url === '/api/public/site-info') {
    return ok({
      'site.name': 'GPT2API 控制台',
      'site.logo_url': '',
      'site.footer': 'Mock preview mode',
      'auth.allow_register': 'false',
    }, config)
  }
  if (method === 'post' && url === '/api/auth/login') {
    return ok({
      user: mockUser,
      token: {
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        expires_in: 86400,
      },
    }, config)
  }
  if (method === 'get' && url === '/api/me') {
    return ok({
      user: mockUser,
      role: 'admin',
      permissions: mockPermissions,
    }, config)
  }
  if (method === 'get' && url === '/api/me/menu') {
    return ok({
      role: 'admin',
      permissions: mockPermissions,
      menu: mockMenu,
    }, config)
  }
  if (method === 'get' && url === '/api/admin/models') {
    return ok({ items: mockModels, total: mockModels.length }, config)
  }
  if (method === 'get' && url === '/api/admin/proxies') {
    return ok({ list: mockProxies, total: mockProxies.length, page: 1, page_size: 500 }, config)
  }
  if (method === 'get' && url === '/api/admin/accounts') {
    const params = (config.params || {}) as Record<string, any>
    const poolID = Number(params.pool_id || 0)
    let list = mockAccounts
    if (poolID > 0) {
      const accountIDs = new Set((mockMembers[poolID] || []).map((x) => x.account_id))
      list = list.filter((x) => accountIDs.has(x.id))
    }
    return ok({ list, total: list.length, page: 1, page_size: 10 }, config)
  }
  if (method === 'get' && url === '/api/admin/accounts/auto-refresh') {
    return ok({ enabled: true, ahead_sec: 86400, threshold: 'AT 距离过期 < 1 天时自动刷新' }, config)
  }
  if (method === 'put' && url === '/api/admin/accounts/auto-refresh') {
    const body = typeof config.data === 'string' ? JSON.parse(config.data) : (config.data || {})
    return ok({ enabled: !!body.enabled, ahead_sec: 86400 }, config)
  }
  if (method === 'post' && url === '/api/admin/accounts/refresh-all') {
    return ok({ total: mockAccounts.length, success: mockAccounts.length, failed: 0, results: [] }, config)
  }
  if (method === 'post' && url === '/api/admin/accounts/probe-quota-all') {
    return ok({ total: mockAccounts.length, success: mockAccounts.length, failed: 0, results: [] }, config)
  }
  return null
}
