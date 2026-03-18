<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi } from '../api/health'

const tab = ref<'users' | 'chats' | 'logs'>('users')
const users = ref<any[]>([])
const chats = ref<any[]>([])
const logs = ref<any[]>([])

onMounted(async () => {
  await Promise.all([loadUsers(), loadChats(), loadLogs()])
})

async function loadUsers() {
  try {
    const { data } = await adminApi.users()
    users.value = data
  } catch (e) { console.error(e) }
}

async function loadChats() {
  try {
    const { data } = await adminApi.chats()
    chats.value = data || []
  } catch (e) { console.error(e) }
}

async function loadLogs() {
  try {
    const { data } = await adminApi.syncLogs()
    logs.value = data || []
  } catch (e) { console.error(e) }
}

async function toggleUserActive(user: any) {
  try {
    await adminApi.updateUser(user.id, { is_active: !user.is_active })
    await loadUsers()
  } catch (e) { console.error(e) }
}

async function exportDB(format: string) {
  try {
    const { data } = await adminApi.exportDB(format)
    const blob = new Blob([data])
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `fitassist_export.${format}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) { console.error(e) }
}

function fmtDate(d: string) {
  return new Date(d).toLocaleString()
}
</script>

<template>
  <div>
    <h1 class="page-title">Admin Panel</h1>

    <div class="tabs">
      <button :class="{ active: tab === 'users' }" @click="tab = 'users'">
        <i class="pi pi-users"></i> Users
      </button>
      <button :class="{ active: tab === 'chats' }" @click="tab = 'chats'">
        <i class="pi pi-comments"></i> Telegram Chats
      </button>
      <button :class="{ active: tab === 'logs' }" @click="tab = 'logs'">
        <i class="pi pi-history"></i> Sync Logs
      </button>
    </div>

    <!-- Users -->
    <div v-if="tab === 'users'" class="card">
      <table class="data-table">
        <thead>
          <tr><th>Username</th><th>Email</th><th>Role</th><th>Active</th><th>Created</th><th>Actions</th></tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td><strong>{{ u.username }}</strong></td>
            <td>{{ u.email || '—' }}</td>
            <td><span class="badge" :class="u.role">{{ u.role }}</span></td>
            <td><span :style="{ color: u.is_active ? 'var(--success)' : 'var(--danger)' }">{{ u.is_active ? 'Yes' : 'No' }}</span></td>
            <td>{{ fmtDate(u.created_at) }}</td>
            <td>
              <button class="btn-sm" @click="toggleUserActive(u)">
                {{ u.is_active ? 'Disable' : 'Enable' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Telegram Chats -->
    <div v-if="tab === 'chats'" class="card">
      <div v-if="chats.length === 0" class="empty-state">
        <i class="pi pi-comments"></i>
        <p>No Telegram chats yet. Start the bot and send /start.</p>
      </div>
      <table v-else class="data-table">
        <thead>
          <tr><th>Chat ID</th><th>Username</th><th>Name</th><th>Status</th><th>Created</th></tr>
        </thead>
        <tbody>
          <tr v-for="c in chats" :key="c.id">
            <td>{{ c.chat_id }}</td>
            <td>{{ c.username || '—' }}</td>
            <td>{{ c.first_name || '—' }}</td>
            <td>
              <span v-if="c.is_blocked" style="color: var(--danger)">Blocked</span>
              <span v-else-if="c.is_approved" style="color: var(--success)">Approved</span>
              <span v-else style="color: var(--warning)">Pending</span>
            </td>
            <td>{{ fmtDate(c.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Sync Logs -->
    <div v-if="tab === 'logs'" class="card">
      <div v-if="logs.length === 0" class="empty-state">
        <i class="pi pi-history"></i>
        <p>No sync logs yet</p>
      </div>
      <table v-else class="data-table">
        <thead>
          <tr><th>Type</th><th>Status</th><th>Records</th><th>Started</th><th>Error</th></tr>
        </thead>
        <tbody>
          <tr v-for="l in logs" :key="l.id">
            <td>{{ l.sync_type || '—' }}</td>
            <td>
              <span :style="{ color: l.status === 'success' ? 'var(--success)' : 'var(--danger)' }">{{ l.status }}</span>
            </td>
            <td>{{ l.records_synced }}</td>
            <td>{{ fmtDate(l.started_at) }}</td>
            <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis">{{ l.error_message || '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Export -->
    <div class="card" style="margin-top: 1.5rem">
      <h3 style="margin-bottom: 0.75rem; font-size: 1rem; font-weight: 600">Database Export</h3>
      <div style="display: flex; gap: 0.5rem">
        <button class="btn-secondary" @click="exportDB('sql')">
          <i class="pi pi-download"></i> Export SQL
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 1rem;
}
.tabs button {
  padding: 0.5rem 1rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 0.375rem;
}
.tabs button.active {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}
.data-table th {
  text-align: left;
  padding: 0.625rem 0.75rem;
  color: var(--text-secondary);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
}
.data-table td {
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid var(--border);
}
.data-table tr:last-child td { border-bottom: none; }

.badge {
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
}
.badge.admin { background: rgba(239, 68, 68, 0.15); color: var(--danger); }
.badge.user { background: rgba(99, 102, 241, 0.15); color: var(--accent); }

.btn-sm {
  padding: 0.25rem 0.5rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  cursor: pointer;
  font-family: inherit;
  font-size: 0.75rem;
}
.btn-sm:hover { background: var(--border); }

.btn-secondary {
  padding: 0.5rem 1rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  cursor: pointer;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}
.btn-secondary:hover { background: var(--border); }
</style>
