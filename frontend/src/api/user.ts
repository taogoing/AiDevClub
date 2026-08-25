import http from './http'
import type { ApiResponse, UserProfile } from '@/types'

export function getMe() {
  return http.get<ApiResponse<UserProfile>>('/api/v1/users/me')
}

export function updateMe(data: { nickname?: string; avatar_url?: string; bio?: string }) {
  return http.put<ApiResponse<UserProfile>>('/api/v1/users/me', data)
}

export function updatePassword(password: string) {
  return http.put<ApiResponse<null>>('/api/v1/users/me/password', { password })
}

export function deleteAccount() {
  return http.delete<ApiResponse<null>>('/api/v1/users/me')
}

export function uploadAvatar(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post<ApiResponse<{ avatar_url: string }>>('/api/v1/users/me/avatar', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
