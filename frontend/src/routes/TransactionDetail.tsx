import {
  AlertCircle,
  ArrowDownLeft,
  ArrowUpRight,
  Check,
  Loader2,
} from 'lucide-react'
import { useTransaction, usePayout, type TransactionEntry } from '../api/hooks'
import BackButton from '../components/BackButton'
import { Layout, Spinner, apiError, formatAmount } from '../components/Layout'


function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Unknown date'

  return date.toLocaleString(undefined, {
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function compactAmount(entry?: TransactionEntry) {
  if (!entry) return ''
  return formatAmount(entry.amount, entry.currency)
}

function entryByDirection(entries: TransactionEntry[], direction: string) {
  return entries.find((entry) => entry.account_type === 'user_wallet' && entry.direction === direction)
    ?? entries.find((entry) => entry.direction === direction)
}

function titleFor(type: string, isCredit: boolean) {
  if (type === 'deposit') return 'To your wallet'
  if (type === 'conversion') return 'Converted balance'
  if (type === 'reversal') return 'Transfer reversed'
  if (isCredit) return 'Money received'
  return 'Bank transfer'
}

function subtitleFor(type: string, isCredit: boolean) {
  if (type === 'deposit') return 'Added'
  if (type === 'conversion') return 'Converted'
  if (type === 'reversal') return 'Returned'
  return isCredit ? 'Received' : 'Sent'
}

export default function TransactionDetail() {
  const id = window.location.pathname.split('/').pop() ?? ''
  const transaction = useTransaction(id)
  const data = transaction.data
  const entries = data?.entries ?? []
  const debitEntry = entryByDirection(entries, 'debit')
  const creditEntry = entryByDirection(entries, 'credit')
  const primaryEntry = debitEntry ?? creditEntry ?? entries[0]
  const secondaryEntry = creditEntry && creditEntry.id !== primaryEntry?.id ? creditEntry : entries.find((entry) => entry.id !== primaryEntry?.id)
  const isCredit = primaryEntry?.direction === 'credit'
  const title = data ? titleFor(data.type, isCredit) : ''
  const subtitle = data ? subtitleFor(data.type, isCredit) : ''
  const transferTime = data ? formatDateTime(data.created_at) : ''
  const canRepeatTransfer = data?.type === 'payout'

  // Fetch live payout status when viewing a payout transaction
  const payoutDetail = usePayout(
    data?.type === 'payout' ? data.reference_id : '',
    data?.type === 'payout',
  )
  const payoutStatus = payoutDetail.data?.status
  const payoutFailed = payoutStatus === 'failed'
  const payoutInProgress = payoutStatus === 'pending' || payoutStatus === 'processing'

  return (
    <Layout>
      <div className="w-full">
        <BackButton href="/transactions" />

        <h1 className="mb-8 text-[34px] font-semibold leading-tight tracking-normal text-[#1f241d]">
          Transaction details
        </h1>

        {transaction.isLoading ? (
          <Spinner />
        ) : transaction.isError ? (
          <div className="rounded-[20px] bg-white p-6 text-sm text-[#be123c] ring-1 ring-[#f2c6ce]">
            {apiError(transaction.error)}
          </div>
        ) : data ? (
          <section className="overflow-hidden rounded-[20px] border border-[#dfe3dc] bg-white">
            <header className="flex items-center justify-between gap-6 px-9 py-8">
              <div className="flex min-w-0 items-center gap-5">
                <span className={`flex size-12 shrink-0 items-center justify-center rounded-full ${payoutFailed ? 'bg-[#fdecea] text-[#c0392b]' : 'bg-[#eef0eb] text-[#11160f]'}`}>
                  {isCredit ? <ArrowDownLeft size={24} /> : <ArrowUpRight size={24} />}
                </span>
                <div className="min-w-0">
                  <h2 className="truncate text-[18px] font-semibold text-[#1f241d]">{title}</h2>
                  <p className="mt-1 text-[14px] text-[#4f554d]">{subtitle}</p>
                </div>
              </div>

              <div className="flex shrink-0 items-center gap-5">
                {canRepeatTransfer ? (
                  <a
                    href="/payout"
                    className="flex h-10 items-center justify-center rounded-full bg-[#92e85b] px-6 text-[14px] font-semibold text-[#173300] transition-colors hover:bg-[#81d94d]"
                  >
                    Repeat
                  </a>
                ) : null}

                <div className="text-right">
                  <p className="text-[18px] font-semibold text-[#11160f]">{compactAmount(primaryEntry)}</p>
                  {secondaryEntry ? (
                    <p className="mt-2 text-[14px] text-[#4f554d]">{compactAmount(secondaryEntry)}</p>
                  ) : null}
                </div>
              </div>
            </header>

            <div className="flex items-center border-y border-[#dfe3dc] px-9">
              <div className="flex gap-7">
                <div className="border-b-2 border-[#263c1c] px-4 py-5 text-[15px] font-semibold text-[#263c1c]">
                  Updates
                </div>
              </div>
            </div>

            <div className="grid gap-14 px-9 py-9">
              <div className="max-w-[880px] space-y-7">
                {data.type === 'deposit' && (
                  <>
                    <TimelineItem complete time={transferTime} text="You initiated a deposit." />
                    {primaryEntry ? (
                      <TimelineItem complete time={transferTime} text={`${compactAmount(primaryEntry)} was credited to your ${primaryEntry.currency} wallet.`} />
                    ) : null}
                    <TimelineItem time={transferTime} title="Deposit complete" text="Your funds are now available in your wallet." />
                  </>
                )}

                {data.type === 'conversion' && (
                  <>
                    <TimelineItem complete time={transferTime} text="You initiated a currency conversion." />
                    {debitEntry ? (
                      <TimelineItem complete time={transferTime} text={`${compactAmount(debitEntry)} was debited from your ${debitEntry.currency} wallet.`} />
                    ) : null}
                    {creditEntry ? (
                      <TimelineItem complete time={transferTime} text={`${compactAmount(creditEntry)} was credited to your ${creditEntry.currency} wallet.`} />
                    ) : null}
                    <TimelineItem time={transferTime} title="Conversion complete" text={debitEntry && creditEntry ? `Converted ${compactAmount(debitEntry)} to ${compactAmount(creditEntry)}.` : 'Your conversion is complete.'} />
                  </>
                )}

                {data.type === 'payout' && (
                  <>
                    <TimelineItem complete time={transferTime} text="You initiated a bank transfer." />
                    {debitEntry ? (
                      <TimelineItem complete time={transferTime} text={`${compactAmount(debitEntry)} was debited from your ${debitEntry.currency} wallet.`} />
                    ) : null}

                    {payoutInProgress && (
                      <TimelineItem
                        pending
                        time={transferTime}
                        title="Transfer in progress"
                        text="Your transfer is being processed by the recipient's bank."
                      />
                    )}

                    {payoutFailed && (
                      <>
                        <TimelineItem
                          failed
                          time={payoutDetail.data?.updated_at ? formatDateTime(payoutDetail.data.updated_at) : transferTime}
                          title="Transfer failed"
                          text={payoutDetail.data?.failure_reason ?? 'The payment could not be completed.'}
                        />
                        {payoutDetail.data?.reversed_at && debitEntry && (
                          <TimelineItem
                            complete
                            time={formatDateTime(payoutDetail.data.reversed_at)}
                            title="Balance restored"
                            text={`${compactAmount(debitEntry)} has been returned to your ${debitEntry.currency} wallet.`}
                          />
                        )}
                      </>
                    )}

                    {!payoutInProgress && !payoutFailed && (
                      <>
                        {creditEntry ? (
                          <TimelineItem complete time={transferTime} text={`${compactAmount(creditEntry)} was sent to the recipient's bank account.`} />
                        ) : null}
                        <TimelineItem time={transferTime} title="Transfer complete" text={`${compactAmount(debitEntry ?? primaryEntry)} was successfully sent.`} />
                      </>
                    )}
                  </>
                )}

                {data.type === 'reversal' && (
                  <>
                    <TimelineItem complete time={transferTime} text="A reversal was initiated for this transaction." />
                    {primaryEntry ? (
                      <TimelineItem complete time={transferTime} text={`${compactAmount(primaryEntry)} was returned to your ${primaryEntry.currency} wallet.`} />
                    ) : null}
                    <TimelineItem time={transferTime} title="Reversal complete" text="The funds have been returned to your wallet." />
                  </>
                )}

                {!['deposit', 'conversion', 'payout', 'reversal'].includes(data.type) && (
                  <>
                    <TimelineItem complete time={transferTime} text="You initiated a transaction." />
                    <TimelineItem time={transferTime} title="Transaction complete" text={`This transaction was completed for ${compactAmount(primaryEntry)}.`} />
                  </>
                )}
              </div>
            </div>
          </section>
        ) : (
          <div className="rounded-[20px] bg-white p-6 text-sm text-[#6f746d] ring-1 ring-[#eceee8]">
            Transaction not found.
          </div>
        )}
      </div>
    </Layout>
  )
}

function TimelineItem({
  complete = false,
  failed = false,
  pending = false,
  time,
  title,
  text,
}: {
  complete?: boolean
  failed?: boolean
  pending?: boolean
  time: string
  title?: string
  text: string
}) {
  const icon = complete ? (
    <Check size={17} strokeWidth={2} className="text-[#173300]" />
  ) : failed ? (
    <AlertCircle size={17} className="text-[#c0392b]" />
  ) : pending ? (
    <Loader2 size={17} className="animate-spin text-[#6b7c65]" />
  ) : (
    <span className="size-3 rounded-full bg-[#11160f]" />
  )

  return (
    <div className="grid grid-cols-[24px_minmax(0,1fr)] gap-5">
      <span className="mt-1 flex size-6 items-center justify-center">
        {icon}
      </span>
      <div>
        <p className="text-[15px] text-[#3c4239]">{time}</p>
        {title ? (
          <h4 className={`mt-2 text-[18px] font-semibold ${failed ? 'text-[#c0392b]' : 'text-[#11160f]'}`}>
            {title}
          </h4>
        ) : null}
        <p className={`mt-1 max-w-[360px] text-[15px] leading-6 ${failed ? 'text-[#c0392b]' : 'text-[#3f453d]'}`}>
          {text}
        </p>
      </div>
    </div>
  )
}
