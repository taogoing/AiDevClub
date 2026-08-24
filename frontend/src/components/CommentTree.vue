<template>
  <div class="comment-tree">
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
            <el-button text size="small" @click="toggleLike(comment)">
              <el-icon><CaretTop /></el-icon> {{ comment.likes_count }}
            </el-button>
            <el-button text size="small" @click="showReplyInput(comment.id)">
              回复
            </el-button>
            <el-button
              v-if="canDelete(comment)"
              text
              size="small"
              type="danger"
              @click="handleDelete(comment.id)"
            >
              删除
            </el-button>
          </div>
          <div v-if="replyingTo === comment.id" class="reply-input">
            <el-input
              v-model="replyContent"
              type="textarea"
              :rows="2"
              placeholder="写下你的回复..."
            />
            <div class="reply-actions">
              <el-button size="small" @click="replyingTo = null">取消</el-button>
              <el-button size="small" type="primary" @click="handleReply(comment.id)">提交</el-button>
            </div>
          </div>
          <div v-if="comment.replies?.length" class="comment-replies">
            <div v-for="reply in comment.replies" :key="reply.id" class="comment-item reply-item">
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
                  <el-button text size="small" @click="toggleLike(reply)">
                    <el-icon><CaretTop /></el-icon> {{ reply.likes_count }}
                  </el-button>
                  <el-button
                    v-if="canDelete(reply)"
                    text
                    size="small"
                    type="danger"
                    @click="handleDelete(reply.id)"
                  >
                    删除
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
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { CaretTop } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { likeComment, deleteComment, createComment } from '@/api/comment'
import { useAuthStore } from '@/stores/auth'
import type { CommentItem } from '@/types'

const props = defineProps<{
  comments: CommentItem[]
  articleId: number
  articleAuthorId: number
}>()

const emit = defineEmits<{ refresh: [] }>()

const auth = useAuthStore()
const replyingTo = ref<number | null>(null)
const replyContent = ref('')

function formatTime(t: string) {
  return new Date(t).toLocaleDateString('zh-CN')
}

function canDelete(comment: CommentItem): boolean {
  if (!auth.isLoggedIn) return false
  return auth.user?.id === comment.author_id || auth.user?.id === props.articleAuthorId
}

async function toggleLike(comment: CommentItem) {
  if (!auth.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  try {
    await likeComment(comment.id)
    emit('refresh')
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

function showReplyInput(id: number) {
  if (!auth.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  replyingTo.value = id
  replyContent.value = ''
}

async function handleReply(parentId: number) {
  if (!replyContent.value.trim()) return
  try {
    await createComment(props.articleId, replyContent.value, parentId)
    replyingTo.value = null
    replyContent.value = ''
    emit('refresh')
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
}

async function handleDelete(id: number) {
  try {
    await ElMessageBox.confirm('确定删除这条评论？', '确认', { type: 'warning' })
    await deleteComment(id)
    emit('refresh')
  } catch { /* cancelled */ }
}
</script>

<style scoped>
.comment-item {
  display: flex;
  gap: 12px;
}

.comment-main {
  display: flex;
  gap: 12px;
  width: 100%;
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
  padding: 8px 0;
}

.comment-tree > .comment-item {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.comment-tree > .comment-item:last-child {
  border-bottom: none;
}
</style>
