<template>
  <div class="home-container">
    <div class="main-area">
      <div class="filter-bar">
        <div class="search-row">
          <el-input
            v-model="keyword"
            placeholder="搜索 Skill..."
            clearable
            size="default"
            style="width: 300px"
            @clear="onSearch"
            @keyup.enter="onSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" @click="onSearch">搜索</el-button>
          <el-button
            v-if="auth.isLoggedIn"
            type="success"
            @click="$router.push('/skills/new')"
          >
            发布 Skill
          </el-button>
        </div>
        <div class="sort-bar">
          <el-select v-model="sortBy" size="small" style="width: 120px">
            <el-option label="最新" value="latest" />
            <el-option label="热门" value="hot" />
            <el-option label="下载量" value="downloads" />
          </el-select>
          <div class="tag-filters">
            <el-button
              :type="!selectedTag ? 'primary' : 'default'"
              size="small"
              @click="selectedTag = 0"
            >
              全部标签
            </el-button>
            <el-button
              v-for="tag in hotTags"
              :key="tag.id"
              :type="selectedTag === tag.id ? 'primary' : 'default'"
              size="small"
              @click="selectedTag = tag.id"
            >
              {{ tag.name }}
            </el-button>
          </div>
        </div>
      </div>
      <div v-loading="loading">
        <SkillCard v-for="skill in skills" :key="skill.id" :skill="skill" />
        <el-empty v-if="!loading && !skills.length" description="暂无 Skill" />
      </div>
      <div class="pagination-wrap" v-if="total > pageSize">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="fetchSkills"
        />
      </div>
    </div>
    <aside class="sidebar">
      <Sidebar />
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import SkillCard from '@/components/SkillCard.vue'
import Sidebar from '@/components/Sidebar.vue'
import { getSkills } from '@/api/skill'
import { getHotTags } from '@/api/tag'
import { useAuthStore } from '@/stores/auth'
import type { SkillSummary, Tag } from '@/types'

const auth = useAuthStore()
const skills = ref<SkillSummary[]>([])
const hotTags = ref<Tag[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const sortBy = ref('latest')
const keyword = ref('')
const selectedTag = ref(0)

onMounted(async () => {
  try {
    const res = await getHotTags()
    hotTags.value = res.data.data
  } catch { /* ignore */ }
  await fetchSkills()
})

watch([sortBy, selectedTag], () => {
  currentPage.value = 1
  fetchSkills()
})

function onSearch() {
  currentPage.value = 1
  fetchSkills()
}

async function fetchSkills() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: currentPage.value,
      page_size: pageSize,
      sort: sortBy.value,
    }
    if (keyword.value) params.keyword = keyword.value
    if (selectedTag.value) params.tag_id = selectedTag.value
    const res = await getSkills(params)
    const data = res.data.data
    skills.value = data.list
    total.value = data.total
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.home-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  display: flex;
  gap: 24px;
}

.main-area {
  flex: 1;
  min-width: 0;
}

.sidebar {
  width: 280px;
  flex-shrink: 0;
}

.filter-bar {
  margin-bottom: 20px;
}

.search-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.sort-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.tag-filters {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

@media (max-width: 768px) {
  .sidebar {
    display: none;
  }

  .home-container {
    padding: 16px;
  }
}
</style>
