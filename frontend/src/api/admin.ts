import http from './http'
import type { ApiResponse, AuthorBrief } from '@/types'

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface AdminUser {
  id: number
  email: string
  nickname: string
  avatar_url: string
  role: 'user' | 'admin'
  role_mutable: boolean
  created_at: string
}

export interface AdminUserQuery {
  keyword?: string
  role?: 'user' | 'admin' | ''
  page?: number
  page_size?: number
}

export interface AdminDashboard {
  total_users: number
  total_articles: number
  total_skills: number
  total_mcp_servers: number
  pending_skills: number
  pending_mcp_servers: number
  pending_reports: number
}

export interface AdminArticle {
  id: number
  title: string
  summary: string
  author: AuthorBrief
  category_id: number
  views: number
  likes_count: number
  comments_count: number
  hidden: boolean
  published_at: string | null
  created_at: string
}

export interface AdminArticleDetail extends AdminArticle {
  content: string
  category_name: string
  favorites_count: number
}

export interface AdminArticleQuery {
  keyword?: string
  visibility?: 'visible' | 'hidden' | ''
  author_id?: number
  page?: number
  page_size?: number
}

export interface AdminComment {
  id: number
  article_id: number
  parent_id: number
  author: AuthorBrief
  content: string
  likes_count: number
  hidden: boolean
  created_at: string
}

export interface AdminCommentQuery {
  keyword?: string
  visibility?: 'visible' | 'hidden' | ''
  page?: number
  page_size?: number
}

export interface AdminResourceComment {
  id: number
  resource_type: string
  resource_id: number
  parent_id: number
  author: AuthorBrief
  content: string
  likes_count: number
  hidden: boolean
  created_at: string
}

export interface AdminResourceCommentQuery {
  keyword?: string
  visibility?: 'visible' | 'hidden' | ''
  resource_type?: 'skill' | 'mcp_server' | ''
  page?: number
  page_size?: number
}

export interface AdminResource {
  id: number
  name: string
  description: string
  author: AuthorBrief
  status: string
  hidden: boolean
  views: number
  downloads: number
  created_at: string
  updated_at: string
}

export interface AdminSkillDetail extends AdminResource {
  repo_url: string
  zip_url: string
  zip_filename: string
  file_size: number
  skill_md: string
  reject_reason?: string
}

export interface AdminMCPServerDetail extends AdminResource {
  repo_url: string
  tools_json: string
  readme: string
  zip_url: string
  zip_filename: string
  file_size: number
  reject_reason?: string
}

export interface AdminResourceQuery {
  keyword?: string
  status?: string
  author_id?: number
  tag_id?: number
  page?: number
  page_size?: number
}

export interface AdminReport {
  id: number
  reporter_id: number
  target_type: string
  target_id: number
  reason: string
  description: string
  status: string
  handler_id: number
  handle_result: string
  created_at: string
  resolved_at: string | null
}

export interface AdminReportTarget {
  id: number
  type: string
  title?: string
  content?: string
  summary?: string
  hidden: boolean
  author_id: number
  author_name: string
  parent_url?: string
}

export interface AdminReportDetail extends AdminReport {
  target: AdminReportTarget
  reporter_name: string
}

export interface AdminAnnouncement {
  id: number
  title: string
  content: string
  admin_id: number
  created_at: string
}

export interface AdminLogItem {
  id: number
  admin_id: number
  admin: AuthorBrief
  action: string
  target_type: string
  target_id: number
  detail: unknown
  created_at: string
}

export interface AdminLogQuery {
  action?: string
  page?: number
  page_size?: number
}

export const getAdminDashboard = () =>
  http.get<ApiResponse<AdminDashboard>>('/api/v1/admin/dashboard')

export const getAdminUsers = (params: AdminUserQuery) =>
  http.get<ApiResponse<PageResult<AdminUser>>>('/api/v1/admin/users', { params })

export const updateAdminUserRole = (id: number, role: 'user' | 'admin') =>
  http.put<ApiResponse<null>>(`/api/v1/admin/users/${id}/role`, { role })

export const getAdminArticles = (params: AdminArticleQuery) =>
  http.get<ApiResponse<PageResult<AdminArticle>>>('/api/v1/admin/articles', { params })

export const getAdminArticle = (id: number) =>
  http.get<ApiResponse<AdminArticleDetail>>(`/api/v1/admin/articles/${id}`)

export const hideAdminArticle = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/articles/${id}/hide`)

export const unhideAdminArticle = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/articles/${id}/unhide`)

export const getAdminComments = (params: AdminCommentQuery) =>
  http.get<ApiResponse<PageResult<AdminComment>>>('/api/v1/admin/comments', { params })

export const hideAdminComment = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/comments/${id}/hide`)

export const unhideAdminComment = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/comments/${id}/unhide`)

export const getAdminResourceComments = (params: AdminResourceCommentQuery) =>
  http.get<ApiResponse<PageResult<AdminResourceComment>>>('/api/v1/admin/resource-comments', { params })

export const hideAdminResourceComment = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/resource-comments/${id}/hide`)

export const unhideAdminResourceComment = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/resource-comments/${id}/unhide`)

export const getAdminSkills = (params: AdminResourceQuery) =>
  http.get<ApiResponse<PageResult<AdminResource>>>('/api/v1/admin/skills', { params })

export const getAdminSkill = (id: number) =>
  http.get<ApiResponse<AdminSkillDetail>>(`/api/v1/admin/skills/${id}`)

export const hideAdminSkill = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/skills/${id}/hide`)

export const unhideAdminSkill = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/skills/${id}/unhide`)

export const reviewAdminSkill = (id: number, data: { approved: boolean; reason?: string }) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/skills/${id}/review`, data)

export const getAdminMCPServers = (params: AdminResourceQuery) =>
  http.get<ApiResponse<PageResult<AdminResource>>>('/api/v1/admin/mcp-servers', { params })

export const getAdminMCPServer = (id: number) =>
  http.get<ApiResponse<AdminMCPServerDetail>>(`/api/v1/admin/mcp-servers/${id}`)

export const hideAdminMCPServer = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/mcp-servers/${id}/hide`)

export const unhideAdminMCPServer = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/mcp-servers/${id}/unhide`)

export const reviewAdminMCPServer = (id: number, data: { approved: boolean; reason?: string }) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/mcp-servers/${id}/review`, data)

export const getAdminReports = (params: { status?: string; page?: number; page_size?: number }) =>
  http.get<ApiResponse<PageResult<AdminReport>>>('/api/v1/admin/reports', { params })

export const getAdminReport = (id: number) =>
  http.get<ApiResponse<AdminReportDetail>>(`/api/v1/admin/reports/${id}`)

export const resolveAdminReport = (id: number, data: { action: string; result: string }) =>
  http.put<ApiResponse<null>>(`/api/v1/admin/reports/${id}/resolve`, data)

export const getAdminAnnouncements = (params: { page?: number; page_size?: number }) =>
  http.get<ApiResponse<PageResult<AdminAnnouncement>>>('/api/v1/admin/announcements', { params })

export const createAdminAnnouncement = (data: { title: string; content: string }) =>
  http.post<ApiResponse<{ id: number }>>('/api/v1/admin/announcements', data)

export const getAdminLogs = (params: AdminLogQuery) =>
  http.get<ApiResponse<PageResult<AdminLogItem>>>('/api/v1/admin/logs', { params })
