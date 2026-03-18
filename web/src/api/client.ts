import axios from 'axios'
import { useAuthStore } from '../stores/auth'
import router from '../router'

const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.accessToken) {
    config.headers.Authorization = `Bearer ${auth.accessToken}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const auth = useAuthStore()
    const original = error.config

    if (error.response?.status === 401 && !original._retry) {
      original._retry = true

      if (auth.refreshToken) {
        try {
          const { data } = await axios.post('/api/auth/refresh', {
            refresh_token: auth.refreshToken,
          })
          auth.setTokens(data.tokens)
          original.headers.Authorization = `Bearer ${data.tokens.access_token}`
          return api(original)
        } catch {
          auth.logout()
          router.push('/login')
        }
      } else {
        auth.logout()
        router.push('/login')
      }
    }

    return Promise.reject(error)
  }
)

export default api
