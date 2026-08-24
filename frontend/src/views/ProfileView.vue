<template>
  <div class="page-container">
    <h2>个人中心</h2>
    <div v-loading="loading" class="profile-content">
      <div class="profile-section">
        <h3>基本信息</h3>
        <div class="avatar-area">
          <el-avatar :size="80" :src="auth.user?.avatar_url || undefined">
            {{ auth.user?.nickname?.charAt(0) || '?' }}
          </el-avatar>
          <el-upload
            :show-file-list="false"
            :before-upload="handleAvatarUpload"
            accept="image/*"
          >
            <el-button size="small">更换头像</el-button>
          </el-upload>
        </div>
        <el-form :model="profileForm" label-width="80px" style="max-width: 500px">
          <el-form-item label="昵称">
            <el-input v-model="profileForm.nickname" />
          </el-form-item>
          <el-form-item label="简介">
            <el-input v-model="profileForm.bio" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleUpdateProfile" :loading="saving">保存</el-button>
          </el-form-item>
        </el-form>
      </div>

      <el-divider />

      <div class="profile-section">
        <h3>修改密码</h3>
        <el-form style="max-width: 500px">
          <el-form-item label="新密码">
            <el-input v-model="newPassword" type="password" show-password placeholder="输入新密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleUpdatePassword" :loading="savingPwd">修改密码</el-button>
          </el-form-item>
        </el-form>
      </div>

      <el-divider />

      <div class="profile-section danger-zone">
        <h3>危险操作</h3>
        <el-button type="danger" @click="handleDeleteAccount">注销账号</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { updateMe, updatePassword, deleteAccount, uploadAvatar } from '@/api/user'
import type { UploadFile } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const savingPwd = ref(false)
const newPassword = ref('')
const profileForm = ref({ nickname: '', bio: '' })

onMounted(async () => {
  loading.value = true
  if (!auth.user) await auth.fetchUser()
  if (auth.user) {
    profileForm.value.nickname = auth.user.nickname
    profileForm.value.bio = auth.user.bio
  }
  loading.value = false
})

async function handleAvatarUpload(file: UploadFile) {
  try {
    const res = await uploadAvatar(file as unknown as File)
    const url = res.data.data.avatar_url
    await updateMe({ avatar_url: url })
    await auth.fetchUser()
    ElMessage.success('头像已更新')
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  }
  return false
}

async function handleUpdateProfile() {
  saving.value = true
  try {
    await updateMe(profileForm.value)
    await auth.fetchUser()
    ElMessage.success('资料已更新')
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function handleUpdatePassword() {
  if (!newPassword.value) {
    ElMessage.warning('请输入新密码')
    return
  }
  savingPwd.value = true
  try {
    await updatePassword(newPassword.value)
    newPassword.value = ''
    ElMessage.success('密码已修改')
  } catch (e: unknown) {
    ElMessage.error((e as Error).message)
  } finally {
    savingPwd.value = false
  }
}

async function handleDeleteAccount() {
  try {
    await ElMessageBox.confirm('注销后无法恢复，确定要注销账号？', '警告', {
      type: 'warning',
      confirmButtonText: '确认注销',
      cancelButtonText: '取消',
    })
    await deleteAccount()
    auth.clearAuth()
    router.push('/')
    ElMessage.success('账号已注销')
  } catch { /* cancelled */ }
}
</script>

<style scoped>
h2 {
  margin-bottom: 24px;
}

h3 {
  margin-bottom: 16px;
  font-size: 16px;
}

.profile-section {
  margin-bottom: 16px;
}

.avatar-area {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.danger-zone {
  padding: 16px;
  border: 1px solid #fde2e2;
  border-radius: 8px;
  background: #fef0f0;
}
</style>
