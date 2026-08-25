<template>
  <div class="announcements-view">
    <h2>公告管理</h2>
    <el-card style="margin-bottom: 20px">
      <h3>发布公告</h3>
      <el-form :model="form" label-width="80px">
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="公告标题" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="form.content" type="textarea" :rows="4" placeholder="公告内容" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="publish" :loading="submitting" :disabled="!form.title.trim() || !form.content.trim()">发布</el-button>
        </el-form-item>
      </el-form>
    </el-card>
    <h3>历史公告</h3>
    <el-table :data="announcements" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="title" label="标题" />
      <el-table-column prop="content" label="内容" show-overflow-tooltip />
      <el-table-column prop="created_at" label="发布时间" width="180">
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminAnnouncements, createAdminAnnouncement, type AdminAnnouncement } from '@/api/admin'

const announcements = ref<AdminAnnouncement[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const submitting = ref(false)
const form = ref({ title: '', content: '' })

async function loadAnnouncements() {
  loading.value = true
  try {
    const res = await getAdminAnnouncements({ page: page.value, page_size: pageSize.value })
    announcements.value = res.data.data.list
    total.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  loadAnnouncements()
}

async function publish() {
  if (!form.value.title.trim() || !form.value.content.trim()) {
    ElMessage.warning('请填写标题和内容')
    return
  }
  try {
    await ElMessageBox.confirm('发布后会向全部用户发送站内通知，确认继续？', '发布公告')
    submitting.value = true
    await createAdminAnnouncement({ title: form.value.title.trim(), content: form.value.content.trim() })
    ElMessage.success('发布成功')
    form.value = { title: '', content: '' }
    await loadAnnouncements()
  } catch {
    // cancelled
  } finally {
    submitting.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(loadAnnouncements)
</script>

<style scoped>
.announcements-view {
  padding: 20px;
}
</style>
