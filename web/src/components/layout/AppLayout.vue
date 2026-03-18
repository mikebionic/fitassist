<script setup lang="ts">
import { useAuthStore } from '../../stores/auth'
import { useThemeStore } from '../../stores/theme'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const theme = useThemeStore()
const router = useRouter()

function handleLogout() {
  auth.logout()
  router.push('/login')
}

const navItems = [
  { label: 'Dashboard', icon: 'pi pi-home', to: '/' },
  { label: 'Sleep', icon: 'pi pi-moon', to: '/sleep' },
  { label: 'Heart Rate', icon: 'pi pi-heart', to: '/heartrate' },
  { label: 'Workouts', icon: 'pi pi-bolt', to: '/workouts' },
  { label: 'AI Assistant', icon: 'pi pi-comments', to: '/ai' },
  { label: 'Settings', icon: 'pi pi-cog', to: '/settings' },
]
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="logo">
          <i class="pi pi-heart-fill" style="color: var(--accent)"></i>
          <span>FitAssist</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          active-class="active"
          :exact="item.to === '/'"
        >
          <i :class="item.icon"></i>
          <span>{{ item.label }}</span>
        </router-link>

        <router-link
          v-if="auth.isAdmin"
          to="/admin"
          class="nav-item"
          active-class="active"
        >
          <i class="pi pi-shield"></i>
          <span>Admin</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <button class="nav-item" @click="theme.toggle()">
          <i :class="theme.dark ? 'pi pi-sun' : 'pi pi-moon'"></i>
          <span>{{ theme.dark ? 'Light' : 'Dark' }} Mode</span>
        </button>
        <button class="nav-item" @click="handleLogout">
          <i class="pi pi-sign-out"></i>
          <span>Logout</span>
        </button>
        <div class="user-info">
          <i class="pi pi-user"></i>
          <span>{{ auth.user?.username }}</span>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
}

.sidebar {
  width: var(--sidebar-width);
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 100;
}

.sidebar-header {
  padding: 1.25rem;
  border-bottom: 1px solid var(--border);
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.25rem;
  font-weight: 700;
}

.sidebar-nav {
  flex: 1;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.75rem;
  border-radius: 8px;
  font-size: 0.875rem;
  color: var(--text-secondary);
  text-decoration: none;
  transition: all 0.15s;
  border: none;
  background: none;
  cursor: pointer;
  width: 100%;
  text-align: left;
  font-family: inherit;
}
.nav-item:hover {
  background: var(--border);
  color: var(--text-primary);
}
.nav-item.active {
  background: var(--accent);
  color: #fff;
}
.nav-item i { font-size: 1rem; width: 1.25rem; text-align: center; }

.sidebar-footer {
  padding: 0.75rem;
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.main-content {
  flex: 1;
  margin-left: var(--sidebar-width);
  padding: 2rem;
  min-height: 100vh;
}
</style>
