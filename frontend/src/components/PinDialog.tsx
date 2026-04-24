import { useEffect, useRef, useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './ui/dialog'

interface PinDialogProps {
  open: boolean
  onConfirm: (pin: string) => void
  onCancel: () => void
  error?: string
  loading?: boolean
}

export function PinDialog({ open, onConfirm, onCancel, error, loading }: PinDialogProps) {
  const [digits, setDigits] = useState(['', '', '', ''])
  const inputs = useRef<Array<HTMLInputElement | null>>([])

  // Reset digits when dialog opens
  useEffect(() => {
    if (open) {
      setDigits(['', '', '', ''])
      setTimeout(() => inputs.current[0]?.focus(), 50)
    }
  }, [open])

  function handleChange(index: number, value: string) {
    const digit = value.replace(/\D/g, '').slice(-1)
    const next = [...digits]
    next[index] = digit
    setDigits(next)
    if (digit && index < 3) {
      inputs.current[index + 1]?.focus()
    }
    if (digit && index === 3) {
      const pin = [...next].join('')
      if (pin.length === 4) onConfirm(pin)
    }
  }

  function handleKeyDown(index: number, e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Backspace' && !digits[index] && index > 0) {
      inputs.current[index - 1]?.focus()
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const pin = digits.join('')
    if (pin.length === 4) onConfirm(pin)
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onCancel() }}>
      <DialogContent className="max-w-[360px] rounded-[24px] p-8">
        <DialogHeader>
          <DialogTitle className="text-center text-[20px] font-semibold text-[#11160f]">
            Enter your PIN
          </DialogTitle>
        </DialogHeader>

        <p className="mt-1 text-center text-[14px] text-[#6b7c65]">
          Confirm this transaction with your 4-digit PIN.
        </p>

        <form onSubmit={handleSubmit} className="mt-6 flex flex-col items-center gap-6">
          <div className="flex gap-3">
            {digits.map((d, i) => (
              <input
                key={i}
                ref={(el) => { inputs.current[i] = el }}
                type="password"
                inputMode="numeric"
                maxLength={1}
                value={d}
                onChange={(e) => handleChange(i, e.target.value)}
                onKeyDown={(e) => handleKeyDown(i, e)}
                disabled={loading}
                className="size-14 rounded-[14px] border border-[#dfe2db] bg-[#f7f8f4] text-center text-[24px] font-bold text-[#11160f] outline-none transition-all focus:border-[#9fe870] focus:bg-white focus:ring-2 focus:ring-[#9fe870]/30 disabled:opacity-50"
              />
            ))}
          </div>

          {error && (
            <p className="text-[13px] font-medium text-[#be123c]">{error}</p>
          )}

          <div className="flex w-full gap-3">
            <button
              type="button"
              onClick={onCancel}
              disabled={loading}
              className="flex-1 h-11 rounded-full border border-[#dfe2db] text-[14px] font-semibold text-[#4f554d] transition-colors hover:bg-[#f7f8f4] disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={digits.join('').length < 4 || loading}
              className="flex-1 h-11 rounded-full bg-[#9fe870] text-[14px] font-semibold text-[#163300] transition-colors hover:bg-[#8fdd5f] disabled:opacity-50"
            >
              {loading ? 'Verifying…' : 'Confirm'}
            </button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
