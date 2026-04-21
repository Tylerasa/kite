import axios from 'axios'

const client = axios.create({
  baseURL: typeof window !== 'undefined' ? '' : (import.meta.env.VITE_API_URL ?? ''),
  headers: { 'Content-Type': 'application/json' },
})

// Attach JWT token to every request
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('kite_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// On 401, clear token and redirect to login
client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('kite_token')
      localStorage.removeItem('kite_user_id')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default client
