<template>
  <div class="page-container">
    <h2>{{ isEdit ? '编辑文章' : '发布文章' }}</h2>
    <el-form :model="form" label-position="top" v-loading="pageLoading">
      <el-form-item label="标题">
        <el-input v-model="form.title" placeholder="请输入标题" maxlength="200" show-word-limit />
      </el-form-item>
      <el-form-item label="摘要">
        <el-input v-model="form.summary" type="textarea" :rows="2" placeholder="请输入摘要（可选）" maxlength="500" />
      </el-form-item>
      <el-form-item label="分类">
        <el-select v-model="form.category_id" placeholder="选择分类">
          <el-option v-for="cat in categories" :key="cat.id" :label="cat.name" :value="cat.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="标签">
        <div class="tag-selector">
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
        </div>
      </el-form-item>
      <el-form-item label="正文">
        <MdEditor
          v-model="form.content"
          :toolbars="toolbars"
          @onUploadImg="handleImageUpload"
          style="height: 500px"
        />
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
import { MdEditor } from 'md-editor-v3'
import type { ToolbarNames } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import { createArticle, updateArticle, getArticle, uploadArticleImage } from '@/api/article'
import { getCategories } from '@/api/category'
import { getTags } from '@/api/tag'
import type { Category, Tag, ArticleForm } from '@/types'

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => !!route.params.id)
const categories = ref<Category[]>([])
const availableTags = ref<Tag[]>([])
const pageLoading = ref(false)
const submitting = ref(false)

const form = ref<ArticleForm>({
  title: '',
  summary: '',
  content: '',
  category_id: 0,
  status: 'published',
  tag_ids: [],
  tag_names: [],
})

const toolbars: ToolbarNames[] = [
  'bold', 'underline', 'italic', 'strikeThrough', '-',
  'title', 'sub', 'sup', 'quote', 'unorderedList', 'orderedList', 'task', '-',
  'code', 'link', 'image', 'table', 'mermaid', 'katex', '-',
  'revoke', 'next', '=',
  'pageFullscreen', 'preview', 'htmlPreview',
]

onMounted(async () => {
  try {
    const [catRes, tagRes] = await Promise.all([getCategories(), getTags()])
    categories.value = catRes.data.data
    availableTags.value = tagRes.data.data
  } catch { /* ignore */ }

  if (isEdit.value) {
    pageLoading.value = true
    try {
      const res = await getArticle(Number(route.params.id))
      const data = res.data.data
      form.value = {
        title: data.title,
        summary: data.summary,
        content: data.content,
        category_id: data.category_id,
        status: 'published',
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
  form.value.tag_names = ids
    .filter((id) => !existingIds.has(id))
    .map((id) => String(id))
  form.value.tag_ids = ids.filter((id) => existingIds.has(id))
}

async function handleImageUpload(files: File[]) {
  const urls: string[] = []
  for (const file of files) {
    try {
      const res = await uploadArticleImage(file)
      urls.push(res.data.data.url)
    } catch (e: unknown) {
      ElMessage.error((e as Error).message)
    }
  }
  return urls
}

async function handleSubmit(status: 'draft' | 'published') {
  if (!form.value.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  if (!form.value.category_id) {
    ElMessage.warning('请选择分类')
    return
  }
  submitting.value = true
  try {
    form.value.status = status
    if (isEdit.value) {
      await updateArticle(Number(route.params.id), form.value)
    } else {
      await createArticle(form.value)
    }
    ElMessage.success(status === 'published' ? '发布成功' : '已保存草稿')
    router.push('/')
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

.tag-selector {
  width: 100%;
}
</style>
