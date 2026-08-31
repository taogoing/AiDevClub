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
        {
          path: 'my/articles',
          name: 'my-articles',
          component: () => import('@/views/MyArticlesView.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'my/skills',
          name: 'my-skills',
          component: () => import('@/views/MySkillsView.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'my/mcps',
          name: 'my-mcps',
          component: () => import('@/views/MyMcpServersView.vue'),
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
        {
          path: 'notifications',
          name: 'notifications',
          component: () => import('@/views/NotificationsView.vue'),
          meta: { requiresAuth: true },
        },
      ],
    },
    {
      path: '/admin',
      component: () => import('@/components/AdminLayout.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
      children: [
        { path: '', name: 'admin-home', redirect: '/admin/dashboard' },
        { path: 'dashboard', name: 'admin-dashboard', component: () => import('@/views/admin/DashboardView.vue') },
        { path: 'users', name: 'admin-users', component: () => import('@/views/admin/UsersView.vue') },
        { path: 'articles', name: 'admin-articles', component: () => import('@/views/admin/ArticlesView.vue') },
        { path: 'comments', name: 'admin-comments', component: () => import('@/views/admin/CommentsView.vue') },
        { path: 'resources', name: 'admin-resources', component: () => import('@/views/admin/ResourcesView.vue') },
        { path: 'tags', name: 'admin-tags', component: () => import('@/views/admin/TagManagement.vue') },
        { path: 'reports', name: 'admin-reports', component: () => import('@/views/admin/ReportsView.vue') },
        { path: 'announcements', name: 'admin-announcements', component: () => import('@/views/admin/AnnouncementsView.vue') },
        { path: 'logs', name: 'admin-logs', component: () => import('@/views/admin/LogsView.vue') },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth) {
    await auth.restoreSession()
    if (!auth.isLoggedIn) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }
  if (to.meta.requiresAdmin && auth.user?.role !== 'admin') {
    return '/'
  }
})

export default router
