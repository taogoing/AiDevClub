import http from './http'
import type { ApiResponse, CommentItem, LikeResult } from '@/types'

export function getComments(articleId: number) {
  return http.get<ApiResponse<CommentItem[]>>(`/api/v1/articles/${articleId}/comments`)
}

export function createComment(articleId: number, content: string, parentId: number | null) {
  return http.post<ApiResponse<CommentItem>>(`/api/v1/articles/${articleId}/comments`, {
    content,
    parent_id: parentId,
  })
}

export function deleteComment(id: number) {
  return http.delete<ApiResponse<null>>(`/api/v1/comments/${id}`)
}

export function likeComment(id: number) {
  return http.post<ApiResponse<LikeResult>>(`/api/v1/comments/${id}/like`)
}
