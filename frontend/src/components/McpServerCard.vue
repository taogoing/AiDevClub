<template>
  <div class="mcp-card" @click="$router.push(`/mcps/${server.id}`)">
    <div class="card-header">
      <h3 class="card-title">{{ server.name }}</h3>
      <el-tag size="small" :type="statusType">{{ statusLabel }}</el-tag>
    </div>
    <p class="card-summary">{{ server.description || '暂无描述' }}</p>
    <div class="card-tags">
      <el-tag v-for="tag in server.tags" :key="tag.id" size="small" type="info">{{ tag.name }}</el-tag>
    </div>
    <div class="card-meta">
      <div class="meta-left">
        <el-avatar :size="18" :src="server.author.avatar_url || undefined">
          {{ server.author.nickname?.charAt(0) || '?' }}
        </el-avatar>
        <span class="author-name">{{ server.author.nickname }}</span>
      </div>
      <div class="meta-right">
        <span><el-icon><View /></el-icon>{{ server.views }}</span>
        <span><el-icon><Download /></el-icon>{{ server.downloads }}</span>
        <span><el-icon><CaretTop /></el-icon>{{ server.likes_count }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { View, Download, ChatDotRound, Star, CaretTop } from '@element-plus/icons-vue'
import type { McpServerSummary } from '@/types'

const props = defineProps<{ server: McpServerSummary }>()

const statusType = computed(() => {
  const map: Record<string, string> = {
    draft: 'info',
    pending_review: 'warning',
    published: 'success',
    rejected: 'danger',
    archived: 'info',
  }
  return (map[props.server.status] || 'info') as any
})

const statusLabel = computed(() => {
  const map: Record<string, string> = {
    draft: '草稿',
    pending_review: '审核中',
    published: '已发布',
    rejected: '已拒绝',
    archived: '已下架',
  }
  return map[props.server.status] || props.server.status
})

function formatTime(t: string | null) {
  if (!t) return ''
  return new Date(t).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.mcp-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.mcp-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  line-height: 1.4;
  flex: 1;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-summary {
  color: #606266;
  font-size: 13px;
  margin: 0 0 12px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
}

.card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 10px;
}

.card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #909399;
  padding-top: 10px;
  border-top: 1px solid #f0f0f0;
}

.meta-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.author-name {
  color: #606266;
  font-size: 12px;
}

.meta-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-right span {
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>
