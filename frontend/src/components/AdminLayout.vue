<template>
  <div class="admin-layout">
    <el-container>
      <el-aside width="200px" class="admin-sidebar">
        <div class="admin-logo">
          <router-link to="/">AIDevClub</router-link>
          <span class="admin-badge">管理后台</span>
        </div>
        <el-menu
          :default-active="activeMenu"
          router
          class="admin-menu"
        >
          <el-menu-item index="/admin/tags">
            <el-icon><PriceTag /></el-icon>
            <span>标签管理</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
      <el-container>
        <el-header class="admin-header">
          <div class="admin-header-left">
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/admin' }">管理后台</el-breadcrumb-item>
              <el-breadcrumb-item>{{ currentSection }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          <div class="admin-header-right">
            <el-button text @click="$router.push('/')">
              <el-icon><HomeFilled /></el-icon>
              返回前台
            </el-button>
            <el-dropdown trigger="click" @command="handleCommand">
              <div class="user-info">
                <el-avatar :size="28" :src="auth.user?.avatar_url || undefined">
                  {{ auth.user?.nickname?.charAt(0) || '?' }}
                </el-avatar>
                <span class="user-name">{{ auth.user?.nickname }}</span>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                  <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </el-header>
        <el-main class="admin-main">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { PriceTag, HomeFilled } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

onMounted(async () => {
  if (auth.isLoggedIn && !auth.user) {
    await auth.fetchUser()
  }
})

const activeMenu = computed(() => route.path)

const currentSection = computed(() => {
  const map: Record<string, string> = {
    '/admin/tags': '标签管理',
  }
  return map[route.path] || '管理'
})

function handleCommand(cmd: string) {
  if (cmd === 'profile') {
    router.push('/users/me')
  } else if (cmd === 'logout') {
    auth.logout().then(() => router.push('/'))
  }
}
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
}

.admin-layout :deep(.el-container) {
  min-height: 100vh;
}

.admin-sidebar {
  background: #304156;
  overflow-y: auto;
}

.admin-logo {
  height: 56px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.admin-logo a {
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  text-decoration: none;
}

.admin-badge {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
  margin-top: 2px;
}

.admin-menu {
  border-right: none;
  background: #304156;
}

.admin-menu :deep(.el-menu-item) {
  color: rgba(255, 255, 255, 0.8);
}

.admin-menu :deep(.el-menu-item:hover),
.admin-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background: #263445;
}

.admin-header {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
}

.admin-header-left {
  display: flex;
  align-items: center;
}

.admin-header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.user-name {
  font-size: 14px;
  color: #606266;
}

.admin-main {
  background: #f5f7fa;
  padding: 24px;
}
</style>
