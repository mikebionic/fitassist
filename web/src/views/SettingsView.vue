<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { mifitApi } from '../api/health'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()

const mifitStatus = ref<any>(null)
const email = ref('')
const password = ref('')
const linking = ref(false)
const syncing = ref(false)
const linkError = ref('')
const linkSuccess = ref('')

onMounted(async () => {
  try {
    const { data } = await mifitApi.status()
    mifitStatus.value = data
  } catch (e) {
    console.error(e)
  }
})

async function linkAccount() {
  linkError.value = ''
  linkSuccess.value = ''
  linking.value = true
  try {
    await mifitApi.link(email.value, password.value)
    linkSuccess.value = 'Account linked successfully!'
    const { data } = await mifitApi.status()
    mifitStatus.value = data
    email.value = ''
    password.value = ''
  } catch (e: any) {
    linkError.value = e.response?.data?.error || 'Failed to link account'
  } finally {
    linking.value = false
  }
}

async function triggerSync() {
  syncing.value = true
  try {
    await mifitApi.sync()
  } catch (e) {
    console.error(e)
  } finally {
    syncing.value = false
  }
}
</script>

<template>
  <div>
    <h1 class="page-title">Settings</h1>

    <div class="settings-section">
      <h2>Profile</h2>
      <div class="card">
        <div class="profile-row">
          <span class="profile-label">Username</span>
          <span>{{ auth.user?.username }}</span>
        </div>
        <div class="profile-row">
          <span class="profile-label">Email</span>
          <span>{{ auth.user?.email || '—' }}</span>
        </div>
        <div class="profile-row">
          <span class="profile-label">Role</span>
          <span class="badge">{{ auth.user?.role }}</span>
        </div>
      </div>
    </div>

    <div class="settings-section">
      <h2>Mi Fitness Account</h2>
      <div class="card">
        <div v-if="mifitStatus?.linked" class="linked-status">
          <div class="status-row">
            <i class="pi pi-check-circle" style="color: var(--success)"></i>
            <span>Connected: <strong>{{ mifitStatus.email }}</strong></span>
          </div>
          <div v-if="mifitStatus.last_sync" class="status-sub">
            Last sync: {{ new Date(mifitStatus.last_sync).toLocaleString() }}
          </div>
          <button class="btn-secondary" @click="triggerSync" :disabled="syncing">
            <i :class="syncing ? 'pi pi-spin pi-spinner' : 'pi pi-sync'"></i>
            Sync Now
          </button>
        </div>

        <div v-else>
          <p style="color: var(--text-secondary); margin-bottom: 1rem; font-size: 0.9rem">
            Link your Xiaomi/Mi Fitness account to sync health data.
          </p>

          <div v-if="linkError" class="error-msg">{{ linkError }}</div>
          <div v-if="linkSuccess" class="success-msg">{{ linkSuccess }}</div>

          <form @submit.prevent="linkAccount" class="link-form">
            <div class="field">
              <label>Xiaomi Email</label>
              <input v-model="email" type="email" required placeholder="your@email.com" />
            </div>
            <div class="field">
              <label>Password</label>
              <input v-model="password" type="password" required placeholder="••••••" />
            </div>
            <button type="submit" class="btn-primary" :disabled="linking">
              <i :class="linking ? 'pi pi-spin pi-spinner' : 'pi pi-link'"></i>
              Link Account
            </button>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-section { margin-bottom: 2rem; }
.settings-section h2 {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
}
.profile-row {
  display: flex;
  justify-content: space-between;
  padding: 0.625rem 0;
  border-bottom: 1px solid var(--border);
  font-size: 0.9rem;
}
.profile-row:last-child { border-bottom: none; }
.profile-label { color: var(--text-secondary); }

.badge {
  background: var(--accent);
  color: #fff;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.linked-status { display: flex; flex-direction: column; gap: 0.75rem; }
.status-row { display: flex; align-items: center; gap: 0.5rem; font-size: 0.9rem; }
.status-sub { font-size: 0.8rem; color: var(--text-secondary); }

.link-form { max-width: 400px; }
.field { margin-bottom: 1rem; }
.field label {
  display: block;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 0.375rem;
}
.field input {
  width: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.9rem;
  font-family: inherit;
  outline: none;
}
.field input:focus { border-color: var(--accent); }

.btn-primary, .btn-secondary {
  padding: 0.6rem 1.25rem;
  border: none;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}
.btn-primary { background: var(--accent); color: #fff; }
.btn-primary:hover { opacity: 0.9; }
.btn-secondary { background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border); }
.btn-secondary:hover { background: var(--border); }
.btn-primary:disabled, .btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

.error-msg {
  background: rgba(239, 68, 68, 0.1);
  color: var(--danger);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  font-size: 0.85rem;
  margin-bottom: 1rem;
}
.success-msg {
  background: rgba(34, 197, 94, 0.1);
  color: var(--success);
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  font-size: 0.85rem;
  margin-bottom: 1rem;
}
</style>
