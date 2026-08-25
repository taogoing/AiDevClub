<template>
  <div class="articles-view">
    <h2>文章管理</h2>
    <el-form :inline="true" @submit.prevent="loadArticles">
      <el-form-item label="关键词">
        <el-input v-model="keyword" placeholder="标题/摘要" clearable @clear="loadArticles" />
      </el-form-item>
      <el-form-item label="可见性">
        <el-select v-model="visibility" clearable placeholder="全部" @change="loadArticles">
          <el-option label="正常" value="visible" />
          <el-option label="隐藏" value="hidden" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="loadArticles">搜索</el-button>
      </el-form-item>
    </el-form>
    <el-table :data="articles" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="title" label="标题" show-overflow-tooltip />
      <el-table-column label="作者" width="120">
        <template #default="{ row }">{{ row.author.nickname }}</template>
      </el-table-column>
      <el-table-column prop="views" label="浏览" width="80" />
      <el-table-column prop="likes_count" label="点赞" width="80" />
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.hidden ? 'danger' : 'success'">{{ row.hidden ? '隐藏' : '正常' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150">
        <template #default="{ row }">
          <el-button size="small" @click="viewArticle(row)">查看</el-button>
          <el-button size="small" :type="row.hidden ? 'success' : 'danger'" @click="toggleHidden(row)">
            {{ row.hidden ? '恢复' : '隐藏' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination
      v-if="total > 0"
      style="margin-top: 20px; justify-content: flex-end"
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="handlePageChange"
    />
    <el-drawer v-model="drawerVisible" title="文章详情" size="50%">
      <div v-if="selectedArticle" v-loading="detailLoading">
        <h3>{{ selectedArticle.title }}</h3>
        <p class="text-gray">作者: {{ selectedArticle.author.nickname }} | 浏览: {{ selectedArticle.views }}</p>
        <div class="content" v-html="selectedArticle.content"></div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminArticles, getAdminArticle, hideAdminArticle, unhideAdminArticle, type AdminArticle, type AdminArticleDetail } from '@/api/admin'

const articles = ref<AdminArticle[]>([])
const keyword = ref('')
const visibility = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const drawerVisible = ref(false)
const detailLoading = ref(false)
const selectedArticle = ref<AdminArticleDetail | null>(null)

async function loadArticles() {
  loading.value = true
  try {
    const res = await getAdminArticles({ keyword: keyword.value, visibility: visibility.value as any, page: page.value, page_size: pageSize.value })
    articles.value = res.data.data.list
    total.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  loadArticles()
}

async function viewArticle(row: AdminArticle) {
  drawerVisible.value = true
  detailLoading.value = true
  try {
    const res = await getAdminArticle(row.id)
    selectedArticle.value = res.data.data
  } catch {
    ElMessage.error('加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function toggleHidden(row: AdminArticle) {
  try {
    await ElMessageBox.confirm(row.hidden ? '确认恢复这篇文章？' : '确认隐藏这篇文章？', '内容管理')
    if (row.hidden) {
      await unhideAdminArticle(row.id)
    } else {
      await hideAdminArticle(row.id)
    }
    ElMessage.success('操作成功')
    await loadArticles()
  } catch {
    // cancelled
  }
}

onMounted(loadArticles)
</script>

<style scoped>
.articles-view {
  padding: 20px;
}
.text-gray {
  color: #999;
  font-size: 12px;
}
.content {
  margin-top: 20px;
  line-height: 1.6;
}
</style>
