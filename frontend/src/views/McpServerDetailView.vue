<template>
  <div class="page-container" v-loading="loading">
    <template v-if="server">
      <div class="detail-header">
        <h1 class="detail-title">{{ server.name }}</h1>
        <el-tag :type="statusType">{{ statusLabel }}</el-tag>
      </div>
      <div class="detail-meta">
        <el-avatar :size="28" :src="server.author.avatar_url || undefined">
          {{ server.author.nickname?.charAt(0) || '?' }}
        </el-avatar>
        <span>{{ server.author.nickname }}</span>
        <el-tag v-for="tag in server.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
        <span class="meta-time">{{ formatTime(server.published_at) }}</span>
      </div>
      <div class="detail-stats">
        <span><el-icon><View /></el-icon> {{ server.views }}</span>
        <span><el-icon><Download /></el-icon> {{ server.downloads }}</span>
        <span><el-icon><ChatDotRound /></el-icon> {{ server.comments_count }}</span>
      </div>
      <div class="detail-description">
        <p>{{ server.description }}</p>
      </div>
      <div v-if="server.repo_url" class="detail-repo">
        <el-link :href="server.repo_url" target="_blank" type="primary">
          <el-icon><Link /></el-icon> 仓库地址
        </el-link>
      </div>

      <div v-if="tools.length" class="tools-section">
        <h3>Tools 清单</h3>
        <el-table :data="tools" border style="width: 100%">
          <el-table-column prop="name" label="名称" width="200" />
          <el-table-column prop="description" label="描述" />
        </el-table>
      </div>

      <div v-if="server.readme" class="readme-section">
        <h3>README</h3>
        <div class="readme-content" v-html="renderedReadme"></div>
      </div>

      <div class="detail-actions">
        <el-button
          v-if="server.zip_url"
          type="primary"
          @click="handleDownload"
        >
          <el-icon><Download /></el-icon> 下载 ({{ formatFileSize(server.file_size) }})
        </el-button>
        <el-button
          :type="server.liked ? 'primary' : 'default'"
          @click="handleLike"
        >
          <el-icon><CaretTop /></el-icon> {{ server.liked ? '已赞' : '点赞' }} {{ server.likes_count }}
        </el-button>
        <el-button
          :type="server.favorited ? 'warning' : 'default'"
          @click="handleFavorite"
        >
          <el-icon><Star /></el-icon> {{ server.favorited ? '已藏' : '收藏' }} {{ server.favorites_count }}
        </el-button>
        <template v-if="isAuthor">
          <el-button type="info" @click="$router.push(`/mcps/${server.id}/edit`)">编辑</el-button>
          <el-button
            v-if="server.status === 'draft' || server.status === 'rejected'"
            type="success"
            @click="handleSubmit"
          >
            提交审核
          </el-button>
          <el-button
            v-if="server.status === 'pending_review'"
            type="warning"
            @click="handleWithdraw"
          >
            撤回
          </el-button>
          <el-button
            v-if="server.status === 'published'"
            type="danger"
            @click="handleArchive"
          >
            下架
          </el-button>
        </template>
      </div>

      <div class="comment-section">
        <h3>评论 ({{ server.comments_count }})</h3>
        <div v-if="auth.isLoggedIn" class="comment-input">
          <el-input
            v-model="commentContent"
            type="textarea"
            :rows="3"
            placeholder="写下你的评论..."
          />
          <el-button type="primary" style="margin-top: 8px" @click="submitComment" :disabled="!commentContent.trim()">
            发表评论
          </el-button>
        </div>
        <el-alert v-else type="info" :closable="false" style="margin-bottom: 16px">
          <router-link to="/login">登录</router-link> 后参与评论
        </el-alert>
        <div class="comment-list">
          <div v-for="comment in comments" :key="comment.id" class="comment-item">
            <div class="comment-main">
              <el-avatar :size="36" :src="comment.author.avatar_url || undefined">
                {{ comment.author.nickname?.charAt(0) || '?' }}
              </el-avatar>
              <div class="comment-body">
                <div class="comment-header">
                  <span class="comment-author">{{ comment.author.nickname }}</span>
                  <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
                </div>
                <div class="comment-content">{{ comment.content }}</div>
                <div class="comment-actions">
                  <el-button text size="small" @click="toggleLikeComment(comment)">
                    <el-icon><CaretTop /></el-icon> {{ comment.likes_count }}
                  </el-button>
                  <el-button text size="small" @click="showReply(comment.id)">回复</el-button>
                </div>
                <div v-if="replyingTo === comment.id" class="reply-input">
                  <el-input v-model="replyContent" type="textarea" :rows="2" placeholder="写下你的回复..." />
                  <div class="reply-actions">
                    <el-button size="small" @click="replyingTo = null">取消</el-button>
                    <el-button size="small" type="primary" @click="handleReply(comment.id)">提交</el-button>
                  </div>
                </div>
                <div v-if="comment.replies?.length" class="comment-replies">
                  <div v-for="reply in comment.replies" :key="reply.id" class="reply-item">
                    <el-avatar :size="28" :src="reply.author.avatar_url || undefined">
                      {{ reply.author.nickname?.charAt(0) || '?' }}
                    </el-avatar>
                    <div class="comment-body">
                      <div class="comment-header">
                        <span class="comment-author">{{ reply.author.nickname }}</span>
                        <span class="comment-time">{{ formatTime(reply.created_at) }}</span>
                      </div>
                      <div class="comment-content">{{ reply.content }}</div>
                      <div class="comment-actions">
                        <el-button text size="small" @click="toggleLikeComment(reply)">
                          <el-icon><CaretTop /></el-icon> {{ reply.likes_count }}
                        </el-button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-if="!comments.length" description="暂无评论" :image-size="80" />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { View, Download, ChatDotRound, CaretTop, Star, Link } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MarkdownIt from 'markdown-it'
import { getMcpServer, likeMcpServer, favoriteMcpServer, downloadMcpServer, submitMcpServer, withdrawMcpServer, archiveMcpServer } from '@/api/mcpServer'
import { getResourceComments, createResourceComment, likeResourceComment } from '@/api/resourceComment'
import { useAuthStore } from '@/stores/auth'
import type { McpServerDetail, ResourceCommentItem } from '@/types'

const route = useRoute()
const auth = useAuthStore()
const server = ref<McpServerDetail | null>(null)
const comments = ref<ResourceCommentItem[]>([])
const loading = ref(false)
const commentContent = ref('')
const replyingTo = ref<number | null>(null)
const replyContent = ref('')

const md = new MarkdownIt({ html: false, linkify: true })

const statusType = computed(() => {
  const map: Record<string, string> = { draft: 'info', pending_review: 'warning', published: 'success', rejected: 'danger', archived: 'info' }
  return (map[server.value?.status || ''] || 'info') as any
})

const statusLabel = computed(() => {
  const map: Record<string, string> = { draft: '草稿', pending_review: '审核中', published: '已发布', rejected: '已拒绝', archived: '已下架' }
  return map[server.value?.status || ''] || server.value?.status || ''
})

const isAuthor = computed(() => auth.isLoggedIn && auth.user?.id === server.value?.author.id)

const tools = computed(() => {
  if (!server.value?.tools_json) return []
  try {
    const parsed = JSON.parse(server.value.tools_json)
    if (Array.isArray(parsed)) return parsed
    return []
  } catch { return [] }
})

const renderedReadme = computed(() => {
  if (!server.value?.readme) return ''
  return md.render(server.value.readme)
})

onMounted(() => fetchData())
watch(() => route.params.id, () => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res = await getMcpServer(id)
    server.value = res.data.data
    await fetchComments()
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function fetchComments() {
  try {
    const id = Number(route.params.id)
    const res = await getResourceComments('mcp-servers', id)
    comments.value = res.data.data
  } catch { /* ignore */ }
}

async function handleLike() {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    const res = await likeMcpServer(server.value!.id)
    const data = res.data.data
    server.value!.liked = data.liked
    server.value!.likes_count = data.likes_count
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

async function handleFavorite() {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    const res = await favoriteMcpServer(server.value!.id)
    const data = res.data.data
    server.value!.favorited = data.favorited
    server.value!.favorites_count = data.favorites_count
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

async function handleDownload() {
  try {
    const res = await downloadMcpServer(server.value!.id)
    if (res.data.data.url) {
      window.open(res.data.data.url, '_blank')
    }
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

async function handleSubmit() {
  try {
    await ElMessageBox.confirm('确定提交审核？', '确认', { type: 'warning' })
    await submitMcpServer(server.value!.id)
    ElMessage.success('已提交审核')
    await fetchData()
  } catch { /* cancelled */ }
}

async function handleWithdraw() {
  try {
    await ElMessageBox.confirm('确定撤回？', '确认', { type: 'warning' })
    await withdrawMcpServer(server.value!.id)
    ElMessage.success('已撤回')
    await fetchData()
  } catch { /* cancelled */ }
}

async function handleArchive() {
  try {
    await ElMessageBox.confirm('确定下架？', '确认', { type: 'warning' })
    await archiveMcpServer(server.value!.id)
    ElMessage.success('已下架')
    await fetchData()
  } catch { /* cancelled */ }
}

async function submitComment() {
  if (!commentContent.value.trim()) return
  try {
    const id = Number(route.params.id)
    await createResourceComment('mcp-servers', id, { content: commentContent.value })
    commentContent.value = ''
    await fetchComments()
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

function showReply(id: number) {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  replyingTo.value = id
  replyContent.value = ''
}

async function handleReply(parentId: number) {
  if (!replyContent.value.trim()) return
  try {
    const id = Number(route.params.id)
    await createResourceComment('mcp-servers', id, { content: replyContent.value, parent_id: parentId })
    replyingTo.value = null
    replyContent.value = ''
    await fetchComments()
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

async function toggleLikeComment(comment: ResourceCommentItem) {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    await likeResourceComment(comment.id)
    await fetchComments()
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

function formatTime(t: string | null) {
  if (!t) return ''
  return new Date(t).toLocaleDateString('zh-CN')
}

function formatFileSize(bytes: number) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(1)} ${units[i]}`
}
</script>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.detail-title {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
}

.detail-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #606266;
  font-size: 14px;
  flex-wrap: wrap;
}

.meta-time {
  color: #909399;
}

.detail-stats {
  display: flex;
  gap: 16px;
  margin: 12px 0;
  color: #909399;
  font-size: 14px;
}

.detail-stats span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.detail-description {
  margin: 20px 0;
  line-height: 1.6;
  color: #606266;
}

.detail-repo {
  margin: 12px 0;
}

.tools-section {
  margin: 24px 0;
}

.tools-section h3 {
  margin-bottom: 12px;
  font-size: 18px;
}

.readme-section {
  margin: 24px 0;
}

.readme-section h3 {
  margin-bottom: 12px;
  font-size: 18px;
}

.readme-content {
  line-height: 1.6;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.detail-actions {
  display: flex;
  gap: 12px;
  padding: 20px 0;
  border-top: 1px solid #ebeef5;
  border-bottom: 1px solid #ebeef5;
  margin: 20px 0;
  flex-wrap: wrap;
}

.comment-section {
  margin-top: 24px;
}

.comment-section h3 {
  margin-bottom: 16px;
  font-size: 18px;
}

.comment-input {
  margin-bottom: 20px;
}

.comment-item {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.comment-item:last-child {
  border-bottom: none;
}

.comment-main {
  display: flex;
  gap: 12px;
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.comment-author {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
}

.comment-time {
  font-size: 12px;
  color: #909399;
}

.comment-content {
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
  word-break: break-word;
}

.comment-actions {
  margin-top: 4px;
  display: flex;
  gap: 4px;
}

.reply-input {
  margin-top: 8px;
}

.reply-actions {
  margin-top: 8px;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.comment-replies {
  margin-top: 12px;
  padding-left: 16px;
  border-left: 2px solid #ebeef5;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.reply-item {
  display: flex;
  gap: 10px;
}
</style>
