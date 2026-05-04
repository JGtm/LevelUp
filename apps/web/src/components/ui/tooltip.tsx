import { useState, type ReactNode } from 'react'

interface TooltipProps {
  content: ReactNode
  children: ReactNode
}

/** Wrapper tooltip hover — affiche `content` au-dessus de `children` au survol. */
export function Tooltip({ content, children }: TooltipProps) {
  const [open, setOpen] = useState(false)
  return (
    <div
      className="relative inline-flex items-center"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
    >
      {children}
      {open && (
        <div
          role="tooltip"
          className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50 w-max max-w-56 rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs text-foreground shadow-lg pointer-events-none"
        >
          {content}
          <div className="absolute top-full left-1/2 -translate-x-1/2 -mt-px border-4 border-transparent border-t-background" />
        </div>
      )}
    </div>
  )
}
