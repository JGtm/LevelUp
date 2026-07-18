import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

interface TooltipProps {
  content: ReactNode
  children: ReactNode
  /** Classes additionnelles sur le wrapper (ex. `w-full` pour s'étirer dans un flex-col). */
  className?: string
  /** Élargit le panneau (`max-w-sm` au lieu de `max-w-56`) pour un contenu riche (légende). */
  wide?: boolean
}

/**
 * Tooltip — wrapper hover.
 *
 * Le tooltip est rendu dans `document.body` via Portal + `position: fixed` pour
 * échapper aux parents `overflow:hidden` (ex. cellules de tableau scrollables).
 * Position calculée au montage du tooltip via `getBoundingClientRect`.
 */
export function Tooltip({ content, children, className, wide }: TooltipProps) {
  const [open, setOpen] = useState(false)
  const anchorRef = useRef<HTMLDivElement | null>(null)
  const tooltipRef = useRef<HTMLDivElement | null>(null)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)

  useLayoutEffect(() => {
    if (!open || !anchorRef.current || !tooltipRef.current) return
    const a = anchorRef.current.getBoundingClientRect()
    const t = tooltipRef.current.getBoundingClientRect()
    // Centré au-dessus de l'ancre, clamp dans la viewport (8px de marge).
    const margin = 8
    let left = a.left + a.width / 2 - t.width / 2
    left = Math.max(margin, Math.min(left, window.innerWidth - t.width - margin))
    // Au-dessus de l'ancre par défaut ; bascule EN DESSOUS si ça déborde le haut de
    // la viewport (contenu haut — ex. légende de badges — près du haut de page).
    let top = a.top - t.height - margin
    if (top < margin) top = a.bottom + margin
    setPos({ top, left })
  }, [open, content])

  return (
    <div
      ref={anchorRef}
      className={`relative inline-flex items-center${className ? ` ${className}` : ''}`}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
    >
      {children}
      {open &&
        createPortal(
          <div
            ref={tooltipRef}
            role="tooltip"
            style={{
              position: 'fixed',
              top: pos?.top ?? -9999,
              left: pos?.left ?? -9999,
              visibility: pos ? 'visible' : 'hidden',
            }}
            className={`z-[9999] w-max ${wide ? 'max-w-sm' : 'max-w-56'} whitespace-normal break-words rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs text-foreground shadow-lg pointer-events-none`}
          >
            {content}
          </div>,
          document.body,
        )}
    </div>
  )
}
