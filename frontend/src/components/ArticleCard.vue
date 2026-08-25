<template>
  <div class="article-card" @click="$router.push(`/articles/${article.id}`)">
    <div class="card-header">
      <h3 class="card-title">
        <el-tag v-if="article.pinned" size="small" type="danger" class="pin-tag">置顶</el-tag>
        {{ article.title }}
      </h3>
    </div>
    <p class="card-summary">{{ article.summary || '暂无摘要' }}</p>
    <div class="card-meta">
      <div class="meta-left">
        <el-avatar :size="20" :src="article.author.avatar_url || undefined">
          {{ article.author.nickname?.charAt(0) || '?' }}
        </el-avatar>
        <span class="author-name">{{ article.author.nickname }}</span>
        <el-tag v-for="tag in article.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
      </div>
      <div class="meta-right">
        <span><el-icon><View /></el-icon> {{ article.views }}</span>
        <span><el-icon><ChatDotRound /></el-icon> {{ article.comments_count }}</span>
        <span><el-icon><Star /></el-icon> {{ article.favorites_count }}</span>
        <span><el-icon><CaretTop /></el-icon> {{ article.likes_count }}</span>
        <span class="time">{{ formatTime(article.published_at) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { View, ChatDotRound, Star, CaretTop } from '@element-plus/icons-vue'
import type { ArticleSummary } from '@/types'

defineProps<{ article: ArticleSummary }>()

function formatTime(t: string | null) {
  if (!t) return ''
  return new Date(t).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.article-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 12px;
  cursor: pointer;
  transition: box-shadow 0.2s;
  border: 1px solid #ebeef5;
}

.article-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.card-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 8px;
}

.pin-tag {
  flex-shrink: 0;
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
