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
        { path: 'search', name: 'search', component: () => import('@/views/SearchView.vue') },
        { path: 'skills', name: 'skills', component: () => import('@/views/SkillsView.vue') },
        { path: 'skills/new', name: 'skill-new', component: () => import('@/views/SkillEditView.vue'), meta: { requiresAuth: true } },
        { path: 'skills/:id', name: 'skill-detail', component: () => import('@/views/SkillDetailView.vue') },
        { path: 'skills/:id/edit', name: 'skill-edit', component: () => import('@/views/SkillEditView.vue'), meta: { requiresAuth: true } },
        { path: 'mcps', name: 'mcps', component: () => import('@/views/McpsView.vue') },
        { path: 'mcps/new', name: 'mcp-new', component: () => import('@/views/McpServerEditView.vue'), meta: { requiresAuth: true } },
        { path: 'mcps/:id', name: 'mcp-detail', component: () => import('@/views/McpServerDetailView.vue') },
        { path: 'mcps/:id/edit', name: 'mcp-edit', component: () => import('@/views/McpServerEditView.vue'), meta: { requiresAuth: true } },
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
    {
      path: '/admin',
      component: () => import('@/components/AdminLayout.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
      children: [
        { path: '', name: 'admin-home', redirect: '/admin/tags' },
        { path: 'tags', name: 'admin-tags', component: () => import('@/views/admin/TagManagement.vue') },
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
