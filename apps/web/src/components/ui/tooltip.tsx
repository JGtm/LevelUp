import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { isKeyboardFocus } from '@/lib/dom/focusVisible'
import { TOOLTIP_OWNER_ATTRIBUTE } from './titleTooltip.logic'
import { placeTooltip, TooltipPanel, type TooltipPlacement } from './tooltip-panel'

interface TooltipProps {
  content: ReactNode
  children: ReactNode
  /** Classes additionnelles sur le wrapper (ex. `w-full` pour s'étirer dans un flex-col). */
  className?: string
  /** Élargit le panneau (`max-w-sm` au lieu de `max-w-56`) pour un contenu riche (légende). */
  wide?: boolean
}

/**
 * Tooltip — wrapper hover et focus clavier : un focus qui ne vient pas du clavier (clic, focus rendu par
 * programme à la fermeture d'un volet) n'ouvre pas l'infobulle.
 *
 * Le tooltip est rendu dans `document.body` via Portal + `position: fixed` pour
 * échapper aux parents `overflow:hidden` (ex. cellules de tableau scrollables).
 * Position calculée au montage du tooltip via `getBoundingClientRect` (`placeTooltip`).
 *
 * Le wrapper porte `TOOLTIP_OWNER_ATTRIBUTE` : un `title` natif posé sous lui ne double pas cette infobulle
 * (la couche des `title` le retire sans l'afficher).
 */
export function Tooltip({ content, children, className, wide }: TooltipProps) {
  const [open, setOpen] = useState(false)
  const anchorRef = useRef<HTMLDivElement | null>(null)
  const tooltipRef = useRef<HTMLDivElement | null>(null)
  const [pos, setPos] = useState<TooltipPlacement | null>(null)

  useLayoutEffect(() => {
    if (!open || !anchorRef.current || !tooltipRef.current) return
    const a = anchorRef.current.getBoundingClientRect()
    const t = tooltipRef.current.getBoundingClientRect()
    setPos(placeTooltip(a, t, { width: window.innerWidth, height: window.innerHeight }))
  }, [open, content])

  return (
    <div
      ref={anchorRef}
      {...{ [TOOLTIP_OWNER_ATTRIBUTE]: '' }}
      className={`relative inline-flex items-center${className ? ` ${className}` : ''}`}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={(e) => {
        if (isKeyboardFocus(e.target)) setOpen(true)
      }}
      onBlur={() => setOpen(false)}
    >
      {children}
      {open &&
        createPortal(
          <TooltipPanel innerRef={tooltipRef} placement={pos} wide={wide}>
            {content}
          </TooltipPanel>,
          document.body,
        )}
    </div>
  )
}
