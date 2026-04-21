import { useState } from 'react'
import { useCreatePayout, usePayout } from '../api/hooks'
import { Layout, card, btn, input, label, ErrorMsg, SuccessMsg, formatAmount } from '../components/Layout'
import { apiError } from '../components/Layout'

const CURRENCIES = ['NGN', 'KES', 'USD', 'GBP', 'EUR']

const statusColor: Record<string, string> = {
  pending: '#888', processing: '#f90', successful: '#2a7', failed: '#e57', review: '#a05',
}

export default function Payout() {
  const [currency, setCurrency] = useState('NGN')
  const [amount, setAmount] = useState('')
  const [accountNumber, setAccountNumber] = useState('')
  const [bankCode, setBankCode] = useState('')
  const [accountName, setAccountName] = useState('')
  const [payoutId, setPayoutId] = useState('')
  const [error, setError] = useState('')

  const createPayout = useCreatePayout()
  const payout = usePayout(payoutId, !!payoutId)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    const amountMinor = Math.round(parseFloat(amount) * 100)
    if (isNaN(amountMinor) || amountMinor <= 0) return

    try {
      const result = await createPayout.mutateAsync({
        source_currency: currency,
        amount: amountMinor,
        recipient_account_number: accountNumber,
        recipient_bank_code: bankCode,
        recipient_account_name: accountName,
      })
      setPayoutId(result.id)
    } catch (err) {
      setError(apiError(err))
    }
  }

  if (payoutId && payout.data) {
    const p = payout.data
    return (
      <Layout>
        <div style={{ maxWidth: 480, margin: '0 auto' }}>
          <h1 style={{ marginBottom: 24 }}>Payout Status</h1>
          <div style={card}>
            {p.compliance_flagged && (
              <div style={{ background: '#fff8e1', borderRadius: 8, padding: '12px 16px', marginBottom: 16, fontSize: 14 }}>
                ⚠️ This payout has been flagged for compliance review (amount exceeds threshold). It will be processed after review.
              </div>
            )}
            <div style={{ marginBottom: 16 }}>
              <span style={{
                background: statusColor[p.status] + '22',
                color: statusColor[p.status],
                borderRadius: 20, padding: '4px 12px', fontSize: 13, fontWeight: 700,
              }}>
                {p.status.toUpperCase()}
              </span>
            </div>
            {[
              ['Amount', formatAmount(p.amount, p.source_currency)],
              ['Recipient', p.recipient_account_name],
              ['Account', p.recipient_account_number],
              ['Bank code', p.recipient_bank_code],
            ].map(([k, v]) => (
              <div key={k} style={{ display: 'flex', justifyContent: 'space-between', padding: '8px 0', borderBottom: '1px solid #f0f0f0', fontSize: 14 }}>
                <span style={{ color: '#666' }}>{k}</span>
                <span style={{ fontWeight: 600 }}>{v}</span>
              </div>
            ))}
            {p.failure_reason && <ErrorMsg msg={`Failed: ${p.failure_reason}${p.reversed_at ? ' (balance restored)' : ''}`} />}
            {p.status === 'successful' && <SuccessMsg msg="Payout delivered successfully." />}
            {(p.status === 'pending' || p.status === 'processing') && (
              <p style={{ marginTop: 12, fontSize: 13, color: '#888' }}>Polling for updates…</p>
            )}
            <button style={{ ...btn('secondary'), marginTop: 16 }} onClick={() => setPayoutId('')}>New payout</button>
          </div>
        </div>
      </Layout>
    )
  }

  return (
    <Layout>
      <div style={{ maxWidth: 480, margin: '0 auto' }}>
        <h1 style={{ marginBottom: 24 }}>Send Payout</h1>
        <div style={card}>
          {error && <ErrorMsg msg={error} />}
          <form onSubmit={handleSubmit}>
            <label style={label}>Source currency</label>
            <select style={{ ...input, background: '#fff' }} value={currency} onChange={e => setCurrency(e.target.value)}>
              {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
            </select>

            <label style={label}>Amount</label>
            <input style={input} type="number" min="0.01" step="0.01" placeholder="0.00"
              value={amount} onChange={e => setAmount(e.target.value)} required />

            <hr style={{ margin: '16px 0', border: 'none', borderTop: '1px solid #f0f0f0' }} />
            <p style={{ fontSize: 13, color: '#888', marginBottom: 12 }}>Recipient bank details</p>

            <label style={label}>Account number</label>
            <input style={input} value={accountNumber} onChange={e => setAccountNumber(e.target.value)} required />

            <label style={label}>Bank code</label>
            <input style={input} placeholder="e.g. 058" value={bankCode} onChange={e => setBankCode(e.target.value)} required />

            <label style={label}>Account name</label>
            <input style={input} value={accountName} onChange={e => setAccountName(e.target.value)} required />

            <button style={btn()} type="submit" disabled={createPayout.isPending}>
              {createPayout.isPending ? 'Sending…' : 'Send payout'}
            </button>
          </form>
        </div>
      </div>
    </Layout>
  )
}
