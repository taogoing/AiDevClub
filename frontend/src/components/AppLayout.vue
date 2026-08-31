<template>
  <div class="app-layout">
    <header class="navbar">
      <div class="navbar-inner">
        <router-link to="/" class="logo">AIDevClub</router-link>
        <nav class="navbar-nav">
          <router-link to="/" class="nav-link" exact-active-class="active">文章</router-link>
          <router-link to="/skills" class="nav-link" active-class="active" exact-active-class="active">Skills Hub</router-link>
          <router-link to="/mcps" class="nav-link" active-class="active" exact-active-class="active">MCP Hub</router-link>
        </nav>
        <div class="navbar-search">
          <el-input
            v-model="searchKeyword"
            :placeholder="searchPlaceholder"
            clearable
            @keyup.enter="handleSearch"
            :prefix-icon="Search"
          />
        </div>
        <div class="navbar-right">
          <template v-if="auth.isLoggedIn">
            <NotificationBell />
            <el-dropdown trigger="click" @command="handleCommand">
              <div class="user-avatar-wrap">
                <el-avatar :size="32" :src="auth.user?.avatar_url || undefined">
                  {{ auth.user?.nickname?.charAt(0) || '?' }}
                </el-avatar>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                  <el-dropdown-item command="my-articles">我的文章</el-dropdown-item>
                  <el-dropdown-item command="new-article">发布文章</el-dropdown-item>
                  <el-dropdown-item divided command="logout">登出</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <el-button type="primary" text @click="$router.push('/login')">登录</el-button>
            <el-button type="primary" @click="$router.push('/register')">注册</el-button>
          </template>
        </div>
      </div>
    </header>
    <main class="main-content">
      <router-view />
    </main>
    <footer class="footer">
      <p>&copy; 2026 AIDevClub. All rights reserved.</p>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import NotificationBell from './NotificationBell.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const searchKeyword = ref('')
const searchType = computed(() => {
  if (route.path === '/skills') return 'skill'
  if (route.path === '/mcps') return 'mcp_server'
  return ''
})
const searchPlaceholder = computed(() => {
  if (searchType.value === 'skill') return '搜索 Skill...'
  if (searchType.value === 'mcp_server') return '搜索 MCP Server...'
  return '搜索文章、Skill、MCP...'
})

onMounted(async () => {
  if (auth.isLoggedIn && !auth.user) {
    await auth.fetchUser()
  }
})

function handleSearch() {
  if (searchKeyword.value.trim()) {
    router.push({
      path: '/search',
      query: {
        q: searchKeyword.value.trim(),
        ...(searchType.value ? { type: searchType.value } : {}),
      },
    })
  }
}

function handleCommand(cmd: string) {
  if (cmd === 'profile') router.push('/users/me')
  else if (cmd === 'my-articles') router.push('/my/articles')
  else if (cmd === 'new-article') router.push('/articles/new')
  else if (cmd === 'logout') {
    auth.logout().then(() => router.push('/'))
  }
}
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.navbar {
  background: rgb(255 255 255 / 92%);
  border-bottom: 1px solid #e4eaf3;
  box-shadow: 0 4px 18px rgb(31 59 91 / 5%);
  position: sticky;
  top: 0;
  z-index: 100;
}

.navbar-inner {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  padding: 0 24px;
  height: 56px;
}

.logo {
  font-size: 20px;
  font-weight: 700;
  color: #1f6feb;
  text-decoration: none;
  margin-right: 40px;
}

.navbar-nav {
  display: flex;
  gap: 32px;
}

.navbar-search {
  margin-left: 32px;
  flex: 1;
  max-width: 300px;
}

.nav-link {
  font-size: 15px;
  color: #606266;
  text-decoration: none;
  padding: 8px 0;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.nav-link:hover {
  color: #409eff;
}

.nav-link.active {
  color: #1f6feb;
  border-bottom-color: #1f6feb;
}

.navbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.user-avatar-wrap {
  cursor: pointer;
}

.main-content {
  flex: 1;
}

.footer {
  text-align: center;
  padding: 24px;
  color: #999;
  font-size: 14px;
  border-top: 1px solid #e4eaf3;
  background: #fff;
}
</style>
