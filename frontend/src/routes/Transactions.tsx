import { useState } from 'react'
import { ChevronRight } from 'lucide-react'
import { Icons } from '../../public/assets/svgs/icons'
import { useTransactions, type Transaction } from '../api/hooks'
import { Layout, Spinner, formatAmount } from '../components/Layout'

const TX_TYPE_CONFIG: Record<string, { title: string; subtitle: string }> = {
  deposit:    { title: 'Kite wallet',       subtitle: 'Added'     },
  payout:     { title: 'Bank transfer',     subtitle: 'Sent'      },
  conversion: { title: 'Currency exchange', subtitle: 'Converted' },
  reversal:   { title: 'Returned transfer', subtitle: 'Returned'  },
}

function transactionIcon(type: string) {
  if (type === 'deposit')    return Icons.deposit
  if (type === 'payout')     return Icons.payout
  if (type === 'reversal')   return Icons.reversed
  if (type === 'conversion') return Icons.reversed
  return Icons.deposit
}

function parseTxDate(value: string): Date | null {
  if (!value) return null
  const d = new Date(value)
  if (isNaN(d.getTime())) return null
  // Ignore Go zero time (year 1)
  if (d.getFullYear() < 2000) return null
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

function formatTime(value: string) {
  const d = parseTxDate(value)
  return d ? d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : ''
}

function groupTransactions(items: Transaction[]) {
  return items.reduce<Array<{ date: string; items: typeof items }>>((groups, item) => {
    const date = formatDateGroup(item.time)
    const existing = groups.find(g => g.date === date)
    if (existing) existing.items.push(item)
    else groups.push({ date, items: [item] })
    return groups
  }, [])
}

export default function Transactions() {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useTransactions(page, 50)

  const groups = groupTransactions(data?.items ?? [])

  return (
    <Layout>
      <div className="mb-8">
        <h1 className="text-[22px] font-bold tracking-tight text-[#11160f]">Activity</h1>
        <p className="mt-1 text-[13px] text-[#6b7c65]">Your full transaction history.</p>
      </div>

      {isLoading ? <Spinner /> : !data?.items?.length ? (
        <div className="py-16 text-center text-[14px] text-[#6b7c65]">
          No transactions yet.{' '}
          <a href="/deposit" className="font-semibold text-[#11160f] underline underline-offset-2">
            Make a deposit
          </a>{' '}to get started.
        </div>
      ) : (
        <div className="space-y-10">
          {groups.map(group => (
            <section key={group.date}>
              {/* Date header */}
              <div className="mb-4 border-b border-[#dfe2db] pb-3 text-[15px] font-medium text-[#3f443d]">
                {group.date}
              </div>

              <div className="space-y-1">
                {group.items.map(tx => {
                  const tc = TX_TYPE_CONFIG[tx.type] ?? TX_TYPE_CONFIG.deposit
                  const Icon = transactionIcon(tx.type)
                  return (
                    <a
                      key={tx.id}
                      href={`/transactions/${tx.id}`}
                      className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 rounded-2xl px-3 py-3 transition-colors hover:bg-[#f7f8f4]"
                    >
                      {/* Icon */}
                      <span className="flex size-12 shrink-0 items-center justify-center rounded-full bg-[#eef0eb] text-[#111923]">
                        <Icon className="size-6" />
                      </span>

                      {/* Title + meta */}
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-[15px] font-semibold text-[#111923]">{tc.title}</p>
                        <p className="mt-0.5 text-[13px] text-[#6b7c65]">
                          {tx.status === 'review' ? 'Under review' : tx.status === 'failed' ? 'Failed' : tc.subtitle}{formatTime(tx.time) ? ` · ${formatTime(tx.time)}` : ''}
                        </p>
                      </div>

                      {/* Amount + chevron */}
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

          {/* Pagination */}
          {(data?.total_pages ?? 1) > 1 && (
            <div className="flex items-center justify-between pt-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="rounded-full bg-[#eef0eb] px-5 py-2.5 text-[13px] font-semibold text-[#11160f] transition-colors hover:bg-[#e7eae2] disabled:opacity-40"
              >
                ← Previous
              </button>
              <span className="text-[13px] text-[#6b7c65]">{page} of {data?.total_pages}</span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={page >= (data?.total_pages ?? 1)}
                className="rounded-full bg-[#eef0eb] px-5 py-2.5 text-[13px] font-semibold text-[#11160f] transition-colors hover:bg-[#e7eae2] disabled:opacity-40"
              >
                Next →
              </button>
            </div>
          )}
        </div>
      )}
    </Layout>
  )
}
