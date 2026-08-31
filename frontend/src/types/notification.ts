export interface NotificationActor {
  id: number
  nickname: string
  avatar_url: string
}

export interface Notification {
  id: number
  user_id: number
  type: string
  title: string
  content: string
  resource_type: string
  resource_id: number
  actor_id: number
  is_read: boolean
  created_at: string
  actor?: NotificationActor
}

export interface NotificationListResult {
  list: Notification[]
  total: number
  page: number
  page_size: number
}

export interface UnreadCountResult {
  count: number
}
