<template>
  <div class="app-layout">
    <header class="navbar">
      <div class="navbar-inner">
        <router-link to="/" class="logo">AIDevClub</router-link>
        <nav class="navbar-nav">
          <router-link to="/" class="nav-link" active-class="active" :class="{ active: $route.path === '/' }">文章</router-link>
          <router-link to="/skills" class="nav-link" active-class="active">Skills Hub</router-link>
          <router-link to="/mcps" class="nav-link" active-class="active">MCP Hub</router-link>
        </nav>
        <div class="navbar-right">
          <template v-if="auth.isLoggedIn">
            <el-dropdown trigger="click" @command="handleCommand">
              <div class="user-avatar-wrap">
                <el-avatar :size="32" :src="auth.user?.avatar_url || undefined">
                  {{ auth.user?.nickname?.charAt(0) || '?' }}
                </el-avatar>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="profile">个人中心</el-dropdown-item>
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
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

onMounted(async () => {
  if (auth.isLoggedIn && !auth.user) {
    await auth.fetchUser()
  }
})

function handleCommand(cmd: string) {
  if (cmd === 'profile') router.push('/users/me')
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
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
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
  color: #409eff;
  text-decoration: none;
  margin-right: 40px;
}

.navbar-nav {
  display: flex;
  gap: 32px;
  flex: 1;
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
  color: #409eff;
  border-bottom-color: #409eff;
}

.navbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
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
  border-top: 1px solid #e4e7ed;
  background: #fff;
}
</style>
