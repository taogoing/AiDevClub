<template>
  <div class="page-container">
    <h2>{{ isEdit ? '编辑 Skill' : '发布 Skill' }}</h2>
    <el-form :model="form" label-position="top" v-loading="pageLoading">
      <el-form-item label="名称">
        <el-input v-model="form.name" placeholder="请输入 Skill 名称" maxlength="200" show-word-limit />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入描述" maxlength="1000" />
      </el-form-item>
      <el-form-item label="仓库地址">
        <el-input v-model="form.repo_url" placeholder="https://github.com/..." />
      </el-form-item>
      <el-form-item label="标签">
        <el-select
          v-model="form.tag_ids"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="选择或输入标签"
          style="width: 100%"
          @change="onTagChange"
        >
          <el-option v-for="tag in availableTags" :key="tag.id" :label="tag.name" :value="tag.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="ZIP 文件">
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :limit="1"
          accept=".zip"
          :on-change="handleFileChange"
          :on-remove="handleFileRemove"
        >
          <el-button type="primary">选择 ZIP 文件</el-button>
          <template #tip>
            <div class="el-upload__tip">仅支持 .zip 文件</div>
          </template>
        </el-upload>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="handleSubmit('published')" :loading="submitting">发布</el-button>
        <el-button @click="handleSubmit('draft')" :loading="submitting">保存草稿</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'
import { createSkill, updateSkill, getSkill, uploadSkillZip } from '@/api/skill'
import { getTags } from '@/api/tag'
import type { Tag } from '@/types'

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => !!route.params.id)
const availableTags = ref<Tag[]>([])
const pageLoading = ref(false)
const submitting = ref(false)
const zipFile = ref<File | null>(null)

const form = ref({
  name: '',
  description: '',
  repo_url: '',
  tag_ids: [] as number[],
  tag_names: [] as string[],
})

onMounted(async () => {
  try {
    const res = await getTags()
    availableTags.value = res.data.data
  } catch { /* ignore */ }

  if (isEdit.value) {
    pageLoading.value = true
    try {
      const res = await getSkill(Number(route.params.id))
      const data = res.data.data
      form.value = {
        name: data.name,
        description: data.description,
        repo_url: data.repo_url,
        tag_ids: data.tags.map((t) => t.id),
        tag_names: [],
      }
    } catch (e: unknown) {
      ElMessage.error((e as Error).message)
    } finally {
      pageLoading.value = false
    }
  }
})

function onTagChange(ids: number[]) {
  const existingIds = new Set(availableTags.value.map((t) => t.id))
  form.value.tag_names = ids.filter((id) => !existingIds.has(id)).map((id) => String(id))
  form.value.tag_ids = ids.filter((id) => existingIds.has(id))
}

function handleFileChange(file: UploadFile) {
  zipFile.value = file.raw || null
}

function handleFileRemove() {
  zipFile.value = null
}

async function handleSubmit(status: string) {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入名称')
    return
  }
  submitting.value = true
  try {
    const payload: Record<string, unknown> = {
      name: form.value.name,
      description: form.value.description,
      repo_url: form.value.repo_url,
      tag_ids: form.value.tag_ids,
      tag_names: form.value.tag_names,
      status,
    }

    let id: number
    if (isEdit.value) {
      id = Number(route.params.id)
      await updateSkill(id, payload)
    } else {
      const res = await createSkill(payload)
      id = res.data.data.id
    }

    if (zipFile.value) {
      await uploadSkillZip(id, zipFile.value)
    }

    ElMessage.success(status === 'published' ? '发布成功' : '已保存草稿')
    router.push('/skills')
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
h2 {
  margin-bottom: 24px;
}
</style>
