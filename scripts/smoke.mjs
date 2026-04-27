#!/usr/bin/env node
/**
 * GPT2API · image-only 冒烟脚本
 *
 * 覆盖:
 *   1. /healthz
 *   2. 空库首位注册 bootstrap admin / 或复用既有 admin 登录
 *   3. /api/me、/api/me/menu
 *   4. 管理后台最小接口: accounts / proxies / account-pools / account-pool-routes / settings
 *   5. 旧 SaaS 路径不再注册
 *   6. /v1 静态 Bearer Token 鉴权 + /v1/models 仅返回 gpt-image-2
 */

import { argv, env, exit } from 'node:process'

const args = parseArgs(argv.slice(2))
const BASE = stripSlash(args.base || env.GPT2API_BASE || 'http://localhost:8080')
const ADMIN_EMAIL = args['admin-email'] || env.GPT2API_ADMIN_EMAIL || `admin+${Date.now()}@smoke.test`
const ADMIN_PASS = args['admin-pass'] || env.GPT2API_ADMIN_PASS || 'Admin123456'
const GATEWAY_TOKEN = args['gateway-token'] || env.GPT2API_GATEWAY_TOKEN || env.JWT_SECRET || ''

let pass = 0
let fail = 0
const results = []

function parseArgs(arr) {
  const out = {}
  for (let i = 0; i < arr.length; i++) {
    const a = arr[i]
    if (!a.startsWith('--')) continue
    const k = a.slice(2)
    const v = arr[i + 1] && !arr[i + 1].startsWith('--') ? arr[++i] : 'true'
    out[k] = v
  }
  return out
}

function stripSlash(u) {
  return u.replace(/\/+$/, '')
}

const CYAN = '\x1b[36m'
const RED = '\x1b[31m'
const GREEN = '\x1b[32m'
const RESET = '\x1b[0m'

function step(title) {
  console.log(`\n${CYAN}▶ ${title}${RESET}`)
}

function ok(msg) {
  pass++
  results.push(['PASS', msg])
  console.log(`  ${GREEN}✓${RESET} ${msg}`)
}

function bad(msg, extra) {
  fail++
  results.push(['FAIL', msg, extra])
  console.log(`  ${RED}✗ ${msg}${RESET}`)
  if (extra) console.log(`    ${extra}`)
}

async function call(method, path, { token, body, headers = {} } = {}) {
  const url = path.startsWith('http') ? path : BASE + path
  const h = { ...headers }
  if (token) h.Authorization = `Bearer ${token}`
  let payload
  if (body !== undefined) {
    h['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch(url, { method, headers: h, body: payload })
  const ct = res.headers.get('content-type') || ''
  const data = ct.includes('application/json') ? await res.json() : await res.text()
  return { status: res.status, body: data }
}

function isEnvelope(v) {
  return v && typeof v === 'object' && 'code' in v && 'data' in v
}

function unwrap(body) {
  return isEnvelope(body) ? body.data : body
}

async function checkHealth() {
  step('1. 健康检查')
  const r = await call('GET', '/healthz')
  if (r.status === 200) ok('/healthz 200 OK')
  else throw new Error(`/healthz 失败: ${r.status}`)
}

async function tryRegister(email, password) {
  return call('POST', '/api/auth/register', {
    body: { email, password, nickname: email.split('@')[0] },
  })
}

async function login(email, password) {
  const r = await call('POST', '/api/auth/login', {
    body: { email, password },
  })
  if (r.status !== 200 || r.body.code !== 0) {
    throw new Error(`login 失败: status=${r.status}`)
  }
  return unwrap(r.body)
}

let adminToken = ''

async function ensureAdmin() {
  step('2. bootstrap admin')

  try {
    const reg = await tryRegister(ADMIN_EMAIL, ADMIN_PASS)
    if (reg.status === 200 && reg.body.code === 0) {
      ok(`bootstrap admin 注册成功: ${ADMIN_EMAIL}`)
    } else if (reg.status === 200 && reg.body.code !== 0) {
      ok(`register 未成功(code=${reg.body.code}), 尝试直接登录复用 admin`)
    } else {
      ok(`register 返回 HTTP ${reg.status}, 尝试直接登录复用 admin`)
    }
  } catch {
    ok('register 不可用, 尝试直接登录复用 admin')
  }

  const loginResp = await login(ADMIN_EMAIL, ADMIN_PASS)
  adminToken = loginResp.token.access_token
  if (loginResp.user.role === 'admin') {
    ok(`admin 登录成功: id=${loginResp.user.id}`)
  } else {
    throw new Error(`登录用户不是 admin, 得到 role=${loginResp.user.role}`)
  }
}

async function checkAdminAPIs() {
  step('3. 管理后台最小接口')

  const me = await call('GET', '/api/me', { token: adminToken })
  if (me.status === 200 && unwrap(me.body)?.user?.email) ok('/api/me 正常')
  else bad('/api/me 异常', JSON.stringify(me.body))

  const menu = await call('GET', '/api/me/menu', { token: adminToken })
  const menuData = unwrap(menu.body)
  if (menu.status === 200 && Array.isArray(menuData?.menu) && menuData.menu.every((item) => item.path?.startsWith('/admin') || item.key === 'admin')) {
    ok('/api/me/menu 仅返回 admin 菜单')
  } else {
    bad('/api/me/menu 异常', JSON.stringify(menu.body))
  }

  const paths = [
    '/api/admin/accounts',
    '/api/admin/proxies',
    '/api/admin/account-pools',
    '/api/admin/account-pool-routes',
    '/api/admin/settings',
  ]
  for (const path of paths) {
    const r = await call('GET', path, { token: adminToken })
    if (r.status === 200) ok(`GET ${path} -> 200`)
    else bad(`GET ${path} 失败`, `status=${r.status}`)
  }
}

async function checkRemovedSaaSEndpoints() {
  step('4. 旧 SaaS 路径已移除')

  const removed = [
    ['GET', '/api/keys'],
    ['GET', '/api/recharge/packages'],
    ['GET', '/api/admin/users'],
    ['GET', '/api/admin/usage/stats'],
    ['GET', '/api/admin/audit'],
    ['GET', '/api/admin/system/backup'],
  ]
  for (const [method, path] of removed) {
    const r = await call(method, path, { token: adminToken })
    if (r.status === 404) ok(`${method} ${path} -> 404`)
    else bad(`${method} ${path} 仍然存在`, `status=${r.status}`)
  }
}

async function checkGateway() {
  step('5. /v1 静态 Bearer Token')

  const noAuth = await call('GET', '/v1/models')
  if (noAuth.status === 401) ok('/v1/models 无 token -> 401')
  else bad('/v1/models 无 token 未拦截', `status=${noAuth.status}`)

  if (!GATEWAY_TOKEN) {
    bad('缺少 gateway token', '请通过 --gateway-token 或 GPT2API_GATEWAY_TOKEN 传入')
    return
  }

  const models = await call('GET', '/v1/models', { token: GATEWAY_TOKEN })
  const data = unwrap(models.body)?.data || unwrap(models.body)
  const ids = Array.isArray(data) ? data.map((item) => item.id) : []
  if (models.status === 200 && ids.length === 1 && ids[0] === 'gpt-image-2') {
    ok('/v1/models 仅返回 gpt-image-2')
  } else {
    bad('/v1/models 返回不符合预期', JSON.stringify(models.body))
  }

  const removed = [
    ['POST', '/v1/chat/completions'],
    ['POST', '/v1/images/edits'],
    ['GET', '/v1/images/tasks/test-task'],
  ]
  for (const [method, path] of removed) {
    const r = await call(method, path, { token: GATEWAY_TOKEN })
    if (r.status === 404) ok(`${method} ${path} -> 404`)
    else bad(`${method} ${path} 仍然存在`, `status=${r.status}`)
  }
}

async function main() {
  try {
    console.log(`BASE=${BASE}`)
    await checkHealth()
    await ensureAdmin()
    await checkAdminAPIs()
    await checkRemovedSaaSEndpoints()
    await checkGateway()
  } catch (e) {
    bad('执行异常', e?.message || String(e))
  }

  console.log(`\nPASS=${pass} FAIL=${fail}`)
  if (fail > 0) exit(1)
}

main()
