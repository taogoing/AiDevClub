<template>
  <div class="dashboard-view">
    <h2>数据看板</h2>
    <el-row :gutter="20" v-loading="loading">
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="用户总数" :value="data?.total_users ?? 0" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="文章总数" :value="data?.total_articles ?? 0" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="Skill 总数" :value="data?.total_skills ?? 0" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="MCP Server 总数" :value="data?.total_mcp_servers ?? 0" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="待审核 Skill" :value="data?.pending_skills ?? 0" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="待审核 MCP Server" :value="data?.pending_mcp_servers ?? 0" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="待处理举报" :value="data?.pending_reports ?? 0" />
        </el-card>
      </el-col>
    </el-row>
    <el-alert v-if="error" :title="error" type="error" show-icon style="margin-top: 20px" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAdminDashboard, type AdminDashboard } from '@/api/admin'

const data = ref<AdminDashboard | null>(null)
const loading = ref(false)
const error = ref('')

async function loadDashboard() {
  loading.value = true
  error.value = ''
  try {
    const res = await getAdminDashboard()
    data.value = res.data.data
  } catch (e: any) {
    error.value = e.response?.data?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<style scoped>
.dashboard-view {
  padding: 20px;
}
</style>
