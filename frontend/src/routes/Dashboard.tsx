import { ChevronRight, Plus, RefreshCw, Send } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { Icons } from '../../public/assets/svgs/icons'
import { useBalances, useTransactions } from '../api/hooks'
import { Layout, Spinner, formatAmount } from '../components/Layout'
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  type CarouselApi,
} from '../components/ui/carousel'

const CURRENCY_NAMES: Record<string, string> = {
  USD: 'US dollar',
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

const TX_TYPE_CONFIG: Record<string, { out: boolean }> = {
  deposit:    { out: false },
  conversion: { out: false },
  payout:     { out: true  },
  reversal:   { out: false },
}

function parseTxDate(value: string): Date | null {
  if (!value) return null
  const d = new Date(value)
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return null
  return d
}

function formatDateGroup(value: string) {
  const d = parseTxDate(value)
  if (!d) return 'Unknown date'
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return 'Today'
  if (d.toDateString() === yesterday.toDateString()) return 'Yesterday'
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: d.getFullYear() !== today.getFullYear() ? 'numeric' : undefined })
}

function ActionButton({
  href,
  label,
  icon,
  primary = false,
}: {
  href: string
  label: string
  icon: ReactNode
  primary?: boolean
}) {
  return (
    <a
      href={href}
      className={`inline-flex h-10 items-center gap-2 rounded-full px-4 text-[14px] font-medium transition-colors ${
        primary
          ? 'bg-[#9fe870] text-[#163300] hover:bg-[#8fdd5f]'
          : 'border border-[#dfe2db] bg-white text-[#242822] hover:bg-[#f7f8f4]'
      }`}
    >
      {icon}
      {label}
    </a>
  )
}

function transactionIcon(type: string) {
  if (type === 'deposit') return Icons.deposit
  if (type === 'payout') return Icons.payout
  if (type === 'reversal') return Icons.reversed
  if (type === 'failed') return Icons.failed
  if (type === 'conversion') return Icons.reversed
  return Icons.deposit
}

function TransactionIcon({ type }: { type: string }) {
  const Icon = transactionIcon(type)
  return (
    <span className="flex size-12 shrink-0 items-center justify-center rounded-full bg-[#eef0eb] text-[#111923]">
      <Icon className="size-6" />
    </span>
  )
}

type ActivityItem = {
  id: string
  type: string
  time: string
  amount: number
  currency: string
  direction: string
}

function activityTitle(type: string, out: boolean) {
  if (type === 'payout') return 'Bank transfer'
  if (type === 'deposit') return 'Kite wallet'
  if (type === 'conversion') return 'Currency exchange'
  if (type === 'reversal') return 'Returned transfer'
  return out ? 'Transfer sent' : 'Wallet activity'
}

function activitySubtitle(type: string, out: boolean) {
  if (type === 'deposit') return 'Added'
  if (type === 'conversion') return 'Converted'
  if (type === 'reversal') return 'Returned'
  return out ? 'Sent' : 'Received'
}


function groupActivities(items: ActivityItem[]) {
  return items.reduce<Array<{ date: string; items: ActivityItem[] }>>((groups, item) => {
    const date = formatDateGroup(item.time)
    const existing = groups.find((group) => group.date === date)
    if (existing) {
      existing.items.push(item)
    } else {
      groups.push({ date, items: [item] })
    }
    return groups
  }, [])
}

function BalanceCard({
  balance,
}: {
  balance: {
    currency: string
    amount: number
  }
}) {
  const CurrencyIcon = CURRENCY_ICONS[balance.currency as keyof typeof CURRENCY_ICONS]

  return (
    <article
      className="h-[210px] w-full rounded-[8px] border border-[#d7dbd3] bg-white p-6"
    >
      <div className="flex h-full flex-col justify-between">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-[13px] text-[#62675f]">
              {CURRENCY_NAMES[balance.currency] ?? balance.currency}
            </p>
            <p className="mt-2 max-w-[240px] text-[clamp(1.75rem,1.2rem+1vw,2.25rem)] font-semibold leading-[1.05] tracking-normal text-[#242822] break-words">
              {formatAmount(balance.amount, balance.currency)}
            </p>
          </div>
          {CurrencyIcon ? <CurrencyIcon className="mt-0.5 size-8 shrink-0" /> : null}
        </div>

        <div className="flex items-end justify-end gap-4">
          <div className="text-right">
            <p
              className="text-[20px] font-black uppercase tracking-[0.08em] text-[#11160f]"
              style={{ fontFamily: '"Lilita One", sans-serif' }}
            >
              KITE
            </p>
          </div>
        </div>
      </div>
    </article>
  )
}

export default function Dashboard() {
  const [carouselApi, setCarouselApi] = useState<CarouselApi>()
  const [selectedIndex, setSelectedIndex] = useState(0)
  const balances = useBalances()
  const transactions = useTransactions(1)

  const balanceList = balances.data ?? []
  const recentTransactions = transactions.data?.items?.slice(0, 8) ?? []
  const activityGroups = groupActivities(recentTransactions)

  useEffect(() => {
    if (!carouselApi) return

    const syncSelected = () => {
      setSelectedIndex(carouselApi.selectedScrollSnap())
    }

    syncSelected()
    carouselApi.on('select', syncSelected)
    carouselApi.on('reInit', syncSelected)

    return () => {
      carouselApi.off('select', syncSelected)
      carouselApi.off('reInit', syncSelected)
    }
  }, [carouselApi])

  return (
    <Layout>
      <div className="w-full">
        <header className="mb-8">
          <p className="text-[13px] font-medium text-[#8a9088]">Dashboard</p>
          <div className="mt-1 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <h1 className="text-[32px] font-semibold tracking-normal text-[#242822]">Wallet overview</h1>
            <div className="flex flex-wrap gap-2">
              <ActionButton href="/payout" label="Send" icon={<Send size={16} />} primary />
              <ActionButton href="/deposit" label="Add money" icon={<Plus size={17} />} />
              <ActionButton href="/convert" label="Convert" icon={<RefreshCw size={16} />} />
            </div>
          </div>
        </header>
        <section className="mb-8">
          <div>
            {balances.isLoading ? (
              <Spinner />
            ) : balanceList.length ? (
              <>
                <Carousel
                  setApi={setCarouselApi}
                  opts={{ align: 'start', loop: false }}
                  className="pb-4 pt-1"
                >
                  <CarouselContent className="-ml-5">
                    {balanceList.map((balance, index) => (
                      <CarouselItem
                        key={balance.currency}
                        className="basis-[480px] pl-5 sm:basis-[480px]"
                      >
                        <BalanceCard balance={balance} />
                      </CarouselItem>
                    ))}
                  </CarouselContent>
                </Carousel>
                <div className="mt-1 flex justify-center gap-1.5">
                  {balanceList.map((balance, index) => (
                    <span
                      key={balance.currency}
                      className={`h-1.5 rounded-full transition-all ${
                        index === selectedIndex ? 'w-6 bg-[#242822]' : 'w-1.5 bg-[#d7dbd3]'
                      }`}
                    />
                  ))}
                </div>
              </>
            ) : (
              <p className="px-5 py-10 text-center text-[14px] text-[#6f746d]">No balances yet.</p>
            )}
          </div>
        </section>

        <section>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-[20px] font-semibold text-[#242822]">Recent activity</h2>
            <a href="/transactions" className="text-[13px] font-medium text-[#314d25] underline underline-offset-4">
              See all
            </a>
          </div>

          <div>
            {transactions.isLoading ? (
              <Spinner />
            ) : recentTransactions.length ? (
              <div className="space-y-10">
                {activityGroups.map((group) => (
                  <section key={group.date}>
                    <div className="mb-4 border-b border-[#dfe2db] pb-3 text-[15px] font-medium text-[#3f443d]">
                      {group.date}
                    </div>

                    <div className="space-y-1">
                      {group.items.map((tx) => {
                        const config = TX_TYPE_CONFIG[tx.type] ?? { label: tx.type, out: false }
                        const out = config.out
                        const time = parseTxDate(tx.time)
                          ? new Date(tx.time).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
                          : ''
                        return (
                          <a
                            key={tx.id}
                            href={`/transactions/${tx.id}`}
                            className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 rounded-2xl px-3 py-3 transition-colors hover:bg-[#f7f8f4]"
                          >
                            <TransactionIcon type={tx.type} />

                            <div className="min-w-0">
                              <p className="truncate text-[15px] font-semibold text-[#111923]">
                                {activityTitle(tx.type, out)}
                              </p>
                              <p className="mt-0.5 text-[13px] text-[#6b7c65]">
                                {tx.status === 'review' ? 'Under review' : tx.status === 'failed' ? 'Failed' : activitySubtitle(tx.type, out)}{time ? ` · ${time}` : ''}
                              </p>
                            </div>

                            <div className="flex items-center gap-2 shrink-0">
                              {tx.amount > 0 && tx.currency ? (
                                <span className={`text-[14px] font-semibold ${tx.direction === 'debit' ? 'text-[#11160f]' : 'text-[#173300]'}`}>
                                  {tx.direction === 'debit' ? '−' : '+'}{formatAmount(tx.amount, tx.currency)}
                                </span>
                              ) : null}
                              <ChevronRight size={16} className="text-[#b0b8ab]" />
                            </div>
                          </a>
                        )
                      })}
                    </div>
                  </section>
                ))}
              </div>
            ) : (
              <p className="py-8 text-center text-[14px] text-[#6f746d]">No transactions yet.</p>
            )}
          </div>
        </section>
      </div>
    </Layout>
  )
}
