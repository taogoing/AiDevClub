import http from './http'
import type { ApiResponse, SkillListResult, SkillDetail, LikeResult, FavoriteResult } from '@/types'

export function getSkills(params: Record<string, unknown>) {
  return http.get<ApiResponse<SkillListResult>>('/api/v1/skills', { params })
}

export function getSkill(id: number) {
  return http.get<ApiResponse<SkillDetail>>(`/api/v1/skills/${id}`)
}

export function createSkill(data: Record<string, unknown>) {
  return http.post<ApiResponse<{ id: number }>>('/api/v1/skills', data)
}

export function updateSkill(id: number, data: Record<string, unknown>) {
  return http.put<ApiResponse<{ id: number }>>(`/api/v1/skills/${id}`, data)
}

export function deleteSkill(id: number) {
  return http.delete<ApiResponse<null>>(`/api/v1/skills/${id}`)
}

export function submitSkill(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/skills/${id}/submit`)
}

export function withdrawSkill(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/skills/${id}/withdraw`)
}

export function archiveSkill(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/skills/${id}/archive`)
}

export function likeSkill(id: number) {
  return http.post<ApiResponse<LikeResult>>(`/api/v1/skills/${id}/like`)
}

export function favoriteSkill(id: number) {
  return http.post<ApiResponse<FavoriteResult>>(`/api/v1/skills/${id}/favorite`)
}
