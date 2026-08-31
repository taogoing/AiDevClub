# 消息通知铃铛功能设计

## 概述

在用户端导航栏添加消息通知铃铛图标，点击后弹出下拉面板展示最近10条通知，支持未读角标、标记已读、查看全部等功能。

## 功能需求

### 1. 通知入口

- 位置：导航栏右侧，用户头像左边（紧挨着头像）
- 图标：信封图标（使用 Element Plus 的 `Bell` 图标）
- 仅登录用户可见

### 2. 未读角标

- 显示在铃铛图标右上角
- 未读数 1-99：显示数字
- 未读数 > 99：显示 "99+"
- 未读数 = 0：不显示角标
- 实时轮询更新（每 30 秒）

### 3. 下拉面板

- **顶部**：标题 "通知" + "全部已读" 按钮
- **列表**：最多显示 10 条通知，每条显示：
  - 通知类型图标
  - 通知标题
  - 通知内容摘要（截断显示）
  - 时间（相对时间，如 "5分钟前"）
  - 未读状态指示（蓝色圆点）
- **底部**："查看全部" 链接，跳转到 `/notifications` 独立页面
- **空状态**：显示 "暂无通知"

### 4. 交互行为

- 点击铃铛图标：打开/关闭下拉面板
- 打开面板时：自动加载未读数和最新10条通知
- 点击单条通知：
  - 标记该通知为已读
  - 跳转到相关页面（根据通知类型和资源ID）
- 点击"全部已读"：批量标记所有通知为已读

### 5. 通知类型

| 类型 | 图标 | 文案示例 |
|------|------|----------|
| 评论文章 | ChatDotRound | "有人评论了你的文章" |
| 回复评论 | ChatLineRound | "有人回复了你的评论" |
| 点赞文章 | Star | "有人点赞了你的文章" |
| 点赞 Skill | Star | "有人点赞了你的 Skill" |
| 点赞 MCP Server | Star | "有人点赞了你的 MCP Server" |
| 点赞评论 | Star | "有人点赞了你的评论" |
| 点赞资源评论 | Star | "有人点赞了你的评论" |
| 资源审核通过 | CircleCheck | "你的资源已通过审核" |
| 资源审核拒绝 | CircleClose | "你的资源未通过审核" |
| 举报处理完成 | Warning | "你的举报已处理" |
| 系统公告 | Bell | "系统公告" |

## 技术方案

### 1. 新增文件

```
frontend/src/
├── api/
│   └── notification.ts          # 通知 API 接口
├── types/
│   └── notification.ts          # 通知类型定义
└── components/
    └── NotificationBell.vue     # 通知铃铛组件
```

### 2. 修改文件

```
frontend/src/
├── components/
│   └── AppLayout.vue            # 添加通知铃铛组件
└── router/
    └── index.ts                 # 添加 /notifications 路由
```

### 3. API 接口

```typescript
// 获取通知列表
GET /api/v1/notifications?page=1&page_size=10

// 获取未读数
GET /api/v1/notifications/unread-count

// 标记单条已读
PUT /api/v1/notifications/:id/read

// 全部标记已读
PUT /api/v1/notifications/read
```

### 4. 类型定义

```typescript
interface Notification {
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
  actor?: {
    id: number
    nickname: string
    avatar_url: string
  }
}

interface NotificationListResult {
  items: Notification[]
  total: number
  page: number
  page_size: number
}
```

## 样式规范

- 铃铛图标：24px，颜色 #606266，hover 时 #409eff
- 角标：红色背景，白色文字，最小宽度 18px，高度 18px
- 下拉面板：宽度 360px，最大高度 480px，带滚动条
- 通知项：hover 背景色 #f5f7fa
- 未读状态：左侧 4px 蓝色边框

## 测试要点

1. 未读数正确显示和更新
2. 下拉面板正确展示通知列表
3. 点击通知正确标记已读并跳转
4. "全部已读"功能正常
5. 未登录时不显示铃铛图标
6. 空状态正确显示
7. 响应式布局在移动端正常显示
