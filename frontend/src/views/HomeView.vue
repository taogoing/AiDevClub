<template>
  <div class="home-container">
    <div class="main-area">
      <div class="filter-bar">
        <div class="category-tabs">
          <el-button :type="!selectedCategory ? 'primary' : 'default'" size="small" @click="selectedCategory = 0">全部</el-button>
          <el-button
            v-for="cat in categories"
            :key="cat.id"
            :type="selectedCategory === cat.id ? 'primary' : 'default'"
            size="small"
            @click="selectedCategory = cat.id"
          >
            {{ cat.name }}
          </el-button>
        </div>
        <div class="sort-bar">
          <el-select v-model="sortBy" size="small" style="width: 100px">
            <el-option label="最新" value="latest" />
            <el-option label="热门" value="hot" />
            <el-option label="置顶" value="pinned" />
          </el-select>
        </div>
      </div>
      <div v-loading="loading">
        <ArticleCard v-for="article in articles" :key="article.id" :article="article" />
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
    <aside class="sidebar">
      <Sidebar />
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import ArticleCard from '@/components/ArticleCard.vue'
import Sidebar from '@/components/Sidebar.vue'
import { getArticles } from '@/api/article'
import { getCategories } from '@/api/category'
import type { ArticleSummary, Category } from '@/types'

const route = useRoute()
const articles = ref<ArticleSummary[]>([])
const categories = ref<Category[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const selectedCategory = ref(0)
const selectedTag = ref(0)
const sortBy = ref('latest')
const keyword = ref('')

onMounted(async () => {
  keyword.value = (route.query.keyword as string) || ''
  selectedTag.value = Number(route.query.tag_id) || 0
  try {
    const res = await getCategories()
    categories.value = res.data.data
  } catch { /* ignore */ }
  await fetchArticles()
})

watch([selectedCategory, selectedTag, sortBy], () => {
  currentPage.value = 1
  fetchArticles()
})

watch(() => route.query.keyword, (val) => {
  keyword.value = (val as string) || ''
  currentPage.value = 1
  fetchArticles()
})

watch(() => route.query.tag_id, (val) => {
  selectedTag.value = Number(val) || 0
  currentPage.value = 1
  fetchArticles()
})

async function fetchArticles() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: currentPage.value,
      page_size: pageSize,
      sort: sortBy.value,
    }
    if (selectedCategory.value) params.category_id = selectedCategory.value
    if (selectedTag.value) params.tag_id = selectedTag.value
    if (keyword.value) params.keyword = keyword.value
    const res = await getArticles(params)
    const data = res.data.data
    articles.value = data.list
    total.value = data.total
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.home-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  display: flex;
  gap: 24px;
}

.main-area {
  flex: 1;
  min-width: 0;
}

.sidebar {
  width: 280px;
  flex-shrink: 0;
}

.filter-bar {
  margin-bottom: 20px;
}

.category-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.sort-bar {
  margin-bottom: 16px;
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

@media (max-width: 768px) {
  .sidebar {
    display: none;
  }
  
  .home-container {
    padding: 16px;
  }
}
</style>
