import http from './http'
import type { ApiResponse, LoginResult } from '@/types'

export function register(email: string, password: string, nickname: string) {
  return http.post<ApiResponse<null>>('/api/v1/auth/register', { email, password, nickname })
}

export function login(email: string, password: string) {
  return http.post<ApiResponse<LoginResult>>('/api/v1/auth/login', { email, password })
}

export function refreshToken(refresh_token: string) {
  return http.post<ApiResponse<LoginResult>>('/api/v1/auth/refresh', { refresh_token })
}

export function logout(refresh_token: string) {
  return http.post<ApiResponse<null>>('/api/v1/auth/logout', { refresh_token })
}
