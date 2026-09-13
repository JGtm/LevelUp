/**
 * TacticalPlanCard — la carte « Plan » de la vue d'analyse (item 5.4).
 *
 * ─── LA PROJECTION A ÉTÉ REFAITE LE 2026-09-13 ───────────────────────────────
 *
 * Constat utilisateur : sur Illusion, le calque était quasi vide et ses quelques cellules
 * tombaient À CÔTÉ du bâtiment. Trois causes empilées, toutes mesurées (cf. l'en-tête de
 * `tacticalView.logic.ts`) : le peintre jetait les cellules d'index NÉGATIF (11 dessinées
 * sur 54 servies), le cadre était la boîte englobante des MORTS et non celle du FOND, et
 * l'axe Y n'était pas inversé.
 *
 * Le plan projette désormais sur LE CADRE DU FOND (`useTacticalMapBackgroundFrame`, le
 * calage publié avec chaque image), exactement comme « Où ça se joue » de la vue Match et
 * comme le rejeu 2D. Une carte sans fond figé retombe sur ses propres bornes : elle
 * n'affiche aucune image, le calque s'y lit seul.
 *
 * LE FOND (`<img>`) ET LE CALQUE DE CHALEUR (`<canvas>`, `drawTacticalHeatmap` de
 * `lib/replay/heatPaint.ts` — noyau partagé avec le rejeu depuis le lot Q7, 2026-09-07)
 * PARTAGENT EXACTEMENT LE MÊME CADRE MONDE : celui du CALAGE du fond. Le conteneur prend son
 * rapport, donc `object-cover` n'y rogne rien, et le calque s'y pose au mètre près.
 *
 * LE CADRE EST POSÉ MÊME QUAND RIEN N'EST PEINT (lot 3.2, 2026-09-09), et sa hauteur est
 * BORNÉE (`PLAN_HAUTEUR_MAX_PX`) : sans repère exploitable il prend le rapport par défaut,
 * et l'état vide se pose PAR-DESSUS. Auparavant, l'état vide remplaçait le cadre, et un
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
import type { BornesMonde, CelluleTactique, EchelleTactique } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import { drawTacticalHeatmap, heatRamp, heatRampDivergent } from '@/lib/replay/heatPaint'
import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundFrame, useTacticalMapBackgroundUrl } from './queries'
import {
  celluleDuClic,
  grilleDuPlan,
  planEmptyReason,
  planEmptyText,
  planLegend,
  rectSelection,
  repereAspect,
  repereDuPlan,
  sourceForQuestion,
  statusMessages,
  TACTICAL_CELL_FLOOR,
  unitForQuestion,
  vueDuPlan,
  PLAN_ASPECT_DEFAUT,
  PLAN_HAUTEUR_MAX_PX,
  type TacticalQuestion,
} from './tacticalView.logic'

export interface TacticalPlanCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  mapId: string
  question: TacticalQuestion
  /** Les cellules SERVEUR, adressées sur l'origine du monde — la projection est faite ici,
   *  parce qu'elle dépend du cadre du fond, que seule cette carte connaît. */
  cellules: readonly CelluleTactique[]
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
  cellules,
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
  // LE CADRE DU FOND EST LE REPÈRE, quand il existe : c'est lui, et lui seul, qui fait
  // coïncider le calque et l'image.
  const cadreFond = useTacticalMapBackgroundFrame(playerSlug, mapId)
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

  // LE REPÈRE : le cadre du fond quand la carte en a un, la boîte englobante des cellules
  // sinon. Le cadre de la carte prend son rapport, hauteur bornée (c'est ce qui ferme le
  // canvas de 13 375 px constaté sur Illusion en 2026-09).
  const repere = repereDuPlan(cadreFond, bornes, pasM)
  const aspect = repere ? repereAspect(repere) : PLAN_ASPECT_DEFAUT
  const cadre = { aspectRatio: aspect, maxWidth: `${aspect * PLAN_HAUTEUR_MAX_PX}px` }
  const grid = repere ? grilleDuPlan(cellules, repere, echelle) : null

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !grid || !repere) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    const vue = vueDuPlan(repere, width)
    if (!vue) return
    drawTacticalHeatmap(ctx, grid, vue, { ramp, k: 1 })
    // LE CADRE DE LA CELLULE CHOISIE, par-dessus le calque (maquette 034b1915,
    // `strokeRect`) : c'est le SEUL retour immédiat au clic — le panneau de détail est
    // plus bas et met plusieurs secondes à se remplir.
    const cadreSel = rectSelection(selected, repere, width)
    if (!cadreSel) return
    // L'ENCRE DU CADRE EST CELLE DU TEXTE DE L'APP (`text-foreground` porté par le canvas,
    // lu par `getComputedStyle`) : un jeton sémantique de couleur dirait une SIGNIFICATION
    // (succès, alerte, camp) que cette sélection n'a pas — elle dit seulement « ici ».
    ctx.strokeStyle = getComputedStyle(canvas).color
    ctx.lineWidth = 2
    ctx.strokeRect(cadreSel.x - 1, cadreSel.y - 1, cadreSel.size + 2, cadreSel.size + 2)
  }, [grid, ramp, repere, selected])

  function handleClick(event: React.MouseEvent<HTMLCanvasElement>) {
    const canvas = canvasRef.current
    if (!canvas || !repere) return
    const rect = canvas.getBoundingClientRect()
    // L'ADRESSE RENDUE EST CELLE DU SERVEUR (ancrée sur l'origine du monde) : c'est elle que
    // `/tactical/{map}/cellule` attend, et celle que portent les cellules de la réponse.
    const cellule = celluleDuClic(
      event.clientX - rect.left,
      event.clientY - rect.top,
      { width: canvas.clientWidth, height: canvas.clientHeight },
      repere,
    )
    if (cellule) onCellSelect(cellule.col, cellule.row)
  }

  const messages = statusMessages(t, matchsEnAttente, matchsNonCuisables)
  const source = sourceForQuestion(t, question)
  // POURQUOI le plan est vide, pas seulement QU'IL l'est : les trois causes n'appellent
  // pas la même action de l'utilisateur (cf. `planEmptyReason`).
  const raisonVide = planEmptyReason(grid?.filled ?? 0, matchsRetenus, matchsFiltres)
  // Les cellules SERVIES mais tombées hors du cadre du fond : dites, jamais avalées — un
  // plan amputé ressemblerait sinon à un plan complet.
  const horsCadre = cellules.length - (grid?.filled ?? 0)
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
            {horsCadre > 0 && (
              <p className="mt-1" data-testid="tactical-plan-off-frame">
                {t.footerOffFrame(horsCadre, cellules.length)}
              </p>
            )}
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
