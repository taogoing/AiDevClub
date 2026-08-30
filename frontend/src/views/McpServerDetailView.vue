<template>
  <div class="page-container" v-loading="loading">
    <template v-if="server">
      <section class="mcp-hero">
        <div class="hero-copy">
          <div class="eyebrow">COMMUNITY MCP SERVER</div>
          <div class="detail-header">
            <h1 class="detail-title">{{ server.name }}</h1>
            <el-tag v-if="server.status !== 'published'" :type="statusType">{{ statusLabel }}</el-tag>
          </div>
          <p class="hero-description">{{ server.description || '暂无描述' }}</p>
          <div class="detail-meta">
            <el-avatar :size="30" :src="server.author.avatar_url || undefined">
              {{ server.author.nickname?.charAt(0) || '?' }}
            </el-avatar>
            <span class="author-name">{{ server.author.nickname }}</span>
            <span class="meta-time">发布于 {{ formatTime(server.published_at) }}</span>
          </div>
          <div class="tag-list">
            <el-tag v-for="tag in server.tags" :key="tag.id" size="small" effect="plain">{{ tag.name }}</el-tag>
          </div>
        </div>
        <div class="hero-side">
          <el-button class="repo-button" type="primary" @click="openRepository">
            <el-icon><Link /></el-icon> 查看 Git 仓库
          </el-button>
          <div class="hero-stats" aria-label="MCP 统计">
            <div class="stat-card"><strong>{{ server.views }}</strong><span>浏览</span></div>
            <div class="stat-card"><strong>{{ server.likes_count }}</strong><span>点赞</span></div>
            <div class="stat-card"><strong>{{ server.favorites_count }}</strong><span>收藏</span></div>
          </div>
        </div>
      </section>
      <div class="install-section">
        <h3>安装指南</h3>
        <p class="section-hint">选择客户端，复制安装命令或 JSON 配置。</p>
        <McpInstallPanel :installations="server.installations" />
      </div>

      <div v-if="server.readme" class="readme-section">
        <h3>补充说明</h3>
        <div class="readme-content" v-html="renderedReadme"></div>
      </div>

      <div class="detail-actions">
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
import { CaretTop, Star, Link } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MarkdownIt from 'markdown-it'
import { getMcpServer, likeMcpServer, favoriteMcpServer, submitMcpServer, withdrawMcpServer, archiveMcpServer } from '@/api/mcpServer'
import { getResourceComments, createResourceComment, likeResourceComment } from '@/api/resourceComment'
import { useAuthStore } from '@/stores/auth'
import type { McpServerDetail, ResourceCommentItem } from '@/types'
import McpInstallPanel from '@/components/McpInstallPanel.vue'

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

function openRepository() {
  if (!server.value?.repo_url) {
    ElMessage.warning('该项目暂未填写 Git 仓库')
    return
  }
  window.open(server.value.repo_url, '_blank', 'noopener,noreferrer')
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

</script>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.install-section {
  margin: 24px 0;
}

.install-section h3 {
  margin-bottom: 12px;
  font-size: 18px;
}

.section-hint { margin: -4px 0 14px; color: #909399; font-size: 14px; }

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

/* Shared resource-detail visual language */
.page-container { max-width: 1120px; padding-top: 32px; }
.mcp-hero { display: flex; justify-content: space-between; gap: 32px; padding: 32px; border: 1px solid #e4eaf3; border-radius: 18px; background: linear-gradient(135deg, #f8fbff 0%, #fff 68%); box-shadow: 0 12px 35px rgb(36 76 130 / 7%); }
.hero-copy { min-width: 0; }
.hero-side { display: flex; flex-direction: column; align-items: flex-end; justify-content: space-between; gap: 26px; flex-shrink: 0; }
.eyebrow { color: #409eff; font-size: 11px; font-weight: 700; letter-spacing: .12em; }
.mcp-hero .detail-header { display: flex; align-items: center; gap: 12px; padding: 0; margin: 8px 0 14px; border: 0; background: transparent; box-shadow: none; }
.mcp-hero .detail-meta { padding: 0; border: 0; background: transparent; }
.mcp-hero .hero-description { max-width: 720px; margin: 0 0 20px; color: #52657d; font-size: 16px; line-height: 1.75; }
.mcp-hero .tag-list { display: flex; flex-wrap: wrap; gap: 7px; margin-top: 16px; }
.mcp-hero .hero-stats { display: flex; gap: 10px; margin: 0; }
.page-container > template + * { min-width: 0; }
.detail-header { display: grid; grid-template-columns: minmax(0, 1fr) auto auto; align-items: center; padding: 28px 32px; margin-bottom: 0; border: 1px solid #e4eaf3; border-radius: 18px 18px 0 0; background: linear-gradient(135deg, #f8fbff, #fff 68%); box-shadow: 0 10px 30px rgb(31 78 121 / 7%); }
.detail-title { color: #1f3b5b; font-size: 30px; letter-spacing: -.02em; }
.hero-stats { display: flex; align-items: stretch; gap: 10px; margin-left: 24px; }
.stat-card { display: flex; min-width: 66px; flex-direction: column; align-items: center; justify-content: center; gap: 2px; padding: 9px 10px; border: 1px solid #e4eaf3; border-radius: 10px; background: rgb(255 255 255 / 80%); }
.stat-card strong { color: #1f3b5b; font-size: 19px; line-height: 1.2; }
.stat-card span { color: #8291a5; font-size: 11px; }
.detail-meta { padding: 16px 32px; border: 1px solid #e4eaf3; border-top: 0; background: #fff; }
.detail-stats { padding: 0 32px 20px; margin: 0; border: 1px solid #e4eaf3; border-top: 0; background: #fff; color: #71849a; }
.detail-description { margin: 20px 0; padding: 22px 28px; border: 1px solid #e4eaf3; border-radius: 14px; background: #fff; color: #52657d; box-shadow: 0 6px 22px rgb(36 76 130 / 5%); }
.install-section, .readme-section, .comment-section { padding: 24px 28px; border: 1px solid #e4eaf3; border-radius: 14px; background: #fff; box-shadow: 0 6px 22px rgb(36 76 130 / 5%); }
.install-section { margin: 20px 0; }
.install-section h3, .readme-section h3, .comment-section h3 { color: #243b53; font-size: 20px; }
.section-hint { color: #71849a; }
.readme-section { margin: 20px 0; }
.readme-content { padding: 20px; border: 1px solid #edf1f6; border-radius: 12px; background: #fbfcfe; color: #52657d; line-height: 1.8; }
.detail-actions { padding: 20px 0; border-color: #e4eaf3; }
.comment-section { margin-top: 20px; }
.comment-item { border-color: #edf1f6; }
@media (max-width: 700px) {
  .page-container { padding: 20px 14px 40px; }
  .mcp-hero { flex-direction: column; padding: 24px 20px; border-radius: 14px; }
  .mcp-hero .detail-header { align-items: center; flex-direction: row; padding: 0; border-radius: 0; }
  .detail-title { font-size: 25px; }
  .detail-meta, .detail-stats { padding-left: 20px; padding-right: 20px; }
  .detail-description, .install-section, .readme-section, .comment-section { padding: 20px; }
  .hero-side { width: 100%; align-items: stretch; gap: 16px; }
  .repo-button { width: 100%; }
  .hero-stats { width: 100%; margin: 0; }
  .stat-card { flex: 1; }
}
</style>
