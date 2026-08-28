<template>
  <div class="page-container">
    <div class="page-header">
      <h2>我的文章</h2>
      <el-button type="primary" @click="$router.push('/articles/new')">
        <el-icon><Plus /></el-icon> 写文章
      </el-button>
    </div>

    <div class="filter-bar">
      <el-radio-group v-model="statusFilter" @change="onStatusChange">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button value="published">已发布</el-radio-button>
        <el-radio-button value="draft">草稿</el-radio-button>
      </el-radio-group>
    </div>

    <div v-loading="loading">
      <div v-for="article in articles" :key="article.id" class="article-item">
        <div class="article-info" @click="$router.push(`/articles/${article.id}`)">
          <div class="article-title-row">
            <el-tag v-if="article.status === 'draft'" size="small" type="info">草稿</el-tag>
            <el-tag v-else size="small" type="success">已发布</el-tag>
            <h3 class="article-title">{{ article.title }}</h3>
          </div>
          <p class="article-summary">{{ article.summary || '暂无摘要' }}</p>
          <div class="article-meta">
            <el-tag v-for="tag in article.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
            <span class="meta-time">{{ formatTime(article.published_at) }}</span>
            <span><el-icon><View /></el-icon> {{ article.views }}</span>
            <span><el-icon><ChatDotRound /></el-icon> {{ article.comments_count }}</span>
          </div>
        </div>
        <div class="article-actions">
          <el-button size="small" @click="$router.push(`/articles/${article.id}/edit`)">编辑</el-button>
          <el-popconfirm title="确定删除这篇文章？" @confirm="handleDelete(article.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </div>
      </div>
      <el-empty v-if="!loading && !articles.length" description="暂无文章" />
    </div>

    <div class="pagination-wrap" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="fetchArticles"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, View, ChatDotRound } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getMyArticles, deleteArticle } from '@/api/article'
import type { ArticleSummary } from '@/types'

const articles = ref<ArticleSummary[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref('')

onMounted(() => fetchArticles())

function onStatusChange() {
  currentPage.value = 1
  fetchArticles()
}

async function fetchArticles() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: currentPage.value,
      page_size: pageSize,
    }
    if (statusFilter.value) params.status = statusFilter.value
    const res = await getMyArticles(params)
    const data = res.data.data
    articles.value = data.list
    total.value = data.total
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await deleteArticle(id)
    ElMessage.success('已删除')
    await fetchArticles()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

function formatTime(t: string | null) {
  if (!t) return ''
  return new Date(t).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.filter-bar {
  margin-bottom: 20px;
}

.article-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fff;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 12px;
  border: 1px solid #ebeef5;
  gap: 16px;
}

.article-info {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}

.article-info:hover .article-title {
  color: #409eff;
}

.article-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.article-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.2s;
}

.article-summary {
  color: #606266;
  font-size: 14px;
  margin: 0 0 8px;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #909399;
  flex-wrap: wrap;
}

.meta-time {
  color: #909399;
}

.article-meta span {
  display: flex;
  align-items: center;
  gap: 3px;
}

.article-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>
