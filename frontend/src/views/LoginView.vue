<template>
  <div class="auth-container">
    <div class="auth-card">
      <h2>登录</h2>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="form.email" placeholder="邮箱" type="email" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" placeholder="密码" type="password" size="large" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" style="width: 100%" @click="handleLogin" :loading="loading">
            登录
          </el-button>
        </el-form-item>
      </el-form>
      <p class="auth-link">
        还没有账号？<router-link to="/register">立即注册</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const loading = ref(false)
const form = ref({ email: '', password: '' })

async function handleLogin() {
  if (!form.value.email || !form.value.password) {
    ElMessage.warning('请填写邮箱和密码')
    return
  }
  loading.value = true
  try {
    const res = await login(form.value.email, form.value.password)
    const data = res.data.data
    auth.setAuth(data.access_token, data.refresh_token)
    await auth.fetchUser()
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: calc(100vh - 200px);
}

.auth-card {
  background: #fff;
  border-radius: 8px;
  padding: 40px;
  width: 400px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.auth-card h2 {
  text-align: center;
  margin-bottom: 24px;
}

.auth-link {
  text-align: center;
  font-size: 14px;
  color: #909399;
}
</style>
