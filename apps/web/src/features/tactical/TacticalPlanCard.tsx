/**
 * TacticalPlanCard — la carte du plan de l'écran unique : bandeau (nom de la carte, aide ⓘ, trois
 * réglages en pilules), bandeau d'état des lectures d'artefact, puis le fond de la carte, le calque
 * de la lecture par-dessus et la rampe verticale de la légende au bord droit.
 *
 * ─── LE FOND NE BOUGE JAMAIS (retours rejeu L2, 2026-09-23) ──────────────────────────────────
 *
 * La carte est TOUJOURS rendue dès qu'une carte est connue, et le cadre + le fond vivent dans
 * `TacticalPlanFond`, qui ne reçoit que la carte et le calage. Ce qui se pose PAR-DESSUS dépend de
 * l'état du plan (`etatDuPlan`, `plan.logic.ts`) :
 *   - `attente`    : l'indicateur de chargement, sur le cadre déjà posé (au rapport par défaut tant
 *                    qu'aucune carte n'est connue) ;
 *   - `relecture`  : l'ancien calque, sa légende et ses messages d'état ESTOMPÉS sous « Mise à
 *                    jour… » (décision Q26) ;
 *   - `echec`      : le message (panne de la lecture ou de son périmètre, composition impossible),
 *                    le fond reste ;
 *   - `hors_filtre`: « Aucun match sur cette carte dans le filtre », sans lecture ;
 *   - `sans_carte` : rien (la colonne des cartes dit pourquoi) ;
 *   - `pret`       : le calque, ou l'état vide du plan (un titre seul).
 *
 * ─── LE REPÈRE EST LE CADRE DU FOND ──────────────────────────────────────────────────────────
 *
 * Le calque projette sur le CALAGE publié avec chaque fond (`useTacticalMapBackgroundFrame`),
 * exactement comme « Occupation du terrain » de la vue match et le rejeu 2D ; une carte sans fond
 * figé retombe sur ses propres bornes. Le fond (`<img>`) et le calque (`<canvas>`,
 * `drawTacticalHeatmap`, noyau partagé de `lib/replay/heatPaint.ts`) partagent le même cadre monde.
 *
 * COULEURS : rampe d'INTENSITÉ pour les grandeurs neutres (des morts, des frags, du temps) ; rampe
 * DIVERGENTE pour les lectures SIGNÉES (« victoires − défaites », « solde ») — leur zéro a deux
 * côtés, et la rampe d'intensité effacerait tout le côté négatif. Unité, source et rampe viennent de
 * la question À LAQUELLE LA LECTURE RÉPOND (`questionServie`), jamais de la question demandée.
 */
import { useEffect, useMemo, useRef } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { Spinner } from '@/components/ui/spinner'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalGrappe, TacticalRaster } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { drawTacticalHeatmap, heatRamp, heatRampDivergent, type TacticalGrid } from '@/lib/replay/heatPaint'

import { RAMPE_HAUTEUR_PX } from './cockpit.logic'
import type { TacticalText } from './i18n'
import { infoDuPlan, legendeDuPlan, MARGE_LEGENDE, rampeVerticale, type EtatDuPlan, type BornesDeLegende } from './plan.logic'
import { useTacticalMapBackgroundFrame } from './queries'
import { positionEtiquette } from './zone.logic'
import { aspectDuPlan, ESTOMPE } from './tacticalLecture.logic'
import { TacticalPlanFond } from './TacticalPlanFond'
import {
  celluleDuClic,
  grilleDuPlan,
  planEmptyReason,
  titreDuPlanVide,
  rectSelection,
  repereDuPlan,
  statusMessages,
  unitForQuestion,
  vueDuPlan,
  type RepereTactique,
  type TacticalQuestion,
  type TacticalQui,
} from './tacticalView.logic'

/** Les trois réglages du bandeau (état local de la vue). */
export interface ReglagesDuPlan {
  question: TacticalQuestion
  onQuestionChange: (question: TacticalQuestion) => void
  qui: TacticalQui
  onQuiChange: (qui: TacticalQui) => void
  escouadeDisponible: boolean
  spawn: string
  onSpawnChange: (spawn: string) => void
  grappes: readonly TacticalGrappe[]
}

export interface TacticalPlanCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  /** La carte affichée ; '' = aucune encore (le cadre garde le rapport par défaut). */
  mapId: string
  /** Le nom de la carte affichée ; '' = aucun titre. */
  titre: string
  /** La question À LAQUELLE `lecture` RÉPOND (`questionServie`). */
  question: TacticalQuestion
  /** La lecture AFFICHÉE : la courante, ou la précédente pendant une relecture. */
  lecture: TacticalRaster | undefined
  etat: EtatDuPlan
  /** En échec : les coéquipiers introuvables (composition impossible), `null` pour une panne. */
  inconnus: string[] | null
  /** Le nom de la zone choisie, posé à côté de sa cellule sur le plan ; `null` tant qu'il n'est pas connu. */
  etiquette: string | null
  reglages: ReglagesDuPlan
  /** La cellule choisie, encadrée sur le plan — `null` tant que rien n'est cliqué. */
  selected: { col: number; row: number } | null
  onCellSelect: (col: number, row: number) => void
}

export function TacticalPlanCard({
  t,
  locale,
  playerSlug,
  mapId,
  titre,
  question,
  lecture,
  etat,
  inconnus,
  etiquette,
  reglages,
  selected,
  onCellSelect,
}: TacticalPlanCardProps) {
  // LE CADRE DU FOND EST LE REPÈRE, quand il existe. Sans lecture, il donne encore le rapport.
  const cadreFond = useTacticalMapBackgroundFrame(playerSlug, mapId)
  const lue = etat === 'pret' || etat === 'relecture' ? lecture : undefined
  const repere = lue ? repereDuPlan(cadreFond, lue.bornes, lue.pas_m) : null
  const peinture = usePeinture(t, locale, question, lue, repere)
  const estompe = etat === 'relecture' ? ` ${ESTOMPE}` : ''
  const peintes = peinture.grid?.filled ?? 0
  const raisonVide = lue ? planEmptyReason(peintes, lue.matchs_retenus, lue.matchs_filtres) : null
  const messages = lue ? statusMessages(t, lue.matchs_en_attente ?? 0, lue.matchs_non_cuisables ?? 0) : []

  return (
    <SectionCard
      title={titre}
      label={titre || undefined}
      titleAdornment={titleWithInfo(lue ? infoDuPlan(t, locale, question, lue, peintes) : null, {
        trailing: <ReglagesEnPilules t={t} locale={locale} {...reglages} />,
        testId: 'tactical-plan-title',
      })}
    >
      {messages.length > 0 && (
        <div role="status" className={`px-3 pt-1.5 text-xs text-warning${estompe}`} data-testid="tactical-plan-status">
          {messages.map((msg) => (
            <p key={msg}>{msg}</p>
          ))}
        </div>
      )}
      <div className="relative mx-3 mb-3 mt-2.5 flex justify-center" style={{ paddingRight: MARGE_LEGENDE }}>
        <TacticalPlanFond playerSlug={playerSlug} mapId={mapId} aspect={aspectDuPlan(cadreFond, repere)}>
          {lue && raisonVide === null && (
            <CalqueDuPlan
              grid={peinture.grid}
              repere={repere}
              ramp={peinture.ramp}
              selected={selected}
              onCellSelect={onCellSelect}
              estompe={estompe}
            />
          )}
          {lue && repere && selected && etiquette && (
            <EtiquetteDeZone selected={selected} repere={repere} texte={etiquette} estompe={estompe} />
          )}
          <AvisSurLeFond t={t} etat={etat} inconnus={inconnus} estompe={estompe}>
            {lue && raisonVide ? titreDuPlanVide(t, raisonVide) : null}
          </AvisSurLeFond>
          <IndicateurDeLecture t={t} etat={etat} />
        </TacticalPlanFond>
        <RampeVerticale
          t={t}
          legende={lue && raisonVide === null ? peinture.legende : null}
          ramp={peinture.ramp}
          unite={unitForQuestion(t, question)}
          estompe={estompe}
        />
      </div>
    </SectionCard>
  )
}

const PILULE =
  'inline-flex h-7 items-stretch overflow-hidden rounded-md border border-input bg-background text-[11.5px] font-normal'
const ETIQUETTE = 'flex items-center whitespace-nowrap border-r border-input bg-muted px-1.5 text-muted-foreground'
const QUI_VALEURS: readonly TacticalQui[] = ['moi', 'escouade', 'adv']

/**
 * ReglagesEnPilules — « Lecture », « Joueurs » et « Réapparition », chacun en pilule (étiquette
 * atténuée + valeur). « Escouade » est DÉSACTIVÉ sans composition dans la barre, et son infobulle
 * le dit : un bouton actif qui ne changerait rien à la lecture mentirait sur ce qu'il fait.
 */
function ReglagesEnPilules({
  t,
  locale,
  question,
  onQuestionChange,
  qui,
  onQuiChange,
  escouadeDisponible,
  spawn,
  onSpawnChange,
  grappes,
}: ReglagesDuPlan & { t: TacticalText; locale: Locale }) {
  const libelleQui: Record<TacticalQui, string> = { moi: t.whoMe, escouade: t.whoSquad, adv: t.whoOpponents }
  return (
    <span className="flex flex-wrap items-center gap-2" data-testid="tactical-plan-pills">
      <label className={PILULE}>
        <span className={ETIQUETTE}>{t.pillReading}</span>
        <select
          value={question}
          onChange={(e) => onQuestionChange(e.target.value as TacticalQuestion)}
          aria-label={t.pillReading}
          className="w-[146px] bg-transparent pl-1.5 pr-0.5 text-foreground"
        >
          {t.analysisQuestions.map((q) => (
            <option key={q.id} value={q.id}>
              {q.label}
            </option>
          ))}
        </select>
      </label>
      <span role="group" aria-label={t.pillPlayers} className={PILULE}>
        <span className={ETIQUETTE}>{t.pillPlayers}</span>
        {QUI_VALEURS.map((valeur, i) => {
          const desactive = valeur === 'escouade' && !escouadeDisponible
          return (
            <button
              key={valeur}
              type="button"
              aria-pressed={qui === valeur}
              disabled={desactive}
              title={desactive ? t.planSquadDisabled : undefined}
              onClick={() => onQuiChange(valeur)}
              className={`px-[7px] text-foreground aria-pressed:bg-primary aria-pressed:font-medium aria-pressed:text-primary-foreground disabled:cursor-not-allowed disabled:text-muted-foreground disabled:opacity-55${i > 0 ? ' border-l border-input' : ''}`}
            >
              {libelleQui[valeur]}
            </button>
          )
        })}
      </span>
      <label className={PILULE}>
        <span className={ETIQUETTE}>{t.pillRespawn}</span>
        <select
          value={spawn}
          onChange={(e) => onSpawnChange(e.target.value)}
          aria-label={t.pillRespawn}
          className="w-[72px] bg-transparent pl-1.5 pr-0.5 text-foreground"
        >
          <option value="">{t.pillRespawnAll}</option>
          {grappes.map((g) => (
            <option key={g.id} value={g.id}>
              {locale === 'fr' ? g.nom_fr : g.nom_en}
            </option>
          ))}
        </select>
      </label>
    </span>
  )
}

/**
 * usePeinture — la légende, la rampe et la grille de peinture d'une lecture. La rampe est résolue
 * une fois par changement de palette d'accessibilité, jamais par cellule ; la légende la prélève.
 */
function usePeinture(
  t: TacticalText,
  locale: Locale,
  question: TacticalQuestion,
  lecture: TacticalRaster | undefined,
  repere: RepereTactique | null,
) {
  const paletteVersion = useColorPaletteVersion()
  const numFmt = useMemo(() => new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 }), [locale])
  const legende = lecture ? legendeDuPlan(lecture.echelle, unitForQuestion(t, question), (n) => numFmt.format(n)) : null
  const modeRampe = legende?.mode ?? 'intensity'
  const ramp = useMemo(() => {
    void paletteVersion
    const tokens = heatmapRampTokens(modeRampe).map(resolveToken)
    return modeRampe === 'divergent' ? heatRampDivergent(tokens) : heatRamp(tokens)
  }, [paletteVersion, modeRampe])
  const grid = lecture && repere ? grilleDuPlan(lecture.cellules ?? [], repere, lecture.echelle) : null
  return { legende, ramp, grid }
}

/**
 * RampeVerticale — la légende au bord droit, indépendante du fond : la rampe peinte en hauteur fixe,
 * la borne haute et l'unité en haut, la borne basse en bas. Sans lecture à décrire, la rampe
 * atténuée et l'unité seules.
 */
function RampeVerticale({
  t,
  legende,
  ramp,
  unite,
  estompe,
}: {
  t: TacticalText
  legende: BornesDeLegende | null
  ramp: readonly string[]
  unite: string
  estompe: string
}) {
  return (
    <div
      className={`pointer-events-none absolute right-0 top-1/2 w-[82px] -translate-y-1/2 text-[11px] leading-[14px] tabular-nums text-muted-foreground${estompe}`}
      style={{ height: RAMPE_HAUTEUR_PX }}
      data-testid="tactical-plan-legend"
    >
      {legende ? (
        <span
          role="img"
          aria-label={t.planLegendLabel(legende.lo, legende.hi)}
          data-mode={legende.mode}
          className="absolute bottom-0 left-0 top-0 w-2 rounded-full"
          style={{ background: rampeVerticale(ramp) }}
        />
      ) : (
        <span className="absolute bottom-0 left-0 top-0 w-2 rounded-full opacity-35" style={{ background: rampeVerticale(ramp) }} />
      )}
      <span className="absolute left-3.5 right-0 top-0">
        {legende && <b className="block font-medium text-foreground">{legende.hiNombre}</b>}
        {unite}
      </span>
      <span className="absolute bottom-0 left-3.5 right-0">
        <b className="block font-medium text-foreground">{legende ? legende.lo : '—'}</b>
      </span>
    </div>
  )
}

/**
 * AvisSurLeFond — le message posé SUR le fond, jamais à sa place : l'échec (composition impossible
 * ou panne), la carte hors du filtre, ou l'état vide du plan (`children`, un titre seul).
 */
function AvisSurLeFond({
  t,
  etat,
  inconnus,
  estompe,
  children,
}: {
  t: TacticalText
  etat: EtatDuPlan
  inconnus: string[] | null
  estompe: string
  children: string | null
}) {
  if (etat === 'echec') {
    return (
      <div className="absolute inset-0 grid place-items-center p-3" data-testid="tactical-plan-avis">
        {inconnus ? (
          <EmptyStateNotice
            title={t.unknownTeammateTitle}
            description={t.unknownTeammateDescription(inconnus.join(', '))}
            className="bg-card"
          />
        ) : (
          <p className="rounded-md border border-border bg-card px-2.5 py-1.5 text-center text-sm text-muted-foreground">
            {t.analysisErrorTitle}
          </p>
        )}
      </div>
    )
  }
  const titre = etat === 'hors_filtre' ? t.planEmptyNoMatchTitle : children
  if (!titre) return null
  return (
    <div
      className={`pointer-events-none absolute inset-0 grid place-items-center p-3${estompe}`}
      data-testid={etat === 'hors_filtre' ? 'tactical-carte-hors-filtre' : 'tactical-plan-vide'}
    >
      <p className="rounded-md border border-border bg-card px-2.5 py-1.5 text-center text-sm text-muted-foreground">{titre}</p>
    </div>
  )
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
    // LE CADRE DE LA CELLULE CHOISIE, par-dessus le calque : le retour immédiat au clic. Son encre
    // est celle du texte de l'app (`text-foreground` lu par `getComputedStyle`) : un jeton
    // sémantique dirait une SIGNIFICATION que cette sélection n'a pas — elle dit seulement « ici ».
    const cadreSel = rectSelection(selected, repere, width)
    if (!cadreSel) return
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
 * IndicateurDeLecture — ce qui se pose PAR-DESSUS le fond selon l'état de la lecture : l'indicateur
 * du premier chargement (visuel seul, sans texte d'attente : l'occupation est dite par `aria-busy`
 * sur le corps de la vue), ou la mention « Mise à jour… » d'une relecture. Jamais À LA PLACE du fond.
 */
function IndicateurDeLecture({ t, etat }: { t: TacticalText; etat: EtatDuPlan }) {
  if (etat === 'attente') {
    return (
      <div className="absolute inset-0 flex items-center justify-center" data-testid="tactical-analysis-pending">
        <Spinner />
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

/** EtiquetteDeZone — le nom de la zone choisie, posé à côté de sa cellule, du côté où il tient. */
function EtiquetteDeZone({
  selected,
  repere,
  texte,
  estompe,
}: {
  selected: { col: number; row: number }
  repere: RepereTactique
  texte: string
  estompe: string
}) {
  const position = positionEtiquette(selected, repere)
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
