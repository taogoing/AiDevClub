<template>
  <div class="skill-card" @click="$router.push(`/skills/${skill.id}`)">
    <div class="card-header">
      <h3 class="card-title">{{ skill.name }}</h3>
      <el-link v-if="skill.repo_url" :href="skill.repo_url" target="_blank" rel="noopener noreferrer" @click.stop>Git 仓库</el-link>
    </div>
    <p class="card-summary">{{ skill.description || '暂无描述' }}</p>
    <div class="card-tags">
      <el-tag v-for="tag in skill.tags" :key="tag.id" size="small" type="info">{{ tag.name }}</el-tag>
    </div>
    <div class="card-meta">
      <div class="meta-left">
        <el-avatar :size="18" :src="skill.author.avatar_url || undefined">
          {{ skill.author.nickname?.charAt(0) || '?' }}
        </el-avatar>
        <span class="author-name">{{ skill.author.nickname }}</span>
      </div>
      <div class="meta-right">
        <span><el-icon><View /></el-icon>{{ skill.views }}</span>
        <span><el-icon><CaretTop /></el-icon>{{ skill.likes_count }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { View, CaretTop } from '@element-plus/icons-vue'
import type { SkillSummary } from '@/types'

defineProps<{ skill: SkillSummary }>()
</script>

<style scoped>
.skill-card {
  background: #fff;
  border-radius: 14px;
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid #e4eaf3;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.skill-card:hover {
  box-shadow: 0 14px 28px rgb(31 78 121 / 12%);
  border-color: #b8d5f6;
  transform: translateY(-4px);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.card-title {
  font-size: 17px;
  font-weight: 600;
  color: #243b53;
  line-height: 1.4;
  flex: 1;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-summary {
  color: #61758a;
  font-size: 14px;
  margin: 0 0 16px;
  line-height: 1.7;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
}

.card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 10px;
}

.card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #909399;
  padding-top: 14px;
  border-top: 1px solid #edf1f6;
}

.meta-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.author-name {
  color: #606266;
  font-size: 12px;
}

.meta-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-right span {
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>
