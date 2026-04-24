import axios from 'axios'

const client = axios.create({
  baseURL: '',
  headers: { 'Content-Type': 'application/json' },
})

// Attach JWT token to every request
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('kite_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// On 401, clear token and redirect to login — but not for auth endpoints
// (wrong password on /auth/login should show an error, not cause a reload)
client.interceptors.response.use(
  (res) => res,
  (err) => {
    const isAuthEndpoint = err.config?.url?.startsWith('/auth/')
    if (err.response?.status === 401 && !isAuthEndpoint) {
      localStorage.removeItem('kite_token')
      localStorage.removeItem('kite_user_id')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default client
