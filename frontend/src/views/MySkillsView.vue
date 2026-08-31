<template>
  <div class="page-container">
    <div class="page-header">
      <h2>我的 Skills</h2>
      <el-button type="primary" @click="$router.push('/skills/new')">
        <el-icon><Plus /></el-icon> 发布 Skill
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
      <div v-for="skill in skills" :key="skill.id" class="resource-item">
        <div class="resource-info" @click="$router.push(`/skills/${skill.id}`)">
          <div class="resource-title-row">
            <el-tag :type="getStatusType(skill.status)" size="small">{{ getStatusText(skill.status) }}</el-tag>
            <h3 class="resource-title">{{ skill.name }}</h3>
          </div>
          <p class="resource-desc">{{ skill.description || '暂无描述' }}</p>
          <div class="resource-meta">
            <el-tag v-for="tag in skill.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
            <span><el-icon><View /></el-icon> {{ skill.views }}</span>
            <span><el-icon><ChatDotRound /></el-icon> {{ skill.comments_count }}</span>
          </div>
        </div>
        <div class="resource-actions">
          <el-button size="small" @click="$router.push(`/skills/${skill.id}/edit`)">编辑</el-button>
          <el-button v-if="skill.status === 'draft' || skill.status === 'rejected' || skill.status === 'archived'" size="small" type="success" @click="handleSubmit(skill.id)">提交审核</el-button>
          <el-button v-if="skill.status === 'pending_review'" size="small" @click="handleWithdraw(skill.id)">撤回</el-button>
          <el-popconfirm title="确定删除这个 Skill？" @confirm="handleDelete(skill.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </div>
      </div>
      <el-empty v-if="!loading && !skills.length" description="暂无 Skill" />
    </div>

    <div class="pagination-wrap" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="fetchSkills"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, View, ChatDotRound } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getMySkills, submitSkill, withdrawSkill, deleteSkill } from '@/api/skill'
import type { SkillSummary } from '@/types'

const skills = ref<SkillSummary[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref('')

onMounted(() => fetchSkills())

function onStatusChange() {
  currentPage.value = 1
  fetchSkills()
}

async function fetchSkills() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: currentPage.value,
      page_size: pageSize,
    }
    if (statusFilter.value) params.status = statusFilter.value
    const res = await getMySkills(params)
    const data = res.data.data
    skills.value = data.list
    total.value = data.total
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function handleSubmit(id: number) {
  try {
    await submitSkill(id)
    ElMessage.success('已提交审核')
    await fetchSkills()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function handleWithdraw(id: number) {
  try {
    await withdrawSkill(id)
    ElMessage.success('已撤回')
    await fetchSkills()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function handleDelete(id: number) {
  try {
    await deleteSkill(id)
    ElMessage.success('已删除')
    await fetchSkills()
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
