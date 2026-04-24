import { useMemo, useState } from 'react'
import { AlertCircle, ChevronDown, ChevronRight, Info, Plus, X } from 'lucide-react'
import { useBalances, useDeposit } from '../api/hooks'
import BackButton from '../components/BackButton'
import { Layout, apiError, formatAmount } from '../components/Layout'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../components/ui/dialog'
import { PinDialog } from '../components/PinDialog'
import { Icons } from '../../public/assets/svgs/icons'

const CURRENCY_NAMES: Record<string, string> = {
  USD: 'United States dollar',
  GBP: 'British pound',
  EUR: 'Euro',
  NGN: 'Nigerian naira',
  KES: 'Kenyan shilling',
}

const CURRENCY_ICONS = {
  USD: Icons.usd,
  GBP: Icons.gbp,
  EUR: Icons.eur,
  NGN: Icons.ngn,
  KES: Icons.kes,
}

const FALLBACK_CURRENCIES = ['USD', 'GBP', 'EUR', 'NGN', 'KES']

function formatInputAmount(raw: string): string {
  if (!raw) return ''
  const [integer, decimal] = raw.split('.')
  const formatted = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return decimal !== undefined ? `${formatted}.${decimal}` : formatted
}

function parseMajorAmount(value: string) {
  const normalized = value.replace(/,/g, '.').replace(/[^\d.]/g, '')
  const number = Number.parseFloat(normalized)
  if (Number.isNaN(number) || number <= 0) return 0
  const minor = Math.round(number * 100)
  if (minor > Number.MAX_SAFE_INTEGER) return -1 // signal: too large
  return minor
}


function currencyIcon(code: string) {
  return CURRENCY_ICONS[code as keyof typeof CURRENCY_ICONS]
}

export default function Deposit() {
  const [currency, setCurrency] = useState('USD')
  const [amount, setAmount] = useState('')
  const [successResult, setSuccessResult] = useState<{ amount: number; currency: string } | null>(null)
  const [chooserOpen, setChooserOpen] = useState(false)
  const [pinOpen, setPinOpen] = useState(false)
  const [pinError, setPinError] = useState('')
  const deposit = useDeposit()
  const balances = useBalances()

  const availableCurrencies = useMemo(() => {
    const fromBalances = (balances.data ?? []).map((balance) => balance.currency)
    return [...new Set([...fromBalances, ...FALLBACK_CURRENCIES])]
  }, [balances.data])

  const SelectedCurrencyIcon = currencyIcon(currency)
  const amountMinor = parseMajorAmount(amount)
  const amountTooLarge = amountMinor === -1
  const canContinue = amountMinor > 0 && !amountTooLarge && !deposit.isPending

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canContinue) return
    setPinError('')
    setPinOpen(true)
  }

  async function handlePinConfirm(pin: string) {
    const idempotencyKey = crypto.randomUUID()
    try {
      const result = await deposit.mutateAsync({ currency, amount: amountMinor, idempotencyKey, pin })
      setPinOpen(false)
      setSuccessResult({ amount: result.amount, currency: result.currency })
      setAmount('')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'response' in err
        ? (err as { response: { data: { error: { message: string } } } }).response?.data?.error?.message
        : ''
      if (msg?.toLowerCase().includes('pin')) {
        setPinError('Incorrect PIN. Please try again.')
      } else {
        setPinOpen(false)
      }
    }
  }

  return (
    <Layout>
      <div className="w-full max-w-[880px]">
        <BackButton href="/dashboard" />


        {successResult ? (
          <div className="rounded-[28px] bg-white px-5 py-12 text-center">
            <div className="mx-auto flex max-w-[300px] flex-col items-center gap-6">

              {/* Animated checkmark */}
              <div className="relative flex items-center justify-center">
                {/* pulse ring */}
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

              {/* Text */}
              <div
                className="flex flex-col items-center gap-1"
                style={{ animation: 'success-fade-up 0.4s 0.2s ease-out both' }}
              >
                <p className="text-[44px] font-black leading-none text-[#11160f]">
                  {formatAmount(successResult.amount, successResult.currency)}
                </p>
                <p className="mt-1 text-[14px] text-[#6b7c65]">
                  added to your <span className="font-semibold text-[#11160f]">Main account</span>
                </p>
              </div>

              {/* Actions */}
              <div
                className="mt-2 flex w-full flex-col gap-2"
                style={{ animation: 'success-fade-up 0.4s 0.35s ease-out both' }}
              >
                <button
                  type="button"
                  onClick={() => setSuccessResult(null)}
                  className="w-full rounded-full bg-[#9fe870] py-3 text-[15px] font-semibold text-[#173300] transition-colors hover:bg-[#8fdd5f]"
                >
                  Add more money
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
        ) : (
        <form onSubmit={handleSubmit}>
          <div className="rounded-[28px] bg-white px-5 py-6">
            <div className="mx-auto flex max-w-[500px] flex-col gap-6">
              <div>
                <p className="text-[13px] text-[#171b18]">
                  You add to <span className="font-semibold">Main account</span>
                </p>

                <div className="mt-3 flex items-center justify-between gap-4">
                  <button
                    type="button"
                    onClick={() => setChooserOpen(true)}
                    className="inline-flex py-2 items-center gap-2 rounded-full bg-[#eef0eb] px-3 text-[#11160f] transition-colors hover:bg-[#e7eae2]"
                  >
                    {SelectedCurrencyIcon
                      ? <SelectedCurrencyIcon className="size-6 shrink-0" />
                      : <span className="size-6 shrink-0 rounded-full bg-[#d7dbd3]" />
                    }
                    <span className="text-[15px] font-semibold">{currency}</span>
                    <ChevronDown size={16} className="text-[#233818]" />
                  </button>

                  <label htmlFor="deposit-amount" className="sr-only">
                    Deposit amount
                  </label>
                  <input
                    id="deposit-amount"
                    inputMode="decimal"
                    autoComplete="off"
                    placeholder="0.00"
                    value={formatInputAmount(amount)}
                    onChange={(e) => { setAmount(e.target.value.replace(/,/g, '').replace(/[^0-9.]/g, '')); deposit.reset() }}
                    className="min-w-0 flex-1 bg-transparent text-right text-[40px] font-black leading-none tracking-normal text-[#7b7b7b] outline-none transition-all duration-150 placeholder:text-[#7b7b7b] focus:text-[56px] focus:text-[#11160f]"
                  />
                </div>
              </div>

              <div className={`rounded-[20px] p-3 transition-colors ${deposit.isError || amountTooLarge ? 'bg-[#fdecea]' : 'bg-[#eef0eb]'}`}>
                <div className={`flex items-center justify-center gap-2 px-2 py-1 text-center text-[13px] ${deposit.isError || amountTooLarge ? 'text-[#c0392b]' : 'text-[#171b18]'}`}>
                  {deposit.isError || amountTooLarge
                    ? <AlertCircle size={15} className="shrink-0" />
                    : <Info size={15} className="shrink-0" />
                  }
                  <p>
                    {amountTooLarge
                      ? 'Amount is too large. Please enter a smaller value.'
                      : deposit.isError
                        ? apiError(deposit.error)
                        : amountMinor > 0
                          ? 'Ready to add money to your wallet'
                          : 'Enter the amount you wish to add'
                    }
                  </p>
                </div>

                <button
                  type="submit"
                  disabled={!canContinue}
                  className={`mt-2 flex py-3 w-full items-center justify-center rounded-full text-[15px] font-semibold transition-colors ${
                    canContinue
                      ? 'bg-[#9fe870] text-[#173300] hover:bg-[#8fdd5f]'
                      : deposit.isError
                        ? 'bg-[#f5c6c2] text-[#c0392b]'
                        : 'bg-[#dfe2db] text-[#969b96]'
                  }`}
                >
                  {deposit.isPending ? 'Processing…' : 'Continue'}
                </button>
              </div>
            </div>
          </div>
        </form>
        )}

        <Dialog open={chooserOpen} onOpenChange={setChooserOpen}>
          <DialogContent className="max-w-[480px] rounded-[28px] px-6 py-8 shadow-[0_24px_80px_rgba(0,0,0,0.15)]">
            <button
              type="button"
              onClick={() => setChooserOpen(false)}
              className="absolute right-5 top-5 flex size-10 items-center justify-center rounded-full bg-[#eef0eb] text-[#233818] transition-colors hover:bg-[#e6e9e2]"
              aria-label="Close currency chooser"
            >
              <X size={18} />
            </button>

            <DialogHeader>
              <DialogTitle className="text-[22px] font-semibold tracking-normal text-[#11160f]">
                Choose a currency
              </DialogTitle>
            </DialogHeader>

            <div className="mt-4 rounded-[20px] border border-[#d9ddd6] p-4">
              <h3 className="text-[14px] font-semibold text-[#11160f]">Main account</h3>

              <div className="mt-3 space-y-1">
                {availableCurrencies.map((code) => {
                  const Icon = currencyIcon(code)
                  const balance = balances.data?.find((item) => item.currency === code)
                  return (
                    <button
                      key={code}
                      type="button"
                      onClick={() => {
                        setCurrency(code)
                        setChooserOpen(false)
                      }}
                      className={`grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-2xl px-2 py-2.5 text-left transition-colors hover:bg-[#f5f6f3] ${code === currency ? 'bg-[#f5f6f3]' : ''}`}
                    >
                      {Icon
                        ? <Icon className="size-7 shrink-0" />
                        : <span className="size-7 shrink-0 rounded-full bg-[#eef0eb]" />
                      }

                      <span className="min-w-0">
                        <span className="block text-[15px] font-semibold text-[#11160f]">
                          {balance ? formatAmount(balance.amount, code) : `0 ${code}`}
                        </span>
                        <span className="block text-[13px] text-[#4f5650]">
                          {CURRENCY_NAMES[code] ?? code}
                        </span>
                      </span>

                      {code === currency
                        ? <span className="size-2 rounded-full bg-[#9fe870]" />
                        : <ChevronRight size={16} className="text-[#b0b8ab]" />
                      }
                    </button>
                  )
                })}

              </div>
            </div>
          </DialogContent>
        </Dialog>

        <PinDialog
          open={pinOpen}
          onConfirm={handlePinConfirm}
          onCancel={() => { setPinOpen(false); setPinError('') }}
          error={pinError}
          loading={deposit.isPending}
        />
      </div>
    </Layout>
  )
}
