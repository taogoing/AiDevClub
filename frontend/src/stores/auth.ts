import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { getMe } from '@/api/user'
import { logout as logoutApi } from '@/api/auth'
import type { UserProfile } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(localStorage.getItem('refresh_token'))
  const user = ref<UserProfile | null>(null)

  const isLoggedIn = computed(() => !!accessToken.value)

  function setAuth(access: string, refresh: string) {
    accessToken.value = access
    refreshToken.value = refresh
    localStorage.setItem('refresh_token', refresh)
  }

  function clearAuth() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem('refresh_token')
  }

  async function fetchUser() {
    try {
      const res = await getMe()
      user.value = res.data.data
    } catch {
      clearAuth()
    }
  }

  async function logout() {
    if (refreshToken.value) {
      try {
        await logoutApi(refreshToken.value)
      } catch { /* ignore */ }
    }
    clearAuth()
  }

  return { accessToken, refreshToken, user, isLoggedIn, setAuth, clearAuth, fetchUser, logout }
})
