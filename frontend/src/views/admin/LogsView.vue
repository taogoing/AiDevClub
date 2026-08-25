<template>
  <div class="logs-view">
    <h2>操作日志</h2>
    <el-form :inline="true" @submit.prevent="loadLogs">
      <el-form-item label="操作">
        <el-select v-model="action" clearable placeholder="全部" @change="loadLogs">
          <el-option label="审核通过" value="approve_resource" />
          <el-option label="审核拒绝" value="reject_resource" />
          <el-option label="隐藏内容" value="hide_content" />
          <el-option label="恢复内容" value="unhide_content" />
          <el-option label="创建标签" value="create_tag" />
          <el-option label="更新标签" value="update_tag" />
          <el-option label="发布公告" value="create_announcement" />
          <el-option label="处理举报" value="resolve_report" />
          <el-option label="修改角色" value="update_user_role" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="loadLogs">搜索</el-button>
      </el-form-item>
    </el-form>
    <el-table :data="logs" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="管理员" width="120">
        <template #default="{ row }">{{ row.admin?.nickname || row.admin_id }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120">
        <template #default="{ row }">{{ getActionText(row.action) }}</template>
      </el-table-column>
      <el-table-column label="目标" width="120">
        <template #default="{ row }">{{ row.target_type }} #{{ row.target_id }}</template>
      </el-table-column>
      <el-table-column label="详情">
        <template #default="{ row }">{{ formatDetail(row.detail) }}</template>
      </el-table-column>
      <el-table-column prop="created_at" label="时间" width="180">
        <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getAdminLogs, type AdminLogItem } from '@/api/admin'

const logs = ref<AdminLogItem[]>([])
const action = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)

function getActionText(action: string) {
  const map: Record<string, string> = {
    approve_resource: '审核通过',
    reject_resource: '审核拒绝',
    hide_content: '隐藏内容',
    unhide_content: '恢复内容',
    create_tag: '创建标签',
    update_tag: '更新标签',
    create_announcement: '发布公告',
    resolve_report: '处理举报',
    update_user_role: '修改角色',
  }
  return map[action] || action
}

function formatDetail(detail: unknown): string {
  if (detail === null || detail === undefined) return ''
  if (typeof detail === 'string') return detail
  try {
    return JSON.stringify(detail, null, 2)
  } catch {
    return String(detail)
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN')
}

async function loadLogs() {
  loading.value = true
  try {
    const res = await getAdminLogs({ action: action.value, page: page.value, page_size: pageSize.value })
    logs.value = res.data.data.list
    total.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  loadLogs()
}

onMounted(loadLogs)
</script>

<style scoped>
.logs-view {
  padding: 20px;
}
</style>
