<template>
  <div class="reports-view">
    <h2>举报处理</h2>
    <el-form :inline="true" @submit.prevent="loadReports">
      <el-form-item label="状态">
        <el-select v-model="status" clearable placeholder="全部" @change="loadReports">
          <el-option label="待处理" value="pending" />
          <el-option label="已处理" value="resolved" />
          <el-option label="已驳回" value="dismissed" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="loadReports">搜索</el-button>
      </el-form-item>
    </el-form>
    <el-table :data="reports" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="举报人" width="100">
        <template #default="{ row }">{{ row.reporter_id }}</template>
      </el-table-column>
      <el-table-column label="目标" width="100">
        <template #default="{ row }">{{ row.target_type }}</template>
      </el-table-column>
      <el-table-column prop="reason" label="原因" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'pending' ? 'warning' : row.status === 'resolved' ? 'success' : 'info'">
            {{ row.status === 'pending' ? '待处理' : row.status === 'resolved' ? '已处理' : '已驳回' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150">
        <template #default="{ row }">
          <el-button size="small" @click="openReport(row)">处理</el-button>
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
    <el-drawer v-model="drawerVisible" title="举报详情" size="50%">
      <div v-if="selectedReport" v-loading="detailLoading">
        <h3>举报信息</h3>
        <p><strong>举报人:</strong> {{ selectedReport.reporter_name }} (ID: {{ selectedReport.reporter_id }})</p>
        <p><strong>原因:</strong> {{ selectedReport.reason }}</p>
        <p><strong>描述:</strong> {{ selectedReport.description }}</p>
        <h3 style="margin-top: 20px">目标内容</h3>
        <p><strong>类型:</strong> {{ selectedReport.target.type }}</p>
        <p v-if="selectedReport.target.title"><strong>标题:</strong> {{ selectedReport.target.title }}</p>
        <p v-if="selectedReport.target.content"><strong>内容:</strong> {{ selectedReport.target.content }}</p>
        <p><strong>作者:</strong> {{ selectedReport.target.author_name }} (ID: {{ selectedReport.target.author_id }})</p>
        <p><strong>状态:</strong> <el-tag :type="selectedReport.target.hidden ? 'danger' : 'success'">{{ selectedReport.target.hidden ? '隐藏' : '正常' }}</el-tag></p>
        <p v-if="selectedReport.target.parent_url"><strong>链接:</strong> <a :href="selectedReport.target.parent_url" target="_blank">{{ selectedReport.target.parent_url }}</a></p>
        <div style="margin-top: 20px">
          <el-input v-model="resolveResult" type="textarea" :rows="3" placeholder="处理结果（可选）" />
          <div style="margin-top: 10px">
            <el-button type="danger" @click="resolve('hide')" :disabled="selectedReport.status !== 'pending'">隐藏内容</el-button>
            <el-button type="success" @click="resolve('unhide')" :disabled="selectedReport.status !== 'pending'">恢复内容</el-button>
            <el-button @click="resolve('dismiss')" :disabled="selectedReport.status !== 'pending'">驳回举报</el-button>
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminReports, getAdminReport, resolveAdminReport, type AdminReport, type AdminReportDetail } from '@/api/admin'

const reports = ref<AdminReport[]>([])
const status = ref('pending')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const drawerVisible = ref(false)
const detailLoading = ref(false)
const selectedReport = ref<AdminReportDetail | null>(null)
const resolveResult = ref('')

async function loadReports() {
  loading.value = true
  try {
    const res = await getAdminReports({ status: status.value, page: page.value, page_size: pageSize.value })
    reports.value = res.data.data.list
    total.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  loadReports()
}

async function openReport(row: AdminReport) {
  drawerVisible.value = true
  detailLoading.value = true
  selectedReport.value = null
  resolveResult.value = ''
  try {
    const res = await getAdminReport(row.id)
    selectedReport.value = res.data.data
  } catch {
    ElMessage.error('加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function resolve(action: string) {
  if (!selectedReport.value) return
  try {
    await ElMessageBox.confirm(`确认${action === 'hide' ? '隐藏' : action === 'unhide' ? '恢复' : '驳回'}？`, '操作确认')
    await resolveAdminReport(selectedReport.value.id, { action, result: resolveResult.value })
    ElMessage.success('处理成功')
    drawerVisible.value = false
    await loadReports()
  } catch {
    // cancelled
  }
}

onMounted(loadReports)
</script>

<style scoped>
.reports-view {
  padding: 20px;
}
</style>
