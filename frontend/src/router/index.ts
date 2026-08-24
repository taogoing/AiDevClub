import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: () => import('@/components/AppLayout.vue'),
      children: [
        { path: '', name: 'home', component: () => import('@/views/HomeView.vue') },
        { path: 'skills', name: 'skills', component: () => import('@/views/SkillsView.vue') },
        { path: 'mcps', name: 'mcps', component: () => import('@/views/McpsView.vue') },
        { path: 'articles/:id', name: 'article-detail', component: () => import('@/views/ArticleDetailView.vue') },
        {
          path: 'articles/new',
          name: 'article-new',
          component: () => import('@/views/ArticleEditView.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'articles/:id/edit',
          name: 'article-edit',
          component: () => import('@/views/ArticleEditView.vue'),
          meta: { requiresAuth: true },
        },
        { path: 'login', name: 'login', component: () => import('@/views/LoginView.vue') },
        { path: 'register', name: 'register', component: () => import('@/views/RegisterView.vue') },
        {
          path: 'users/me',
          name: 'profile',
          component: () => import('@/views/ProfileView.vue'),
          meta: { requiresAuth: true },
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth) {
    const auth = useAuthStore()
    if (!auth.isLoggedIn) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }
})

export default router
