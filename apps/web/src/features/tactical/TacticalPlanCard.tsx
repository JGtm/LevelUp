/**
 * TacticalPlanCard — la carte « Plan » de la vue d'analyse (item 5.4).
 *
 * LE FOND (`<img>`, MÊME convention que `TacticalMapTile`) ET LE CALQUE DE CHALEUR
 * (`<canvas>`, `drawTacticalHeatmap` de `lib/replay/heatPaint.ts` — noyau partagé avec le
 * rejeu depuis le lot Q7, 2026-09-07) PARTAGENT EXACTEMENT LE MÊME
 * CADRE : le conteneur est mis à l'aspect-ratio DU MONDE (bornes du raster), jamais un
 * 16:9 fixe — sinon `object-cover` rognerait l'image sur un axe que le calque, lui, ne
 * rogne pas, et les deux se désaligneraient au clic. Sans bornes valides, rien n'est
 * peint (état vide : `EmptyStateNotice`).
 *
 * COULEURS : rampe d'INTENSITÉ (bleu → rouge → violet), MÊME token que la carte de
 * chaleur du rejeu (`useReplayHeatmap.ts`) — grandeur neutre (des morts, des kills, du
 * temps), pas une performance, donc jamais la rampe « à connotation » du dépôt.
 */
import { useEffect, useMemo, useRef } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { BornesMonde } from '@/lib/api/types'

import { drawTacticalHeatmap, heatRamp, type TacticalGrid } from '@/lib/replay/heatPaint'
import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundUrl } from './queries'
import {
  cellFromClick,
  sourceForQuestion,
  statusMessages,
  TACTICAL_CELL_FLOOR,
  unitForQuestion,
  type TacticalQuestion,
} from './tacticalView.logic'

export interface TacticalPlanCardProps {
  t: TacticalText
  playerSlug: string
  mapId: string
  question: TacticalQuestion
  grid: TacticalGrid | null
  bornes: BornesMonde
  pasM: number
  matchsEnAttente: number
  matchsNonCuisables: number
  onCellSelect: (col: number, row: number) => void
}

export function TacticalPlanCard({
  t,
  playerSlug,
  mapId,
  question,
  grid,
  bornes,
  pasM,
  matchsEnAttente,
  matchsNonCuisables,
  onCellSelect,
}: TacticalPlanCardProps) {
  const fond = useTacticalMapBackgroundUrl(playerSlug, mapId)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  // Rampe précalculée PAR THÈME, résolue une fois par changement de palette d'accessibilité
  // (même patron que `useReplayHeatmap.ts`) — jamais recalculée par cellule.
  const paletteVersion = useColorPaletteVersion()
  const ramp = useMemo(() => {
    void paletteVersion
    return heatRamp(heatmapRampTokens('intensity').map(resolveToken))
  }, [paletteVersion])

  const bornesValides = bornes.valide && bornes.max_x > bornes.min_x && bornes.max_y > bornes.min_y
  const aspect = bornesValides
    ? (bornes.max_x - bornes.min_x) / (bornes.max_y - bornes.min_y)
    : 16 / 9

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !grid || !bornesValides) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    const scale = width / (bornes.max_x - bornes.min_x)
    drawTacticalHeatmap(ctx, grid, { topLeftWorld: { x: 0, y: 0 }, scale }, { ramp, k: 1 })
  }, [grid, ramp, bornes, bornesValides])

  function handleClick(event: React.MouseEvent<HTMLCanvasElement>) {
    const canvas = canvasRef.current
    if (!canvas || !bornesValides) return
    const rect = canvas.getBoundingClientRect()
    const cellule = cellFromClick(
      event.clientX - rect.left,
      event.clientY - rect.top,
      canvas.clientWidth,
      canvas.clientHeight,
      bornes,
      pasM,
    )
    if (cellule) onCellSelect(cellule.col, cellule.row)
  }

  const messages = statusMessages(t, matchsEnAttente, matchsNonCuisables)
  const unite = unitForQuestion(t, question)
  const source = sourceForQuestion(t, question)
  const aucuneCellule = !grid || grid.filled === 0

  return (
    <SectionCard
      title={t.planTitle}
      label={t.planTitle}
      footer={
        !aucuneCellule ? (
          <div className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
            <p>{unite}</p>
            <p className="mt-1">{t.footerFloor(TACTICAL_CELL_FLOOR)}</p>
            <p className="mt-1">{source}</p>
          </div>
        ) : undefined
      }
    >
      <div className="p-3">
        {messages.length > 0 && (
          <div role="status" className="mb-2 flex flex-col gap-1 text-xs text-warning">
            {messages.map((msg) => (
              <p key={msg}>{msg}</p>
            ))}
          </div>
        )}
        {aucuneCellule ? (
          <EmptyStateNotice title={t.planEmptyTitle} description={t.planEmptyDescription} />
        ) : (
          <div
            className="relative w-full overflow-hidden rounded-md bg-muted"
            style={{ aspectRatio: aspect }}
          >
            {fond && (
              <img src={fond} alt="" aria-hidden className="h-full w-full object-cover" />
            )}
            <canvas
              ref={canvasRef}
              className="absolute inset-0 h-full w-full cursor-crosshair"
              onClick={handleClick}
              data-testid="tactical-plan-canvas"
            />
          </div>
        )}
      </div>
    </SectionCard>
  )
}
