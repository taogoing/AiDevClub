<template>
  <div class="skill-card" @click="$router.push(`/skills/${skill.id}`)">
    <div class="card-header">
      <h3 class="card-title">{{ skill.name }}</h3>
      <el-tag size="small" :type="statusType">{{ statusLabel }}</el-tag>
    </div>
    <p class="card-summary">{{ skill.description || '暂无描述' }}</p>
    <div class="card-meta">
      <div class="meta-left">
        <el-avatar :size="20" :src="skill.author.avatar_url || undefined">
          {{ skill.author.nickname?.charAt(0) || '?' }}
        </el-avatar>
        <span class="author-name">{{ skill.author.nickname }}</span>
        <el-tag v-for="tag in skill.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
      </div>
      <div class="meta-right">
        <span><el-icon><View /></el-icon> {{ skill.views }}</span>
        <span><el-icon><Download /></el-icon> {{ skill.downloads }}</span>
        <span><el-icon><ChatDotRound /></el-icon> {{ skill.comments_count }}</span>
        <span><el-icon><Star /></el-icon> {{ skill.favorites_count }}</span>
        <span><el-icon><CaretTop /></el-icon> {{ skill.likes_count }}</span>
        <span class="time">{{ formatTime(skill.published_at) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { View, Download, ChatDotRound, Star, CaretTop } from '@element-plus/icons-vue'
import type { SkillSummary } from '@/types'

const props = defineProps<{ skill: SkillSummary }>()

const statusType = computed(() => {
  const map: Record<string, string> = {
    draft: 'info',
    pending_review: 'warning',
    published: 'success',
    rejected: 'danger',
    archived: 'info',
  }
  return (map[props.skill.status] || 'info') as any
})

const statusLabel = computed(() => {
  const map: Record<string, string> = {
    draft: '草稿',
    pending_review: '审核中',
    published: '已发布',
    rejected: '已拒绝',
    archived: '已下架',
  }
  return map[props.skill.status] || props.skill.status
})

function formatTime(t: string | null) {
  if (!t) return ''
  return new Date(t).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.skill-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 12px;
  cursor: pointer;
  transition: box-shadow 0.2s;
  border: 1px solid #ebeef5;
}

.skill-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.card-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.card-summary {
  color: #606266;
  font-size: 14px;
  margin: 8px 0 12px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  color: #909399;
  flex-wrap: wrap;
  gap: 8px;
}

.meta-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.author-name {
  color: #606266;
}

.meta-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.meta-right span {
  display: flex;
  align-items: center;
  gap: 3px;
}

.time {
  margin-left: 4px;
}
</style>
