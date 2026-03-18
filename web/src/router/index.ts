import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('../components/layout/AppLayout.vue'),
    children: [
      { path: '', name: 'Dashboard', component: () => import('../views/DashboardView.vue') },
      { path: 'sleep', name: 'Sleep', component: () => import('../views/SleepView.vue') },
      { path: 'heartrate', name: 'HeartRate', component: () => import('../views/HeartRateView.vue') },
      { path: 'workouts', name: 'Workouts', component: () => import('../views/WorkoutsView.vue') },
      { path: 'ai', name: 'AI Assistant', component: () => import('../views/AIAssistantView.vue') },
      { path: 'settings', name: 'Settings', component: () => import('../views/SettingsView.vue') },
      { path: 'admin', name: 'Admin', component: () => import('../views/AdminView.vue'), meta: { admin: true } },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()

  if (!to.meta.public && !auth.isLoggedIn) {
    return '/login'
  }

  if (to.meta.admin && !auth.isAdmin) {
    return '/'
  }
})

export default router
