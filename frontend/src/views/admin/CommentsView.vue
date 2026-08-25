<template>
  <div class="comments-view">
    <h2>评论管理</h2>
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <el-tab-pane label="文章评论" name="article">
        <el-form :inline="true" @submit.prevent="loadArticleComments">
          <el-form-item label="关键词">
            <el-input v-model="articleKeyword" placeholder="内容" clearable @clear="loadArticleComments" />
          </el-form-item>
          <el-form-item label="可见性">
            <el-select v-model="articleVisibility" clearable placeholder="全部" @change="loadArticleComments">
              <el-option label="正常" value="visible" />
              <el-option label="隐藏" value="hidden" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadArticleComments">搜索</el-button>
          </el-form-item>
        </el-form>
        <el-table :data="articleComments" v-loading="articleLoading" stripe>
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="content" label="内容" show-overflow-tooltip />
          <el-table-column label="作者" width="120">
            <template #default="{ row }">{{ row.author.nickname }}</template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.hidden ? 'danger' : 'success'">{{ row.hidden ? '隐藏' : '正常' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button size="small" :type="row.hidden ? 'success' : 'danger'" @click="toggleArticleComment(row)">
                {{ row.hidden ? '恢复' : '隐藏' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-if="articleTotal > 0"
          style="margin-top: 20px; justify-content: flex-end"
          :current-page="articlePage"
          :page-size="articlePageSize"
          :total="articleTotal"
          layout="total, prev, pager, next"
          @current-change="handleArticlePageChange"
        />
      </el-tab-pane>
      <el-tab-pane label="资源评论" name="resource">
        <el-form :inline="true" @submit.prevent="loadResourceComments">
          <el-form-item label="关键词">
            <el-input v-model="resourceKeyword" placeholder="内容" clearable @clear="loadResourceComments" />
          </el-form-item>
          <el-form-item label="资源类型">
            <el-select v-model="resourceType" clearable placeholder="全部" @change="loadResourceComments">
              <el-option label="Skill" value="skill" />
              <el-option label="MCP Server" value="mcp_server" />
            </el-select>
          </el-form-item>
          <el-form-item label="可见性">
            <el-select v-model="resourceVisibility" clearable placeholder="全部" @change="loadResourceComments">
              <el-option label="正常" value="visible" />
              <el-option label="隐藏" value="hidden" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadResourceComments">搜索</el-button>
          </el-form-item>
        </el-form>
        <el-table :data="resourceComments" v-loading="resourceLoading" stripe>
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="content" label="内容" show-overflow-tooltip />
          <el-table-column label="作者" width="120">
            <template #default="{ row }">{{ row.author.nickname }}</template>
          </el-table-column>
          <el-table-column label="类型" width="100">
            <template #default="{ row }">{{ row.resource_type === 'skill' ? 'Skill' : 'MCP' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.hidden ? 'danger' : 'success'">{{ row.hidden ? '隐藏' : '正常' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button size="small" :type="row.hidden ? 'success' : 'danger'" @click="toggleResourceComment(row)">
                {{ row.hidden ? '恢复' : '隐藏' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-if="resourceTotal > 0"
          style="margin-top: 20px; justify-content: flex-end"
          :current-page="resourcePage"
          :page-size="resourcePageSize"
          :total="resourceTotal"
          layout="total, prev, pager, next"
          @current-change="handleResourcePageChange"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminComments, getAdminResourceComments, hideAdminComment, unhideAdminComment, hideAdminResourceComment, unhideAdminResourceComment, type AdminComment, type AdminResourceComment } from '@/api/admin'

const activeTab = ref('article')

const articleComments = ref<AdminComment[]>([])
const articleKeyword = ref('')
const articleVisibility = ref('')
const articlePage = ref(1)
const articlePageSize = ref(20)
const articleTotal = ref(0)
const articleLoading = ref(false)

const resourceComments = ref<AdminResourceComment[]>([])
const resourceKeyword = ref('')
const resourceType = ref('')
const resourceVisibility = ref('')
const resourcePage = ref(1)
const resourcePageSize = ref(20)
const resourceTotal = ref(0)
const resourceLoading = ref(false)

async function loadArticleComments() {
  articleLoading.value = true
  try {
    const res = await getAdminComments({ keyword: articleKeyword.value, visibility: articleVisibility.value as any, page: articlePage.value, page_size: articlePageSize.value })
    articleComments.value = res.data.data.list
    articleTotal.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    articleLoading.value = false
  }
}

function handleArticlePageChange(p: number) {
  articlePage.value = p
  loadArticleComments()
}

async function toggleArticleComment(row: AdminComment) {
  try {
    await ElMessageBox.confirm(row.hidden ? '确认恢复这条评论？' : '确认隐藏这条评论？', '评论管理')
    if (row.hidden) {
      await unhideAdminComment(row.id)
    } else {
      await hideAdminComment(row.id)
    }
    ElMessage.success('操作成功')
    await loadArticleComments()
  } catch {
    // cancelled
  }
}

async function loadResourceComments() {
  resourceLoading.value = true
  try {
    const res = await getAdminResourceComments({ keyword: resourceKeyword.value, resource_type: resourceType.value as any, visibility: resourceVisibility.value as any, page: resourcePage.value, page_size: resourcePageSize.value })
    resourceComments.value = res.data.data.list
    resourceTotal.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    resourceLoading.value = false
  }
}

function handleResourcePageChange(p: number) {
  resourcePage.value = p
  loadResourceComments()
}

async function toggleResourceComment(row: AdminResourceComment) {
  try {
    await ElMessageBox.confirm(row.hidden ? '确认恢复这条评论？' : '确认隐藏这条评论？', '评论管理')
    if (row.hidden) {
      await unhideAdminResourceComment(row.id)
    } else {
      await hideAdminResourceComment(row.id)
    }
    ElMessage.success('操作成功')
    await loadResourceComments()
  } catch {
    // cancelled
  }
}

function handleTabChange(tab: string) {
  if (tab === 'article') {
    loadArticleComments()
  } else {
    loadResourceComments()
  }
}

onMounted(() => {
  loadArticleComments()
})
</script>

<style scoped>
.comments-view {
  padding: 20px;
}
</style>
