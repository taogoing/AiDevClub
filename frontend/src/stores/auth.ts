import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { getMe } from '@/api/user'
import { refreshToken as refreshTokenApi, logout as logoutApi } from '@/api/auth'
import type { UserProfile } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(localStorage.getItem('refresh_token'))
  const user = ref<UserProfile | null>(null)
  let restorePromise: Promise<void> | null = null

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

  async function restoreSession(): Promise<void> {
    if (user.value) return
    if (restorePromise) return restorePromise
    restorePromise = (async () => {
      if (!accessToken.value && refreshToken.value) {
        try {
          const res = await refreshTokenApi(refreshToken.value)
          const data = res.data.data
          setAuth(data.access_token, data.refresh_token)
        } catch {
          clearAuth()
          return
        }
      }
      if (!accessToken.value) return
      await fetchUser()
    })().finally(() => { restorePromise = null })
    return restorePromise
  }

  async function logout() {
    if (refreshToken.value) {
      try {
        await logoutApi(refreshToken.value)
      } catch { /* ignore */ }
    }
    clearAuth()
  }

  return { accessToken, refreshToken, user, isLoggedIn, setAuth, clearAuth, fetchUser, restoreSession, logout }
})
