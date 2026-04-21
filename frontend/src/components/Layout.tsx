import { type ReactNode } from 'react'

const styles: Record<string, React.CSSProperties> = {
  app: { minHeight: '100vh', background: '#f8f9fa' },
  nav: {
    background: '#1a1a2e', color: '#fff', padding: '0 24px',
    display: 'flex', alignItems: 'center', justifyContent: 'space-between', height: 56,
  },
  brand: { fontWeight: 700, fontSize: 20, letterSpacing: 1 },
  navLinks: { display: 'flex', gap: 24, fontSize: 14 },
  navLink: { color: '#ccc', cursor: 'pointer' },
  main: { maxWidth: 960, margin: '0 auto', padding: '32px 16px' },
}

export function Layout({ children }: { children: ReactNode }) {
  const isLoggedIn = !!localStorage.getItem('kite_token')

  function logout() {
    localStorage.removeItem('kite_token')
    localStorage.removeItem('kite_user_id')
    window.location.href = '/login'
  }

  return (
    <div style={styles.app}>
      <nav style={styles.nav}>
        <span style={styles.brand}>✦ Kite</span>
        {isLoggedIn && (
          <div style={styles.navLinks}>
            <a href="/dashboard" style={styles.navLink}>Dashboard</a>
            <a href="/deposit" style={styles.navLink}>Deposit</a>
            <a href="/convert" style={styles.navLink}>Convert</a>
            <a href="/payout" style={styles.navLink}>Payout</a>
            <a href="/transactions" style={styles.navLink}>History</a>
            <span style={{ ...styles.navLink, color: '#e57' }} onClick={logout}>Logout</span>
          </div>
        )}
      </nav>
      <main style={styles.main}>{children}</main>
    </div>
  )
}

export const card: React.CSSProperties = {
  background: '#fff', borderRadius: 12, padding: 24,
  boxShadow: '0 1px 4px rgba(0,0,0,0.08)', marginBottom: 16,
}

export const btn = (variant: 'primary' | 'secondary' = 'primary'): React.CSSProperties => ({
  padding: '10px 20px', borderRadius: 8, border: 'none', cursor: 'pointer', fontSize: 14, fontWeight: 600,
  background: variant === 'primary' ? '#1a1a2e' : '#eee',
  color: variant === 'primary' ? '#fff' : '#333',
})

export const input: React.CSSProperties = {
  width: '100%', padding: '10px 12px', borderRadius: 8,
  border: '1px solid #ddd', fontSize: 14, marginBottom: 12, display: 'block',
}

export const label: React.CSSProperties = { fontSize: 13, fontWeight: 600, marginBottom: 4, display: 'block' }

export function ErrorMsg({ msg }: { msg: string }) {
  return <div style={{ color: '#e57', background: '#fff0f0', padding: '10px 14px', borderRadius: 8, marginBottom: 12, fontSize: 14 }}>{msg}</div>
}

export function SuccessMsg({ msg }: { msg: string }) {
  return <div style={{ color: '#2a7', background: '#f0fff4', padding: '10px 14px', borderRadius: 8, marginBottom: 12, fontSize: 14 }}>{msg}</div>
}

export function Spinner() {
  return <div style={{ textAlign: 'center', padding: 40, color: '#888' }}>Loading…</div>
}

export function formatAmount(amount: number, currency: string): string {
  const display = (amount / 100).toFixed(2)
  const symbols: Record<string, string> = { USD: '$', GBP: '£', EUR: '€', NGN: '₦', KES: 'KSh' }
  return `${symbols[currency] ?? ''}${Number(display).toLocaleString()}`
}

export function apiError(err: unknown): string {
  if (err && typeof err === 'object' && 'response' in err) {
    const r = (err as { response: { data: { error: { message: string } } } }).response
    return r?.data?.error?.message ?? 'Something went wrong.'
  }
  return 'Something went wrong.'
}
