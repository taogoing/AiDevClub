<template>
  <div class="assistant-shell">
    <el-button class="assistant-trigger" type="primary" plain @click="visible = !visible">
      <el-icon><ChatDotRound /></el-icon>
      <span>AI 助手</span>
      <el-tag size="small" type="success" effect="light">新</el-tag>
    </el-button>

    <transition name="assistant-popover">
      <section v-if="visible" class="assistant-panel" aria-label="AI 助手">
        <header class="assistant-header">
          <div class="assistant-heading">
            <div class="assistant-icon"><el-icon><MagicStick /></el-icon></div>
            <div>
              <strong>AI 助手</strong>
              <p>{{ contextLabel }}</p>
            </div>
          </div>
          <el-button text circle aria-label="关闭 AI 助手" @click="visible = false">
            <el-icon><Close /></el-icon>
          </el-button>
        </header>

        <div ref="messagesEl" class="assistant-messages" aria-live="polite">
          <div v-if="messages.length === 0" class="assistant-empty">
            <div class="empty-orb"><el-icon><ChatLineRound /></el-icon></div>
            <strong>问问帖子里的内容</strong>
            <p>我会基于已索引的文章回答，并给出参考章节。</p>
            <div class="suggestions">
              <button v-for="suggestion in suggestions" :key="suggestion" @click="question = suggestion">
                {{ suggestion }}
              </button>
            </div>
          </div>

          <article v-for="message in messages" :key="message.id" :class="['message', `message-${message.role}`]">
            <div v-if="message.role === 'assistant'" class="message-avatar"><el-icon><MagicStick /></el-icon></div>
            <div class="message-body">
              <p class="message-text">{{ message.content }}</p>
              <div v-if="message.citations?.length" class="citations">
                <span class="citation-label">参考内容</span>
                <button v-for="(citation, index) in message.citations" :key="`${message.id}-${index}`" class="citation">
                  {{ citation.title || '相关帖子' }}<span v-if="citation.heading_path"> · {{ formatHeading(citation.heading_path) }}</span>
                </button>
              </div>
            </div>
          </article>

          <div v-if="loading" class="message message-assistant">
            <div class="message-avatar"><el-icon><MagicStick /></el-icon></div>
            <div class="message-body typing"><i></i><i></i><i></i></div>
          </div>
        </div>

        <div class="assistant-composer">
          <el-input
            v-model="question"
            type="textarea"
            :rows="2"
            resize="none"
            maxlength="500"
            show-word-limit
            :disabled="loading"
            placeholder="输入你想了解的问题..."
            @keydown.enter.exact.prevent="submit"
          />
          <div class="composer-footer">
            <span>AI 可能会出错，请结合原文判断</span>
            <el-button type="primary" :loading="loading" :disabled="!question.trim()" @click="submit">
              <el-icon v-if="!loading"><Promotion /></el-icon> 发送
            </el-button>
          </div>
        </div>
      </section>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { ChatDotRound, ChatLineRound, Close, MagicStick, Promotion } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { askAiAssistant, type AiCitation } from '@/api/aiAssistant'

interface Message {
  id: number
  role: 'user' | 'assistant'
  content: string
  citations?: AiCitation[]
}

const props = withDefaults(defineProps<{ articleId?: number; contextLabel?: string }>(), {
  contextLabel: '全站文章知识库',
})

const visible = ref(false)
const question = ref('')
const loading = ref(false)
const messages = ref<Message[]>([])
const messagesEl = ref<HTMLElement | null>(null)
const suggestions = ['这篇文章的核心观点是什么？', '有哪些实践建议？', '能用简单的方式解释吗？']

watch(messages, () => nextTick(() => { messagesEl.value?.scrollTo({ top: messagesEl.value.scrollHeight, behavior: 'smooth' }) }), { deep: true })

async function submit() {
  const content = question.value.trim()
  if (!content || loading.value) return
  messages.value.push({ id: Date.now(), role: 'user', content })
  question.value = ''
  loading.value = true
  try {
    const response = await askAiAssistant({ question: content, ...(props.articleId ? { article_id: props.articleId } : {}) })
    const data = response.data.data
    messages.value.push({ id: Date.now() + 1, role: 'assistant', content: data.answer, citations: data.citations })
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : 'AI 助手暂时无法回答，请稍后重试'
    messages.value.push({ id: Date.now() + 1, role: 'assistant', content: `暂时没有得到答案：${message}` })
    ElMessage.warning('AI 助手请求失败')
  } finally {
    loading.value = false
  }
}

function formatHeading(path: string | string[]) {
  return Array.isArray(path) ? path.join(' / ') : path
}
</script>

<style scoped>
.assistant-shell { position: relative; }
.assistant-trigger { border-radius: 999px; padding: 8px 13px; font-weight: 600; }
.assistant-trigger .el-icon { margin-right: 5px; }
.assistant-trigger .el-tag { margin-left: 6px; border: 0; font-size: 10px; }
.assistant-panel { position: absolute; z-index: 1200; top: calc(100% + 12px); right: 0; width: min(380px, calc(100vw - 28px)); overflow: hidden; border: 1px solid #dfe8f3; border-radius: 18px; background: #fff; box-shadow: 0 18px 50px rgb(31 59 91 / 18%); }
.assistant-header { display: flex; align-items: center; justify-content: space-between; padding: 16px 17px; border-bottom: 1px solid #edf1f6; background: linear-gradient(135deg, #f6fbff, #fff); }
.assistant-heading { display: flex; align-items: center; gap: 10px; }
.assistant-icon, .empty-orb, .message-avatar { display: grid; place-items: center; color: #1689d7; background: #e9f6ff; }
.assistant-icon { width: 36px; height: 36px; border-radius: 11px; font-size: 19px; }
.assistant-heading strong { color: #1f3b5b; font-size: 16px; }
.assistant-heading p { margin-top: 2px; color: #8b9aab; font-size: 12px; }
.assistant-messages { min-height: 250px; max-height: 400px; overflow-y: auto; padding: 18px 16px 8px; background: #fbfdff; }
.assistant-empty { display: flex; flex-direction: column; align-items: center; padding: 20px 8px 14px; text-align: center; }
.empty-orb { width: 48px; height: 48px; margin-bottom: 12px; border-radius: 15px; font-size: 24px; }
.assistant-empty strong { color: #34495e; font-size: 15px; }
.assistant-empty p { max-width: 260px; margin: 5px 0 15px; color: #8b9aab; font-size: 12px; }
.suggestions { display: flex; flex-wrap: wrap; justify-content: center; gap: 7px; }
.suggestions button, .citation { border: 1px solid #dcebf7; border-radius: 999px; color: #2581ba; background: #f4faff; cursor: pointer; font-size: 12px; }
.suggestions button { padding: 6px 10px; }
.message { display: flex; gap: 8px; margin-bottom: 14px; }
.message-user { justify-content: flex-end; }
.message-body { max-width: 85%; }
.message-user .message-body { padding: 9px 12px; border-radius: 14px 14px 3px 14px; color: #fff; background: #1689d7; }
.message-assistant .message-body { padding: 9px 12px; border: 1px solid #e7eef5; border-radius: 3px 14px 14px; background: #fff; }
.message-avatar { flex: 0 0 27px; width: 27px; height: 27px; border-radius: 9px; font-size: 15px; }
.message-text { white-space: pre-wrap; word-break: break-word; color: #40566c; font-size: 13px; line-height: 1.65; }
.message-user .message-text { color: #fff; }
.citations { display: flex; flex-direction: column; gap: 5px; margin-top: 9px; padding-top: 8px; border-top: 1px solid #edf1f6; }
.citation-label { color: #94a3b5; font-size: 11px; }
.citation { width: fit-content; padding: 4px 8px; text-align: left; }
.typing { display: flex; align-items: center; gap: 4px; min-height: 26px; }
.typing i { width: 5px; height: 5px; border-radius: 50%; background: #8dc8eb; animation: blink 1s infinite ease-in-out; }
.typing i:nth-child(2) { animation-delay: .15s; }.typing i:nth-child(3) { animation-delay: .3s; }
@keyframes blink { 0%, 80%, 100% { opacity: .35; transform: translateY(0); } 40% { opacity: 1; transform: translateY(-3px); } }
.assistant-composer { padding: 11px 13px 13px; border-top: 1px solid #eaf0f6; background: #fff; }
.assistant-composer :deep(.el-textarea__inner) { padding: 9px 10px; border: 0; box-shadow: none; background: #f5f8fb; }
.composer-footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-top: 8px; }
.composer-footer span { color: #a0adba; font-size: 10px; }
.composer-footer .el-button { padding: 8px 13px; border-radius: 9px; }
.assistant-popover-enter-active, .assistant-popover-leave-active { transition: opacity .18s ease, transform .18s ease; transform-origin: top right; }
.assistant-popover-enter-from, .assistant-popover-leave-to { opacity: 0; transform: translateY(-6px) scale(.98); }
@media (max-width: 700px) { .assistant-panel { position: fixed; top: 64px; right: 14px; } }
</style>
