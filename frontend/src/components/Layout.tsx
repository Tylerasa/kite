import { useState, useEffect, useCallback, type ReactNode } from 'react'
import { LogOut, Menu, X } from 'lucide-react'
import { Icons } from '../../public/assets/svgs/icons'

function NavItem({
  href,
  icon,
  label,
  activePath,
  onClick,
}: {
  href: string
  icon: ReactNode
  label: string
  activePath: string
  onClick?: () => void
}) {
  const active = activePath === href || (href === '/dashboard' && activePath === '/')
  return (
    <a
      href={href}
      onClick={onClick}
      className={`flex min-h-10 items-center gap-4 rounded-full px-4 text-[14px] font-normal transition-colors ${
        active
          ? 'bg-[#eef0eb] text-[#253b1f]'
          : 'text-[#5f635d] hover:bg-[#f4f5f1] hover:text-[#253b1f]'
      }`}
    >
      <span className="flex size-5 shrink-0 items-center justify-center">{icon}</span>
      <span className="min-w-0 leading-5">{label}</span>
    </a>
  )
}

const NAV_ITEMS = [
  { href: '/dashboard', icon: <Icons.home className="size-5" />, label: 'Home' },
  { href: '/deposit',   icon: <Icons.deposit className="size-5" />, label: 'Add money' },
  { href: '/convert',   icon: <Icons.transfer className="size-5" />, label: 'Convert' },
  { href: '/payout',    icon: <Icons.send className="size-5" />, label: 'Send' },
  { href: '/transactions', icon: <Icons.transactions className="size-5" />, label: 'Transactions' },
]

const INACTIVITY_MS = 5 * 60 * 1000 // 5 minutes
const LAST_ACTIVE_KEY = 'kite_last_active'

function logout() {
  localStorage.removeItem('kite_token')
  localStorage.removeItem('kite_user_id')
  localStorage.removeItem('kite_name')
  localStorage.removeItem(LAST_ACTIVE_KEY)
  window.location.href = '/login'
}

function useInactivityLock() {
  useEffect(() => {
    if (!localStorage.getItem('kite_token')) return

    const touch = () => localStorage.setItem(LAST_ACTIVE_KEY, String(Date.now()))

    const EVENTS = ['mousemove', 'mousedown', 'keydown', 'scroll', 'touchstart'] as const
    EVENTS.forEach(e => window.addEventListener(e, touch, { passive: true }))
    touch() // record immediately on mount

    const interval = setInterval(() => {
      const last = Number(localStorage.getItem(LAST_ACTIVE_KEY) ?? 0)
      if (Date.now() - last >= INACTIVITY_MS) {
        sessionStorage.setItem('kite_logout_reason', 'inactivity')
        logout()
      }
    }, 30_000) // check every 30 s

    return () => {
      EVENTS.forEach(e => window.removeEventListener(e, touch))
      clearInterval(interval)
    }
  }, [])
}

function SidebarFooter() {
  const name = localStorage.getItem('kite_name') ?? ''
  const initials = name
    ? name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
    : 'ME'

  return (
    <button
      type="button"
      onClick={logout}
      className="group flex w-full items-center gap-2.5 rounded-full px-3 py-2 text-left transition-colors hover:bg-[#f4f5f1]"
    >
      <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-[#9fe870] text-[11px] font-bold text-[#163300]">
        {initials}
      </span>
      <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-[#5f635d] group-hover:text-[#253b1f]">
        Log out
      </span>
      <LogOut size={14} className="shrink-0 text-[#9aa097] group-hover:text-[#5f635d]" />
    </button>
  )
}

export function Layout({ children }: { children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const isLoggedIn = !!localStorage.getItem('kite_token')
  useInactivityLock()
  const activePath = window.location.pathname

  if (!isLoggedIn) {
    return <div className="min-h-screen bg-white font-sans text-[#1f1f1f]">{children}</div>
  }

  return (
    <div className="min-h-screen bg-white font-sans text-[#1f1f1f]">

      {/* ── Mobile top bar ───────────────────────────────────── */}
      <header className="sticky top-0 z-30 flex items-center justify-between border-b border-[#eef0eb] bg-white px-4 py-3 lg:hidden">
        <a href="/dashboard" className="flex items-center text-[22px] font-semibold tracking-normal text-[#163300]">
          <span className="mr-1 text-[18px] leading-none">↗</span>
          Kite
        </a>
        <button
          type="button"
          onClick={() => setMenuOpen(true)}
          className="flex size-9 items-center justify-center rounded-full bg-[#eef0eb] text-[#253b1f]"
          aria-label="Open menu"
        >
          <Menu size={18} />
        </button>
      </header>

      {/* ── Mobile drawer backdrop ───────────────────────────── */}
      {menuOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/25 lg:hidden"
          onClick={() => setMenuOpen(false)}
        />
      )}

      {/* ── Mobile drawer ────────────────────────────────────── */}
      <div
        className={`fixed inset-y-0 left-0 z-50 flex w-[240px] flex-col bg-white px-3 pb-8 pt-6 transition-transform duration-300 ease-in-out lg:hidden ${
          menuOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="mb-8 flex items-center justify-between px-2">
          <a href="/dashboard" className="flex items-center text-[22px] font-semibold tracking-normal text-[#163300]">
            <span className="mr-1 text-[18px] leading-none">↗</span>
            Kite
          </a>
          <button
            type="button"
            onClick={() => setMenuOpen(false)}
            className="flex size-8 items-center justify-center rounded-full bg-[#eef0eb] text-[#253b1f]"
            aria-label="Close menu"
          >
            <X size={16} />
          </button>
        </div>

        <nav className="flex flex-1 flex-col gap-1">
          {NAV_ITEMS.map(item => (
            <NavItem
              key={item.href}
              href={item.href}
              icon={item.icon}
              label={item.label}
              activePath={activePath}
              onClick={() => setMenuOpen(false)}
            />
          ))}
        </nav>

        <SidebarFooter />
      </div>

      {/* ── Desktop layout ───────────────────────────────────── */}
      <div className="flex max-w-[1280px]">
        <aside className="hidden w-[200px] shrink-0 px-2 pb-8 pt-10 lg:flex lg:flex-col lg:sticky lg:top-0 lg:h-screen">
          <a href="/dashboard" className="mb-8 flex items-center px-3 text-[24px] font-semibold tracking-normal text-[#163300]">
            <span className="mr-1 text-[20px] leading-none">↗</span>
            Kite
          </a>

          <nav className="flex flex-1 flex-col gap-1">
            {NAV_ITEMS.map(item => (
              <NavItem
                key={item.href}
                href={item.href}
                icon={item.icon}
                label={item.label}
                activePath={activePath}
              />
            ))}
          </nav>

          <SidebarFooter />
        </aside>

        <div className="min-w-0 flex-1 p-4 lg:pt-[40px] lg:pl-8 lg:pr-10">
          <main className="w-full">{children}</main>
        </div>
      </div>
    </div>
  )
}

export function InfoBanner({ msg, onDismiss }: { msg: string; onDismiss?: () => void }) {
  const [visible, setVisible] = useState(true)
  const dismiss = useCallback(() => {
    setVisible(false)
    onDismiss?.()
  }, [onDismiss])

  if (!visible) return null

  return (
    <div className="mb-6 flex items-start gap-4 rounded-2xl bg-[#f0f0f0] px-4 py-4 pr-3">
      <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full bg-[#3c3c3c] text-white text-[13px] font-bold">
        i
      </span>
      <p className="flex-1 text-[14px] leading-[1.55] text-[#3c3c3c]">{msg}</p>
      <button
        type="button"
        onClick={dismiss}
        aria-label="Dismiss"
        className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full text-[#6b6b6b] transition-colors hover:bg-[#e2e2e2]"
      >
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
          <path d="M1 1l10 10M11 1L1 11" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round"/>
        </svg>
      </button>
    </div>
  )
}

export function Spinner() {
  return (
    <div className="flex items-center justify-center py-14 text-sm font-medium text-[#747870]">
      Loading...
    </div>
  )
}

export function ErrorBanner({ msg }: { msg: string }) {
  return (
    <div className="mb-5 rounded-xl border border-[#fecdd3] bg-[#fff0f2] px-4 py-3 text-sm text-[#be123c]">
      {msg}
    </div>
  )
}

export function SuccessBanner({ msg }: { msg: string }) {
  return (
    <div className="mb-5 rounded-xl border border-[#bbf7d0] bg-[#f0fdf4] px-4 py-3 text-sm text-[#15803d]">
      {msg}
    </div>
  )
}

export function formatAmount(amount: number, currency: string): string {
  const symbols: Record<string, string> = { USD: '$', GBP: '£', EUR: '€', NGN: '₦', KES: 'KSh ' }
  return `${symbols[currency] ?? ''}${(amount / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export function apiError(err: unknown): string {
  if (err && typeof err === 'object' && 'response' in err) {
    const r = (err as { response: { data: { error: { message: string } } } }).response
    return r?.data?.error?.message ?? 'Something went wrong.'
  }
  return 'Something went wrong.'
}
