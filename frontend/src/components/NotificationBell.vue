<template>
  <div class="notification-bell" ref="bellRef">
    <el-badge :value="unreadCount" :hidden="unreadCount === 0" :max="99" class="bell-badge">
      <el-button :icon="Bell" circle @click="togglePanel" />
    </el-badge>
    
    <transition name="fade">
      <div v-if="showPanel" class="notification-panel">
        <div class="panel-header">
          <span class="panel-title">通知</span>
          <el-button text size="small" @click="handleMarkAllRead">全部已读</el-button>
        </div>
        
        <div class="panel-body">
          <div v-if="notifications.length === 0" class="empty-state">
            暂无通知
          </div>
          <div v-else class="notification-list">
            <div
              v-for="item in notifications"
              :key="item.id"
              class="notification-item"
              :class="{ unread: !item.is_read }"
              @click="handleNotificationClick(item)"
            >
              <div class="item-icon">
                <el-icon :size="16">
                  <component :is="getNotificationIcon(item.type)" />
                </el-icon>
              </div>
              <div class="item-content">
                <div class="item-title">{{ item.title }}</div>
                <div class="item-desc">{{ truncateContent(item.content) }}</div>
                <div class="item-time">{{ formatTime(item.created_at) }}</div>
              </div>
              <div v-if="!item.is_read" class="unread-dot"></div>
            </div>
          </div>
        </div>
        
        <div class="panel-footer">
          <router-link to="/notifications" class="view-all-link" @click="showPanel = false">
            查看全部
          </router-link>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, ChatDotRound, ChatLineRound, Star, CircleCheck, CircleClose, Warning } from '@element-plus/icons-vue'
import { getNotifications, getUnreadCount, markAsRead, markAllAsRead } from '@/api/notification'
import type { Notification } from '@/types/notification'

const router = useRouter()
const bellRef = ref<HTMLElement>()
const showPanel = ref(false)
const unreadCount = ref(0)
const notifications = ref<Notification[]>([])
const pollTimer = ref<number>()

const togglePanel = () => {
  showPanel.value = !showPanel.value
  if (showPanel.value) {
    fetchNotifications()
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

const fetchNotifications = async () => {
  try {
    const res = await getNotifications({ page: 1, page_size: 10 })
    notifications.value = res.data.data.list || []
  } catch (error) {
    console.error('Failed to fetch notifications:', error)
  }
}

const handleNotificationClick = async (item: Notification) => {
  if (!item.is_read) {
    try {
      await markAsRead(item.id)
      item.is_read = true
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    } catch (error) {
      console.error('Failed to mark as read:', error)
    }
  }
  
  // 根据资源类型跳转
  if (item.resource_type && item.resource_id) {
    const routeMap: Record<string, string> = {
      article: `/articles/${item.resource_id}`,
      skill: `/skills/${item.resource_id}`,
      mcp_server: `/mcps/${item.resource_id}`,
    }
    const path = routeMap[item.resource_type]
    if (path) {
      router.push(path)
    }
  }
  
  showPanel.value = false
}

const handleMarkAllRead = async () => {
  try {
    await markAllAsRead()
    notifications.value.forEach(item => item.is_read = true)
    unreadCount.value = 0
  } catch (error) {
    console.error('Failed to mark all as read:', error)
  }
}

const getNotificationIcon = (type: string) => {
  const iconMap: Record<string, any> = {
    comment_article: ChatDotRound,
    reply_comment: ChatLineRound,
    like_article: Star,
    like_skill: Star,
    like_mcp_server: Star,
    like_comment: Star,
    like_resource_comment: Star,
    resource_approved: CircleCheck,
    resource_rejected: CircleClose,
    report_resolved: Warning,
    announcement: Bell,
  }
  return iconMap[type] || Bell
}

const truncateContent = (content: string) => {
  if (!content) return ''
  return content.length > 50 ? content.substring(0, 50) + '...' : content
}

const formatTime = (time: string) => {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}天前`
  
  return date.toLocaleDateString()
}

const handleClickOutside = (event: MouseEvent) => {
  if (bellRef.value && !bellRef.value.contains(event.target as Node)) {
    showPanel.value = false
  }
}

onMounted(() => {
  fetchUnreadCount()
  pollTimer.value = window.setInterval(fetchUnreadCount, 30000)
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
  }
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.notification-bell {
  position: relative;
}

.bell-badge :deep(.el-badge__content) {
  top: 8px;
  right: 12px;
}

.notification-panel {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 8px;
  width: 360px;
  max-height: 480px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgb(0 0 0 / 15%);
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #ebeef5;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  max-height: 360px;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
  color: #909399;
  font-size: 14px;
}

.notification-list {
  padding: 8px 0;
}

.notification-item {
  display: flex;
  align-items: flex-start;
  padding: 12px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
  position: relative;
}

.notification-item:hover {
  background-color: #f5f7fa;
}

.notification-item.unread {
  background-color: #f0f7ff;
}

.notification-item.unread::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background-color: #409eff;
}

.item-icon {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  color: #606266;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  color: #303133;
  margin-bottom: 4px;
}

.item-desc {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
  line-height: 1.4;
}

.item-time {
  font-size: 12px;
  color: #c0c4cc;
}

.unread-dot {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #409eff;
  margin-left: 8px;
  margin-top: 6px;
}

.panel-footer {
  padding: 12px 16px;
  border-top: 1px solid #ebeef5;
  text-align: center;
}

.view-all-link {
  color: #409eff;
  text-decoration: none;
  font-size: 14px;
}

.view-all-link:hover {
  text-decoration: underline;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
