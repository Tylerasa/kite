import { useState } from 'react'
import { useTransactions } from '../api/hooks'
import { Layout, card, Spinner, btn } from '../components/Layout'

const typeColor: Record<string, string> = {
  deposit: '#2a7', conversion: '#48a', payout: '#e57', reversal: '#f90',
}

export default function Transactions() {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useTransactions(page)

  return (
    <Layout>
      <h1 style={{ marginBottom: 24 }}>Transaction History</h1>
      <div style={card}>
        {isLoading ? <Spinner /> : (
          <>
            {!data?.items?.length ? (
              <p style={{ color: '#888', textAlign: 'center', padding: 24 }}>No transactions yet.</p>
            ) : data.items.map(tx => (
              <a href={`/transactions/${tx.id}`} key={tx.id} style={{
                display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                padding: '14px 0', borderBottom: '1px solid #f0f0f0', textDecoration: 'none', color: 'inherit',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                  <span style={{
                    background: (typeColor[tx.type] ?? '#888') + '22',
                    color: typeColor[tx.type] ?? '#888',
                    borderRadius: 20, padding: '2px 10px', fontSize: 12, fontWeight: 700, textTransform: 'uppercase',
                  }}>
                    {tx.type}
                  </span>
                  <span style={{ fontSize: 13, color: '#888' }}>{new Date(tx.created_at).toLocaleString()}</span>
                </div>
                <span style={{ fontSize: 13, color: '#1a1a2e' }}>Details →</span>
              </a>
            ))}

            {data && data.total_pages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 16 }}>
                <button style={btn('secondary')} onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>
                  ← Previous
                </button>
                <span style={{ fontSize: 13, color: '#888' }}>Page {page} of {data.total_pages}</span>
                <button style={btn('secondary')} onClick={() => setPage(p => p + 1)} disabled={page >= data.total_pages}>
                  Next →
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </Layout>
  )
}
