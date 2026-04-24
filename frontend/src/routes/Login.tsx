import { useState } from 'react'
import { useLogin } from '../api/hooks'
import { apiError, InfoBanner } from '../components/Layout'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const login = useLogin()

  const [logoutReason] = useState(() => {
    const reason = sessionStorage.getItem('kite_logout_reason')
    if (reason) sessionStorage.removeItem('kite_logout_reason')
    return reason
  })

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    login.mutate({ email, password }, {
      onSuccess(result) {
        localStorage.setItem('kite_token', result.token)
        localStorage.setItem('kite_user_id', result.user_id)
        localStorage.setItem('kite_name', result.name)
        window.location.href = '/dashboard'
      },
    })
  }

  return (
    <div className="min-h-screen bg-white flex flex-col font-sans text-[#1b1b1b] justify-evenly">

      {/* Form */}
      <div className="flex flex-col items-center pt-16 px-4">
        <div className="w-full max-w-120">

          <h1 className="text-[26px] font-bold text-center tracking-tight mb-1.5">
            Welcome back
          </h1>
          <p className="text-sm text-[#757575] text-center mb-8">
            New to Kite?{' '}
            <a href="/signup" className="text-[#1b1b1b] font-semibold underline underline-offset-2">
              Sign up
            </a>
          </p>

          {logoutReason === 'inactivity' && (
            <InfoBanner msg="We logged you out because you were inactive for 5 minutes — it's to help keep your account secure." />
          )}

          {login.isError && (
            <div className="bg-[#fff0f2] border border-[#fecdd3] text-[#be123c] rounded-md px-4 py-3 text-sm mb-5">
              {apiError(login.error)}
            </div>
          )}

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email" className="text-[13px] text-[#757575] font-medium">
                Your email address
              </Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                autoComplete="email"
                required
                className="h-12 text-[15px] border-[#b8c0c8] focus:border-[#6c6c6b] focus-visible:ring-0 focus-visible:border-[#1b1b1b] rounded-lg pr-11"

              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password" className="text-[13px] text-[#757575] font-medium">
                Your password
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPw ? 'text' : 'password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                  className="h-12 text-[15px] border-[#b8c0c8] focus:border-[#6c6c6b] focus-visible:ring-0 focus-visible:border-[#1b1b1b] rounded-lg pr-11"
                />
                <button
                  type="button"
                  onClick={() => setShowPw(v => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-[#757575] flex items-center"
                  aria-label={showPw ? 'Hide password' : 'Show password'}
                >
                  {showPw
                    ? <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                    : <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                  }
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={login.isPending}
              className="w-full h-12 mt-2 rounded-full font-bold text-base cursor-pointer disabled:cursor-default transition-colors"
              style={{ background: login.isPending ? '#c8f5a0' : '#9fe870', color: '#163300' }}
            >
              {login.isPending ? 'Logging in…' : 'Log in'}
            </button>

            <div className="flex items-center gap-3 my-1">
              <div className="h-px flex-1 bg-[#e8ecef]" />
              <span className="text-xs text-[#aaa]">or</span>
              <div className="h-px flex-1 bg-[#e8ecef]" />
            </div>

            <a
              href="/signup"
              className="flex w-full h-12 items-center justify-center rounded-full border border-[#d0d5db] text-[15px] font-semibold text-[#1b1b1b] hover:bg-[#f5f5f5] transition-colors"
            >
              Create an account
            </a>
          </form>
        </div>
      </div>
    </div>
  )
}
