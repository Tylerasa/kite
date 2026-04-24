import { ChevronLeft } from 'lucide-react'
import { cn } from '../lib/utils'

type BackButtonProps = {
  href?: string
  className?: string
  isMobile?: boolean
  label?: string
}

export default function BackButton({
  href,
  className,
  isMobile = false,
  label = 'Go back',
}: BackButtonProps) {
  function handleClick() {
    if (window.history.length > 1) {
      window.history.back()
      return
    }

    if (href) {
      window.location.href = href
    }
  }

  const classes = cn(
    'mb-4 flex size-12 items-center justify-center rounded-full bg-[#16330014] text-gray-700 transition-colors hover:bg-[#16330024]',
    href && !isMobile && 'hidden lg:flex',
    className,
  )

  if (href && !isMobile) {
    return (
      <a href={href} className={classes} aria-label={label}>
        <ChevronLeft className="size-5" />
      </a>
    )
  }

  return (
    <button type="button" onClick={handleClick} className={classes} aria-label={label}>
      <ChevronLeft className="size-5" />
    </button>
  )
}
