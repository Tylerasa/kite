import { useState, useEffect, useRef } from 'react'

function formatInputAmount(raw: string): string {
  if (!raw) return ''
  const [integer, decimal] = raw.split('.')
  const formatted = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return decimal !== undefined ? `${formatted}.${decimal}` : formatted
}
import { AlertCircle, ArrowRight, ChevronDown, Info, RefreshCw } from 'lucide-react'
import { useCreateQuote, useExecuteConversion, type Quote } from '../api/hooks'
import BackButton from '../components/BackButton'
import { Layout, formatAmount, apiError } from '../components/Layout'
import { PinDialog } from '../components/PinDialog'
import { Icons } from '../../public/assets/svgs/icons'

const CURRENCIES = ['USD', 'GBP', 'EUR', 'NGN', 'KES']

const CURRENCY_NAMES: Record<string, string> = {
  USD: 'United States dollar',
  GBP: 'British pound',
  EUR: 'Euro',
  NGN: 'Nigerian naira',
  KES: 'Kenyan shilling',
}

const CURRENCY_ICONS: Record<string, React.FC<{ className?: string }>> = {
  USD: Icons.usd,
  GBP: Icons.gbp,
  EUR: Icons.eur,
  NGN: Icons.ngn,
  KES: Icons.kes,
}

type Step = 'input' | 'quoted' | 'done'

function CurrencyPill({
  value,
  onChange,
  open,
  onToggle,
  disabledCurrency,
}: {
  value: string
  onChange: (c: string) => void
  open: boolean
  onToggle: () => void
  disabledCurrency?: string
}) {
  const Icon = CURRENCY_ICONS[value]
  return (
    <div className="relative">
      <button
        type="button"
        onClick={onToggle}
        className="inline-flex items-center gap-2 rounded-full bg-[#eef0eb] py-2 px-3 text-[#11160f] transition-colors hover:bg-[#e7eae2]"
      >
        {Icon
          ? <Icon className="size-6 shrink-0" />
          : <span className="size-6 shrink-0 rounded-full bg-[#d7dbd3]" />
        }
        <span className="text-[15px] font-semibold">{value}</span>
        <ChevronDown size={16} className="text-[#233818]" />
      </button>

      {open && (
        <div className="absolute left-0 top-full z-20 mt-2 w-48 rounded-[20px] bg-white py-1.5 shadow-[0_8px_32px_rgba(0,0,0,0.12)]">
          {CURRENCIES.map(c => {
            const CIcon = CURRENCY_ICONS[c]
            const selected = c === value
            const disabled = c === disabledCurrency
            return (
              <button
                key={c}
                type="button"
                disabled={disabled}
                onClick={() => { onChange(c); onToggle() }}
                className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors ${disabled ? 'cursor-not-allowed opacity-35' : `hover:bg-[#f5f6f3] ${selected ? 'bg-[#f5f6f3]' : ''}`}`}
              >
                {CIcon
                  ? <CIcon className="size-5 shrink-0" />
                  : <span className="size-5 shrink-0 rounded-full bg-[#eef0eb]" />
                }
                <span>
                  <span className={`block text-[14px] text-[#11160f] ${selected ? 'font-semibold' : 'font-medium'}`}>{c}</span>
                  <span className="block text-[11px] text-[#6b7c65]">{CURRENCY_NAMES[c]}</span>
                </span>
                {selected && (
                  <span className="ml-auto size-1.5 rounded-full bg-[#9fe870]" />
                )}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

export default function Convert() {
  const [from, setFrom] = useState('USD')
  const [to, setTo] = useState('NGN')
  const [amount, setAmount] = useState('')
  const [step, setStep] = useState<Step>('input')
  const [quote, setQuote] = useState<Quote | null>(null)
  const [secondsLeft, setSecondsLeft] = useState(0)
  const [error, setError] = useState('')
  const [fromOpen, setFromOpen] = useState(false)
  const [toOpen, setToOpen] = useState(false)
  const [pinOpen, setPinOpen] = useState(false)
  const [pinError, setPinError] = useState('')
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

  function handleExecute() {
    if (!quote) return
    setError('')
    setPinError('')
    setPinOpen(true)
  }

  async function handlePinConfirm(pin: string) {
    if (!quote) return
    try {
      await execute.mutateAsync({ quoteId: quote.id, pin })
      clearInterval(timerRef.current!)
      setPinOpen(false)
      setStep('done')
    } catch (err) {
      const msg = apiError(err)
      if (msg.toLowerCase().includes('pin')) {
        setPinError('Incorrect PIN. Please try again.')
      } else {
        setPinOpen(false)
        if (msg.includes('expired')) { setStep('input'); setQuote(null) }
        setError(msg)
      }
    }
  }

  function reset() {
    setStep('input'); setQuote(null); setError(''); setAmount('')
    timerRef.current && clearInterval(timerRef.current)
  }

  const amountMinor = Math.round(parseFloat(amount) * 100)
  const canQuote = !isNaN(amountMinor) && amountMinor > 0 && !createQuote.isPending
  const urgency = secondsLeft < 10

  return (
    <Layout>
      <div className="w-full max-w-[480px]">
        <BackButton href="/dashboard" />

        {/* ── Step 1: Input ───────────────────────────────────────── */}
        {step === 'input' && (
          <form onSubmit={handleGetQuote}>
            <div className="rounded-[28px] bg-white px-5 py-6">
              <div className="flex flex-col gap-6">

                {/* From row */}
                <div>
                  <p className="text-[13px] text-[#171b18]">You convert <span className="font-semibold">from</span></p>
                  <div className="mt-3 flex items-center justify-between gap-4">
                    <CurrencyPill value={from} onChange={(c) => { if (c === to) setTo(from); setFrom(c) }} open={fromOpen} onToggle={() => { setFromOpen(o => !o); setToOpen(false) }} disabledCurrency={to} />
                    <input
                      inputMode="decimal"
                      autoComplete="off"
                      placeholder="0.00"
                      value={formatInputAmount(amount)}
                      onChange={e => setAmount(e.target.value.replace(/,/g, '').replace(/[^0-9.]/g, ''))}
                      className="min-w-0 flex-1 bg-transparent text-right text-[40px] font-black leading-none tracking-normal text-[#7b7b7b] outline-none transition-all duration-150 placeholder:text-[#7b7b7b] focus:text-[56px] focus:text-[#11160f]"
                    />
                  </div>
                </div>

                {/* Arrow divider */}
                <div className="flex items-center gap-3">
                  <div className="h-px flex-1 bg-[#eef0eb]" />
                  <span className="flex size-8 items-center justify-center rounded-full bg-[#eef0eb]">
                    <ArrowRight size={15} className="text-[#4f5650]" />
                  </span>
                  <div className="h-px flex-1 bg-[#eef0eb]" />
                </div>

                {/* To row */}
                <div>
                  <p className="text-[13px] text-[#171b18]">You receive <span className="font-semibold">in</span></p>
                  <div className="mt-3">
                    <CurrencyPill value={to} onChange={(c) => { if (c === from) setFrom(to); setTo(c) }} open={toOpen} onToggle={() => { setToOpen(o => !o); setFromOpen(false) }} disabledCurrency={from} />
                  </div>
                </div>

                {/* Info + submit */}
                <div className="flex flex-col gap-2">
                  <div className={`flex items-center justify-center gap-2 px-1 py-1 text-center text-[13px] ${error ? 'text-[#c0392b]' : 'text-[#6b7c65]'}`}>
                    {error ? <AlertCircle size={15} className="shrink-0" /> : <Info size={15} className="shrink-0" />}
                    <p>{error || (canQuote ? `Converting ${from} → ${to}` : 'Enter an amount to get a live rate')}</p>
                  </div>
                  <button
                    type="submit"
                    disabled={!canQuote}
                    className={`flex w-full items-center justify-center gap-2 rounded-full py-3 text-[15px] font-semibold transition-colors ${
                      canQuote ? 'bg-[#9fe870] text-[#173300] hover:bg-[#8fdd5f]' : 'bg-[#dfe2db] text-[#969b96]'
                    }`}
                  >
                    <RefreshCw size={15} />
                    {createQuote.isPending ? 'Getting rate…' : 'Get live rate'}
                  </button>
                </div>

              </div>
            </div>
          </form>
        )}

        {/* ── Step 2: Quote ───────────────────────────────────────── */}
        {step === 'quoted' && quote && (
          <div className="rounded-[28px] bg-white px-5 py-6">
            <div className="flex flex-col gap-5">

              {/* Hero amounts */}
              <div className="text-center">
                <p className="text-[13px] text-[#6b7c65]">You convert</p>
                <div className="mt-1 flex items-center justify-center gap-3">
                  <span className="text-[32px] font-black text-[#11160f]">{formatAmount(quote.amount_in, quote.from_currency)}</span>
                  <ArrowRight size={20} className="text-[#6b7c65]" />
                  <span className="text-[32px] font-black text-[#173300]">{formatAmount(quote.amount_out, quote.to_currency)}</span>
                </div>
              </div>

              {/* Timer */}
              <div className="flex items-center justify-between px-1">
                <span className="text-[13px] text-[#6b7c65]">Rate locked for</span>
                <span className={`text-[18px] font-black tabular-nums ${urgency ? 'text-[#c0392b]' : 'text-[#173300]'}`}>
                  {secondsLeft}s
                </span>
              </div>

              {/* Detail rows */}
              <div>
                {[
                  ['Exchange rate', `1 ${quote.from_currency} = ${parseFloat(quote.quoted_rate).toFixed(4)} ${quote.to_currency}`],
                  ['Fee', formatAmount(quote.fee, quote.to_currency)],
                ].map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between px-1 py-2.5">
                    <span className="text-[13px] text-[#6b7c65]">{k}</span>
                    <span className="text-[13px] font-semibold text-[#11160f]">{v}</span>
                  </div>
                ))}
              </div>

              {/* Error inline */}
              {error && (
                <div className="flex items-center gap-2 px-1 text-[13px] text-[#c0392b]">
                  <AlertCircle size={15} className="shrink-0" />
                  {error}
                </div>
              )}

              {/* Actions */}
              <div className="flex flex-col gap-2">
                <button
                  type="button"
                  onClick={handleExecute}
                  disabled={execute.isPending}
                  className="w-full rounded-full bg-[#9fe870] py-3 text-[15px] font-semibold text-[#173300] transition-colors hover:bg-[#8fdd5f] disabled:opacity-60"
                >
                  {execute.isPending ? 'Converting…' : 'Confirm conversion'}
                </button>
                <button
                  type="button"
                  onClick={reset}
                  className="w-full rounded-full bg-[#f2f4ef] py-3 text-[15px] font-semibold text-[#4f5650] transition-colors hover:bg-[#eaede7]"
                >
                  Cancel
                </button>
              </div>

            </div>
          </div>
        )}

        {/* ── Step 3: Done ────────────────────────────────────────── */}
        {step === 'done' && quote && (
          <div className="rounded-[28px] bg-white px-5 py-12 text-center">
            <div className="mx-auto flex max-w-[300px] flex-col items-center gap-6">

              <div className="relative flex items-center justify-center">
                <span
                  className="absolute size-[88px] rounded-full bg-[#a6ea6c]"
                  style={{ animation: 'success-ring 0.7s 0.35s cubic-bezier(0.4,0,0.6,1) forwards', opacity: 0 }}
                />
                <svg
                  viewBox="0 0 80 80"
                  className="size-[88px]"
                  style={{ animation: 'success-circle 0.5s cubic-bezier(0.34,1.56,0.64,1) forwards' }}
                >
                  <circle cx="40" cy="40" r="40" fill="#a6ea6c" />
                  <polyline
                    points="22,41 34,54 58,27"
                    fill="none" stroke="#173300" strokeWidth="5.5"
                    strokeLinecap="round" strokeLinejoin="round"
                    strokeDasharray="58" strokeDashoffset="58"
                    style={{ animation: 'success-check 0.4s 0.35s ease-out forwards' }}
                  />
                </svg>
              </div>

              <div className="flex flex-col items-center gap-2" style={{ animation: 'success-fade-up 0.4s 0.2s ease-out both' }}>
                <div className="flex items-center gap-2">
                  <span className="text-[28px] font-black text-[#11160f]">{formatAmount(quote.amount_in, quote.from_currency)}</span>
                  <ArrowRight size={18} className="text-[#6b7c65]" />
                  <span className="text-[28px] font-black text-[#173300]">{formatAmount(quote.amount_out, quote.to_currency)}</span>
                </div>
                <p className="text-[14px] text-[#6b7c65]">conversion complete</p>
              </div>

              <div className="mt-1 flex w-full flex-col gap-2" style={{ animation: 'success-fade-up 0.4s 0.35s ease-out both' }}>
                <button
                  type="button"
                  onClick={reset}
                  className="w-full rounded-full bg-[#9fe870] py-3 text-[15px] font-semibold text-[#173300] transition-colors hover:bg-[#8fdd5f]"
                >
                  Convert again
                </button>
                <a
                  href="/dashboard"
                  className="w-full rounded-full bg-[#f2f4ef] py-3 text-[15px] font-semibold text-[#4f5650] transition-colors hover:bg-[#eaede7]"
                >
                  Back to home
                </a>
              </div>

            </div>
          </div>
        )}
      <PinDialog
        open={pinOpen}
        onConfirm={handlePinConfirm}
        onCancel={() => { setPinOpen(false); setPinError('') }}
        error={pinError}
        loading={execute.isPending}
      />
      </div>
    </Layout>
  )
}
