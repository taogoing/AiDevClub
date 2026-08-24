import axios from 'axios'
import type { ApiResponse } from '@/types'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

const http = axios.create({
  baseURL: '',
  timeout: 15000,
})

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

http.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.accessToken) {
    config.headers.Authorization = `Bearer ${auth.accessToken}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const res = response.data as ApiResponse<unknown>
    if (res.code !== 0) {
      return Promise.reject(new Error(res.message || 'error'))
    }
    return response
  },
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      const auth = useAuthStore()
      if (!auth.refreshToken) {
        auth.clearAuth()
        router.push('/login')
        return Promise.reject(error)
      }

      if (isRefreshing) {
        return new Promise((resolve) => {
          pendingRequests.push((token: string) => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            resolve(http(originalRequest))
          })
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const res = await axios.post('/api/v1/auth/refresh', {
          refresh_token: auth.refreshToken,
        })
        const data = res.data.data
        auth.setAuth(data.access_token, data.refresh_token)
        pendingRequests.forEach((cb) => cb(data.access_token))
        pendingRequests = []
        originalRequest.headers.Authorization = `Bearer ${data.access_token}`
        return http(originalRequest)
      } catch {
        auth.clearAuth()
        router.push('/login')
        return Promise.reject(error)
      } finally {
        isRefreshing = false
      }
    }
    const msg = error.response?.data?.message || error.message
    return Promise.reject(new Error(msg))
  },
)

export default http
