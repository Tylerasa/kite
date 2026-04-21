import { useBalances, useTransactions } from '../api/hooks'
import { Layout, card, Spinner, formatAmount } from '../components/Layout'

const CURRENCY_FLAGS: Record<string, string> = {
  USD: '🇺🇸', GBP: '🇬🇧', EUR: '🇪🇺', NGN: '🇳🇬', KES: '🇰🇪',
}

export default function Dashboard() {
  const balances = useBalances()
  const transactions = useTransactions(1)

  return (
    <Layout>
      <h1 style={{ marginBottom: 24 }}>Dashboard</h1>

      {/* Balances */}
      <div style={card}>
        <h3 style={{ marginBottom: 16 }}>Your Balances</h3>
        {balances.isLoading ? <Spinner /> : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 12 }}>
            {balances.data?.map(b => (
              <div key={b.currency} style={{
                background: '#f8f9fa', borderRadius: 10, padding: '16px 20px',
                borderLeft: '4px solid #1a1a2e',
              }}>
                <div style={{ fontSize: 22, marginBottom: 4 }}>{CURRENCY_FLAGS[b.currency]}</div>
                <div style={{ fontSize: 12, color: '#888', marginBottom: 2 }}>{b.currency}</div>
                <div style={{ fontSize: 20, fontWeight: 700 }}>{formatAmount(b.amount, b.currency)}</div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Quick actions */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 24 }}>
        {[
          { label: '+ Deposit', href: '/deposit', bg: '#e8f5e9' },
          { label: '⇄ Convert', href: '/convert', bg: '#e3f2fd' },
          { label: '↗ Payout', href: '/payout', bg: '#fce4ec' },
        ].map(a => (
          <a key={a.href} href={a.href} style={{
            background: a.bg, borderRadius: 8, padding: '12px 20px',
            fontWeight: 600, fontSize: 14, flex: 1, textAlign: 'center',
          }}>
            {a.label}
          </a>
        ))}
      </div>

      {/* Recent transactions */}
      <div style={card}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
          <h3>Recent Transactions</h3>
          <a href="/transactions" style={{ fontSize: 13, color: '#1a1a2e', fontWeight: 600 }}>View all →</a>
        </div>
        {transactions.isLoading ? <Spinner /> : (
          transactions.data?.items?.length ? (
            transactions.data.items.slice(0, 5).map(tx => (
              <div key={tx.id} style={{
                display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                padding: '12px 0', borderBottom: '1px solid #f0f0f0',
              }}>
                <div>
                  <div style={{ fontWeight: 600, textTransform: 'capitalize' }}>{tx.type.replace('_', ' ')}</div>
                  <div style={{ fontSize: 12, color: '#888' }}>{new Date(tx.created_at).toLocaleString()}</div>
                </div>
                <a href={`/transactions/${tx.id}`} style={{ fontSize: 12, color: '#1a1a2e' }}>Details →</a>
              </div>
            ))
          ) : (
            <p style={{ color: '#888', textAlign: 'center', padding: 24 }}>No transactions yet. Make a deposit to get started.</p>
          )
        )}
      </div>
    </Layout>
  )
}
