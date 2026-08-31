<template>
  <div class="notifications-page">
    <div class="header">
      <h1>通知中心</h1>
      <div class="actions">
        <el-badge :value="unreadCount" :hidden="unreadCount === 0" class="item">
          <el-button size="small" @click="refreshNotifications">刷新</el-button>
        </el-badge>
        <el-button type="primary" size="small" @click="markAllAsRead" :disabled="unreadCount === 0">
          全部标为已读
        </el-button>
      </div>
    </div>

    <div class="notification-list" v-loading="loading">
      <el-empty v-if="!loading && notifications.length === 0" description="暂无通知" />
      
      <div
        v-for="notification in notifications"
        :key="notification.id"
        class="notification-item"
        :class="{ unread: !notification.is_read }"
        @click="handleNotificationClick(notification)"
      >
        <div class="avatar">
          <el-avatar :size="40" :src="notification.actor?.avatar_url">
            {{ notification.actor?.nickname?.charAt(0) || 'U' }}
          </el-avatar>
        </div>
        <div class="content">
          <div class="title">{{ notification.title }}</div>
          <div class="message">{{ notification.content }}</div>
          <div class="meta">
            <span class="time">{{ formatTime(notification.created_at) }}</span>
            <el-tag size="small" :type="getTagType(notification.type)">{{ getTypeName(notification.type) }}</el-tag>
          </div>
        </div>
        <div class="status">
          <el-tag v-if="!notification.is_read" type="danger" size="small">未读</el-tag>
          <el-button
            v-if="!notification.is_read"
            type="primary"
            link
            size="small"
            @click.stop="markAsRead(notification)"
          >
            标为已读
          </el-button>
        </div>
      </div>
    </div>

    <div class="pagination" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getNotifications, getUnreadCount, markAsRead as markAsReadApi, markAllAsRead as markAllAsReadApi } from '@/api/notification'
import type { Notification } from '@/types/notification'

const notifications = ref<Notification[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const unreadCount = ref(0)

const formatTime = (time: string) => {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`
  
  return date.toLocaleDateString()
}

const getTagType = (type: string) => {
  const types: Record<string, string> = {
    'like': 'danger',
    'comment': 'primary',
    'system': 'warning',
    'announcement': 'success'
  }
  return (types[type] || 'info') as any
}

const getTypeName = (type: string) => {
  const names: Record<string, string> = {
    'like': '点赞',
    'comment': '评论',
    'system': '系统',
    'announcement': '公告'
  }
  return names[type] || type
}

const fetchNotifications = async () => {
  loading.value = true
  try {
    const res = await getNotifications({
      page: currentPage.value,
      page_size: pageSize.value
    })
    notifications.value = res.data.data.list || []
    total.value = res.data.data.total
  } catch (error) {
    console.error('Failed to fetch notifications:', error)
    ElMessage.error('获取通知失败')
  } finally {
    loading.value = false
  }
}

const fetchUnreadCount = async () => {
  try {
    const res = await getUnreadCount()
    unreadCount.value = res.data.data.count
  } catch (error) {
    console.error('Failed to fetch unread count:', error)
  }
}

const refreshNotifications = () => {
  currentPage.value = 1
  fetchNotifications()
  fetchUnreadCount()
}

const markAsRead = async (notification: Notification) => {
  try {
    await markAsReadApi(notification.id)
    notification.is_read = true
    unreadCount.value = Math.max(0, unreadCount.value - 1)
    ElMessage.success('已标为已读')
  } catch (error) {
    console.error('Failed to mark as read:', error)
    ElMessage.error('操作失败')
  }
}

const markAllAsRead = async () => {
  try {
    await markAllAsReadApi()
    notifications.value.forEach(n => n.is_read = true)
    unreadCount.value = 0
    ElMessage.success('已全部标为已读')
  } catch (error) {
    console.error('Failed to mark all as read:', error)
    ElMessage.error('操作失败')
  }
}

const handleNotificationClick = (notification: Notification) => {
  if (!notification.is_read) {
    markAsRead(notification)
  }
  
  // 根据资源类型跳转
  if (notification.resource_type && notification.resource_id) {
    const routes: Record<string, string> = {
      'article': `/articles/${notification.resource_id}`,
      'skill': `/skills/${notification.resource_id}`,
      'mcp_server': `/mcp-servers/${notification.resource_id}`
    }
    const route = routes[notification.resource_type]
    if (route) {
      window.open(route, '_blank')
    }
  }
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  fetchNotifications()
}

onMounted(() => {
  fetchNotifications()
  fetchUnreadCount()
})
</script>

<style scoped>
.notifications-page {
  padding: 24px;
  max-width: 800px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header h1 {
  margin: 0;
  color: #303133;
}

.actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.notification-list {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.notification-item {
  display: flex;
  align-items: flex-start;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background-color 0.2s;
}

.notification-item:hover {
  background-color: #f5f7fa;
}

.notification-item.unread {
  background-color: #f0f9ff;
}

.notification-item:last-child {
  border-bottom: none;
}

.avatar {
  margin-right: 16px;
  flex-shrink: 0;
}

.content {
  flex: 1;
  min-width: 0;
}

.title {
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.message {
  color: #606266;
  font-size: 14px;
  margin-bottom: 8px;
  line-height: 1.5;
}

.meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.time {
  color: #909399;
  font-size: 12px;
}

.status {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  margin-left: 16px;
  flex-shrink: 0;
}

.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}
</style>