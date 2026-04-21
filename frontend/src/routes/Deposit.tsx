import { useState } from 'react'
import { useDeposit } from '../api/hooks'
import { Layout, card, btn, input, label, ErrorMsg, SuccessMsg, formatAmount } from '../components/Layout'
import { apiError } from '../components/Layout'

const CURRENCIES = ['USD', 'GBP', 'EUR', 'NGN', 'KES']

export default function Deposit() {
  const [currency, setCurrency] = useState('USD')
  const [amount, setAmount] = useState('')
  const [success, setSuccess] = useState('')
  const deposit = useDeposit()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSuccess('')
    const amountMinor = Math.round(parseFloat(amount) * 100)
    if (isNaN(amountMinor) || amountMinor <= 0) return

    // Generate a new idempotency key per submission attempt
    const idempotencyKey = crypto.randomUUID()

    try {
      const result = await deposit.mutateAsync({ currency, amount: amountMinor, idempotencyKey })
      setSuccess(`Deposited ${formatAmount(result.amount, result.currency)} successfully!`)
      setAmount('')
    } catch (_) {}
  }

  return (
    <Layout>
      <div style={{ maxWidth: 480, margin: '0 auto' }}>
        <h1 style={{ marginBottom: 24 }}>Deposit Funds</h1>
        <div style={card}>
          {deposit.isError && <ErrorMsg msg={apiError(deposit.error)} />}
          {success && <SuccessMsg msg={success} />}
          <form onSubmit={handleSubmit}>
            <label style={label}>Currency</label>
            <select style={{ ...input, background: '#fff' }} value={currency} onChange={e => setCurrency(e.target.value)}>
              {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
            </select>

            <label style={label}>Amount</label>
            <input
              style={input}
              type="number"
              min="0.01"
              step="0.01"
              placeholder="0.00"
              value={amount}
              onChange={e => setAmount(e.target.value)}
              required
            />

            <p style={{ fontSize: 12, color: '#888', marginBottom: 16 }}>
              This simulates an inbound bank deposit. In production, this would be triggered by a webhook.
            </p>

            <button style={btn()} type="submit" disabled={deposit.isPending}>
              {deposit.isPending ? 'Processing…' : 'Deposit'}
            </button>
          </form>
        </div>
      </div>
    </Layout>
  )
}
