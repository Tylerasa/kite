import { Layout, card, Spinner } from '../components/Layout'

export default function TransactionDetail() {
  const id = window.location.pathname.split('/').pop() ?? ''

  return (
    <Layout>
      <div style={{ maxWidth: 560, margin: '0 auto' }}>
        <a href="/transactions" style={{ fontSize: 13, color: '#888', display: 'block', marginBottom: 16 }}>← Back to history</a>
        <h1 style={{ marginBottom: 24 }}>Transaction Detail</h1>
        <div style={card}>
          <p style={{ fontSize: 13, color: '#888', wordBreak: 'break-all' }}>ID: {id}</p>
          <p style={{ marginTop: 16, color: '#555', fontSize: 14 }}>
            Full transaction detail (ledger entries, amounts, timestamps) would be fetched here from
            a <code>/wallets/transactions/{id}</code> endpoint. The base data is visible in the history list.
          </p>
        </div>
      </div>
    </Layout>
  )
}
