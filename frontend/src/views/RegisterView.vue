<template>
  <div class="auth-container">
    <div class="auth-card">
      <h2>注册</h2>
      <el-form :model="form" @submit.prevent="handleRegister">
        <el-form-item>
          <el-input v-model="form.email" placeholder="邮箱" type="email" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" placeholder="密码" type="password" size="large" show-password />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.nickname" placeholder="昵称（可选）" size="large" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" style="width: 100%" @click="handleRegister" :loading="loading">
            注册
          </el-button>
        </el-form-item>
      </el-form>
      <p class="auth-link">
        已有账号？<router-link to="/login">去登录</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'

const router = useRouter()
const loading = ref(false)
const form = ref({ email: '', password: '', nickname: '' })

async function handleRegister() {
  if (!form.value.email || !form.value.password) {
    ElMessage.warning('请填写邮箱和密码')
    return
  }
  loading.value = true
  try {
    await register(form.value.email, form.value.password, form.value.nickname)
    ElMessage.success('注册成功，请登录')
    router.push('/login')
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
