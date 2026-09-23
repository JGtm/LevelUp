/**
 * TacticalPlanCard — la carte « Plan » de la vue d'analyse (item 5.4).
 *
 * ─── LE FOND NE BOUGE PLUS (retours rejeu L2, 2026-09-23) ───────────────────────
 *
 * Constat utilisateur : changer de question ou cocher une session faisait CLIGNOTER le fond
 * de carte. La carte n'était rendue qu'une fois la lecture servie : chaque nouvelle clé de
 * cache la démontait, fond compris, puis la reconstruisait. Désormais la carte est TOUJOURS
 * rendue, et le cadre + le fond vivent dans `TacticalPlanFond`, qui ne reçoit que la carte
 * et le calage. La lecture (`lecture`) est optionnelle et son ÉTAT (`etat`, cf.
 * `tacticalLecture.logic.ts`) décide de ce qui se pose PAR-DESSUS le fond :
 *   - `attente`   : l'indicateur de chargement, sur le cadre déjà posé ;
 *   - `relecture` : l'ancien calque, sa légende et son pied ESTOMPÉS sous « Mise à jour… »
 *                   (décision Q26) ;
 *   - `echec`     : rien (le message d'échec est rendu par la vue, au-dessus) ;
 *   - `pret`      : le calque.
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
 * calage publié avec chaque image), exactement comme « Occupation du terrain » de la vue Match et
 * comme le rejeu 2D. Une carte sans fond figé retombe sur ses propres bornes : elle
 * n'affiche aucune image, le calque s'y lit seul.
 *
 * LE FOND (`<img>`) ET LE CALQUE DE CHALEUR (`<canvas>`, `drawTacticalHeatmap` de
 * `lib/replay/heatPaint.ts` — noyau partagé avec le rejeu depuis le lot Q7, 2026-09-07)
 * PARTAGENT EXACTEMENT LE MÊME CADRE MONDE : celui du CALAGE du fond.
 *
 * LE CADRE EST POSÉ MÊME QUAND RIEN N'EST PEINT (lot 3.2, 2026-09-09), et sa hauteur est
 * BORNÉE (`PLAN_HAUTEUR_MAX_PX`) : sans repère exploitable il prend le rapport par défaut,
 * et l'état vide se pose PAR-DESSUS.
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
 * Unité et source viennent de la question À LAQUELLE LA LECTURE RÉPOND (`question`, cf.
 * `questionServie`), jamais de la question demandée.
 */
import { useEffect, useMemo, useRef } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { Spinner } from '@/components/ui/spinner'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalRaster } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import {
  drawTacticalHeatmap,
  heatRamp,
  heatRampDivergent,
  type TacticalGrid,
} from '@/lib/replay/heatPaint'
import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundFrame } from './queries'
import { aspectDuPlan, type TacticalEtatLecture } from './tacticalLecture.logic'
import { TacticalPlanFond } from './TacticalPlanFond'
import {
  celluleDuClic,
  grilleDuPlan,
  planEmptyReason,
  planEmptyText,
  planLegend,
  rectSelection,
  repereDuPlan,
  sourceForQuestion,
  statusMessages,
  TACTICAL_CELL_FLOOR,
  unitForQuestion,
  vueDuPlan,
  type RepereTactique,
  type TacticalQuestion,
} from './tacticalView.logic'

/** Classe d'estompage d'une réponse PRÉCÉDENTE pendant la relecture (décision Q26). */
const ESTOMPE = 'opacity-50'

export interface TacticalPlanCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  mapId: string
  /** La question À LAQUELLE `lecture` RÉPOND (`questionServie`) — elle décide de l'unité,
   *  de la source et de la rampe ; jamais la question demandée. */
  question: TacticalQuestion
  /** La lecture AFFICHÉE : la réponse courante, ou la précédente pendant une relecture.
   *  `undefined` = aucune donnée (premier chargement, échec) : le cadre et le fond restent. */
  lecture: TacticalRaster | undefined
  etat: TacticalEtatLecture
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
  lecture,
  etat,
  selected,
  onCellSelect,
}: TacticalPlanCardProps) {
  // LE CADRE DU FOND EST LE REPÈRE, quand il existe : c'est lui, et lui seul, qui fait
  // coïncider le calque et l'image. Sans lecture, il donne encore le rapport du cadre.
  const cadreFond = useTacticalMapBackgroundFrame(playerSlug, mapId)
  const repere = lecture ? repereDuPlan(cadreFond, lecture.bornes, lecture.pas_m) : null
  const peinture = usePeinture(t, locale, question, lecture, repere)
  const estompe = etat === 'relecture' ? ` ${ESTOMPE}` : ''

  // POURQUOI le plan est vide, pas seulement QU'IL l'est (cf. `planEmptyReason`).
  const raisonVide = lecture
    ? planEmptyReason(peinture.grid?.filled ?? 0, lecture.matchs_retenus, lecture.matchs_filtres)
    : null
  const messages = lecture
    ? statusMessages(t, lecture.matchs_en_attente ?? 0, lecture.matchs_non_cuisables ?? 0)
    : []

  return (
    <SectionCard
      title={t.planTitle}
      label={t.planTitle}
      footer={
        lecture && raisonVide === null ? (
          <PiedDuPlan
            t={t}
            lecture={lecture}
            question={question}
            divergent={peinture.modeRampe === 'divergent'}
            peintes={peinture.grid?.filled ?? 0}
            estompe={estompe}
          />
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
        <TacticalPlanFond playerSlug={playerSlug} mapId={mapId} aspect={aspectDuPlan(cadreFond, repere)}>
          {lecture && raisonVide ? (
            // L'ÉTAT VIDE SE POSE SUR LE CADRE, il ne le remplace pas : la carte garde la
            // taille qu'elle aura une fois remplie, et le message dit ce qui manque
            // au-dessus du fond plutôt qu'à la place de tout.
            <div className={`absolute inset-0 flex items-center justify-center p-3${estompe}`}>
              <EmptyStateNotice {...planEmptyText(t, raisonVide, lecture.matchs_retenus, lecture.pas_m)} />
            </div>
          ) : (
            lecture && (
              <CalqueDuPlan
                grid={peinture.grid}
                repere={repere}
                ramp={peinture.ramp}
                selected={selected}
                onCellSelect={onCellSelect}
                estompe={estompe}
              />
            )
          )}
          <IndicateurDeLecture t={t} etat={etat} />
        </TacticalPlanFond>
        {lecture && raisonVide === null && peinture.legende && (
          <LegendeDuPlan t={t} legende={peinture.legende} estompe={estompe} />
        )}
      </div>
    </SectionCard>
  )
}

/** LegendeDuPlan — la rampe avec ses deux bornes et l'unité (maquette 034b1915). */
function LegendeDuPlan({
  t,
  legende,
  estompe,
}: {
  t: TacticalText
  legende: { lo: string; hi: string; mode: 'intensity' | 'divergent' }
  estompe: string
}) {
  return (
    <div
      className={`mt-2 flex flex-wrap items-center gap-2${estompe}`}
      data-testid="tactical-plan-legend"
    >
      <span className="font-mono text-2xs tabular-nums text-muted-foreground">{legende.lo}</span>
      <span
        role="img"
        aria-label={t.planLegendLabel(legende.lo, legende.hi)}
        className="h-2 min-w-32 flex-1 rounded-full"
        style={{ background: rampeCss(legende.mode) }}
      />
      <span className="font-mono text-2xs tabular-nums text-muted-foreground">{legende.hi}</span>
    </div>
  )
}

/**
 * usePeinture — la légende, la rampe et la grille de peinture d'une lecture.
 *
 * Rampe précalculée PAR THÈME, résolue une fois par changement de palette d'accessibilité
 * (même patron que `useReplayHeatmap.ts`) — jamais recalculée par cellule.
 */
function usePeinture(
  t: TacticalText,
  locale: Locale,
  question: TacticalQuestion,
  lecture: TacticalRaster | undefined,
  repere: RepereTactique | null,
) {
  const paletteVersion = useColorPaletteVersion()
  const numFmt = useMemo(
    () => new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 }),
    [locale],
  )
  const legende = lecture
    ? planLegend(lecture.echelle, unitForQuestion(t, question), (n) => numFmt.format(n))
    : null
  const modeRampe = legende?.mode ?? 'intensity'
  const ramp = useMemo(() => {
    void paletteVersion
    const tokens = heatmapRampTokens(modeRampe).map(resolveToken)
    return modeRampe === 'divergent' ? heatRampDivergent(tokens) : heatRamp(tokens)
  }, [paletteVersion, modeRampe])
  const grid = lecture && repere ? grilleDuPlan(lecture.cellules ?? [], repere, lecture.echelle) : null
  return { legende, modeRampe, ramp, grid }
}

/** CalqueDuPlan — le `<canvas>` du calque de chaleur, posé sur le fond, et son clic. */
function CalqueDuPlan({
  grid,
  repere,
  ramp,
  selected,
  onCellSelect,
  estompe,
}: {
  grid: TacticalGrid | null
  repere: RepereTactique | null
  ramp: readonly string[]
  selected: { col: number; row: number } | null
  onCellSelect: (col: number, row: number) => void
  estompe: string
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

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

  return (
    <canvas
      ref={canvasRef}
      className={`absolute inset-0 h-full w-full cursor-crosshair text-foreground transition-opacity${estompe}`}
      onClick={handleClick}
      data-testid="tactical-plan-canvas"
    />
  )
}

/**
 * IndicateurDeLecture — ce qui se pose PAR-DESSUS le fond selon l'état de la lecture :
 * l'indicateur du premier chargement, ou la mention « Mise à jour… » d'une relecture.
 * Jamais À LA PLACE du fond.
 */
function IndicateurDeLecture({ t, etat }: { t: TacticalText; etat: TacticalEtatLecture }) {
  if (etat === 'attente') {
    return (
      <div
        className="absolute inset-0 flex items-center justify-center"
        data-testid="tactical-analysis-pending"
      >
        <Spinner label={t.loading} />
      </div>
    )
  }
  if (etat === 'relecture') {
    return (
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        <p
          role="status"
          className="rounded-md border border-border bg-card px-3 py-1.5 text-sm text-foreground shadow-sm"
          data-testid="tactical-analysis-updating"
        >
          {t.analysisUpdating}
        </p>
      </div>
    )
  }
  return null
}

/** PiedDuPlan — la phrase d'échelle, les dénominateurs, le pas, les cellules hors cadre. */
function PiedDuPlan({
  t,
  lecture,
  question,
  divergent,
  peintes,
  estompe,
}: {
  t: TacticalText
  lecture: TacticalRaster
  question: TacticalQuestion
  divergent: boolean
  peintes: number
  estompe: string
}) {
  const cellules = lecture.cellules ?? []
  // Les cellules SERVIES mais tombées hors du cadre du fond : dites, jamais avalées — un
  // plan amputé ressemblerait sinon à un plan complet.
  const horsCadre = cellules.length - peintes
  return (
    <div className={`border-t border-border px-3 py-2 text-xs text-muted-foreground${estompe}`}>
      <p data-testid="tactical-plan-scale-note">
        {divergent ? t.planScaleDivergent(TACTICAL_CELL_FLOOR) : t.planScaleQuantile}
      </p>
      <p className="mt-1" data-testid="tactical-plan-retained">
        {t.planFooterRetained(
          lecture.matchs_retenus,
          lecture.matchs_filtres,
          sourceForQuestion(t, question),
        )}
      </p>
      <p className="mt-1" data-testid="tactical-plan-grid-step">
        {t.footerGrid(lecture.pas_m)} · {t.footerFloor(TACTICAL_CELL_FLOOR)}
      </p>
      {horsCadre > 0 && (
        <p className="mt-1" data-testid="tactical-plan-off-frame">
          {t.footerOffFrame(horsCadre, cellules.length)}
        </p>
      )}
    </div>
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
