# 消息通知铃铛功能实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 在用户端导航栏添加消息通知铃铛图标，点击后弹出下拉面板展示最近10条通知，支持未读角标、标记已读、查看全部等功能。

**架构：** 新增通知 API 模块、类型定义和 NotificationBell 组件，在 AppLayout 中集成。组件包含图标、角标和下拉面板，支持实时轮询更新未读数。

**技术栈：** Vue 3、TypeScript、Element Plus、Axios

---

### 任务 1：创建通知类型定义

**文件：**
- 创建：`frontend/src/types/notification.ts`

- [ ] **步骤 1：创建通知类型定义文件**

```typescript
// frontend/src/types/notification.ts

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
  items: Notification[]
  total: number
  page: number
  page_size: number
}

export interface UnreadCountResult {
  count: number
}
```

- [ ] **步骤 2：验证类型定义**

运行：`cd frontend && npx tsc --noEmit`
预期：无错误

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/types/notification.ts
git commit -m "feat: add notification type definitions"
```

---

### 任务 2：创建通知 API 模块

**文件：**
- 创建：`frontend/src/api/notification.ts`

- [ ] **步骤 1：创建通知 API 模块**

```typescript
// frontend/src/api/notification.ts
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
```

- [ ] **步骤 2：验证 API 模块**

运行：`cd frontend && npx tsc --noEmit`
预期：无错误

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/api/notification.ts
git commit -m "feat: add notification API module"
```

---

### 任务 3：创建 NotificationBell 组件基础结构

**文件：**
- 创建：`frontend/src/components/NotificationBell.vue`

- [ ] **步骤 1：创建组件基础结构和数据**

```vue
<!-- frontend/src/components/NotificationBell.vue -->
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
import { Bell, ChatDotRound, ChatLineRound, Star, CircleCheck, CircleClose, Warning } from '@element-plus/icons-vue'
import { getNotifications, getUnreadCount, markAsRead, markAllAsRead } from '@/api/notification'
import type { Notification } from '@/types/notification'

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
    notifications.value = res.data.data.items
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
  // TODO: 跳转到相关页面
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
```

- [ ] **步骤 2：验证组件语法**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无错误

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/components/NotificationBell.vue
git commit -m "feat: add NotificationBell component"
```

---

### 任务 4：在 AppLayout 中集成通知铃铛

**文件：**
- 修改：`frontend/src/components/AppLayout.vue:20-27`

- [ ] **步骤 1：导入 NotificationBell 组件**

在 `<script setup>` 中添加导入：

```typescript
import NotificationBell from './NotificationBell.vue'
```

- [ ] **步骤 2：在模板中添加通知铃铛**

修改模板，在用户头像左边添加通知铃铛：

```vue
<template v-if="auth.isLoggedIn">
  <NotificationBell />
  <el-dropdown trigger="click" @command="handleCommand">
    <div class="user-avatar-wrap">
      <el-avatar :size="32" :src="auth.user?.avatar_url || undefined">
        {{ auth.user?.nickname?.charAt(0) || '?' }}
      </el-avatar>
    </div>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="profile">个人中心</el-dropdown-item>
        <el-dropdown-item command="my-articles">我的文章</el-dropdown-item>
        <el-dropdown-item command="new-article">发布文章</el-dropdown-item>
        <el-dropdown-item divided command="logout">登出</el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>
```

- [ ] **步骤 3：验证集成**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无错误

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/components/AppLayout.vue
git commit -m "feat: integrate NotificationBell into AppLayout"
```

---

### 任务 5：添加通知路由和跳转逻辑

**文件：**
- 修改：`frontend/src/router/index.ts`
- 修改：`frontend/src/components/NotificationBell.vue:63-65`

- [ ] **步骤 1：添加通知页面路由**

在 `router/index.ts` 中添加路由：

```typescript
{
  path: '/notifications',
  name: 'notifications',
  component: () => import('@/views/NotificationsView.vue'),
  meta: { requiresAuth: true },
},
```

- [ ] **步骤 2：创建通知页面占位组件**

创建 `frontend/src/views/NotificationsView.vue`：

```vue
<template>
  <div class="notifications-page">
    <h1>通知中心</h1>
    <p>完整通知页面开发中...</p>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.notifications-page {
  padding: 24px;
  max-width: 800px;
  margin: 0 auto;
}

h1 {
  margin-bottom: 16px;
  color: #303133;
}
</style>
```

- [ ] **步骤 3：更新 NotificationBell 中的跳转逻辑**

修改 `handleNotificationClick` 方法，添加跳转逻辑：

```typescript
import { useRouter } from 'vue-router'

const router = useRouter()

// 在 handleNotificationClick 中添加跳转逻辑
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
```

- [ ] **步骤 4：验证路由配置**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无错误

- [ ] **步骤 5：Commit**

```bash
git add frontend/src/router/index.ts frontend/src/views/NotificationsView.vue frontend/src/components/NotificationBell.vue
git commit -m "feat: add notification route and page placeholder"
```

---

### 任务 6：最终验证和测试

**文件：**
- 无新增修改

- [ ] **步骤 1：运行类型检查**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无错误

- [ ] **步骤 2：运行构建**

运行：`cd frontend && npm run build`
预期：构建成功

- [ ] **步骤 3：运行开发服务器测试**

运行：`cd frontend && npm run dev`
手动测试：
1. 登录用户
2. 验证导航栏显示通知铃铛图标
3. 验证未读角标正确显示
4. 点击铃铛打开下拉面板
5. 验证通知列表正确展示
6. 点击通知验证标记已读
7. 点击"全部已读"验证批量标记
8. 点击"查看全部"验证跳转

- [ ] **步骤 4：Commit 最终版本**

```bash
git add -A
git commit -m "feat: complete notification bell feature"
```
