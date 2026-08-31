<template>
  <div class="page-container" v-loading="loading">
    <template v-if="article">
      <div class="article-hero">
        <div class="hero-copy">
          <div class="eyebrow">ARTICLE</div>
          <h1 class="article-title">{{ article.title }}</h1>
          <div class="article-meta">
            <el-avatar :size="28" :src="article.author.avatar_url || undefined">
              {{ article.author.nickname?.charAt(0) || '?' }}
            </el-avatar>
            <span class="author-name">{{ article.author.nickname }}</span>
            <span class="meta-sep">·</span>
            <span class="meta-time">{{ formatTime(article.published_at) }}</span>
          </div>
          <div class="tag-list">
            <el-tag v-for="tag in article.tags" :key="tag.id" size="small" type="info">{{ tag.name }}</el-tag>
          </div>
        </div>
        <div class="hero-stats">
          <div class="stat-card">
            <strong>{{ article.views }}</strong>
            <span>阅读</span>
          </div>
          <div class="stat-card">
            <strong>{{ article.likes_count }}</strong>
            <span>点赞</span>
          </div>
          <div class="stat-card">
            <strong>{{ article.comments_count }}</strong>
            <span>评论</span>
          </div>
        </div>
      </div>

      <div class="article-content-card">
        <div class="article-content" v-html="renderedContent"></div>
      </div>

      <div class="article-actions-card">
        <el-button
          :type="article.liked ? 'primary' : 'default'"
          @click="handleLike"
        >
          <el-icon><CaretTop /></el-icon> {{ article.liked ? '已赞' : '点赞' }}
        </el-button>
        <el-button
          :type="article.favorited ? 'warning' : 'default'"
          @click="handleFavorite"
        >
          <el-icon><Star /></el-icon> {{ article.favorited ? '已藏' : '收藏' }}
        </el-button>
        <el-button
          v-if="isAuthor"
          type="info"
          @click="$router.push(`/articles/${article.id}/edit`)"
        >
          编辑
        </el-button>
      </div>

      <div class="comment-section">
        <h3>评论 ({{ article.comments_count }})</h3>
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
        <CommentTree
          :comments="comments"
          :article-id="article.id"
          :article-author-id="article.author.id"
          @refresh="fetchComments"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { CaretTop, Star } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import CommentTree from '@/components/CommentTree.vue'
import { getArticle, likeArticle, favoriteArticle } from '@/api/article'
import { getComments, createComment } from '@/api/comment'
import { useAuthStore } from '@/stores/auth'
import type { ArticleDetail, CommentItem } from '@/types'

const route = useRoute()
const auth = useAuthStore()
const article = ref<ArticleDetail | null>(null)
const comments = ref<CommentItem[]>([])
const loading = ref(false)
const commentContent = ref('')

const md = new MarkdownIt({
  html: false,
  linkify: true,
  highlight(str: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${hljs.highlight(str, { language: lang }).value}</code></pre>`
      } catch { /* fallback */ }
    }
    return `<pre class="hljs"><code>${MarkdownIt().utils.escapeHtml(str)}</code></pre>`
  },
})

const renderedContent = computed(() => {
  if (!article.value?.content) return ''
  return md.render(article.value.content)
})

const isAuthor = computed(() => auth.isLoggedIn && auth.user?.id === article.value?.author.id)

onMounted(() => fetchData())
watch(() => route.params.id, () => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res = await getArticle(id)
    article.value = res.data.data
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
    const res = await getComments(id)
    comments.value = res.data.data
  } catch { /* ignore */ }
}

async function handleLike() {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    const res = await likeArticle(article.value!.id)
    const data = res.data.data
    article.value!.liked = data.liked
    article.value!.likes_count = data.likes_count
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function handleFavorite() {
  if (!auth.isLoggedIn) { ElMessage.warning('请先登录'); return }
  try {
    const res = await favoriteArticle(article.value!.id)
    const data = res.data.data
    article.value!.favorited = data.favorited
    article.value!.favorites_count = data.favorites_count
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function submitComment() {
  if (!commentContent.value.trim()) return
  try {
    await createComment(article.value!.id, commentContent.value, null)
    commentContent.value = ''
    await fetchComments()
    article.value!.comments_count = comments.value.reduce(
      (sum, c) => sum + 1 + (c.replies?.length || 0), 0,
    )
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

function formatTime(t: string | null) {
  if (!t) return ''
  return new Date(t).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.page-container { max-width: 900px; margin: 0 auto; padding: 32px 24px 48px; }

.article-hero {
  display: flex;
  justify-content: space-between;
  gap: 32px;
  padding: 32px;
  margin-bottom: 20px;
  border: 1px solid #e4eaf3;
  border-radius: 18px;
  background: linear-gradient(135deg, #f8fbff 0%, #fff 62%);
  box-shadow: 0 12px 35px rgb(36 76 130 / 7%);
}

.hero-copy { min-width: 0; flex: 1; }

.eyebrow {
  color: #409eff;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
  margin-bottom: 8px;
}

.article-title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 14px;
  color: #1f3b5b;
  line-height: 1.35;
}

.article-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #606266;
  font-size: 14px;
  flex-wrap: wrap;
}

.author-name { color: #34495e; font-weight: 600; }
.meta-sep { color: #c0c4cc; }
.meta-time { color: #909399; }

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 14px;
}

.hero-stats {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  flex-shrink: 0;
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  min-width: 66px;
  padding: 12px 14px;
  border: 1px solid #e4eaf3;
  border-radius: 10px;
  background: rgb(255 255 255 / 80%);
}

.stat-card strong {
  color: #1f3b5b;
  font-size: 20px;
  line-height: 1.2;
}

.stat-card span {
  color: #8291a5;
  font-size: 12px;
}

.article-content-card {
  padding: 28px 32px;
  margin-bottom: 20px;
  border: 1px solid #e4eaf3;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 6px 22px rgb(36 76 130 / 5%);
}

.article-content {
  line-height: 1.8;
  font-size: 16px;
  color: #52657d;
}

.article-content :deep(h1) {
  font-size: 22px;
  font-weight: 700;
  margin: 28px 0 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #edf1f6;
  color: #243b53;
}

.article-content :deep(h1:first-child) { margin-top: 0; }

.article-content :deep(h2) {
  font-size: 19px;
  font-weight: 600;
  margin: 24px 0 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid #edf1f6;
  color: #243b53;
}

.article-content :deep(h3) {
  font-size: 17px;
  font-weight: 600;
  margin: 20px 0 8px;
  color: #243b53;
}

.article-content :deep(p) {
  margin: 10px 0;
}

.article-content :deep(ul),
.article-content :deep(ol) {
  padding-left: 24px;
  margin: 10px 0;
}

.article-content :deep(li) {
  margin: 5px 0;
}

.article-content :deep(blockquote) {
  margin: 16px 0;
  padding: 12px 16px;
  border-left: 4px solid #409eff;
  background: #f4f7fb;
  color: #5a6a7a;
  border-radius: 0 8px 8px 0;
}

.article-content :deep(blockquote p) {
  margin: 0;
}

.article-content :deep(pre) {
  margin: 16px 0;
  border-radius: 8px;
  overflow-x: auto;
}

.article-content :deep(pre code) {
  display: block;
  padding: 16px 20px;
  background: #1e1e2e;
  color: #cdd6f4;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace;
  font-size: 14px;
  line-height: 1.6;
  border-radius: 8px;
  overflow-x: auto;
}

.article-content :deep(:not(pre) > code) {
  padding: 2px 6px;
  margin: 0 2px;
  background: #f1f5f9;
  color: #d14;
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
  font-size: 14px;
  border-radius: 4px;
}

.article-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 12px 0;
}

.article-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 16px 0;
}

.article-content :deep(th),
.article-content :deep(td) {
  padding: 10px 14px;
  border: 1px solid #edf1f6;
  text-align: left;
}

.article-content :deep(th) {
  background: #f5f7fa;
  font-weight: 600;
  color: #243b53;
}

.article-content :deep(tr:hover) {
  background: #f9fafc;
}

.article-content :deep(a) {
  color: #409eff;
}

.article-actions-card {
  display: flex;
  gap: 12px;
  padding: 20px 28px;
  border: 1px solid #e4eaf3;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 6px 22px rgb(36 76 130 / 5%);
  margin-bottom: 20px;
}

.comment-section {
  padding: 24px 28px;
  border: 1px solid #e4eaf3;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 6px 22px rgb(36 76 130 / 5%);
}

.comment-section h3 {
  margin-bottom: 16px;
  font-size: 20px;
  color: #243b53;
}

.comment-input {
  margin-bottom: 20px;
}

@media (max-width: 700px) {
  .page-container { padding: 20px 14px 40px; }
  .article-hero { flex-direction: column; padding: 24px 20px; border-radius: 14px; }
  .hero-stats { justify-content: flex-start; padding-top: 4px; }
  .article-content-card, .article-actions-card, .comment-section { padding: 20px; }
}
</style>
