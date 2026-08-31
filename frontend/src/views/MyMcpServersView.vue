<template>
  <div class="page-container">
    <div class="page-header">
      <h2>我的 MCP</h2>
      <el-button type="primary" @click="$router.push('/mcps/new')">
        <el-icon><Plus /></el-icon> 发布 MCP
      </el-button>
    </div>

    <div class="filter-bar">
      <el-radio-group v-model="statusFilter" @change="onStatusChange">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button value="published">已发布</el-radio-button>
        <el-radio-button value="draft">草稿</el-radio-button>
        <el-radio-button value="pending_review">待审核</el-radio-button>
        <el-radio-button value="rejected">已拒绝</el-radio-button>
      </el-radio-group>
    </div>

    <div v-loading="loading">
      <div v-for="server in servers" :key="server.id" class="resource-item">
        <div class="resource-info" @click="$router.push(`/mcps/${server.id}`)">
          <div class="resource-title-row">
            <el-tag :type="getStatusType(server.status)" size="small">{{ getStatusText(server.status) }}</el-tag>
            <h3 class="resource-title">{{ server.name }}</h3>
          </div>
          <p class="resource-desc">{{ server.description || '暂无描述' }}</p>
          <div class="resource-meta">
            <el-tag v-for="tag in server.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
            <span><el-icon><View /></el-icon> {{ server.views }}</span>
            <span><el-icon><ChatDotRound /></el-icon> {{ server.comments_count }}</span>
          </div>
        </div>
        <div class="resource-actions">
          <el-button size="small" @click="$router.push(`/mcps/${server.id}/edit`)">编辑</el-button>
          <el-button v-if="server.status === 'draft' || server.status === 'rejected' || server.status === 'archived'" size="small" type="success" @click="handleSubmit(server.id)">提交审核</el-button>
          <el-button v-if="server.status === 'pending_review'" size="small" @click="handleWithdraw(server.id)">撤回</el-button>
          <el-popconfirm title="确定删除这个 MCP？" @confirm="handleDelete(server.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </div>
      </div>
      <el-empty v-if="!loading && !servers.length" description="暂无 MCP" />
    </div>

    <div class="pagination-wrap" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="fetchServers"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, View, ChatDotRound } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getMyMcpServers, submitMcpServer, withdrawMcpServer, deleteMcpServer } from '@/api/mcpServer'
import type { McpServerSummary } from '@/types'

const servers = ref<McpServerSummary[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref('')

onMounted(() => fetchServers())

function onStatusChange() {
  currentPage.value = 1
  fetchServers()
}

async function fetchServers() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: currentPage.value,
      page_size: pageSize,
    }
    if (statusFilter.value) params.status = statusFilter.value
    const res = await getMyMcpServers(params)
    const data = res.data.data
    servers.value = data.list
    total.value = data.total
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function handleSubmit(id: number) {
  try {
    await submitMcpServer(id)
    ElMessage.success('已提交审核')
    await fetchServers()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function handleWithdraw(id: number) {
  try {
    await withdrawMcpServer(id)
    ElMessage.success('已撤回')
    await fetchServers()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function handleDelete(id: number) {
  try {
    await deleteMcpServer(id)
    ElMessage.success('已删除')
    await fetchServers()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

function getStatusType(status: string) {
  switch (status) {
    case 'published': return 'success'
    case 'pending_review': return 'warning'
    case 'draft': return 'info'
    case 'rejected': return 'danger'
    case 'archived': return 'info'
    default: return 'info'
  }
}

function getStatusText(status: string) {
  switch (status) {
    case 'published': return '已发布'
    case 'pending_review': return '待审核'
    case 'draft': return '草稿'
    case 'rejected': return '已拒绝'
    case 'archived': return '已下架'
    default: return status
  }
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
.resource-item {
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
.resource-info {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}
.resource-info:hover .resource-title {
  color: #409eff;
}
.resource-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.resource-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.2s;
}
.resource-desc {
  color: #606266;
  font-size: 14px;
  margin: 0 0 8px;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.resource-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #909399;
  flex-wrap: wrap;
}
.resource-meta span {
  display: flex;
  align-items: center;
  gap: 3px;
}
.resource-actions {
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
