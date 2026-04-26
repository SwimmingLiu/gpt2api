import { http } from './http'

export interface AccountPool {
  id: number
  code: string
  name: string
  pool_type: string
  description: string
  enabled: boolean
  dispatch_strategy: string
  sticky_ttl_sec: number
  created_at?: string
  updated_at?: string
}

export interface AccountPoolMember {
  id: number
  pool_id: number
  account_id: number
  enabled: boolean
  weight: number
  priority: number
  max_parallel: number
  note: string
  created_at?: string
  updated_at?: string
}

export interface AccountPoolRoute {
  id: number
  model_id: number
  pool_id: number
  fallback_pool_id: number
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export interface PagedList<T> {
  items: T[]
  total?: number
}

export function listPools(): Promise<PagedList<AccountPool>> {
  return http.get('/api/admin/account-pools')
}

export function getPool(id: number): Promise<AccountPool> {
  return http.get(`/api/admin/account-pools/${id}`)
}

export function createPool(body: Partial<AccountPool>): Promise<AccountPool> {
  return http.post('/api/admin/account-pools', body)
}

export function updatePool(id: number, body: Partial<AccountPool>): Promise<AccountPool> {
  return http.patch(`/api/admin/account-pools/${id}`, body)
}

export function deletePool(id: number) {
  return http.delete(`/api/admin/account-pools/${id}`)
}

export function listMembers(poolId: number): Promise<PagedList<AccountPoolMember>> {
  return http.get(`/api/admin/account-pools/${poolId}/members`)
}

export function createMember(poolId: number, body: Partial<AccountPoolMember>): Promise<AccountPoolMember> {
  return http.post(`/api/admin/account-pools/${poolId}/members`, body)
}

export function updateMember(poolId: number, memberId: number, body: Partial<AccountPoolMember>): Promise<AccountPoolMember> {
  return http.patch(`/api/admin/account-pools/${poolId}/members/${memberId}`, body)
}

export function deleteMember(poolId: number, memberId: number) {
  return http.delete(`/api/admin/account-pools/${poolId}/members/${memberId}`)
}

export function listRoutes(): Promise<PagedList<AccountPoolRoute>> {
  return http.get('/api/admin/account-pool-routes')
}

export function putRoute(modelId: number, body: Partial<AccountPoolRoute>): Promise<AccountPoolRoute> {
  return http.put(`/api/admin/account-pool-routes/${modelId}`, body)
}

export function deleteRoute(modelId: number) {
  return http.delete(`/api/admin/account-pool-routes/${modelId}`)
}
