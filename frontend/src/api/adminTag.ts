import http from './http'
import type { ApiResponse, Tag } from '@/types'

export interface AdminTag extends Tag {
  description: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface AdminTagListResult {
  items: AdminTag[]
  total: number
  page: number
  page_size: number
}

export interface AdminTagListParams {
  keyword?: string
  status?: 'all' | 'enabled' | 'disabled'
  page?: number
  page_size?: number
}

export function getAdminTags(params: AdminTagListParams = {}) {
  return http.get<ApiResponse<AdminTagListResult>>('/api/v1/admin/tags', { params })
}

export function createTag(data: { name: string; description?: string }) {
  return http.post<ApiResponse<AdminTag>>('/api/v1/admin/tags', data)
}

export function updateTag(id: number, data: { name?: string; description?: string; enabled?: boolean }) {
  return http.put<ApiResponse<null>>(`/api/v1/admin/tags/${id}`, data)
}
