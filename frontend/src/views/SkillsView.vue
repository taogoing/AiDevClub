<template>
  <div class="home-container">
    <div class="main-area">
      <el-empty description="Skills Hub 即将上线" :image-size="120">
        <template #description>
          <p>Skills Hub 正在建设中，敬请期待</p>
        </template>
      </el-empty>
    </div>
    <aside class="sidebar">
      <Sidebar />
    </aside>
  </div>
</template>

<script setup lang="ts">
import Sidebar from '@/components/Sidebar.vue'
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
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.sidebar {
  width: 280px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .sidebar {
    display: none;
  }
}
</style>
