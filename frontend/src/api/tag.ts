import http from './http'
import type { ApiResponse, Tag } from '@/types'

export function getTags(params: { prefix?: string; hot?: number; limit?: number; type?: string } = {}) {
  return http.get<ApiResponse<Tag[]>>('/api/v1/tags', { params })
}

export function getHotTags(limit = 20, type?: string) {
  return http.get<ApiResponse<Tag[]>>('/api/v1/tags', { params: { hot: 1, limit, type } })
}
