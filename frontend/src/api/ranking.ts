import http from './http'
import type { ApiResponse } from '@/types'

export interface RankingResult {
  ids: number[]
  page: number
  page_size: number
}

export function getArticleRanking(params: { type?: string; page?: number; page_size?: number } = {}) {
  return http.get<ApiResponse<RankingResult>>('/api/v1/articles/ranking', { params })
}

export function getSkillRanking(params: { type?: string; page?: number; page_size?: number } = {}) {
  return http.get<ApiResponse<RankingResult>>('/api/v1/skills/ranking', { params })
}

export function getMcpServerRanking(params: { type?: string; page?: number; page_size?: number } = {}) {
  return http.get<ApiResponse<RankingResult>>('/api/v1/mcp-servers/ranking', { params })
}
