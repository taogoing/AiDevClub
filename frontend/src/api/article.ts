import http from './http'
import type { ApiResponse, ArticleListResult, ArticleDetail, ArticleForm, LikeResult, FavoriteResult } from '@/types'

export function getArticles(params: Record<string, unknown>) {
  return http.get<ApiResponse<ArticleListResult>>('/api/v1/articles', { params })
}

export function getMyArticles(params: Record<string, unknown>) {
  return http.get<ApiResponse<ArticleListResult>>('/api/v1/articles/mine', { params })
}

export function getArticle(id: number) {
  return http.get<ApiResponse<ArticleDetail>>(`/api/v1/articles/${id}`)
}

export function createArticle(data: ArticleForm) {
  return http.post<ApiResponse<ArticleDetail>>('/api/v1/articles', data)
}

export function updateArticle(id: number, data: ArticleForm) {
  return http.put<ApiResponse<ArticleDetail>>(`/api/v1/articles/${id}`, data)
}

export function deleteArticle(id: number) {
  return http.delete<ApiResponse<null>>(`/api/v1/articles/${id}`)
}

export function uploadArticleImage(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post<ApiResponse<{ url: string }>>('/api/v1/articles/images', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function likeArticle(id: number) {
  return http.post<ApiResponse<LikeResult>>(`/api/v1/articles/${id}/like`)
}

export function favoriteArticle(id: number) {
  return http.post<ApiResponse<FavoriteResult>>(`/api/v1/articles/${id}/favorite`)
}
