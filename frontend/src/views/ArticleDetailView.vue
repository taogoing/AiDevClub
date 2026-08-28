<template>
  <div class="page-container" v-loading="loading">
    <template v-if="article">
      <article class="article-detail">
        <h1 class="article-title">{{ article.title }}</h1>
        <div class="article-meta">
          <el-avatar :size="28" :src="article.author.avatar_url || undefined">
            {{ article.author.nickname?.charAt(0) || '?' }}
          </el-avatar>
          <span>{{ article.author.nickname }}</span>
          <el-tag v-for="tag in article.tags" :key="tag.id" size="small">{{ tag.name }}</el-tag>
          <span class="meta-time">{{ formatTime(article.published_at) }}</span>
        </div>
        <div class="article-stats">
          <span><el-icon><View /></el-icon> {{ article.views }}</span>
          <span><el-icon><ChatDotRound /></el-icon> {{ article.comments_count }}</span>
        </div>
        <div class="article-content" v-html="renderedContent"></div>
        <div class="article-actions">
          <el-button
            :type="article.liked ? 'primary' : 'default'"
            @click="handleLike"
          >
            <el-icon><CaretTop /></el-icon> {{ article.liked ? '已赞' : '点赞' }} {{ article.likes_count }}
          </el-button>
          <el-button
            :type="article.favorited ? 'warning' : 'default'"
            @click="handleFavorite"
          >
            <el-icon><Star /></el-icon> {{ article.favorited ? '已藏' : '收藏' }} {{ article.favorites_count }}
          </el-button>
          <el-button
            v-if="isAuthor"
            type="info"
            @click="$router.push(`/articles/${article.id}/edit`)"
          >
            编辑
          </el-button>
        </div>
      </article>
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
import { View, ChatDotRound, CaretTop, Star } from '@element-plus/icons-vue'
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
.article-title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 16px;
  color: #303133;
}

.article-meta {
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

.article-stats {
  display: flex;
  gap: 16px;
  margin: 12px 0;
  color: #909399;
  font-size: 14px;
}

.article-stats span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.article-content {
  margin: 24px 0;
  line-height: 1.5;
}

.article-actions {
  display: flex;
  gap: 12px;
  padding: 20px 0;
  border-top: 1px solid #ebeef5;
  border-bottom: 1px solid #ebeef5;
  margin: 20px 0;
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
</style>
