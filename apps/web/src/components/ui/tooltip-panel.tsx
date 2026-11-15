import type { ReactNode, Ref } from 'react'

/** Marge entre l'ancre et l'infobulle, et entre l'infobulle et le bord de la fenêtre. */
const MARGIN = 8

/** Rectangle d'ancre : le sous-ensemble de `DOMRect` dont le placement a besoin. */
export interface AnchorRect {
  left: number
  top: number
  width: number
  bottom: number
}

export interface TooltipSize {
  width: number
  height: number
}

export interface TooltipPlacement {
  top: number
  left: number
}

/**
 * Place une infobulle centrée au-dessus de son ancre, bornée à la fenêtre (marge de 8 px). Au-dessus par
 * défaut ; bascule EN DESSOUS si elle déborde le haut de la fenêtre (contenu haut près du haut de page),
 * puis reste dans la fenêtre si le dessous déborde aussi. Fonction pure : partagée par `Tooltip` et par la
 * couche des `title` natifs.
 */
export function placeTooltip(anchor: AnchorRect, tip: TooltipSize, viewport: TooltipSize): TooltipPlacement {
  let left = anchor.left + anchor.width / 2 - tip.width / 2
  left = Math.max(MARGIN, Math.min(left, viewport.width - tip.width - MARGIN))
  let top = anchor.top - tip.height - MARGIN
  if (top < MARGIN) top = anchor.bottom + MARGIN
  top = Math.max(MARGIN, Math.min(top, viewport.height - tip.height - MARGIN))
  return { top, left }
}

interface TooltipPanelProps {
  children: ReactNode
  /** Position fixe calculée par `placeTooltip` ; `null` tant que la mesure n'est pas faite (panneau caché). */
  placement: TooltipPlacement | null
  innerRef?: Ref<HTMLDivElement>
  id?: string
  /** Élargit le panneau (`max-w-sm` au lieu de `max-w-56`) pour un contenu riche (légende). */
  wide?: boolean
  /** Respecte les retours à la ligne du texte (un `title` natif peut en porter). */
  keepLineBreaks?: boolean
}

/**
 * Panneau d'infobulle — l'habillage unique de `Tooltip` et de la couche des `title` natifs. À rendre dans un
 * portail sur `document.body` : `position: fixed` échappe aux parents `overflow:hidden`.
 */
export function TooltipPanel({ children, placement, innerRef, id, wide, keepLineBreaks }: TooltipPanelProps) {
  return (
    <div
      ref={innerRef}
      id={id}
      role="tooltip"
      style={{
        position: 'fixed',
        top: placement?.top ?? -9999,
        left: placement?.left ?? -9999,
        visibility: placement ? 'visible' : 'hidden',
      }}
      className={`z-[9999] w-max ${wide ? 'max-w-sm' : 'max-w-56'} ${keepLineBreaks ? 'whitespace-pre-line' : 'whitespace-normal'} break-words rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs text-foreground shadow-lg pointer-events-none`}
    >
      {children}
    </div>
  )
}
