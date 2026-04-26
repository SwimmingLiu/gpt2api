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
}

export interface ModelPoolRoute {
  id?: number
  model_id: number
  pool_id: number
  fallback_pool_id: number
  enabled: boolean
}

export function listAccountPools() {
  return http.get<any, { items: AccountPool[]; total: number }>('/api/admin/account-pools')
}

export function getAccountPool(id: number) {
  return http.get<any, AccountPool>(`/api/admin/account-pools/${id}`)
}

export function createAccountPool(body: {
  code: string
  name: string
  pool_type?: string
  description?: string
  enabled?: boolean
  dispatch_strategy?: string
  sticky_ttl_sec?: number
}) {
  return http.post<any, AccountPool>('/api/admin/account-pools', body)
}

export function updateAccountPool(id: number, body: {
  name?: string
  pool_type?: string
  description?: string
  enabled?: boolean
  dispatch_strategy?: string
  sticky_ttl_sec?: number
}) {
  return http.patch<any, AccountPool>(`/api/admin/account-pools/${id}`, body)
}

export function deleteAccountPool(id: number) {
  return http.delete<any, { deleted: number }>(`/api/admin/account-pools/${id}`)
}

export function listPoolMembers(poolID: number) {
  return http.get<any, { items: AccountPoolMember[]; total: number }>(`/api/admin/account-pools/${poolID}/members`)
}

export function createPoolMember(poolID: number, body: {
  account_id: number
  enabled?: boolean
  weight?: number
  priority?: number
  max_parallel?: number
  note?: string
}) {
  return http.post<any, AccountPoolMember>(`/api/admin/account-pools/${poolID}/members`, body)
}

export function updatePoolMember(poolID: number, memberID: number, body: {
  enabled?: boolean
  weight?: number
  priority?: number
  max_parallel?: number
  note?: string
}) {
  return http.patch<any, AccountPoolMember>(`/api/admin/account-pools/${poolID}/members/${memberID}`, body)
}

export function deletePoolMember(poolID: number, memberID: number) {
  return http.delete<any, { deleted: number; pool_id: number }>(`/api/admin/account-pools/${poolID}/members/${memberID}`)
}

export function listModelPoolRoutes() {
  return http.get<any, { items: ModelPoolRoute[]; total: number }>('/api/admin/account-pool-routes')
}

export function putModelPoolRoute(modelID: number, body: {
  pool_id: number
  fallback_pool_id?: number
  enabled?: boolean
}) {
  return http.put<any, ModelPoolRoute>(`/api/admin/account-pool-routes/${modelID}`, body)
}

export function deleteModelPoolRoute(modelID: number) {
  return http.delete<any, { deleted: number }>(`/api/admin/account-pool-routes/${modelID}`)
}
