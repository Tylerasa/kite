import { useState } from 'react'
import { useSignup } from '../api/hooks'
import { Layout, card, btn, input, label, ErrorMsg } from '../components/Layout'
import { apiError } from '../components/Layout'

export default function Signup() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const signup = useSignup()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    try {
      const result = await signup.mutateAsync({ email, password })
      localStorage.setItem('kite_token', result.token)
      localStorage.setItem('kite_user_id', result.user_id)
      window.location.href = '/dashboard'
    } catch (_) {}
  }

  return (
    <Layout>
      <div style={{ maxWidth: 400, margin: '60px auto' }}>
        <div style={card}>
          <h2 style={{ marginBottom: 20 }}>Create your Kite account</h2>
          {signup.isError && <ErrorMsg msg={apiError(signup.error)} />}
          <form onSubmit={handleSubmit}>
            <label style={label}>Email</label>
            <input style={input} type="email" value={email} onChange={e => setEmail(e.target.value)} required />
            <label style={label}>Password <span style={{ color: '#999', fontWeight: 400 }}>(min 8 chars)</span></label>
            <input style={input} type="password" value={password} minLength={8} onChange={e => setPassword(e.target.value)} required />
            <button style={btn()} type="submit" disabled={signup.isPending}>
              {signup.isPending ? 'Creating account…' : 'Create account'}
            </button>
          </form>
          <p style={{ marginTop: 16, fontSize: 14, color: '#666' }}>
            Already have an account? <a href="/login" style={{ color: '#1a1a2e', fontWeight: 600 }}>Sign in</a>
          </p>
        </div>
      </div>
    </Layout>
  )
}
