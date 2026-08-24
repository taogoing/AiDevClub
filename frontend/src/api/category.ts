import http from './http'
import type { ApiResponse, Category } from '@/types'

export function getCategories() {
  return http.get<ApiResponse<Category[]>>('/api/v1/categories')
}
