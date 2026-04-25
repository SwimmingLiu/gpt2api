export type ImportPaneKind = 'access_token' | 'cpa' | 'sub2api' | 'manual'

export type TokenImportMode = 'at' | 'rt' | 'st'

export interface SelectOption {
  label: string
  value: number
  disabled?: boolean
}

export interface ImportAdvancedOptions {
  update_existing: boolean
  default_proxy_id?: number
  target_pool_id?: number
  resolve_identity: boolean
  kick_refresh: boolean
  kick_quota_probe: boolean
}

export interface ImportDialogResultRow {
  index: number
  source_type?: string
  source_ref?: string
  email: string
  status: 'created' | 'updated' | 'skipped' | 'failed'
  reason?: string
  warnings?: string[]
  id?: number
}

export interface AccessTokenImportModel {
  mode: TokenImportMode
  tokens_text: string
  client_id: string
}

export interface FileImportModel {
  text: string
  files: File[]
}

export interface ManualAccountForm {
  email: string
  auth_token: string
  refresh_token: string
  session_token: string
  client_id: string
  account_type: string
  plan_type: string
  daily_image_quota?: number
  notes: string
}

export interface AccessTokenDialogSubmitPayload {
  kind: 'access_token'
  advanced: ImportAdvancedOptions
  payload: AccessTokenImportModel
}

export interface CPAImportDialogSubmitPayload {
  kind: 'cpa'
  advanced: ImportAdvancedOptions
  payload: FileImportModel
}

export interface Sub2APIImportDialogSubmitPayload {
  kind: 'sub2api'
  advanced: ImportAdvancedOptions
  payload: FileImportModel
}

export interface ManualDialogSubmitPayload {
  kind: 'manual'
  advanced: ImportAdvancedOptions
  payload: ManualAccountForm
}

export type DialogSubmitPayload =
  | AccessTokenDialogSubmitPayload
  | CPAImportDialogSubmitPayload
  | Sub2APIImportDialogSubmitPayload
  | ManualDialogSubmitPayload
