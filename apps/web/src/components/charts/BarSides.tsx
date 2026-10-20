/**
 * BarSides — LA VALEUR QUI NE TIENT PAS DANS SON SEGMENT SE LIT À CÔTÉ DE LA BARRE, sur la même
 * ligne, centrée sur la hauteur de SA barre : pastille de la couleur du segment + valeur
 * (décision du 2026-10-15 ; jamais au-dessus de la barre, jamais seulement en infobulle, règle S3).
 *
 *  - barre camp contre camp : à l'extérieur, du côté de son camp (`BarWithSides`, gouttière à
 *    gauche et à droite) ;
 *  - barre empilée : juste après son extrémité (`SideValue` posée par l'appelant).
 *
 * LA PLACE EST RÉSERVÉE d'avance, par carte (`sideSlotPx`, calculée sur les TEXTES qui peuvent
 * s'y écrire et non sur ce qui déborde, donc sans boucle avec la mesure `segmentLabelFit`) :
 * toutes les barres d'une carte ont la même largeur, qu'une valeur déborde ou non. Les axes
 * gradués de la carte reprennent les mêmes marges (`TrackAxis`, `inset`).
 */
import type { ReactNode } from 'react'

/** Largeur moyenne d'un glyphe de 12 px (chiffres tabulaires). */
const GLYPH_PX = 6.6
/** Pastille (9 px) et son espace (5 px). */
const DOT_PX = 14
/** Respiration entre la barre et la valeur. */
export const SIDE_GAP_PX = 6
/** Plafond : une valeur plus longue déborde vers l'extérieur plutôt que d'écraser la barre. */
const SIDE_MAX_PX = 112

/** La largeur réservée pour des textes (0 : rien à réserver). */
// eslint-disable-next-line react-refresh/only-export-components
export function sideSlotPx(texts: readonly string[]): number {
  const longest = texts.reduce((m, t) => Math.max(m, t.length), 0)
  return longest === 0 ? 0 : Math.min(SIDE_MAX_PX, Math.ceil(longest * GLYPH_PX) + DOT_PX + SIDE_GAP_PX)
}

/** Une valeur placée à côté d'une barre : pastille de la couleur du segment, puis la valeur. */
export function SideValue({ color, testId, children }: { color: string; testId?: string; children: ReactNode }) {
  return (
    <span className="inline-flex items-center whitespace-nowrap text-xs leading-none tabular-nums text-foreground" data-testid={testId}>
      <span className="mr-[5px] inline-block h-[9px] w-[9px] shrink-0 rounded-[2px]" style={{ backgroundColor: color }} aria-hidden />
      {children}
    </span>
  )
}

/**
 * Les valeurs d'une barre empilée dont la longueur varie : posées juste après son extrémité
 * (`endPct`, en % de la barre), sur sa ligne. À placer dans un conteneur `relative` dont le
 * `paddingRight` vaut `slotPx` : la barre occupe alors la largeur utile, et la place à droite de
 * son bout le plus long est réservée.
 */
export function AfterBar({ endPct, slotPx, children }: { endPct: number; slotPx: number; children: ReactNode }) {
  return (
    <div
      className="absolute inset-y-0 flex items-center gap-2 whitespace-nowrap"
      style={{ left: `calc((100% - ${slotPx}px) * ${endPct / 100} + ${SIDE_GAP_PX}px)` }}
      data-side="after"
    >
      {children}
    </div>
  )
}

/**
 * La barre entre deux gouttières : à gauche la valeur du camp de gauche (collée à la barre), à
 * droite celle du camp de droite. La rangée prend la hauteur de la barre : les valeurs s'y
 * centrent sur elle. `leftPx` / `rightPx` à 0 : pas de gouttière de ce côté.
 */
export function BarWithSides({
  leftPx,
  rightPx,
  left,
  right,
  children,
}: {
  leftPx: number
  rightPx: number
  left?: ReactNode
  right?: ReactNode
  children: ReactNode
}) {
  return (
    <div className="flex items-center">
      {leftPx > 0 && (
        <div className="flex shrink-0 items-center justify-end" style={{ width: leftPx, paddingRight: SIDE_GAP_PX }} data-side="left">
          {left}
        </div>
      )}
      <div className="relative min-w-0 flex-1">{children}</div>
      {rightPx > 0 && (
        <div className="flex shrink-0 items-center justify-start" style={{ width: rightPx, paddingLeft: SIDE_GAP_PX }} data-side="right">
          {right}
        </div>
      )}
    </div>
  )
}
