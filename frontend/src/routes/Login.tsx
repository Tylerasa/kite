import { useState } from 'react'
import { useLogin } from '../api/hooks'
import { Layout, card, btn, input, label, ErrorMsg } from '../components/Layout'
import { apiError } from '../components/Layout'

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const login = useLogin()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    try {
      const result = await login.mutateAsync({ email, password })
      localStorage.setItem('kite_token', result.token)
      localStorage.setItem('kite_user_id', result.user_id)
      window.location.href = '/dashboard'
    } catch (_) {}
  }

  return (
    <Layout>
      <div style={{ maxWidth: 400, margin: '60px auto' }}>
        <div style={card}>
          <h2 style={{ marginBottom: 20 }}>Sign in to Kite</h2>
          {login.isError && <ErrorMsg msg={apiError(login.error)} />}
          <form onSubmit={handleSubmit}>
            <label style={label}>Email</label>
            <input style={input} type="email" value={email} onChange={e => setEmail(e.target.value)} required />
            <label style={label}>Password</label>
            <input style={input} type="password" value={password} onChange={e => setPassword(e.target.value)} required />
            <button style={btn()} type="submit" disabled={login.isPending}>
              {login.isPending ? 'Signing in…' : 'Sign in'}
            </button>
          </form>
          <p style={{ marginTop: 16, fontSize: 14, color: '#666' }}>
            Don't have an account? <a href="/signup" style={{ color: '#1a1a2e', fontWeight: 600 }}>Sign up</a>
          </p>
        </div>
      </div>
    </Layout>
  )
}
