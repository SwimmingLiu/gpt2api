import { http } from './http'
import type { ImportSummary } from './accounts'

export type AccountImportSourceType = 'sub2api' | 'cpa'
export type AccountImportSourceAuthMode = 'password' | 'api_key' | 'bearer'

export interface AccountImportSource {
  id: number
  source_type: AccountImportSourceType
  name: string
  base_url: string
  enabled: boolean
  auth_mode: AccountImportSourceAuthMode
  email: string
  group_id: string
  default_proxy_id: number
  target_pool_id: number
  has_api_key: boolean
  has_password: boolean
  has_secret_key: boolean
  created_at: string
  updated_at: string
}

export interface CreateAccountImportSourceBody {
  source_type: AccountImportSourceType
  name: string
  base_url: string
  enabled?: boolean
  auth_mode: AccountImportSourceAuthMode
  email?: string
  group_id?: string
  api_key?: string
  password?: string
  secret_key?: string
  default_proxy_id?: number
  target_pool_id?: number
}

export interface RemoteSub2APIGroup {
  id: string
  name: string
  description?: string
  platform?: string
  status?: string
  account_count?: number
  active_account_count?: number
}

export interface RemoteSub2APIAccount {
  id: string
  name: string
  email: string
  plan_type?: string
  status?: string
  expires_at?: string
  has_refresh_token?: boolean
}

export interface RemoteCPAFile {
  name: string
  email?: string
}

export interface RemoteImportBody {
  account_ids?: string[]
  file_names?: string[]
  update_existing?: boolean
  default_proxy_id?: number
  target_pool_id?: number
  resolve_identity?: boolean
  kick_refresh?: boolean
  kick_quota_probe?: boolean
}

export function listAccountImportSources() {
  return http.get<any, { items: AccountImportSource[]; total: number }>('/api/admin/account-import-sources')
}

export function createAccountImportSource(body: CreateAccountImportSourceBody) {
  return http.post<any, AccountImportSource>('/api/admin/account-import-sources', body)
}

export function listRemoteSub2APIGroups(sourceID: number) {
  return http.get<any, { items: RemoteSub2APIGroup[]; total: number }>(
    `/api/admin/account-import-sources/${sourceID}/sub2api/groups`,
  )
}

export function listRemoteSub2APIAccounts(sourceID: number) {
  return http.get<any, { items: RemoteSub2APIAccount[]; total: number }>(
    `/api/admin/account-import-sources/${sourceID}/sub2api/accounts`,
  )
}

export function listRemoteCPAFiles(sourceID: number) {
  return http.get<any, { items: RemoteCPAFile[]; total: number }>(
    `/api/admin/account-import-sources/${sourceID}/cpa/files`,
  )
}

export function importRemoteAccountSource(sourceID: number, body: RemoteImportBody) {
  return http.post<any, ImportSummary>(`/api/admin/account-import-sources/${sourceID}/import`, body)
}
