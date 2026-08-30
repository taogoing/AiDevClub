<template>
  <div class="install-panel">
    <div v-if="installations.length" class="client-tabs">
      <button
        v-for="item in installations"
        :key="item.client"
        type="button"
        :class="{ active: activeClient === item.client }"
        @click="activeClient = item.client"
      >
        {{ clientLabel(item.client) }}
      </button>
    </div>

    <div v-if="activeInstallation" class="panel-body">
      <div class="panel-toolbar">
        <div class="mode-tabs">
          <button type="button" :class="{ active: mode === 'command' }" @click="mode = 'command'">命令</button>
          <button type="button" :class="{ active: mode === 'json' }" @click="mode = 'json'">JSON</button>
        </div>
        <div v-if="mode === 'json'" class="os-tabs">
          <button type="button" :class="{ active: os === 'unix' }" @click="os = 'unix'">macOS / Linux</button>
          <button type="button" :class="{ active: os === 'windows' }" @click="os = 'windows'">Windows</button>
        </div>
      </div>

      <div class="code-box">
        <div class="code-header">
          <span>{{ mode === 'command' ? 'terminal' : 'JSON' }}</span>
          <el-button text type="primary" @click="copyContent">复制</el-button>
        </div>
        <pre><code>{{ displayedContent || '该客户端暂未提供此配置' }}</code></pre>
      </div>
      <el-alert
        title="配置中的密码和 Token 仅为占位符，请复制后在本地替换；本站不会收集或保存你的密钥。"
        type="info"
        :closable="false"
        show-icon
      />
    </div>
    <el-empty v-else description="暂未提供安装配置" :image-size="72" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { McpInstallation } from '@/types'

const props = defineProps<{ installations: McpInstallation[] }>()
const activeClient = ref('')
const mode = ref<'command' | 'json'>('command')
const os = ref<'unix' | 'windows'>('unix')

watch(
  () => props.installations,
  (items) => {
    if (!items.some((item) => item.client === activeClient.value)) {
      activeClient.value = items[0]?.client || ''
    }
  },
  { immediate: true },
)

const activeInstallation = computed(() =>
  props.installations.find((item) => item.client === activeClient.value),
)

const displayedContent = computed(() => {
  const item = activeInstallation.value
  if (!item) return ''
  if (mode.value === 'command') return item.command || ''
  const config = os.value === 'windows' ? (item.windows_config || item.config) : item.config
  return config ? JSON.stringify(config, null, 2) : ''
})

function clientLabel(client: string) {
  const labels: Record<string, string> = {
    cursor: 'Cursor',
    'claude-code': 'Claude Code',
    codex: 'Codex',
    trae: 'Trae CN',
    'trae-global': 'Trae Global',
    cline: 'Cline',
    windsurf: 'Windsurf',
  }
  return labels[client] || client
}

async function copyContent() {
  if (!displayedContent.value) {
    ElMessage.warning('当前没有可复制的配置')
    return
  }
  await navigator.clipboard.writeText(displayedContent.value)
  ElMessage.success('已复制')
}
</script>

<style scoped>
.install-panel { border: 1px solid #dfe4ec; border-radius: 10px; overflow: hidden; }
.client-tabs { display: flex; overflow-x: auto; border-bottom: 1px solid #ebeef5; }
.client-tabs button { padding: 13px 17px; white-space: nowrap; border: 0; border-bottom: 2px solid transparent; color: #606266; background: #fff; cursor: pointer; }
.client-tabs button.active { color: #409eff; border-bottom-color: #409eff; background: #f8fbff; font-weight: 600; }
.panel-body { padding: 16px; }
.panel-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.mode-tabs, .os-tabs { display: flex; gap: 4px; padding: 3px; border-radius: 7px; background: #f2f4f7; }
.mode-tabs button, .os-tabs button { padding: 6px 14px; border: 0; border-radius: 5px; color: #606266; background: transparent; cursor: pointer; }
.mode-tabs button.active, .os-tabs button.active { color: #409eff; background: #fff; box-shadow: 0 1px 3px rgb(0 0 0 / 8%); }
.code-box { margin-bottom: 14px; border: 1px solid #ebeef5; border-radius: 8px; overflow: hidden; background: #fafcff; }
.code-header { display: flex; align-items: center; justify-content: space-between; padding: 5px 12px; border-bottom: 1px solid #ebeef5; color: #909399; font-size: 12px; }
pre { min-height: 90px; margin: 0; padding: 16px; overflow: auto; white-space: pre-wrap; word-break: break-word; font: 13px/1.65 Consolas, monospace; }
@media (max-width: 640px) { .panel-toolbar { align-items: flex-start; flex-direction: column; } }
</style>
