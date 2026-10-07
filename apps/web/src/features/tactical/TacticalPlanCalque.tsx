/**
 * TacticalPlanCalque — ce qui se pose SUR le fond du plan et bouge avec le cadrage : le `<canvas>`
 * des zones nommées et de la chaleur, ses gestes (clic, glisser, molette) et le nom de la zone
 * choisie. Le cadrage vient de `useCadrageDuPlan`.
 *
 * ─── LE CADRAGE EST CELUI DU REJEU 2D, TEL QUEL ────────────────────────────────────────────────
 *
 * `useReplayZoom` (paliers 1x à 3x, centre reborné), `useReplayDrag` (glisser dès qu'on a grossi),
 * `useReplayWheelZoom` (molette ancrée sur le curseur) et, posée par la carte du plan,
 * `ReplayZoomControl` (− / + / croix / retour, dans l'angle bas-droit) : les mêmes que le rejeu et
 * qu'« Occupation du terrain » de la vue match. Ils ne connaissent qu'une scène, une fenêtre et une
 * toile. La scène est le cadre du fond ; la fenêtre visible en sort (`visibleBounds`) et TOUT s'y
 * projette — la chaleur, les zones, la sélection, l'étiquette, et l'image du fond (`cadrageDuFond`).
 *
 * UN GLISSER N'EST PAS UN CLIC : le relâcher d'un glisser déclenche aussi `click` ; un clic dont le
 * pointeur a bougé de plus de `SEUIL_GLISSER_PX` depuis l'appui ne choisit donc aucune zone.
 *
 * LA CHALEUR SE PEINT LISSÉE (`chaleurLissee.ts`) : des zones de chaleur, plus des carrés. Le repère
 * de la zone choisie est un anneau autour de sa cellule (la cellule reste l'adresse du détail).
 */
import { useEffect, useMemo, useRef, useState, type PointerEvent, type RefObject } from 'react'

import { useReplayDrag } from '@/features/match-replay/hooks/useReplayDrag'
import { useReplayWheelZoom } from '@/features/match-replay/hooks/useReplayWheelZoom'
import type { ReplayZoom } from '@/features/match-replay/hooks/useReplayZoom'
import type { ReplayBounds } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import type { TacticalGrid } from '@/lib/replay/heatPaint'

import { lisserLaGrille } from './chaleurLissee'
import { peindreLePlan } from './planPaint'
import { positionEtiquette } from './zone.logic'
import { pointDuClic, rectSelection, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'

/** Au-delà de ce déplacement du pointeur entre l'appui et le relâcher, le geste est un glisser. */
const SEUIL_GLISSER_PX = 4
/** Rayon de l'anneau de la zone choisie, en côtés de cellule : il entoure la cellule entière. */
const ANNEAU_RAYON_CELLULE = 0.75

/** useTailleDuCanvas — la taille affichée du canvas, suivie : le glisser et la molette convertissent des pixels en mètres. */
function useTailleDuCanvas(ref: RefObject<HTMLCanvasElement | null>) {
  const [taille, setTaille] = useState({ width: 0, height: 0 })
  useEffect(() => {
    const canvas = ref.current
    if (!canvas) return
    const lire = () => setTaille({ width: canvas.clientWidth, height: canvas.clientHeight })
    lire()
    // Le patron du dépôt : l'observateur est optionnel (jsdom ne le fournit pas).
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(lire)
    ro.observe(canvas)
    return () => ro.disconnect()
  }, [ref])
  return taille
}

export interface CalqueDuPlanProps {
  /** La grille servie, réindexée sur le repère ; `null` = rien à peindre. */
  grid: TacticalGrid | null
  repere: RepereTactique | null
  fenetre: ReplayBounds
  zoom: ReplayZoom
  ramp: readonly string[]
  zones: readonly CalloutZoneReady[]
  locale: Locale
  selected: { col: number; row: number } | null
  /** Le point MONDE sous un clic (pas un glisser) ; la vue décide de la cellule qu'il retient. */
  onPointSelect: (point: { x: number; y: number }) => void
  estompe: string
}

/** CalqueDuPlan — le `<canvas>` des zones nommées et de la chaleur lissée, posé sur le fond, et ses gestes. */
export function CalqueDuPlan({
  grid,
  repere,
  fenetre,
  zoom,
  ramp,
  zones,
  locale,
  selected,
  onPointSelect,
  estompe,
}: CalqueDuPlanProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const taille = useTailleDuCanvas(canvasRef)
  const view = useMemo(() => ({ bounds: fenetre, width: taille.width, height: taille.height, pad: 0 }), [fenetre, taille])
  const drag = useReplayDrag(zoom, view)
  useReplayWheelZoom(canvasRef, zoom, view)
  const lissee = useMemo(() => (grid ? lisserLaGrille(grid) : null), [grid])
  const appui = useRef<{ x: number; y: number } | null>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    if (!lissee || !repere) return
    // L'ENCRE DU TEXTE (`text-foreground` lue par `getComputedStyle`) : contours des zones et repère
    // de la sélection ne disent aucune SIGNIFICATION qu'un jeton sémantique porterait — « ce lieu », « ici ».
    const encre = getComputedStyle(canvas).color
    peindreLePlan(ctx, repere, width, { grid: lissee, ramp, zones, locale, encre }, fenetre)
    // L'ANNEAU DE LA ZONE CHOISIE, par-dessus les calques : le retour immédiat au clic.
    const cadre = rectSelection(selected, repere, width, fenetre)
    if (!cadre) return
    ctx.strokeStyle = encre
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.arc(cadre.x + cadre.size / 2, cadre.y + cadre.size / 2, cadre.size * ANNEAU_RAYON_CELLULE, 0, 2 * Math.PI)
    ctx.stroke()
  }, [lissee, ramp, repere, fenetre, selected, zones, locale, taille])

  function handlePointerDown(event: PointerEvent<HTMLCanvasElement>) {
    appui.current = { x: event.clientX, y: event.clientY }
    drag.onPointerDown(event)
  }

  function handleClick(event: React.MouseEvent<HTMLCanvasElement>) {
    const depart = appui.current
    appui.current = null
    if (depart && Math.hypot(event.clientX - depart.x, event.clientY - depart.y) > SEUIL_GLISSER_PX) return
    const canvas = canvasRef.current
    if (!canvas || !repere) return
    const rect = canvas.getBoundingClientRect()
    // L'ADRESSE RETENUE EST CELLE DU SERVEUR (ancrée sur l'origine du monde) : c'est elle que
    // `/tactical/{map}/cellule` attend ; la vue la tire du point monde (`choixDuClic`).
    const point = pointDuClic(
      event.clientX - rect.left,
      event.clientY - rect.top,
      { width: canvas.clientWidth, height: canvas.clientHeight },
      fenetre,
    )
    if (point) onPointSelect(point)
  }

  let curseur = ' cursor-crosshair'
  if (zoom.canPan) curseur = drag.dragging ? ' cursor-grabbing' : ' cursor-grab'
  return (
    <canvas
      ref={canvasRef}
      className={`absolute inset-0 h-full w-full touch-none text-foreground transition-opacity${curseur}${estompe}`}
      onPointerDown={handlePointerDown}
      onPointerMove={drag.onPointerMove}
      onPointerUp={drag.onPointerUp}
      onPointerCancel={drag.onPointerUp}
      onClick={handleClick}
      data-testid="tactical-plan-canvas"
    />
  )
}

/** EtiquetteDeZone — le nom de la zone choisie, posé à côté de sa cellule, du côté où il tient. */
export function EtiquetteDeZone({
  selected,
  repere,
  fenetre,
  texte,
  estompe,
}: {
  selected: { col: number; row: number }
  repere: RepereTactique
  fenetre: FenetreDuPlan
  texte: string
  estompe: string
}) {
  const position = positionEtiquette(selected, repere, fenetre)
  if (!position) return null
  return (
    <span
      className={`pointer-events-none absolute z-[2] whitespace-nowrap rounded border border-border bg-card px-[5px] py-px text-[11px] leading-[14px] text-foreground${estompe}`}
      style={position}
      data-testid="tactical-zone-label"
    >
      {texte}
    </span>
  )
}
