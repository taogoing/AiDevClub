<template>
  <div class="tag-management">
    <el-card class="page-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">标签管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            创建标签
          </el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-input
          v-model="keyword"
          placeholder="搜索标签名称"
          clearable
          style="width: 240px"
          @input="debouncedLoadTags"
        />
        <el-select v-model="status" style="width: 140px" @change="loadTags">
          <el-option label="全部状态" value="all" />
          <el-option label="已启用" value="enabled" />
          <el-option label="已禁用" value="disabled" />
        </el-select>
      </div>

      <el-table
        :data="tags"
        v-loading="loading"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="usage_count" label="使用次数" width="100" align="center" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
              {{ row.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="showEditDialog(row)">
              编辑
            </el-button>
            <el-button
              :type="row.enabled ? 'danger' : 'success'"
              link
              size="small"
              @click="toggleTag(row)"
            >
              {{ row.enabled ? '禁用' : '启用' }}
            </el-button>
            <el-button type="danger" link size="small" @click="deleteTag(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadTags"
          @current-change="loadTags"
        />
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editingTag ? '编辑标签' : '创建标签'"
      width="480px"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="80px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入标签名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="请输入标签描述（可选）"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { getAdminTags, createTag, updateTag, deleteTag as apiDeleteTag } from '@/api/adminTag'
import type { AdminTag } from '@/api/adminTag'

const tags = ref<AdminTag[]>([])
const keyword = ref('')
const status = ref<'all' | 'enabled' | 'disabled'>('all')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)

const dialogVisible = ref(false)
const editingTag = ref<AdminTag | null>(null)
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = ref({
  name: '',
  description: '',
})

const rules: FormRules = {
  name: [
    { required: true, message: '请输入标签名称', trigger: 'blur' },
    { max: 64, message: '名称不能超过64个字符', trigger: 'blur' },
  ],
}

let debounceTimer: ReturnType<typeof setTimeout> | null = null
function debouncedLoadTags() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    page.value = 1
    loadTags()
  }, 300)
}

const loadTags = async () => {
  loading.value = true
  try {
    const res = await getAdminTags({
      keyword: keyword.value,
      status: status.value,
      page: page.value,
      page_size: pageSize.value,
    })
    const data = res.data.data
    tags.value = data.items || []
    total.value = data.total
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  editingTag.value = null
  form.value = { name: '', description: '' }
  dialogVisible.value = true
}

const showEditDialog = (tag: AdminTag) => {
  editingTag.value = tag
  form.value = { name: tag.name, description: tag.description || '' }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()

  submitting.value = true
  try {
    if (editingTag.value) {
      await updateTag(editingTag.value.id, {
        name: form.value.name,
        description: form.value.description,
      })
      ElMessage.success('更新成功')
    } else {
      await createTag({
        name: form.value.name,
        description: form.value.description,
      })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadTags()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const toggleTag = async (tag: AdminTag) => {
  const action = tag.enabled ? '禁用' : '启用'
  try {
    await ElMessageBox.confirm(
      `确定要${action}标签「${tag.name}」吗？`,
      '确认操作',
      { type: 'warning' }
    )
    await updateTag(tag.id, { enabled: !tag.enabled })
    ElMessage.success(`${action}成功`)
    loadTags()
  } catch {
    // cancelled
  }
}

const deleteTag = async (tag: AdminTag) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除标签「${tag.name}」吗？此操作不可恢复！`,
      '确认删除',
      { type: 'error', confirmButtonText: '确定删除', cancelButtonText: '取消' }
    )
    await apiDeleteTag(tag.id)
    ElMessage.success('删除成功')
    loadTags()
  } catch (e: any) {
    if (e !== 'cancel' && e?.message) {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

onMounted(loadTags)
</script>

<style scoped>
.tag-management {
  max-width: 1200px;
}

.page-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
