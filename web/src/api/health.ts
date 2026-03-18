import api from './client'

export interface DashboardSummary {
  steps_today: number | null
  calories_today: number | null
  distance_today: number | null
  sleep_last_night_min: number | null
  avg_hr_today: number | null
  last_hr: number | null
}

export interface StepsData {
  id: number
  date: string
  total_steps: number | null
  distance_m: number | null
  calories: number | null
  active_minutes: number | null
}

export interface SleepData {
  id: number
  date: string
  sleep_start: string | null
  sleep_end: string | null
  duration_min: number | null
  deep_min: number | null
  light_min: number | null
  rem_min: number | null
  awake_min: number | null
}

export interface HeartRateData {
  measured_at: string
  bpm: number
  type: string
}

export interface WorkoutData {
  id: number
  workout_type: string
  started_at: string
  ended_at: string | null
  duration_sec: number | null
  distance_m: number | null
  calories: number | null
  avg_heartrate: number | null
  max_heartrate: number | null
  avg_pace: number | null
}

export interface StressData {
  measured_at: string
  value: number
}

export interface SpO2Data {
  measured_at: string
  value: number
}

export const healthApi = {
  dashboard: () => api.get<DashboardSummary>('/health/dashboard'),
  steps: (from: string, to: string) => api.get<StepsData[]>(`/health/steps?from=${from}&to=${to}`),
  sleep: (from: string, to: string) => api.get<SleepData[]>(`/health/sleep?from=${from}&to=${to}`),
  heartrate: (from: string, to: string) => api.get<HeartRateData[]>(`/health/heartrate?from=${from}&to=${to}`),
  workouts: (from: string, to: string) => api.get<WorkoutData[]>(`/health/workouts?from=${from}&to=${to}`),
  workout: (id: number) => api.get<WorkoutData>(`/health/workouts/${id}`),
  stress: (from: string, to: string) => api.get<StressData[]>(`/health/stress?from=${from}&to=${to}`),
  spo2: (from: string, to: string) => api.get<SpO2Data[]>(`/health/spo2?from=${from}&to=${to}`),
}

export const mifitApi = {
  status: () => api.get('/mifit/status'),
  link: (email: string, password: string) => api.post('/mifit/link', { email, password }),
  sync: () => api.post('/mifit/sync'),
}

export const adminApi = {
  users: (limit = 50, offset = 0) => api.get(`/admin/users?limit=${limit}&offset=${offset}`),
  updateUser: (id: string, data: any) => api.patch(`/admin/users/${id}`, data),
  chats: () => api.get('/admin/chats'),
  updateChat: (id: number, data: any) => api.patch(`/admin/chats/${id}`, data),
  syncLogs: (limit = 50) => api.get(`/admin/sync-logs?limit=${limit}`),
  exportDB: (format: string) => api.get(`/admin/export?format=${format}`, { responseType: 'blob' }),
}
