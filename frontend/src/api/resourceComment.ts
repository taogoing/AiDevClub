import http from './http'
import type { ApiResponse, ResourceCommentItem, LikeResult } from '@/types'

export function getResourceComments(resourceType: string, resourceId: number) {
  return http.get<ApiResponse<ResourceCommentItem[]>>(`/api/v1/${resourceType}/${resourceId}/comments`)
}

export function createResourceComment(resourceType: string, resourceId: number, data: { content: string; parent_id?: number }) {
  return http.post<ApiResponse<{ id: number }>>(`/api/v1/${resourceType}/${resourceId}/comments`, data)
}

export function deleteResourceComment(id: number) {
  return http.delete<ApiResponse<null>>(`/api/v1/resource-comments/${id}`)
}

export function likeResourceComment(id: number) {
  return http.post<ApiResponse<LikeResult>>(`/api/v1/resource-comments/${id}/like`)
}
