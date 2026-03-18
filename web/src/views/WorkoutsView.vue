<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { healthApi, type WorkoutData } from '../api/health'

const workouts = ref<WorkoutData[]>([])
const loading = ref(true)

function daysAgo(n: number) {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

onMounted(async () => {
  try {
    const { data } = await healthApi.workouts(daysAgo(90), daysAgo(0))
    workouts.value = data || []
  } finally {
    loading.value = false
  }
})

function fmtDuration(sec: number | null) {
  if (!sec) return '—'
  const m = Math.floor(sec / 60)
  const s = sec % 60
  if (m >= 60) return `${Math.floor(m / 60)}h ${m % 60}m`
  return `${m}m ${s}s`
}

function fmtDate(d: string) {
  return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

function fmtTime(d: string) {
  return new Date(d).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function fmtDistance(m: number | null) {
  if (!m) return '—'
  return (m / 1000).toFixed(2) + ' km'
}

function typeIcon(t: string): string {
  const map: Record<string, string> = {
    running: 'pi-directions-run',
    walking: 'pi-map',
    cycling: 'pi-car',
    swimming_pool: 'pi-wave-pulse',
    yoga: 'pi-sun',
    strength: 'pi-bolt',
    hiking: 'pi-compass',
  }
  return map[t] || 'pi-bolt'
}

function typeLabel(t: string): string {
  return t.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Workouts</h1>
    </div>

    <div v-if="loading" class="empty-state">
      <i class="pi pi-spin pi-spinner"></i>
      <p>Loading...</p>
    </div>

    <div v-else-if="workouts.length === 0" class="empty-state">
      <i class="pi pi-bolt"></i>
      <p>No workouts recorded yet</p>
    </div>

    <div v-else class="workout-list">
      <div v-for="w in workouts" :key="w.id" class="workout-card">
        <div class="workout-header">
          <div class="workout-type">
            <i :class="'pi ' + typeIcon(w.workout_type)"></i>
            <span>{{ typeLabel(w.workout_type) }}</span>
          </div>
          <div class="workout-date">
            {{ fmtDate(w.started_at) }} {{ fmtTime(w.started_at) }}
          </div>
        </div>
        <div class="workout-stats">
          <div class="ws">
            <span class="ws-label">Duration</span>
            <span class="ws-value">{{ fmtDuration(w.duration_sec) }}</span>
          </div>
          <div class="ws">
            <span class="ws-label">Distance</span>
            <span class="ws-value">{{ fmtDistance(w.distance_m) }}</span>
          </div>
          <div class="ws">
            <span class="ws-label">Calories</span>
            <span class="ws-value">{{ w.calories ?? '—' }}</span>
          </div>
          <div class="ws">
            <span class="ws-label">Avg HR</span>
            <span class="ws-value">{{ w.avg_heartrate ?? '—' }} bpm</span>
          </div>
          <div class="ws">
            <span class="ws-label">Max HR</span>
            <span class="ws-value">{{ w.max_heartrate ?? '—' }} bpm</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.workout-list { display: flex; flex-direction: column; gap: 0.75rem; }

.workout-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
}

.workout-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.workout-type {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  font-size: 1rem;
}
.workout-type i { color: var(--accent); }

.workout-date {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.workout-stats {
  display: flex;
  gap: 2rem;
  flex-wrap: wrap;
}

.ws {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}
.ws-label {
  font-size: 0.7rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 600;
}
.ws-value {
  font-size: 0.95rem;
  font-weight: 600;
}
</style>
