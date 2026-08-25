<template>
  <div class="users-view">
    <h2>用户管理</h2>
    <el-form :inline="true" @submit.prevent="loadUsers">
      <el-form-item label="关键词">
        <el-input v-model="keyword" placeholder="邮箱/昵称" clearable @clear="loadUsers" />
      </el-form-item>
      <el-form-item label="角色">
        <el-select v-model="role" clearable placeholder="全部" @change="loadUsers">
          <el-option label="用户" value="user" />
          <el-option label="管理员" value="admin" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="loadUsers">搜索</el-button>
      </el-form-item>
    </el-form>
    <el-table :data="users" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="email" label="邮箱" />
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column prop="role" label="角色" width="100">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'info'">{{ row.role }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150">
        <template #default="{ row }">
          <el-button v-if="row.role_mutable" size="small" @click="changeRole(row, row.role === 'admin' ? 'user' : 'admin')">
            {{ row.role === 'admin' ? '降级' : '升级' }}
          </el-button>
          <span v-else class="text-gray">不可修改</span>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminUsers, updateAdminUserRole, type AdminUser } from '@/api/admin'

const users = ref<AdminUser[]>([])
const keyword = ref('')
const role = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)

async function loadUsers() {
  loading.value = true
  try {
    const res = await getAdminUsers({ keyword: keyword.value, role: role.value as any, page: page.value, page_size: pageSize.value })
    users.value = res.data.data.list
    total.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  loadUsers()
}

async function changeRole(row: AdminUser, newRole: 'user' | 'admin') {
  try {
    await ElMessageBox.confirm(`确认将 ${row.nickname} 的角色改为 ${newRole}？`, '修改角色')
    await updateAdminUserRole(row.id, newRole)
    ElMessage.success('修改成功')
    await loadUsers()
  } catch {
    // cancelled
  }
}

onMounted(loadUsers)
</script>

<style scoped>
.users-view {
  padding: 20px;
}
.text-gray {
  color: #999;
  font-size: 12px;
}
</style>
