<template>
  <div class="page-container skill-page" v-loading="loading">
    <template v-if="skill">
      <section class="skill-hero">
        <div class="hero-copy">
          <div class="eyebrow">COMMUNITY SKILL</div>
          <div class="detail-header">
            <h1 class="detail-title">{{ skill.name }}</h1>
            <el-tag v-if="skill.status !== 'published'" :type="statusType">{{ statusLabel }}</el-tag>
          </div>
          <p class="hero-description">{{ skill.description || '暂无描述' }}</p>
          <div class="detail-meta">
            <el-avatar :size="30" :src="skill.author.avatar_url || undefined">
              {{ skill.author.nickname?.charAt(0) || '?' }}
            </el-avatar>
            <span class="author-name">{{ skill.author.nickname }}</span>
            <span class="meta-time">发布于 {{ formatTime(skill.published_at) }}</span>
          </div>
          <div class="tag-list">
            <el-tag v-for="tag in skill.tags" :key="tag.id" size="small" effect="plain">{{ tag.name }}</el-tag>
          </div>
        </div>
        <div class="hero-stats">
          <div><strong>{{ skill.views }}</strong><span>浏览</span></div>
          <div><strong>{{ skill.likes_count }}</strong><span>点赞</span></div>
          <div><strong>{{ skill.comments_count }}</strong><span>评论</span></div>
        </div>
      </section>
      <div class="detail-stats mobile-stats">
        <span><el-icon><View /></el-icon> {{ skill.views }}</span>
        <span><el-icon><ChatDotRound /></el-icon> {{ skill.comments_count }}</span>
      </div>
      <section class="skill-repo-card" v-if="skill.repo_url">
        <div><span class="repo-kicker">SOURCE REPOSITORY</span><strong>从 Git 仓库获取完整 Skill</strong></div>
        <el-button type="primary" plain tag="a" :href="skill.repo_url" target="_blank" rel="noopener noreferrer">
          <el-icon><Link /></el-icon> 查看 Git 仓库
        </el-button>
      </section>
      <section v-if="skill.skill_md" class="skill-content">
        <div class="content-heading">
          <div class="content-title-wrap">
            <span class="content-icon"><el-icon><Document /></el-icon></span>
            <div>
              <span class="section-kicker">DOCUMENTATION</span>
              <h2>详细说明</h2>
            </div>
          </div>
        </div>
        <div class="markdown-card">
          <div class="markdown-content" v-html="renderedSkillMD"></div>
        </div>
      </section>
      <div class="detail-actions">
        <el-button
          :type="skill.liked ? 'primary' : 'default'"
          @click="handleLike"
        >
          <el-icon><CaretTop /></el-icon> {{ skill.liked ? '已赞' : '点赞' }} {{ skill.likes_count }}
        </el-button>
        <el-button
          :type="skill.favorited ? 'warning' : 'default'"
          @click="handleFavorite"
        >
          <el-icon><Star /></el-icon> {{ skill.favorited ? '已藏' : '收藏' }} {{ skill.favorites_count }}
        </el-button>
        <template v-if="isAuthor">
          <el-button type="info" @click="$router.push(`/skills/${skill.id}/edit`)">编辑</el-button>
          <el-button
            v-if="skill.status === 'draft' || skill.status === 'rejected'"
            type="success"
            @click="handleSubmit"
          >
            提交审核
          </el-button>
          <el-button
            v-if="skill.status === 'pending_review'"
            type="warning"
            @click="handleWithdraw"
          >
            撤回
          </el-button>
          <el-button
            v-if="skill.status === 'published'"
            type="danger"
            @click="handleArchive"
          >
            下架
          </el-button>
        </template>
      </div>
      <div class="comment-section">
        <h3>评论 ({{ skill.comments_count }})</h3>
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
import MarkdownIt from 'markdown-it'
import { View, ChatDotRound, CaretTop, Star, Link, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSkill, likeSkill, favoriteSkill, submitSkill, withdrawSkill, archiveSkill } from '@/api/skill'
import { getResourceComments, createResourceComment, likeResourceComment } from '@/api/resourceComment'
import { useAuthStore } from '@/stores/auth'
import type { SkillDetail, ResourceCommentItem } from '@/types'

const route = useRoute()
const auth = useAuthStore()
const skill = ref<SkillDetail | null>(null)
const comments = ref<ResourceCommentItem[]>([])
const loading = ref(false)
const commentContent = ref('')
const replyingTo = ref<number | null>(null)
const replyContent = ref('')
const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

const statusType = computed(() => {
  const map: Record<string, string> = { draft: 'info', pending_review: 'warning', published: 'success', rejected: 'danger', archived: 'info' }
  return (map[skill.value?.status || ''] || 'info') as any
})

const statusLabel = computed(() => {
  const map: Record<string, string> = { draft: '草稿', pending_review: '审核中', published: '已发布', rejected: '已拒绝', archived: '已下架' }
  return map[skill.value?.status || ''] || skill.value?.status || ''
})

const isAuthor = computed(() => auth.isLoggedIn && auth.user?.id === skill.value?.author.id)
const renderedSkillMD = computed(() => md.render(skill.value?.skill_md || ''))

onMounted(() => fetchData())
watch(() => route.params.id, () => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res = await getSkill(id)
    skill.value = res.data.data
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
    const res = await getResourceComments('skills', id)
    comments.value = res.data.data
  } catch { /* ignore */ }
}

async function handleLike() {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    const res = await likeSkill(skill.value!.id)
    const data = res.data.data
    skill.value!.liked = data.liked
    skill.value!.likes_count = data.likes_count
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

async function handleFavorite() {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    const res = await favoriteSkill(skill.value!.id)
    const data = res.data.data
    skill.value!.favorited = data.favorited
    skill.value!.favorites_count = data.favorites_count
  } catch (e: unknown) { ElMessage.error((e as Error).message) }
}

async function handleSubmit() {
  try {
    await ElMessageBox.confirm('确定提交审核？', '确认', { type: 'warning' })
    await submitSkill(skill.value!.id)
    ElMessage.success('已提交审核')
    await fetchData()
  } catch { /* cancelled */ }
}

async function handleWithdraw() {
  try {
    await ElMessageBox.confirm('确定撤回？', '确认', { type: 'warning' })
    await withdrawSkill(skill.value!.id)
    ElMessage.success('已撤回')
    await fetchData()
  } catch { /* cancelled */ }
}

async function handleArchive() {
  try {
    await ElMessageBox.confirm('确定下架？', '确认', { type: 'warning' })
    await archiveSkill(skill.value!.id)
    ElMessage.success('已下架')
    await fetchData()
  } catch { /* cancelled */ }
}

async function submitComment() {
  if (!commentContent.value.trim()) return
  try {
    const id = Number(route.params.id)
    await createResourceComment('skills', id, { content: commentContent.value })
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
    await createResourceComment('skills', id, { content: replyContent.value, parent_id: parentId })
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
.skill-page { max-width: 1120px; margin: 0 auto; padding-bottom: 48px; }
.skill-hero { display: flex; justify-content: space-between; gap: 32px; padding: 32px; margin-bottom: 20px; border: 1px solid #e4eaf3; border-radius: 18px; background: linear-gradient(135deg, #f8fbff 0%, #fff 62%); box-shadow: 0 12px 35px rgb(36 76 130 / 7%); }
.hero-copy { min-width: 0; }
.eyebrow, .section-kicker, .repo-kicker { color: #409eff; font-size: 11px; font-weight: 700; letter-spacing: .12em; }
.detail-header { margin: 8px 0 14px; }
.hero-description { max-width: 720px; margin: 0 0 20px; color: #52657d; font-size: 16px; line-height: 1.75; }
.detail-meta { gap: 9px; }
.author-name { color: #34495e; font-weight: 600; }
.tag-list { display: flex; flex-wrap: wrap; gap: 7px; margin-top: 16px; }
.hero-stats { display: flex; align-items: center; gap: 22px; flex-shrink: 0; padding: 0 4px; }
.hero-stats div { display: flex; flex-direction: column; align-items: center; gap: 4px; min-width: 54px; }
.hero-stats strong { color: #1f3b5b; font-size: 22px; }
.hero-stats span { color: #8291a5; font-size: 12px; }
.mobile-stats { display: none; }
.skill-repo-card, .skill-content, .comment-section { padding: 24px 28px; border: 1px solid #e4eaf3; border-radius: 14px; background: #fff; box-shadow: 0 6px 22px rgb(36 76 130 / 5%); }
.skill-repo-card { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; }
.skill-repo-card div { display: flex; flex-direction: column; gap: 5px; }
.skill-repo-card strong { color: #263b53; font-size: 15px; }
.skill-content { margin-bottom: 20px; }
.content-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding-bottom: 18px; border-bottom: 1px solid #edf1f6; }
.content-title-wrap { display: flex; align-items: center; gap: 12px; }
.content-icon { display: inline-flex; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 10px; color: #409eff; background: #edf5ff; font-size: 18px; }
.content-heading h2 { margin: 5px 0 0; color: #243b53; font-size: 21px; }
.markdown-card { margin-top: 18px; padding: 4px 22px 20px; border: 1px solid #edf1f6; border-radius: 12px; background: #fbfcfe; }
.markdown-content { color: #52657d; font-size: 15px; line-height: 1.8; }
.markdown-content :deep(h1), .markdown-content :deep(h2), .markdown-content :deep(h3) { margin: 22px 0 8px; color: #243b53; line-height: 1.35; }
.markdown-content :deep(h1:first-child), .markdown-content :deep(h2:first-child) { margin-top: 0; }
.markdown-content :deep(p) { margin: 9px 0; }
.markdown-content :deep(ul), .markdown-content :deep(ol) { padding-left: 24px; }
.markdown-content :deep(code) { padding: 2px 5px; border-radius: 4px; background: #f1f5f9; color: #d14; font-family: Consolas, monospace; }
.markdown-content :deep(pre) { padding: 14px 16px; overflow: auto; border-radius: 8px; background: #f6f8fb; }
.markdown-content :deep(a) { color: #409eff; }
.comment-section { margin-top: 20px; }
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
@media (max-width: 700px) {
  .skill-hero { flex-direction: column; padding: 24px 20px; }
  .hero-stats { justify-content: flex-start; padding-top: 4px; }
  .mobile-stats { display: none; }
  .skill-repo-card, .skill-content, .comment-section { padding: 20px; }
  .skill-repo-card { align-items: flex-start; flex-direction: column; }
  .content-heading { align-items: flex-start; }
  .markdown-card { padding: 4px 16px 16px; }
}
</style>
