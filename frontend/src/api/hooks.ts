import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import client from './client'

// --- Types ---
export interface Balance { currency: string; amount: number; display: string }
export interface Transaction { id: string; type: string; reference_id: string; created_at: string }
export interface HistoryResult { items: Transaction[]; total: number; page: number; total_pages: number }
export interface Quote {
  id: string; from_currency: string; to_currency: string
  market_rate: string; quoted_rate: string
  amount_in: number; amount_out: number; fee: number
  expires_at: string; seconds_left: number
}
export interface Conversion {
  id: string; from_currency: string; to_currency: string
  amount_in: number; amount_out: number; quoted_rate: string; fee: number; status: string; created_at: string
}
export interface Payout {
  id: string; source_currency: string; amount: number; status: string
  recipient_account_number: string; recipient_bank_code: string; recipient_account_name: string
  compliance_flagged: boolean; failure_reason?: string; reversed_at?: string
  created_at: string; updated_at: string
}
export interface Deposit {
  id: string; currency: string; amount: number; status: string
  idempotency_key: string; created_at: string
}

// --- Auth ---
export function useSignup() {
  return useMutation({
    mutationFn: (data: { email: string; password: string }) =>
      client.post<{ token: string; user_id: string }>('/auth/signup', data).then(r => r.data),
  })
}

export function useLogin() {
  return useMutation({
    mutationFn: (data: { email: string; password: string }) =>
      client.post<{ token: string; user_id: string }>('/auth/login', data).then(r => r.data),
  })
}

// --- Wallet ---
export function useBalances() {
  return useQuery({
    queryKey: ['balances'],
    queryFn: () => client.get<{ balances: Balance[] }>('/wallets/balances').then(r => r.data.balances),
    refetchInterval: 10_000,
  })
}

export function useTransactions(page = 1) {
  return useQuery({
    queryKey: ['transactions', page],
    queryFn: () => client.get<HistoryResult>(`/wallets/transactions?page=${page}&limit=20`).then(r => r.data),
  })
}

// --- Deposits ---
export function useDeposit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ currency, amount, idempotencyKey }: { currency: string; amount: number; idempotencyKey: string }) =>
      client.post<Deposit>('/deposits', { currency, amount }, {
        headers: { 'Idempotency-Key': idempotencyKey },
      }).then(r => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['balances'] }),
  })
}

// --- FX Conversion ---
export function useCreateQuote() {
  return useMutation({
    mutationFn: (data: { from_currency: string; to_currency: string; amount_in: number }) =>
      client.post<Quote>('/conversions/quote', data).then(r => r.data),
  })
}

export function useExecuteConversion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (quoteId: string) =>
      client.post<Conversion>('/conversions/execute', { quote_id: quoteId }).then(r => r.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['balances'] })
      qc.invalidateQueries({ queryKey: ['transactions'] })
    },
  })
}

// --- Payouts ---
export function useCreatePayout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: {
      source_currency: string; amount: number
      recipient_account_number: string; recipient_bank_code: string; recipient_account_name: string
    }) => client.post<Payout>('/payouts', data).then(r => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['balances'] }),
  })
}

export function usePayout(id: string, enabled = true) {
  return useQuery({
    queryKey: ['payout', id],
    queryFn: () => client.get<Payout>(`/payouts/${id}`).then(r => r.data),
    enabled: !!id && enabled,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'pending' || status === 'processing' ? 2000 : false
    },
  })
}
