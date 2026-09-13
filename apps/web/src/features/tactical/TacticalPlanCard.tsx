/**
 * TacticalPlanCard — la carte « Plan » de la vue d'analyse (item 5.4).
 *
 * LE FOND (`<img>`, MÊME convention que `TacticalMapTile`) ET LE CALQUE DE CHALEUR
 * (`<canvas>`, `drawTacticalHeatmap` de `lib/replay/heatPaint.ts` — noyau partagé avec le
 * rejeu depuis le lot Q7, 2026-09-07) PARTAGENT EXACTEMENT LE MÊME
 * CADRE : le conteneur est mis à l'aspect-ratio DU MONDE (bornes du raster), jamais un
 * 16:9 fixe — sinon `object-cover` rognerait l'image sur un axe que le calque, lui, ne
 * rogne pas, et les deux se désaligneraient au clic.
 *
 * LE CADRE EST POSÉ MÊME QUAND RIEN N'EST PEINT (lot 3.2, 2026-09-09), et sa hauteur est
 * BORNÉE (`planFrameStyle`) : sans bornes exploitables il prend le rapport du fond, et
 * l'état vide se pose PAR-DESSUS. Auparavant, l'état vide remplaçait le cadre, et un
 * rapport très allongé rendait un canvas de 1 070 x 13 375 px.
 *
 * COULEURS : rampe d'INTENSITÉ (bleu → rouge → violet), MÊME token que la carte de
 * chaleur du rejeu (`useReplayHeatmap.ts`) — grandeur neutre (des morts, des kills, du
 * temps), pas une performance, donc jamais la rampe « à connotation » du dépôt.
 *
 * SAUF POUR « OÙ JE GAGNE », QUI EST UNE LECTURE SIGNÉE : l'écart victoires − défaites a
 * un zéro et deux côtés, et la maquette 034b1915 le peint sur une rampe DIVERGENTE
 * (rouge → neutre → vert). Servi par la rampe d'intensité, tout le côté défaite
 * disparaissait — une valeur ≤ 0 y est traitée comme « jamais atteinte ».
 *
 * LÉGENDE ET PIED sont ceux de la maquette : la rampe avec ses deux bornes et l'unité, la
 * phrase d'échelle, puis les dénominateurs (« N matchs retenus sur M filtrés · source »).
 */
import { useEffect, useMemo, useRef } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { BornesMonde, EchelleTactique } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import {
  drawTacticalHeatmap,
  heatRamp,
  heatRampDivergent,
  type TacticalGrid,
} from '@/lib/replay/heatPaint'
import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundUrl } from './queries'
import {
  cellFromClick,
  planCanvasView,
  planEmptyReason,
  planEmptyText,
  planFrameStyle,
  planLegend,
  planSelectionRect,
  sourceForQuestion,
  statusMessages,
  TACTICAL_CELL_FLOOR,
  unitForQuestion,
  type TacticalQuestion,
} from './tacticalView.logic'

export interface TacticalPlanCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  mapId: string
  question: TacticalQuestion
  grid: TacticalGrid | null
  bornes: BornesMonde
  /** L'échelle publiée par la lecture — elle décide de la rampe ET des bornes affichées. */
  echelle: EchelleTactique
  pasM: number
  /** Les deux dénominateurs publiés par la lecture — ils DISENT pourquoi un plan est vide
   *  (périmètre vide, aucune mesure, ou densité insuffisante). */
  matchsFiltres: number
  matchsRetenus: number
  matchsEnAttente: number
  matchsNonCuisables: number
  /** La cellule choisie, encadrée sur le plan — `null` tant que rien n'est cliqué. */
  selected: { col: number; row: number } | null
  onCellSelect: (col: number, row: number) => void
}

export function TacticalPlanCard({
  t,
  locale,
  playerSlug,
  mapId,
  question,
  grid,
  bornes,
  echelle,
  pasM,
  matchsFiltres,
  matchsRetenus,
  matchsEnAttente,
  matchsNonCuisables,
  selected,
  onCellSelect,
}: TacticalPlanCardProps) {
  const fond = useTacticalMapBackgroundUrl(playerSlug, mapId)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  // Rampe précalculée PAR THÈME, résolue une fois par changement de palette d'accessibilité
  // (même patron que `useReplayHeatmap.ts`) — jamais recalculée par cellule.
  const paletteVersion = useColorPaletteVersion()
  const numFmt = useMemo(
    () => new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 }),
    [locale],
  )
  const unite = unitForQuestion(t, question)
  const legende = planLegend(echelle, unite, (n) => numFmt.format(n))
  const modeRampe = legende.mode
  const ramp = useMemo(() => {
    void paletteVersion
    const tokens = heatmapRampTokens(modeRampe).map(resolveToken)
    return modeRampe === 'divergent' ? heatRampDivergent(tokens) : heatRamp(tokens)
  }, [paletteVersion, modeRampe])

  const bornesValides = bornes.valide && bornes.max_x > bornes.min_x && bornes.max_y > bornes.min_y
  // LE CADRE EST TOUJOURS POSÉ, plein ou vide : rapport des bornes quand elles disent
  // quelque chose, rapport du fond sinon, hauteur bornée dans les deux cas
  // (cf. `planFrameStyle` — c'est ce qui ferme le canvas de 13 375 px).
  const cadre = planFrameStyle(bornes)

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
    const vue = planCanvasView(bornes, width)
    if (!vue) return
    drawTacticalHeatmap(ctx, grid, vue, { ramp, k: 1 })
    // LE CADRE DE LA CELLULE CHOISIE, par-dessus le calque (maquette 034b1915,
    // `strokeRect`) : c'est le SEUL retour immédiat au clic — le panneau de détail est
    // plus bas et met plusieurs secondes à se remplir.
    const cadreSel = planSelectionRect(selected, bornes, pasM, width)
    if (!cadreSel) return
    // L'ENCRE DU CADRE EST CELLE DU TEXTE DE L'APP (`text-foreground` porté par le canvas,
    // lu par `getComputedStyle`) : un jeton sémantique de couleur dirait une SIGNIFICATION
    // (succès, alerte, camp) que cette sélection n'a pas — elle dit seulement « ici ».
    ctx.strokeStyle = getComputedStyle(canvas).color
    ctx.lineWidth = 2
    ctx.strokeRect(cadreSel.x - 1, cadreSel.y - 1, cadreSel.size + 2, cadreSel.size + 2)
  }, [grid, ramp, bornes, bornesValides, selected, pasM])

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
  const source = sourceForQuestion(t, question)
  // POURQUOI le plan est vide, pas seulement QU'IL l'est : les trois causes n'appellent
  // pas la même action de l'utilisateur (cf. `planEmptyReason`).
  const raisonVide = planEmptyReason(grid?.filled ?? 0, matchsRetenus, matchsFiltres)
  const aucuneCellule = raisonVide !== null

  return (
    <SectionCard
      title={t.planTitle}
      label={t.planTitle}
      footer={
        !aucuneCellule ? (
          <div className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
            <p data-testid="tactical-plan-scale-note">
              {modeRampe === 'divergent'
                ? t.planScaleDivergent(TACTICAL_CELL_FLOOR)
                : t.planScaleQuantile}
            </p>
            <p className="mt-1" data-testid="tactical-plan-retained">
              {t.planFooterRetained(matchsRetenus, matchsFiltres, source)}
            </p>
            <p className="mt-1" data-testid="tactical-plan-grid-step">
              {t.footerGrid(pasM)} · {t.footerFloor(TACTICAL_CELL_FLOOR)}
            </p>
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
        <div
          className="relative w-full overflow-hidden rounded-md bg-muted"
          style={cadre}
          data-testid="tactical-plan-frame"
        >
          {fond && <img src={fond} alt="" aria-hidden className="h-full w-full object-cover" />}
          {raisonVide ? (
            // L'ÉTAT VIDE SE POSE SUR LE CADRE, il ne le remplace pas : la carte garde la
            // taille qu'elle aura une fois remplie, et le message dit ce qui manque
            // au-dessus du fond plutôt qu'à la place de tout.
            <div className="absolute inset-0 flex items-center justify-center p-3">
              <EmptyStateNotice {...planEmptyText(t, raisonVide, matchsRetenus, pasM)} />
            </div>
          ) : (
            <canvas
              ref={canvasRef}
              className="absolute inset-0 h-full w-full cursor-crosshair text-foreground"
              onClick={handleClick}
              data-testid="tactical-plan-canvas"
            />
          )}
        </div>
        {!aucuneCellule && (
          <div
            className="mt-2 flex flex-wrap items-center gap-2"
            data-testid="tactical-plan-legend"
          >
            <span className="font-mono text-2xs tabular-nums text-muted-foreground">
              {legende.lo}
            </span>
            <span
              role="img"
              aria-label={t.planLegendLabel(legende.lo, legende.hi)}
              className="h-2 min-w-32 flex-1 rounded-full"
              style={{ background: rampeCss(modeRampe) }}
            />
            <span className="font-mono text-2xs tabular-nums text-muted-foreground">
              {legende.hi}
            </span>
          </div>
        )}
      </div>
    </SectionCard>
  )
}

/**
 * rampeCss — le dégradé CSS de la rampe de légende, construit sur les MÊMES jetons que le
 * calque peint (`heatmapRampTokens`). Une légende dont les couleurs viendraient d'ailleurs
 * décrirait une autre carte que celle qu'on regarde.
 */
function rampeCss(mode: 'intensity' | 'divergent'): string {
  const arrets = heatmapRampTokens(mode).map((token) => tokenCssVar(token))
  return `linear-gradient(90deg, ${arrets.join(', ')})`
}
