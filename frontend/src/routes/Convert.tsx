import { useState, useEffect, useRef } from 'react'
import { useCreateQuote, useExecuteConversion, type Quote } from '../api/hooks'
import { Layout, card, btn, input, label, ErrorMsg, SuccessMsg, formatAmount } from '../components/Layout'
import { apiError } from '../components/Layout'

const CURRENCIES = ['USD', 'GBP', 'EUR', 'NGN', 'KES']
type Step = 'input' | 'quoted' | 'done'

export default function Convert() {
  const [from, setFrom] = useState('USD')
  const [to, setTo] = useState('NGN')
  const [amount, setAmount] = useState('')
  const [step, setStep] = useState<Step>('input')
  const [quote, setQuote] = useState<Quote | null>(null)
  const [secondsLeft, setSecondsLeft] = useState(0)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const createQuote = useCreateQuote()
  const execute = useExecuteConversion()

  function startCountdown(secs: number) {
    setSecondsLeft(secs)
    timerRef.current && clearInterval(timerRef.current)
    timerRef.current = setInterval(() => {
      setSecondsLeft(prev => {
        if (prev <= 1) {
          clearInterval(timerRef.current!)
          setStep('input')
          setQuote(null)
          setError('Quote expired. Please request a new one.')
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }

  useEffect(() => () => { timerRef.current && clearInterval(timerRef.current) }, [])

  async function handleGetQuote(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    const amountMinor = Math.round(parseFloat(amount) * 100)
    if (isNaN(amountMinor) || amountMinor <= 0) return

    try {
      const q = await createQuote.mutateAsync({ from_currency: from, to_currency: to, amount_in: amountMinor })
      setQuote(q)
      setStep('quoted')
      startCountdown(q.seconds_left)
    } catch (err) {
      setError(apiError(err))
    }
  }

  async function handleExecute() {
    if (!quote) return
    setError('')
    try {
      await execute.mutateAsync(quote.id)
      clearInterval(timerRef.current!)
      setStep('done')
      setSuccess(`Converted ${formatAmount(quote.amount_in, quote.from_currency)} → ${formatAmount(quote.amount_out, quote.to_currency)}`)
    } catch (err) {
      const msg = apiError(err)
      if (msg.includes('expired')) {
        setStep('input')
        setQuote(null)
      }
      setError(msg)
    }
  }

  function reset() {
    setStep('input')
    setQuote(null)
    setError('')
    setSuccess('')
    setAmount('')
    timerRef.current && clearInterval(timerRef.current)
  }

  return (
    <Layout>
      <div style={{ maxWidth: 480, margin: '0 auto' }}>
        <h1 style={{ marginBottom: 24 }}>Convert Currency</h1>

        {step === 'input' && (
          <div style={card}>
            {error && <ErrorMsg msg={error} />}
            <form onSubmit={handleGetQuote}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div>
                  <label style={label}>From</label>
                  <select style={{ ...input, background: '#fff' }} value={from} onChange={e => setFrom(e.target.value)}>
                    {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                <div>
                  <label style={label}>To</label>
                  <select style={{ ...input, background: '#fff' }} value={to} onChange={e => setTo(e.target.value)}>
                    {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
              </div>
              <label style={label}>Amount ({from})</label>
              <input style={input} type="number" min="0.01" step="0.01" placeholder="0.00"
                value={amount} onChange={e => setAmount(e.target.value)} required />
              <button style={btn()} type="submit" disabled={createQuote.isPending}>
                {createQuote.isPending ? 'Getting rate…' : 'Get Quote'}
              </button>
            </form>
          </div>
        )}

        {step === 'quoted' && quote && (
          <div style={card}>
            <div style={{
              background: secondsLeft < 10 ? '#fff3e0' : '#f0f7ff',
              borderRadius: 8, padding: '12px 16px', marginBottom: 16,
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            }}>
              <span style={{ fontSize: 14, color: '#555' }}>Quote expires in</span>
              <span style={{ fontWeight: 700, fontSize: 18, color: secondsLeft < 10 ? '#e65' : '#1a1a2e' }}>
                {secondsLeft}s
              </span>
            </div>

            <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: 16 }}>
              {[
                ['You send', formatAmount(quote.amount_in, quote.from_currency)],
                ['You receive', formatAmount(quote.amount_out, quote.to_currency)],
                ['Exchange rate', `1 ${quote.from_currency} = ${parseFloat(quote.quoted_rate).toFixed(4)} ${quote.to_currency}`],
                ['Fee', formatAmount(quote.fee, quote.to_currency)],
              ].map(([k, v]) => (
                <tr key={k} style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <td style={{ padding: '10px 0', color: '#666', fontSize: 14 }}>{k}</td>
                  <td style={{ padding: '10px 0', fontWeight: 600, textAlign: 'right' }}>{v}</td>
                </tr>
              ))}
            </table>

            {error && <ErrorMsg msg={error} />}
            <div style={{ display: 'flex', gap: 10 }}>
              <button style={btn()} onClick={handleExecute} disabled={execute.isPending}>
                {execute.isPending ? 'Converting…' : 'Confirm conversion'}
              </button>
              <button style={btn('secondary')} onClick={reset}>Cancel</button>
            </div>
          </div>
        )}

        {step === 'done' && (
          <div style={card}>
            <SuccessMsg msg={success} />
            <div style={{ display: 'flex', gap: 10 }}>
              <button style={btn()} onClick={reset}>Convert again</button>
              <a href="/dashboard" style={{ ...btn('secondary'), display: 'inline-block', textAlign: 'center' }}>Dashboard</a>
            </div>
          </div>
        )}
      </div>
    </Layout>
  )
}
