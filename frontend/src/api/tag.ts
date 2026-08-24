import http from './http'
import type { ApiResponse, Tag } from '@/types'

export function getTags(keyword?: string) {
  return http.get<ApiResponse<Tag[]>>('/api/v1/tags', { params: keyword ? { keyword } : {} })
}

export function getHotTags() {
  return http.get<ApiResponse<Tag[]>>('/api/v1/tags', { params: { hot: 1 } })
}
