/**
 * TacticalPlanCard — la carte du plan de l'écran unique : bandeau (nom de la carte, aide ⓘ, trois
 * réglages en pilules), bandeau d'état des lectures d'artefact, puis le fond de la carte, le calque
 * de la lecture par-dessus, la commande de zoom dans l'angle bas-droit et la rampe verticale de la
 * légende au bord droit.
 *
 * ─── LE FOND NE BOUGE JAMAIS (retours rejeu L2, 2026-09-23) ──────────────────────────────────
 *
 * La carte est TOUJOURS rendue dès qu'une carte est connue, et le cadre + le fond vivent dans
 * `TacticalPlanFond`, qui ne reçoit que la carte, le calage, la hauteur et le cadrage. Ce qui se
 * pose PAR-DESSUS dépend de l'état du plan (`etatDuPlan`, `plan.logic.ts`) :
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
 * ─── LE REPÈRE EST LE CADRE DU FOND, LA VUE EST SA FENÊTRE ───────────────────────────────────
 *
 * Le calque projette sur le CALAGE publié avec chaque fond (`useTacticalMapBackgroundFrame`),
 * exactement comme « Occupation du terrain » de la vue match et le rejeu 2D ; une carte sans fond
 * figé retombe sur ses propres bornes. Le zoom (`useCadrageDuPlan`, les gestes du rejeu 2D) en tire
 * la FENÊTRE visible, sur laquelle se projettent l'image du fond, la chaleur, les zones nommées
 * (peintre du rejeu, `planPaint.ts`), la sélection et l'étiquette. La boîte prend la hauteur que la
 * fenêtre du navigateur lui laisse (`useHauteurDuPlan`), au rapport du fond.
 *
 * COULEURS : rampe d'INTENSITÉ pour les grandeurs neutres (des morts, des frags, du temps) ; rampe
 * DIVERGENTE pour les lectures SIGNÉES (« victoires − défaites », « solde ») — leur zéro a deux
 * côtés, et la rampe d'intensité effacerait tout le côté négatif. Unité, source et rampe viennent de
 * la question À LAQUELLE LA LECTURE RÉPOND (`questionServie`), jamais de la question demandée.
 */
import { useMemo, useRef } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { ReplayZoomControl } from '@/features/match-replay/ui/ReplayZoomControl'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalGrappe, TacticalRaster } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { normalizeCalloutZones } from '@/lib/replay/calloutsPaint'
import { heatRamp, heatRampDivergent } from '@/lib/replay/heatPaint'

import { RAMPE_HAUTEUR_PX } from './cockpit.logic'
import type { TacticalText } from './i18n'
import {
  cadrageDuFond,
  infoDuPlan,
  legendeDuPlan,
  MARGE_LEGENDE,
  rampeVerticale,
  type BornesDeLegende,
  type EtatDuPlan,
} from './plan.logic'
import { useTacticalMapBackgroundFrame } from './queries'
import { aspectDuPlan, ESTOMPE } from './tacticalLecture.logic'
import { AvisSurLeFond, IndicateurDeLecture } from './TacticalPlanAvis'
import { CalqueDuPlan, EtiquetteDeZone } from './TacticalPlanCalque'
import { TacticalPlanFond } from './TacticalPlanFond'
import {
  grilleDuPlan,
  planEmptyReason,
  titreDuPlanVide,
  repereDuPlan,
  statusMessages,
  unitForQuestion,
  type RepereTactique,
  type TacticalQuestion,
  type TacticalQui,
} from './tacticalView.logic'
import { useCadrageDuPlan } from './useCadrageDuPlan'
import { useHauteurDuPlan } from './useHauteurDuPlan'

/** Les trois réglages du bandeau (état local de la vue). */
export interface ReglagesDuPlan {
  question: TacticalQuestion
  onQuestionChange: (question: TacticalQuestion) => void
  /** La pilule « Lecture » est survolée ou prend le focus : l'intention de changer de lecture. */
  onQuestionIntent?: () => void
  qui: TacticalQui
  onQuiChange: (qui: TacticalQui) => void
  escouadeDisponible: boolean
  /** « Escouade » demandé sans composition : la lecture reste sur « Moi » et le bandeau le dit. */
  escouadeEnAttente: boolean
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
  /** La cellule choisie, entourée sur le plan — `null` tant que rien n'est cliqué. */
  selected: { col: number; row: number } | null
  /** Le point MONDE d'un clic sur le plan ; la vue retient la cellule servie qu'il vise. */
  onPointSelect: (point: { x: number; y: number }) => void
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
  onPointSelect,
}: TacticalPlanCardProps) {
  // LE CADRE DU FOND EST LE REPÈRE, quand il existe. Sans lecture, il donne encore le rapport.
  const cadreFond = useTacticalMapBackgroundFrame(playerSlug, mapId)
  const lue = etat === 'pret' || etat === 'relecture' ? lecture : undefined
  const repere = useMemo(() => (lue ? repereDuPlan(cadreFond, lue.bornes, lue.pas_m) : null), [cadreFond, lue])
  const { scene, zoom, fenetre } = useCadrageDuPlan(repere, cadreFond)
  const peinture = usePeinture(t, locale, question, lue, repere)
  const zones = useMemo(() => normalizeCalloutZones(lue?.zones), [lue?.zones])
  const boite = useRef<HTMLDivElement>(null)
  const hauteurMax = useHauteurDuPlan(boite)
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
      <div ref={boite} className="relative mx-3 mb-3 mt-2.5 flex justify-center" style={{ paddingRight: MARGE_LEGENDE }}>
        <TacticalPlanFond
          playerSlug={playerSlug}
          mapId={mapId}
          aspect={aspectDuPlan(cadreFond, repere)}
          hauteurMax={hauteurMax}
          cadrage={cadrageDuFond(scene, fenetre)}
        >
          {lue && raisonVide === null && (
            <CalqueDuPlan
              grid={peinture.grid}
              repere={repere}
              fenetre={fenetre}
              zoom={zoom}
              ramp={peinture.ramp}
              zones={zones}
              locale={locale}
              selected={selected}
              onPointSelect={onPointSelect}
              estompe={estompe}
            />
          )}
          {repere && selected && etiquette && (
            <EtiquetteDeZone selected={selected} repere={repere} fenetre={fenetre} texte={etiquette} estompe={estompe} />
          )}
          <AvisSurLeFond t={t} etat={etat} inconnus={inconnus} estompe={estompe}>
            {lue && raisonVide ? titreDuPlanVide(t, raisonVide) : null}
          </AvisSurLeFond>
          <IndicateurDeLecture t={t} etat={etat} />
          {mapId && <ReplayZoomControl zoom={zoom} locale={locale} />}
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
 * atténuée + valeur). « Escouade » reste CLIQUABLE sans composition : le clic ouvre le sélecteur de
 * composition de la barre (`onQuiChange`, cf. `useReglages`), la valeur reste atténuée et un texte
 * visible dit qu'aucune composition n'est choisie — la lecture, elle, reste sur « Moi ».
 */
function ReglagesEnPilules({
  t,
  locale,
  question,
  onQuestionChange,
  onQuestionIntent,
  qui,
  onQuiChange,
  escouadeDisponible,
  escouadeEnAttente,
  spawn,
  onSpawnChange,
  grappes,
}: ReglagesDuPlan & { t: TacticalText; locale: Locale }) {
  const libelleQui: Record<TacticalQui, string> = { moi: t.whoMe, escouade: t.whoSquad, adv: t.whoOpponents }
  return (
    <span className="flex flex-wrap items-center gap-2" data-testid="tactical-plan-pills">
      <label className={PILULE} onPointerEnter={onQuestionIntent} onFocus={onQuestionIntent}>
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
          const sansComposition = valeur === 'escouade' && !escouadeDisponible
          return (
            <button
              key={valeur}
              type="button"
              aria-pressed={qui === valeur}
              onClick={() => onQuiChange(valeur)}
              className={`px-[7px] aria-pressed:bg-primary aria-pressed:font-medium aria-pressed:text-primary-foreground ${sansComposition ? 'text-muted-foreground' : 'text-foreground'}${i > 0 ? ' border-l border-input' : ''}`}
            >
              {libelleQui[valeur]}
            </button>
          )
        })}
      </span>
      {escouadeEnAttente && (
        <span role="status" className="text-xs text-muted-foreground" data-testid="tactical-escouade-attente">
          {t.planSquadPending}
        </span>
      )}
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
  const grid = useMemo(
    () => (lecture && repere ? grilleDuPlan(lecture.cellules ?? [], repere, lecture.echelle) : null),
    [lecture, repere],
  )
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
