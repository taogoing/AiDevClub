<template>
  <div class="page-container">
    <h2>{{ isEdit ? '编辑 MCP' : '发布 MCP' }}</h2>
    <el-form :model="form" label-position="top" v-loading="pageLoading">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="例如：MySQL" maxlength="100" show-word-limit />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" placeholder="简要说明这个 MCP 能做什么" maxlength="500" show-word-limit />
      </el-form-item>
      <el-form-item label="Git 仓库" required>
        <el-input v-model="form.repo_url" placeholder="https://github.com/owner/repository" />
      </el-form-item>

      <el-form-item label="安装配置" required>
        <div class="install-editors">
          <div v-for="(item, index) in form.installations" :key="index" class="install-editor">
            <div class="editor-header">
              <strong>客户端配置 {{ index + 1 }}</strong>
              <el-button v-if="form.installations.length > 1" text type="danger" @click="removeInstallation(index)">删除</el-button>
            </div>
            <el-select v-model="item.client" placeholder="选择客户端" style="width: 220px">
              <el-option v-for="option in clientOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
            <el-input v-model="item.command" class="editor-field" placeholder="安装命令，例如：npx -y ..." />
            <el-input v-model="item.config" class="editor-field" type="textarea" :rows="7" placeholder="macOS / Linux JSON 配置（可选）" />
            <el-input v-model="item.windows_config" class="editor-field" type="textarea" :rows="7" placeholder="Windows JSON 配置（可选；留空时使用上面的配置）" />
          </div>
          <el-button plain type="primary" @click="addInstallation">添加客户端配置</el-button>
        </div>
      </el-form-item>

      <el-form-item label="补充说明（Markdown，可选）">
        <el-input v-model="form.readme" type="textarea" :rows="9" placeholder="补充使用说明、环境变量或注意事项" />
      </el-form-item>
      <el-form-item label="标签（可选）">
        <el-select v-model="form.tag_ids" multiple filterable placeholder="选择标签" style="width: 100%">
          <el-option v-for="tag in availableTags" :key="tag.id" :label="tag.name" :value="tag.id" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="submitting" @click="handleSubmit(true)">保存并提交审核</el-button>
        <el-button :loading="submitting" @click="handleSubmit(false)">保存草稿</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createMcpServer, getMcpServer, submitMcpServer, updateMcpServer } from '@/api/mcpServer'
import { getTags } from '@/api/tag'
import type { McpInstallation, Tag } from '@/types'

type InstallationEditor = {
  client: McpInstallation['client']
  command: string
  config: string
  windows_config: string
}

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => !!route.params.id)
const availableTags = ref<Tag[]>([])
const pageLoading = ref(false)
const submitting = ref(false)

const clientOptions = [
  { value: 'cursor', label: 'Cursor' },
  { value: 'claude-code', label: 'Claude Code' },
  { value: 'codex', label: 'Codex' },
  { value: 'trae', label: 'Trae CN' },
  { value: 'trae-global', label: 'Trae Global' },
  { value: 'cline', label: 'Cline' },
  { value: 'windsurf', label: 'Windsurf' },
] as const

function blankInstallation(): InstallationEditor {
  return { client: 'cursor', command: '', config: '', windows_config: '' }
}

const form = ref({
  name: '',
  description: '',
  repo_url: '',
  installations: [blankInstallation()],
  readme: '',
  tag_ids: [] as number[],
})

onMounted(async () => {
  try {
    const res = await getTags()
    availableTags.value = res.data.data
  } catch { /* 标签加载失败不阻止保存 */ }

  if (!isEdit.value) return
  pageLoading.value = true
  try {
    const res = await getMcpServer(Number(route.params.id))
    const data = res.data.data
    form.value = {
      name: data.name,
      description: data.description,
      repo_url: data.repo_url,
      installations: data.installations.length
        ? data.installations.map((item) => ({
            client: item.client,
            command: item.command || '',
            config: item.config ? JSON.stringify(item.config, null, 2) : '',
            windows_config: item.windows_config ? JSON.stringify(item.windows_config, null, 2) : '',
          }))
        : [blankInstallation()],
      readme: data.readme,
      tag_ids: data.tags.map((tag) => tag.id),
    }
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    pageLoading.value = false
  }
})

function addInstallation() {
  if (form.value.installations.length >= clientOptions.length) return ElMessage.warning('已添加全部支持的客户端')
  const used = new Set(form.value.installations.map((item) => item.client))
  const next = clientOptions.find((item) => !used.has(item.value))
  form.value.installations.push({ ...blankInstallation(), client: next?.value || 'cursor' })
}

function removeInstallation(index: number) {
  form.value.installations.splice(index, 1)
}

function parseConfig(value: string, label: string) {
  if (!value.trim()) return undefined
  try {
    const parsed = JSON.parse(value)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error()
    return parsed as Record<string, unknown>
  } catch {
    throw new Error(`${label}不是合法的 JSON 对象`)
  }
}

function buildInstallations(): McpInstallation[] {
  const clients = new Set<string>()
  return form.value.installations.map((item) => {
    if (clients.has(item.client)) throw new Error('同一个客户端只能配置一次')
    clients.add(item.client)
    const config = parseConfig(item.config, `${item.client} 的 macOS / Linux 配置`)
    const windowsConfig = parseConfig(item.windows_config, `${item.client} 的 Windows 配置`)
    if (!item.command.trim() && !config && !windowsConfig) throw new Error(`${item.client} 至少填写一种安装方式`)
    return {
      client: item.client,
      command: item.command.trim() || undefined,
      config,
      windows_config: windowsConfig,
    }
  })
}

async function handleSubmit(shouldSubmit: boolean) {
  if (!form.value.name.trim()) return ElMessage.warning('请输入名称')
  if (!/^https:\/\//i.test(form.value.repo_url.trim())) return ElMessage.warning('请输入有效的 HTTPS Git 仓库地址')

  submitting.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      description: form.value.description.trim(),
      repo_url: form.value.repo_url.trim(),
      installations: buildInstallations(),
      readme: form.value.readme,
      tag_ids: form.value.tag_ids,
      tag_names: [] as string[],
    }

    let id: number
    if (isEdit.value) {
      id = Number(route.params.id)
      await updateMcpServer(id, payload)
    } else {
      const res = await createMcpServer(payload)
      id = res.data.data.id
    }
    if (shouldSubmit) await submitMcpServer(id)

    ElMessage.success(shouldSubmit ? '已提交审核' : '已保存草稿')
    router.push(`/mcps/${id}`)
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
h2 { margin-bottom: 24px; }
.install-editors { width: 100%; display: flex; flex-direction: column; gap: 14px; }
.install-editor { padding: 16px; border: 1px solid #dfe4ec; border-radius: 9px; background: #fafcff; }
.editor-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.editor-field { margin-top: 10px; }
</style>
