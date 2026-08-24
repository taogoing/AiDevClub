<template>
  <div class="sidebar-content">
    <div class="sidebar-section">
      <h3 class="section-title">
        <el-icon><Star /></el-icon> 热门标签
      </h3>
      <div class="tag-cloud">
        <el-tag
          v-for="tag in hotTags"
          :key="tag.id"
          size="small"
          class="tag-item"
          @click="handleTagClick(tag.id)"
        >
          {{ tag.name }}
        </el-tag>
        <el-empty v-if="!hotTags.length" description="暂无标签" :image-size="60" />
      </div>
    </div>

    <div class="sidebar-section">
      <h3 class="section-title">
        <el-icon><TrendCharts /></el-icon> 热门文章
      </h3>
      <div class="hot-articles">
        <div
          v-for="(article, index) in hotArticles"
          :key="article.id"
          class="hot-article-item"
          @click="$router.push(`/articles/${article.id}`)"
        >
          <span class="rank" :class="`rank-${index + 1}`">{{ index + 1 }}</span>
          <span class="hot-title">{{ article.title }}</span>
        </div>
        <el-empty v-if="!hotArticles.length" description="暂无文章" :image-size="60" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Star, TrendCharts } from '@element-plus/icons-vue'
import { getHotTags } from '@/api/tag'
import { getArticles } from '@/api/article'
import type { Tag, ArticleSummary } from '@/types'

const router = useRouter()
const hotTags = ref<Tag[]>([])
const hotArticles = ref<ArticleSummary[]>([])

onMounted(async () => {
  try {
    const [tagRes, articleRes] = await Promise.all([
      getHotTags(),
      getArticles({ page: 1, page_size: 5, sort: 'hot' }),
    ])
    hotTags.value = tagRes.data.data
    hotArticles.value = articleRes.data.data.list
  } catch { /* ignore */ }
})

function handleTagClick(tagId: number) {
  router.push({ name: 'home', query: { tag_id: tagId } })
}
</script>

<style scoped>
.sidebar-content {
  position: sticky;
  top: 80px;
}

.sidebar-section {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  border: 1px solid #ebeef5;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  cursor: pointer;
  transition: all 0.2s;
}

.tag-item:hover {
  color: #409eff;
  border-color: #409eff;
}

.hot-articles {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.hot-article-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  cursor: pointer;
  padding: 4px 0;
}

.hot-article-item:hover .hot-title {
  color: #409eff;
}

.rank {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: #e4e7ed;
  color: #909399;
  font-size: 12px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

.rank-1 {
  background: #f56c6c;
  color: #fff;
}

.rank-2 {
  background: #e6a23c;
  color: #fff;
}

.rank-3 {
  background: #409eff;
  color: #fff;
}

.hot-title {
  font-size: 14px;
  color: #606266;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  transition: color 0.2s;
}
</style>
