<template>
  <div class="resource-sidebar">
    <div class="sidebar-section">
      <h3 class="section-title">
        <el-icon><TrendCharts /></el-icon> 热门{{ type === 'skill' ? 'Skill' : 'MCP' }}
      </h3>
      <div class="hot-resources">
        <div
          v-for="(item, index) in hotResources"
          :key="item.id"
          class="hot-resource-item"
          @click="handleResourceClick(item.id)"
        >
          <span class="rank" :class="`rank-${index + 1}`">{{ index + 1 }}</span>
          <span class="hot-title">{{ item.name }}</span>
        </div>
        <el-empty v-if="!hotResources.length" description="暂无资源" :image-size="60" />
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { TrendCharts } from '@element-plus/icons-vue'
import { getSkillRanking } from '@/api/skill'
import { getMcpServerRanking } from '@/api/mcpServer'
import type { HotSkillBrief, HotMcpServerBrief } from '@/types'

const props = defineProps<{
  type: 'skill' | 'mcp'
}>()

const router = useRouter()
const hotResources = ref<(HotSkillBrief | HotMcpServerBrief)[]>([])

onMounted(async () => {
  await fetchData()
})

watch(() => props.type, () => {
  fetchData()
})

async function fetchData() {
  try {
    if (props.type === 'skill') {
      const res = await getSkillRanking(1, 5)
      hotResources.value = res.data.data.skills ?? []
    } else {
      const res = await getMcpServerRanking(1, 5)
      hotResources.value = res.data.data.mcp_servers ?? []
    }
  } catch {
    hotResources.value = []
  }
}

function handleResourceClick(id: number) {
  if (props.type === 'skill') {
    router.push({ name: 'skill-detail', params: { id } })
  } else {
    router.push({ name: 'mcp-detail', params: { id } })
  }
}
</script>

<style scoped>
.resource-sidebar {
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

.hot-resources {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.hot-resource-item {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 0;
}

.hot-resource-item:hover .hot-title {
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
  flex: 1;
  font-size: 13px;
  color: #606266;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  transition: color 0.2s;
}

</style>
