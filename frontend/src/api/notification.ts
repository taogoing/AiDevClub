import http from '@/utils/http'
import type { ApiResponse } from '@/types'
import type { NotificationListResult, UnreadCountResult } from '@/types/notification'

export const getNotifications = (params: { page?: number; page_size?: number }) =>
  http.get<ApiResponse<NotificationListResult>>('/api/v1/notifications', { params })

export const getUnreadCount = () =>
  http.get<ApiResponse<UnreadCountResult>>('/api/v1/notifications/unread-count')

export const markAsRead = (id: number) =>
  http.put<ApiResponse<null>>(`/api/v1/notifications/${id}/read`)

export const markAllAsRead = () =>
  http.put<ApiResponse<null>>('/api/v1/notifications/read')
