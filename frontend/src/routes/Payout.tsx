import { useEffect, useMemo, useRef, useState } from 'react'

function formatInputAmount(raw: string): string {
  if (!raw) return ''
  const [integer, decimal] = raw.split('.')
  const formatted = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return decimal !== undefined ? `${formatted}.${decimal}` : formatted
}
import { AlertCircle, CheckCircle2, ChevronDown, Info, Loader2, Search, Send, X } from 'lucide-react'
import { useAccountInquiry, useCreatePayout, useInstitutions, usePayout, type Institution } from '../api/hooks'
import BackButton from '../components/BackButton'
import { Layout, ErrorBanner, formatAmount, apiError } from '../components/Layout'
import { PinDialog } from '../components/PinDialog'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../components/ui/dialog'
import { Icons } from '../../public/assets/svgs/icons'

const CURRENCIES = ['NGN', 'KES', 'USD', 'GBP', 'EUR']

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

const STATUS_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  pending:    { label: 'Pending',      color: '#b45309', bg: '#fffbeb' },
  processing: { label: 'Processing',  color: '#0369a1', bg: '#eff6ff' },
  successful: { label: 'Successful',  color: '#15803d', bg: '#f0fdf4' },
  failed:     { label: 'Failed',      color: '#be123c', bg: '#fff0f2' },
  review:     { label: 'Under review',color: '#7c3aed', bg: '#faf5ff' },
}

function Field({
  label,
  id,
  value,
  onChange,
  placeholder,
  type = 'text',
}: {
  label: string
  id: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  type?: string
}) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-[12px] font-medium text-[#6b7c65]">
        {label}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        placeholder={placeholder}
        onChange={e => onChange(e.target.value)}
        required
        className="border-b border-[#d4d9d0] bg-transparent pb-2 text-[15px] text-[#11160f] outline-none transition-colors placeholder:text-[#b0b8ab] focus:border-[#11160f]"
      />
    </div>
  )
}

export default function Payout() {
  const [currency, setCurrency] = useState('NGN')
  const [amount, setAmount]     = useState('')
  const [accountNumber, setAccountNumber] = useState('')
  const [selectedBank, setSelectedBank] = useState<Institution | null>(null)
  const [bankPickerOpen, setBankPickerOpen] = useState(false)
  const [bankSearch, setBankSearch] = useState('')
  const [bankCode, setBankCode] = useState('')
  const [payoutId, setPayoutId] = useState('')
  const [error, setError]       = useState('')
  const [currencyOpen, setCurrencyOpen] = useState(false)
  const [pinOpen, setPinOpen] = useState(false)
  const [pinError, setPinError] = useState('')

  type InquiryState =
    | { status: 'idle' }
    | { status: 'loading' }
    | { status: 'resolved'; name: string; bankName: string }
    | { status: 'error'; message: string }
  const [inquiry, setInquiry] = useState<InquiryState>({ status: 'idle' })

  const accountInquiry = useAccountInquiry()
  const createPayout   = useCreatePayout()
  const payout         = usePayout(payoutId, !!payoutId)
  const institutions = useInstitutions(currency)
  const timerRef       = useRef<ReturnType<typeof setTimeout>>(undefined)
  const institutionList: Institution[] = Array.isArray(institutions.data)
    ? institutions.data.filter((item): item is Institution => !!item && typeof item === 'object')
    : []

  const filteredInstitutions: Institution[] = useMemo(() => {
    const q = bankSearch.toLowerCase().trim()
    if (!q) return institutionList

    return institutionList.filter((institution) => {
      const name = String(institution.name ?? '').toLowerCase()
      const bankCode = String(institution.bank_code ?? '').toLowerCase()
      return name.includes(q) || bankCode.includes(q)
    })
  }, [institutionList, bankSearch])

  const hasInstitutions = currency === 'NGN' || currency === 'KES'
  const effectiveBankCode = selectedBank?.bank_code ?? bankCode

  // Reset bank selection when currency changes
  useEffect(() => {
    setSelectedBank(null)
    setBankSearch('')
    setBankCode('')
    setAccountNumber('')
    setInquiry({ status: 'idle' })
  }, [currency])

  // Auto-lookup when account number and bank are both present
  useEffect(() => {
    setInquiry({ status: 'idle' })
    clearTimeout(timerRef.current)
    if (!accountNumber || !effectiveBankCode) return

    setInquiry({ status: 'loading' })
    timerRef.current = setTimeout(async () => {
      try {
        const res = await accountInquiry.mutateAsync({ currency, bank_code: effectiveBankCode, account_number: accountNumber })
        setInquiry({ status: 'resolved', name: res.account_name, bankName: res.bank_name })
      } catch (err) {
        setInquiry({ status: 'error', message: apiError(err) })
      }
    }, 600)

    return () => clearTimeout(timerRef.current)
  }, [accountNumber, effectiveBankCode, currency])

  const SelectedIcon = CURRENCY_ICONS[currency]
  const amountMinor  = Math.round(parseFloat(amount) * 100)
  const resolvedName = inquiry.status === 'resolved' ? inquiry.name : ''
  const canSend      = !isNaN(amountMinor) && amountMinor > 0 && inquiry.status === 'resolved' && !createPayout.isPending

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!canSend || inquiry.status !== 'resolved') return
    setPinError('')
    setPinOpen(true)
  }

  async function handlePinConfirm(pin: string) {
    if (inquiry.status !== 'resolved' || !effectiveBankCode) return
    try {
      const result = await createPayout.mutateAsync({
        source_currency: currency,
        amount: amountMinor,
        recipient_account_number: accountNumber,
        recipient_bank_code: effectiveBankCode,
        recipient_account_name: inquiry.name,
        pin,
      })
      setPinOpen(false)
      setPayoutId(result.id)
    } catch (err) {
      const msg = apiError(err)
      if (msg.toLowerCase().includes('pin')) {
        setPinError('Incorrect PIN. Please try again.')
      } else {
        setPinOpen(false)
        setError(msg)
      }
    }
  }

  // ── Status view ──────────────────────────────────────────────────────────────
  if (payoutId && payout.data) {
    const p   = payout.data
    const sc  = STATUS_CONFIG[p.status] ?? STATUS_CONFIG.pending
    const done = p.status === 'successful'
    const failed = p.status === 'failed'
    const inProgress = p.status === 'pending' || p.status === 'processing'

    return (
      <Layout>
        <div className="w-full max-w-[480px]">
          <div className="rounded-[28px] bg-white px-5 py-12 text-center">
            <div className="mx-auto flex max-w-[300px] flex-col items-center gap-6">

              {/* Icon */}
              {done ? (
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
                      fill="none"
                      stroke="#173300"
                      strokeWidth="5.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeDasharray="58"
                      strokeDashoffset="58"
                      style={{ animation: 'success-check 0.4s 0.35s ease-out forwards' }}
                    />
                  </svg>
                </div>
              ) : failed ? (
                <div className="flex size-[88px] items-center justify-center rounded-full bg-[#fdecea]">
                  <AlertCircle size={36} className="text-[#c0392b]" />
                </div>
              ) : (
                <div className="flex size-[88px] items-center justify-center rounded-full bg-[#eef0eb]">
                  <Loader2 size={36} className="animate-spin text-[#6b7c65]" />
                </div>
              )}

              {/* Amount + recipient headline */}
              <div
                className="flex flex-col items-center gap-2"
                style={{ animation: 'success-fade-up 0.4s 0.2s ease-out both' }}
              >
                <p className="text-[44px] font-black leading-none text-[#11160f]">
                  {formatAmount(p.amount, p.source_currency)}
                </p>
                <p className="text-[14px] text-[#6b7c65]">
                  {inProgress ? 'being sent to' : done ? 'sent to' : 'failed to'}{' '}
                  <span className="font-semibold text-[#11160f]">{p.recipient_account_name}</span>
                </p>
                <span
                  className="inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-[12px] font-semibold"
                  style={{ background: sc.bg, color: sc.color }}
                >
                  <span className="size-1.5 rounded-full" style={{ background: sc.color }} />
                  {sc.label}
                </span>
              </div>

              {/* Compliance notice */}
              {p.compliance_flagged && (
                <div
                  className="w-full rounded-[16px] bg-[#fffbeb] px-4 py-3 text-left text-[13px] text-[#b45309]"
                  style={{ animation: 'success-fade-up 0.4s 0.25s ease-out both' }}
                >
                  Flagged for compliance review — will be processed after manual review.
                </div>
              )}

              {/* Bank details */}
              <div
                className="w-full"
                style={{ animation: 'success-fade-up 0.4s 0.3s ease-out both' }}
              >
                {[
                  ['Account', p.recipient_account_number],
                  ['Bank code', p.recipient_bank_code],
                ].map(([k, v], i, arr) => (
                  <div
                    key={k}
                    className={`flex items-center justify-between py-2.5 ${i < arr.length - 1 ? 'border-b border-[#f0f1ee]' : ''}`}
                  >
                    <span className="text-[13px] text-[#6b7c65]">{k}</span>
                    <span className="text-[13px] font-semibold text-[#11160f]">{v}</span>
                  </div>
                ))}
              </div>

              {p.failure_reason && (
                <div className="w-full rounded-[16px] bg-[#fdecea] px-4 py-3 text-left text-[13px] text-[#c0392b]">
                  {p.failure_reason}{p.reversed_at ? ' — your balance has been restored.' : ''}
                </div>
              )}

              {inProgress && (
                <p className="text-[13px] text-[#6b7c65]">Checking for updates automatically…</p>
              )}

              {/* Actions */}
              <div
                className="mt-1 flex w-full flex-col gap-2"
                style={{ animation: 'success-fade-up 0.4s 0.4s ease-out both' }}
              >
                <button
                  type="button"
                  onClick={() => setPayoutId('')}
                  className="w-full rounded-full bg-[#9fe870] py-3 text-[15px] font-semibold text-[#173300] transition-colors hover:bg-[#8fdd5f]"
                >
                  New payout
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
        </div>
      </Layout>
    )
  }

  // ── Form view ────────────────────────────────────────────────────────────────
  return (
    <Layout>
      <div className="w-full max-w-[480px]">
        <BackButton href="/dashboard" />

        <form onSubmit={handleSubmit}>
          <div className="rounded-[28px] bg-white px-5 py-6">
            <div className="flex flex-col gap-6">

              {/* Amount row */}
              <div>
                <p className="text-[13px] text-[#171b18]">
                  You send from <span className="font-semibold">Main account</span>
                </p>

                <div className="mt-3 flex items-center justify-between gap-4">
                  {/* Currency selector */}
                  <div className="relative">
                    <button
                      type="button"
                      onClick={() => setCurrencyOpen(o => !o)}
                      className="inline-flex items-center gap-2 rounded-full bg-[#eef0eb] py-2 px-3 text-[#11160f] transition-colors hover:bg-[#e7eae2]"
                    >
                      {SelectedIcon
                        ? <SelectedIcon className="size-6 shrink-0" />
                        : <span className="size-6 shrink-0 rounded-full bg-[#d7dbd3]" />
                      }
                      <span className="text-[15px] font-semibold">{currency}</span>
                      <ChevronDown size={16} className="text-[#233818]" />
                    </button>

                    {currencyOpen && (
                      <div className="absolute left-0 top-full z-20 mt-2 w-48 rounded-[20px] bg-white py-1.5 shadow-[0_8px_32px_rgba(0,0,0,0.12)]">
                        {CURRENCIES.map(c => {
                          const Icon = CURRENCY_ICONS[c]
                          const selected = c === currency
                          return (
                            <button
                              key={c}
                              type="button"
                              onClick={() => { setCurrency(c); setCurrencyOpen(false) }}
                              className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-[#f5f6f3] ${selected ? 'bg-[#f5f6f3]' : ''}`}
                            >
                              {Icon
                                ? <Icon className="size-5 shrink-0" />
                                : <span className="size-5 shrink-0 rounded-full bg-[#eef0eb]" />
                              }
                              <span>
                                <span className={`block text-[14px] text-[#11160f] ${selected ? 'font-semibold' : 'font-medium'}`}>{c}</span>
                                <span className="block text-[11px] text-[#6b7c65]">{CURRENCY_NAMES[c] ?? c}</span>
                              </span>
                              {selected && <span className="ml-auto size-1.5 rounded-full bg-[#9fe870]" />}
                            </button>
                          )
                        })}
                      </div>
                    )}
                  </div>

                  {/* Amount input */}
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

              {/* Recipient details */}
              <div className="rounded-[20px] bg-[#f7f8f6] px-5 py-4">
                <p className="mb-4 text-[11px] font-semibold uppercase tracking-widest text-[#6b7c65]">
                  Recipient bank details
                </p>
                <div className="flex flex-col gap-5">
                  <Field
                    label="Account number"
                    id="accountNumber"
                    value={accountNumber}
                    onChange={v => setAccountNumber(v.replace(/\D/g, ''))}
                    placeholder="Account number"
                  />

                  {/* Bank picker / free-text */}
                  {hasInstitutions ? (
                    <div className="flex flex-col gap-1">
                      <p className="text-[12px] font-medium text-[#6b7c65]">Bank</p>
                      <button
                        type="button"
                        onClick={() => setBankPickerOpen(true)}
                        className="flex items-center justify-between border-b border-[#d4d9d0] pb-2 text-left outline-none transition-colors hover:border-[#11160f]"
                      >
                        <span className={`text-[15px] ${selectedBank ? 'text-[#11160f]' : 'text-[#b0b8ab]'}`}>
                          {selectedBank ? selectedBank.name : 'Select bank'}
                        </span>
                        <ChevronDown size={16} className="shrink-0 text-[#6b7c65]" />
                      </button>
                    </div>
                  ) : (
                    <Field
                      label="Bank / routing code"
                      id="bankCode"
                      value={bankCode}
                      onChange={setBankCode}
                      placeholder="e.g. SWIFT, routing number"
                    />
                  )}

                  {/* Resolved account name */}
                  {inquiry.status !== 'idle' && (
                    <div className="flex flex-col gap-1">
                      <p className="text-[12px] font-medium text-[#6b7c65]">Account name</p>
                      {inquiry.status === 'loading' && (
                        <div className="flex items-center gap-2 py-1 text-[14px] text-[#6b7c65]">
                          <Loader2 size={15} className="animate-spin" />
                          Resolving…
                        </div>
                      )}
                      {inquiry.status === 'resolved' && (
                        <div className="flex flex-col gap-0.5">
                          <div className="flex items-center gap-2">
                            <CheckCircle2 size={16} className="shrink-0 text-[#15803d]" />
                            <span className="text-[15px] font-semibold text-[#11160f]">{inquiry.name}</span>
                          </div>
                          <p className="pl-6 text-[12px] text-[#6b7c65]">{inquiry.bankName}</p>
                        </div>
                      )}
                      {inquiry.status === 'error' && (
                        <div className="flex items-center gap-2 text-[13px] text-[#c0392b]">
                          <AlertCircle size={15} className="shrink-0" />
                          {inquiry.message}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>

              {/* Info + submit */}
              <div className={`rounded-[20px] p-3 transition-colors ${error ? 'bg-[#fdecea]' : 'bg-[#eef0eb]'}`}>
                <div className={`flex items-center justify-center gap-2 px-2 py-1 text-center text-[13px] ${error ? 'text-[#c0392b]' : 'text-[#171b18]'}`}>
                  {error
                    ? <AlertCircle size={15} className="shrink-0" />
                    : <Info size={15} className="shrink-0" />
                  }
                  <p>
                    {error
                      ? error
                      : inquiry.status === 'loading'
                        ? 'Verifying account…'
                        : inquiry.status === 'error'
                          ? 'Could not verify account'
                          : canSend
                            ? `Sending to ${resolvedName}`
                            : 'Enter account number and select a bank'}
                  </p>
                </div>

                <button
                  type="submit"
                  disabled={!canSend}
                  className={`mt-2 flex w-full items-center justify-center gap-2 rounded-full py-3 text-[15px] font-semibold transition-colors ${
                    canSend
                      ? 'bg-[#9fe870] text-[#173300] hover:bg-[#8fdd5f]'
                      : error
                        ? 'bg-[#f5c6c2] text-[#c0392b]'
                        : 'bg-[#dfe2db] text-[#969b96]'
                  }`}
                >
                  <Send size={15} />
                  {createPayout.isPending ? 'Sending…' : 'Send payout'}
                </button>
              </div>

            </div>
          </div>
        </form>

        {/* Bank picker dialog */}
        <Dialog open={bankPickerOpen} onOpenChange={open => { setBankPickerOpen(open); if (!open) setBankSearch('') }}>
          <DialogContent className="max-w-[480px] rounded-[28px] px-6 py-8 shadow-[0_24px_80px_rgba(0,0,0,0.15)]">
            <button
              type="button"
              onClick={() => { setBankPickerOpen(false); setBankSearch('') }}
              className="absolute right-5 top-5 flex size-10 items-center justify-center rounded-full bg-[#eef0eb] text-[#233818] transition-colors hover:bg-[#e6e9e2]"
              aria-label="Close bank picker"
            >
              <X size={18} />
            </button>

            <DialogHeader>
              <DialogTitle className="text-[22px] font-semibold tracking-normal text-[#11160f]">
                Select bank
              </DialogTitle>
            </DialogHeader>

            {/* Search */}
            <div className="mt-4 flex items-center gap-2 rounded-[14px] border border-[#d9ddd6] px-3 py-2.5">
              <Search size={15} className="shrink-0 text-[#9aa097]" />
              <input
                autoFocus
                type="text"
                placeholder="Search by name or code…"
                value={bankSearch}
                onChange={e => setBankSearch(e.target.value)}
                className="min-w-0 flex-1 bg-transparent text-[14px] text-[#11160f] outline-none placeholder:text-[#b0b8ab]"
              />
              {bankSearch && (
                <button type="button" onClick={() => setBankSearch('')} className="shrink-0 text-[#9aa097]">
                  <X size={14} />
                </button>
              )}
            </div>

            {/* List */}
            <div className="mt-3 max-h-[400px] overflow-y-auto rounded-[20px] border border-[#d9ddd6]">
              {institutions.isLoading ? (
                <div className="flex items-center justify-center py-8 text-[14px] text-[#6b7c65]">
                  <Loader2 size={16} className="mr-2 animate-spin" />
                  Loading banks…
                </div>
              ) : filteredInstitutions.length === 0 ? (
                <p className="py-8 text-center text-[14px] text-[#9aa097]">No banks found</p>
              ) : (
                filteredInstitutions.map(inst => {
                  const selected = selectedBank?.bank_code === inst.bank_code
                  return (
                    <button
                      key={inst.bank_code}
                      type="button"
                      onClick={() => { setSelectedBank(inst); setBankPickerOpen(false); setBankSearch('') }}
                      className={`flex w-full items-center justify-between px-4 py-3 text-left transition-colors hover:bg-[#f5f6f3] ${selected ? 'bg-[#f5f6f3]' : ''}`}
                    >
                      <span className="min-w-0">
                        <span className={`block truncate text-[14px] text-[#11160f] ${selected ? 'font-semibold' : 'font-medium'}`}>
                          {inst.name}
                        </span>
                        <span className="block text-[11px] text-[#6b7c65]">{inst.bank_code}</span>
                      </span>
                      {selected && <span className="ml-3 size-2 shrink-0 rounded-full bg-[#9fe870]" />}
                    </button>
                  )
                })
              )}
            </div>
          </DialogContent>
        </Dialog>

        <PinDialog
          open={pinOpen}
          onConfirm={handlePinConfirm}
          onCancel={() => { setPinOpen(false); setPinError('') }}
          error={pinError}
          loading={createPayout.isPending}
        />
      </div>
    </Layout>
  )
}
