import { API_CONFIG } from './config'
import { HttpClient } from './httpClient'

const authClient = new HttpClient(API_CONFIG.gatewayBaseUrl)

export interface RegisterTaxProfile {
  inn: string
  oktmo: string
  last_name: string
  first_name: string
  middle_name: string
  jurisdiction: string
  timezone: string
  phone: string
  wallets: string[]
  tax_residency_status: string
  taxpayer_type: string
}

export interface RegisterRequest {
  email: string
  password: string
  tax_profile: RegisterTaxProfile
}

export interface LoginRequest {
  email: string
  password: string
  ip: string | null
  ua: string | null
}

export interface RefreshRequest {
  refresh_token: string
}

export interface AuthUser {
  id: string
  email: string
  status: 'Active' | 'Pending' | 'Blocked'
  created_at: string
}

export interface AuthSession {
  id: string
  user_id: string
  status: 'Active' | 'Revoked' | 'Closed'
  created_at: string
  last_seen_at: string
  ip: string | null
  user_agent: string | null
}

export interface TokensResponse {
  access_token: string
  refresh_token: string
  access_expires_in: number
  refresh_expires_in: number
}

export interface LoginResult {
  user: AuthUser
  session: AuthSession
  tokens: TokensResponse
}

export interface AckResponse {
  ok: boolean
}

export async function registerUser(payload: RegisterRequest): Promise<AckResponse> {
  return authClient.request<AckResponse>('/auth/register', {
    method: 'POST',
    body: payload,
    auth: false,
    refreshOnUnauthorized: false,
  })
}

export async function loginUser(payload: LoginRequest): Promise<LoginResult> {
  return authClient.request<LoginResult>('/auth/login', {
    method: 'POST',
    body: payload,
    auth: false,
    refreshOnUnauthorized: false,
  })
}

export async function refreshAuthTokens(payload: RefreshRequest): Promise<TokensResponse> {
  return authClient.request<TokensResponse>('/auth/refresh', {
    method: 'POST',
    body: payload,
    auth: false,
    refreshOnUnauthorized: false,
  })
}

export async function logoutUser(): Promise<AckResponse> {
  return authClient.request<AckResponse>('/auth/logout', {
    method: 'POST',
    refreshOnUnauthorized: false,
  })
}

export async function verifyEmail(token: string): Promise<AckResponse> {
  return authClient.request<AckResponse>('/auth/verify', {
    method: 'GET',
    query: { token },
    auth: false,
    refreshOnUnauthorized: false,
  })
}
