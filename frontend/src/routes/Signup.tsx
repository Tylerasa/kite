import { useState } from 'react'
import { useSignup } from '../api/hooks'
import { apiError } from '../components/Layout'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export default function Signup() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pin, setPin] = useState('')
  const [showPw, setShowPw] = useState(false)
  const signup = useSignup()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    try {
      const result = await signup.mutateAsync({ name, email, password, pin })
      localStorage.setItem('kite_token', result.token)
      localStorage.setItem('kite_user_id', result.user_id)
      localStorage.setItem('kite_name', result.name)
      window.location.href = '/dashboard'
    } catch (_) {}
  }

  return (
    <div className="min-h-screen bg-white flex flex-col font-sans text-[#1b1b1b] justify-evenly">

      {/* Form */}
      <div className="flex flex-col items-center pt-16 px-4">
        <div className="w-full max-w-120">

          <h1 className="text-[26px] font-bold text-center tracking-tight mb-1.5">
            Create your account
          </h1>
          <p className="text-sm text-[#757575] text-center mb-8">
            Already have an account?{' '}
            <a href="/login" className="text-[#1b1b1b] font-semibold underline underline-offset-2">
              Log in
            </a>
          </p>

          {signup.isError && (
            <div className="bg-[#fff0f2] border border-[#fecdd3] text-[#be123c] rounded-md px-4 py-3 text-sm mb-5">
              {apiError(signup.error)}
            </div>
          )}

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="name" className="text-[13px] text-[#757575] font-medium">
                Your full name
              </Label>
              <Input
                id="name"
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                autoComplete="name"
                required
                className="h-12 text-[15px] border-[#b8c0c8] focus:border-[#6c6c6b] focus-visible:ring-0 focus-visible:border-[#1b1b1b] rounded-lg"
              />
            </div>

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
                className="h-12 text-[15px] border-[#b8c0c8] focus:border-[#6c6c6b] focus-visible:ring-0 focus-visible:border-[#1b1b1b] rounded-lg"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password" className="text-[13px] text-[#757575] font-medium">
                Password <span className="font-normal">(min 8 characters)</span>
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPw ? 'text' : 'password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  autoComplete="new-password"
                  minLength={8}
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

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="pin" className="text-[13px] text-[#757575] font-medium">
                Transaction PIN <span className="font-normal">(4 digits)</span>
              </Label>
              <Input
                id="pin"
                type="password"
                inputMode="numeric"
                value={pin}
                onChange={e => setPin(e.target.value.replace(/\D/g, '').slice(0, 4))}
                autoComplete="off"
                maxLength={4}
                required
                placeholder="••••"
                className="h-12 text-[15px] border-[#b8c0c8] focus:border-[#6c6c6b] focus-visible:ring-0 focus-visible:border-[#1b1b1b] rounded-lg tracking-[0.4em]"
              />
            </div>

            <button
              type="submit"
              disabled={signup.isPending}
              className="w-full h-12 mt-2 rounded-full font-bold text-base cursor-pointer disabled:cursor-default transition-colors"
              style={{ background: signup.isPending ? '#c8f5a0' : '#9fe870', color: '#163300' }}
            >
              {signup.isPending ? 'Creating account…' : 'Create account'}
            </button>

            <div className="flex items-center gap-3 my-1">
              <div className="h-px flex-1 bg-[#e8ecef]" />
              <span className="text-xs text-[#aaa]">or</span>
              <div className="h-px flex-1 bg-[#e8ecef]" />
            </div>

            <a
              href="/login"
              className="flex w-full h-12 items-center justify-center rounded-full border border-[#d0d5db] text-[15px] font-semibold text-[#1b1b1b] hover:bg-[#f5f5f5] transition-colors"
            >
              Log in instead
            </a>
          </form>

        </div>
      </div>
    </div>
  )
}
