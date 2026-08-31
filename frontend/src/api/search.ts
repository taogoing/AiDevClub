import http from './http'
import type { ApiResponse, SearchResponse } from '@/types'

export interface SearchParams {
  q: string
  type?: 'article' | 'skill' | 'mcp_server' | ''
  tag_id?: number
  page?: number
  page_size?: number
}

export function search(params: SearchParams) {
  return http.get<ApiResponse<SearchResponse>>('/api/v1/search', { params })
}
