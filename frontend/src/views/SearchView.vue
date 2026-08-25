<template>
  <div class="search-page">
    <div class="search-header">
      <h1 class="search-title">搜索结果：{{ keyword }}</h1>
      <div class="type-tabs">
        <el-radio-group v-model="searchType" @change="handleSearch">
          <el-radio-button label="">全部 ({{ totalCount }})</el-radio-button>
          <el-radio-button label="article">文章 ({{ counts.article }})</el-radio-button>
          <el-radio-button label="skill">Skill ({{ counts.skill }})</el-radio-button>
          <el-radio-button label="mcp_server">MCP Server ({{ counts.mcp_server }})</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <div class="search-results">
      <div v-if="loading" class="loading-wrap">
        <el-skeleton :rows="5" animated />
      </div>
      <template v-else>
        <div v-for="item in results" :key="`${item.type}-${item.id}`" class="result-item">
          <div class="result-type">
            <el-tag size="small" :type="getTypeTagType(item.type)">{{ getTypeLabel(item.type) }}</el-tag>
          </div>
          <router-link :to="getItemLink(item)" class="result-title" v-html="item.title"></router-link>
          <p class="result-summary" v-html="item.summary"></p>
          <div class="result-meta">
            <span>{{ item.views }} 浏览</span>
            <span>{{ item.likes_count }} 点赞</span>
          </div>
        </div>
        <el-empty v-if="!results.length" description="未找到相关内容" />
      </template>
    </div>

    <div class="pagination-wrap" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="handleSearch"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { search } from '@/api/search'
import type { SearchResult } from '@/types'

const route = useRoute()

const keyword = ref('')
const searchType = ref('')
const results = ref<SearchResult[]>([])
const total = ref(0)
const totalCount = ref(0)
const counts = ref({ article: 0, skill: 0, mcp_server: 0 })
const page = ref(1)
const pageSize = 20
const loading = ref(false)

const handleSearch = async () => {
  loading.value = true
  try {
    const res = await search({
      q: keyword.value,
      type: searchType.value as any,
      page: page.value,
      page_size: pageSize,
    })

    const data = res.data.data
    results.value = data.items || []
    total.value = data.total
    totalCount.value = data.counts
      ? data.counts.article + data.counts.skill + data.counts.mcp_server
      : data.total

    if (data.counts) {
      counts.value = data.counts
    }
  } catch {
    results.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const getItemLink = (item: SearchResult) => {
  switch (item.type) {
    case 'article':
      return `/articles/${item.id}`
    case 'skill':
      return `/skills/${item.id}`
    case 'mcp_server':
      return `/mcps/${item.id}`
  }
}

const getTypeLabel = (type: string) => {
  switch (type) {
    case 'article': return '文章'
    case 'skill': return 'Skill'
    case 'mcp_server': return 'MCP'
    default: return type
  }
}

const getTypeTagType = (type: string): '' | 'success' | 'warning' | 'info' => {
  switch (type) {
    case 'article': return ''
    case 'skill': return 'success'
    case 'mcp_server': return 'warning'
    default: return 'info'
  }
}

onMounted(() => {
  keyword.value = (route.query.q as string) || ''
  searchType.value = (route.query.type as string) || ''
  handleSearch()
})

watch(() => route.query, (newQuery) => {
  keyword.value = (newQuery.q as string) || ''
  searchType.value = (newQuery.type as string) || ''
  page.value = 1
  handleSearch()
})
</script>

<style scoped>
.search-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.search-header {
  margin-bottom: 24px;
}

.search-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
}

.type-tabs {
  margin-bottom: 16px;
}

.search-results {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.loading-wrap {
  padding: 24px;
}

.result-item {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  transition: box-shadow 0.2s;
}

.result-item:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.result-type {
  margin-bottom: 8px;
}

.result-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  text-decoration: none;
  display: block;
  margin-bottom: 8px;
}

.result-title:hover {
  color: #409eff;
}

.result-title :deep(mark) {
  background: #fff3cd;
  color: #303133;
  padding: 0 2px;
  border-radius: 2px;
}

.result-summary {
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
  margin-bottom: 12px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.result-summary :deep(mark) {
  background: #fff3cd;
  color: #606266;
  padding: 0 2px;
  border-radius: 2px;
}

.result-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #909399;
}

.pagination-wrap {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}
</style>
