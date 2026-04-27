import { http } from './http'

export interface LoginReq {
  email: string
  password: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in?: number
}

export interface LoginResp {
  user: UserInfo
  token: TokenPair
}

export interface UserInfo {
  id: number
  email: string
  nickname: string
  role: string
  status: string
  group_id: number
  created_at?: string
  last_login_at?: string
}

export function login(req: LoginReq): Promise<LoginResp> {
  return http.post('/api/auth/login', req)
}

export interface MeResp {
  user: UserInfo
  role: string
  permissions: string[]
}

export function getMe(): Promise<MeResp> {
  return http.get('/api/me')
}

export interface MenuItem {
  key: string
  title: string
  icon?: string
  path?: string
  children?: MenuItem[]
}

export interface MenuResp {
  role: string
  menu: MenuItem[]
  permissions: string[]
}

export function getMenu(): Promise<MenuResp> {
  return http.get('/api/me/menu')
}
