import http from './http'
import type { ApiResponse, McpServerListResult, McpServerDetail, LikeResult, FavoriteResult } from '@/types'

export function getMcpServers(params: Record<string, unknown>) {
  return http.get<ApiResponse<McpServerListResult>>('/api/v1/mcp-servers', { params })
}

export function getMyMcpServers(params: Record<string, unknown>) {
  return http.get<ApiResponse<McpServerListResult>>('/api/v1/mcp-servers/mine', { params })
}

export function getMcpServer(id: number) {
  return http.get<ApiResponse<McpServerDetail>>(`/api/v1/mcp-servers/${id}`)
}

export function createMcpServer(data: Record<string, unknown>) {
  return http.post<ApiResponse<{ id: number }>>('/api/v1/mcp-servers', data)
}

export function updateMcpServer(id: number, data: Record<string, unknown>) {
  return http.put<ApiResponse<{ id: number }>>(`/api/v1/mcp-servers/${id}`, data)
}

export function deleteMcpServer(id: number) {
  return http.delete<ApiResponse<null>>(`/api/v1/mcp-servers/${id}`)
}

export function submitMcpServer(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/mcp-servers/${id}/submit`)
}

export function withdrawMcpServer(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/mcp-servers/${id}/withdraw`)
}

export function archiveMcpServer(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/mcp-servers/${id}/archive`)
}

export function likeMcpServer(id: number) {
  return http.post<ApiResponse<LikeResult>>(`/api/v1/mcp-servers/${id}/like`)
}

export function favoriteMcpServer(id: number) {
  return http.post<ApiResponse<FavoriteResult>>(`/api/v1/mcp-servers/${id}/favorite`)
}
