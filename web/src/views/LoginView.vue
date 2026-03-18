<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

const isRegister = ref(false)
const username = ref('')
const password = ref('')
const email = ref('')
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''
  loading.value = true

  try {
    if (isRegister.value) {
      await auth.register(username.value, password.value, email.value)
      isRegister.value = false
      error.value = ''
    } else {
      await auth.login(username.value, password.value)
      router.push('/')
    }
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Something went wrong'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <i class="pi pi-heart-fill" style="font-size: 2rem; color: var(--accent)"></i>
        <h1>FitAssist</h1>
        <p>AI Health Assistant</p>
      </div>

      <form @submit.prevent="handleSubmit" class="login-form">
        <h2>{{ isRegister ? 'Create Account' : 'Sign In' }}</h2>

        <div v-if="error" class="error-msg">{{ error }}</div>

        <div class="field">
          <label>Username</label>
          <input v-model="username" type="text" required autocomplete="username" placeholder="admin" />
        </div>

        <div v-if="isRegister" class="field">
          <label>Email</label>
          <input v-model="email" type="email" placeholder="you@example.com" />
        </div>

        <div class="field">
          <label>Password</label>
          <input v-model="password" type="password" required autocomplete="current-password" placeholder="••••••" />
        </div>

        <button type="submit" class="btn-primary" :disabled="loading">
          <i v-if="loading" class="pi pi-spin pi-spinner"></i>
          {{ isRegister ? 'Register' : 'Login' }}
        </button>

        <p class="toggle-text">
          {{ isRegister ? 'Already have an account?' : "Don't have an account?" }}
          <a href="#" @click.prevent="isRegister = !isRegister">
            {{ isRegister ? 'Sign In' : 'Register' }}
          </a>
        </p>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
}
.login-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 2.5rem;
  width: 100%;
  max-width: 400px;
}
.login-header {
  text-align: center;
  margin-bottom: 2rem;
}
.login-header h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-top: 0.5rem;
}
.login-header p {
  color: var(--text-secondary);
  font-size: 0.9rem;
}
.login-form h2 {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 1.25rem;
}
.field {
  margin-bottom: 1rem;
}
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
  transition: border-color 0.15s;
}
.field input:focus {
  border-color: var(--accent);
}
.btn-primary {
  width: 100%;
  padding: 0.7rem;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  transition: opacity 0.15s;
}
.btn-primary:hover { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.toggle-text {
  text-align: center;
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-top: 1rem;
}
.error-msg {
  background: rgba(239, 68, 68, 0.1);
  color: var(--danger);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  font-size: 0.85rem;
  margin-bottom: 1rem;
}
</style>
