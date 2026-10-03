/**
 * useLocalFilterBar — barre de filtres locale réutilisable.
 *
 * Pattern factorisé depuis SynthesisPage : state pending → committed avec
 * cascade locale (expérience, playlists, modes), barre période/saison, et
 * useFiltersPreview pour les counts cascade-aware.
 *
 * Consommé par `UnifiedCitationsPage` (features/citations), `PalmaresRelationsPage`
 * (features/palmares) et `TacticalPage` (features/tactical) — liste vérifiée le
 * 2026-09-06 ; celle d'avant nommait trois pages qui ne l'appellent pas. Chaque page
 * consomme le hook, l'utilise pour ses requêtes et rend `bar` au sommet de son layout.
 *
 * Le hook n'écrit dans AUCUN store global — l'état reste 100% local à la page,
 * cohérent avec le pattern « 1 page = 1 scope de filtres ».
 *
 * ─── ÉTAT COMMITTED PILOTABLE PAR L'APPELANT (option `committed`) ────────────
 *
 * Ajout 2026-09-06 (onglet Tactique) : une page qui persiste son scope dans
 * l'URL (`usePageScope`) doit pouvoir POSSÉDER l'état committed — sinon l'URL et
 * la barre portent deux vérités, et le retour navigateur remet les pills sans
 * changer les requêtes. L'option lève l'état committed chez l'appelant ; le
 * PENDING, lui, reste interne (il est éphémère par nature : ce que l'utilisateur
 * est en train de régler avant « Analyser »).
 *
 * Le pending est un CALQUE sur le committed (`overlay`), pas une copie : sans
 * modification en cours, il SUIT le committed — donc un retour navigateur qui
 * change l'URL remet aussi les pills, sans le moindre effet de synchronisation.
 * Une copie initialisée au montage serait restée figée sur l'état d'arrivée.
 */
import { useMemo, useState, type ReactNode } from 'react'
import { useFiltersPreview } from '@/features/filters/queries'
import { PeriodePill, SaisonPill, DEFAULT_PERIOD } from '@/components/shell/FilterOmnibar'
import { ViewDropdown } from '@/features/_shared/ViewDropdown'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import { MultiSelectFilter, type MultiSelectOption } from '@/features/explorer/MultiSelectFilter'
import { ExperienceDropdown, type Experience } from '@/features/_shared/ExperienceDropdown'
import type { CascadeInput, FilterContextInput, PeriodInput, SessionOption } from '@/lib/api/types'
import { EXPERIENCE_TO_CASCADE, setsEqual } from '@/features/_shared/experienceCascade'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'

export interface LocalFilterBarLabels {
  experience: string
  experienceAll: string
  experienceRanked: string
  experienceUnranked: string
  playlists: string
  modes: string
  reset: string
  analyser?: string
}

/** Libellés du contrôle Vue (solo / escouade). Optionnel : fourni uniquement par
 *  les pages qui exposent la segmentation match_context (ex. hub Relations).
 *  Non exporté : consommé via le champ `viewLabels` (typé structurellement par
 *  l'appelant) — évite un export mort (knip-ratchet). */
export interface LocalFilterBarViewLabels {
  view: string
  viewAll: string
  viewSolo: string
  viewSquad: string
}

/** Vue match_context côté UI ('all' = les deux). Partagé avec ViewDropdown. */
export type MatchView = 'all' | 'solo' | 'squad'

/** État COMMITTED de la barre — tout ce qu'un clic sur « Analyser » applique.
 *  Nommé pour être portable : c'est ce qu'une page persiste dans son URL. */
export interface LocalFilterBarState {
  period: PeriodInput
  experience: Experience
  playlists: string[]
  modes: string[]
  view: MatchView
}

/** L'état committed vu de l'extérieur : sa valeur et le seul moyen de la changer. */
export interface LocalFilterBarCommitted {
  value: LocalFilterBarState
  onCommit: (next: LocalFilterBarState) => void
}

interface UseLocalFilterBarOptions {
  playerSlug: string
  labels: LocalFilterBarLabels
  /** Active le contrôle Vue solo/escouade (mappé sur cascade match_context). */
  viewLabels?: LocalFilterBarViewLabels
  /** État committed PILOTÉ par l'appelant (URL, store…). Absent = état interne. */
  committed?: LocalFilterBarCommitted
  /** Contrôles insérés dans la barre, après les filtres de cascade (sessions,
   *  composition…). Rendus dans la MÊME ligne sticky : une seconde barre en
   *  dessous donnerait deux zones de filtres pour un seul scope.
   *
   *  C'est une FONCTION et non un `ReactNode` : ces contrôles ont besoin de ce que
   *  le hook vient de charger (les sessions disponibles). Un nœud construit par
   *  l'appelant AVANT l'appel ne pourrait pas les lire — il les capturerait avant
   *  qu'elles existent. */
  extras?: (ctx: LocalFilterBarExtrasContext) => ReactNode
  /** true si les `extras` portent eux aussi un filtre actif. Sans cela, le bouton
   *  « Réinitialiser » n'est même pas rendu quand SEULS les extras filtrent — la
   *  barre annonce « aucun filtre actif » sur une lecture pourtant restreinte. */
  extrasActifs?: boolean
  /** Vide les filtres portés par les `extras`. Appelé PAR le bouton
   *  « Réinitialiser », en plus de la remise à zéro des champs du hook : sinon ↺
   *  laisse la moitié du scope en place, et la lecture reste filtrée. */
  onResetExtras?: () => void
}

/** Ce que le hook met à disposition des contrôles supplémentaires. */
export interface LocalFilterBarExtrasContext {
  /** Sessions disponibles sous le contexte en cours de réglage. */
  sessionOptions: SessionOption[]
}

/** L'état par défaut : aucun filtre. Une seule définition, partagée par
 *  l'initialisation et par « Réinitialiser ». */
export const LOCAL_FILTER_BAR_DEFAUT: LocalFilterBarState = {
  period: DEFAULT_PERIOD,
  experience: 'all',
  playlists: [],
  modes: [],
  view: 'all',
}

const MATCH_VIEW_TO_CONTEXT: Record<MatchView, 'solo' | 'squad' | 'all'> = {
  all: 'all',
  solo: 'solo',
  squad: 'squad',
}

interface UseLocalFilterBarResult {
  /** FilterContext committed à envoyer au backend (utiliser dans request.filters). */
  committedFilterContext: FilterContextInput
  /** Période committed pour les query keys / scope hash. */
  committedPeriod: PeriodInput
  /** Hash stable du contexte committed (pour queryKey TanStack). */
  committedHash: string
  /** true si l'utilisateur a au moins un filtre actif. */
  hasActiveFilters: boolean
  /** Sessions disponibles sous le contexte EN COURS de réglage — la réponse de
   *  `/filters/resolve` que le hook interroge déjà pour ses counts. Exposées
   *  plutôt que refetchées par l'appelant : deux requêtes pour une seule liste
   *  donneraient deux comptes de sessions sur la même page. */
  sessionOptions: SessionOption[]
  /** Élément JSX de la barre sticky à rendre dans le layout de la page. */
  bar: ReactNode
}

/** Construit un FilterContextInput minimal à partir d'une période, d'une cascade
 *  et d'une vue match_context (solo/escouade). 'all' → match_context omis. */
function buildContext(period: PeriodInput, cascade: CascadeInput, view: MatchView): FilterContextInput {
  const ctx: FilterContextInput = {
    filter_mode: 'period',
    period,
    sessions: { picked_sessions: [], gap_minutes: 120 },
    cascade,
  }
  if (view !== 'all') ctx.match_context = MATCH_VIEW_TO_CONTEXT[view]
  return ctx
}

/** FNV-1a 32 bits — même algo que computeHash dans createFilterStore.ts. */
function hashContext(ctx: FilterContextInput): string {
  const s = JSON.stringify(ctx) ?? ''
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h.toString(16).padStart(8, '0')
}

export function useLocalFilterBar({
  playerSlug,
  labels,
  viewLabels,
  committed,
  extras,
  extrasActifs,
  onResetExtras,
}: UseLocalFilterBarOptions): UseLocalFilterBarResult {
  // Défaut i18n du bouton « Analyser » quand l'appelant ne fournit pas de libellé
  // (le littéral FR figé cassait le bilinguisme — I2, 2026-07-05).
  const locale = useAppShellStore((s) => s.locale)
  const analyserLabel = labels.analyser ?? formatMessage(commonManifest, 'common.filter.analyser', locale)
  // ÉTAT COMMITTED : interne par défaut, PILOTÉ par l'appelant si `committed` est
  // fourni (page qui persiste son scope dans l'URL).
  const [committedLocal, setCommittedLocal] = useState<LocalFilterBarState>(LOCAL_FILTER_BAR_DEFAUT)
  const etatCommitted = committed?.value ?? committedLocal
  const appliquer = committed?.onCommit ?? setCommittedLocal

  // PENDING = un CALQUE sur le committed. `null` = rien en cours de réglage, donc
  // le pending SUIT le committed (retour navigateur, reset externe) sans effet de
  // synchronisation.
  const [calque, setCalque] = useState<Partial<LocalFilterBarState> | null>(null)
  const pending: LocalFilterBarState = useMemo(
    () => ({ ...etatCommitted, ...(calque ?? {}) }),
    [etatCommitted, calque],
  )
  const regler = (patch: Partial<LocalFilterBarState>) =>
    setCalque((prev) => ({ ...(prev ?? {}), ...patch }))

  const pendingPeriod = pending.period
  const pendingExperience = pending.experience
  const pendingView = pending.view
  const pendingPlaylists = useMemo(() => new Set(pending.playlists), [pending.playlists])
  const pendingModes = useMemo(() => new Set(pending.modes), [pending.modes])

  const committedPeriod = etatCommitted.period
  const committedExperience = etatCommitted.experience
  const committedView = etatCommitted.view
  const committedPlaylists = useMemo(() => new Set(etatCommitted.playlists), [etatCommitted.playlists])
  const committedModes = useMemo(() => new Set(etatCommitted.modes), [etatCommitted.modes])

  const setPendingPeriod = (p: PeriodInput) => regler({ period: p })
  const setPendingExperience = (e: Experience) => regler({ experience: e })
  const setPendingView = (v: MatchView) => regler({ view: v })

  const [activePopover, setActivePopover] = useState<'periode' | 'saison' | null>(null)
  const togglePopover = (which: 'periode' | 'saison') =>
    setActivePopover((cur) => (cur === which ? null : which))
  const closeAll = () => setActivePopover(null)

  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)

  const pendingCascade: CascadeInput = useMemo(() => ({
    experience_types: EXPERIENCE_TO_CASCADE[pendingExperience],
    playlists: Array.from(pendingPlaylists),
    modes: Array.from(pendingModes),
    maps: [],
  }), [pendingExperience, pendingPlaylists, pendingModes])

  const committedCascade: CascadeInput = useMemo(() => ({
    experience_types: EXPERIENCE_TO_CASCADE[committedExperience],
    playlists: Array.from(committedPlaylists),
    modes: Array.from(committedModes),
    maps: [],
  }), [committedExperience, committedPlaylists, committedModes])

  const pendingFilterContext = useMemo(
    () => buildContext(pendingPeriod, pendingCascade, pendingView),
    [pendingPeriod, pendingCascade, pendingView],
  )

  const committedFilterContext = useMemo(
    () => buildContext(committedPeriod, committedCascade, committedView),
    [committedPeriod, committedCascade, committedView],
  )

  const committedHash = useMemo(() => hashContext(committedFilterContext), [committedFilterContext])

  const { data: previewData } = useFiltersPreview(playerSlug, pendingFilterContext)
  const available = previewData?.available_options

  const experienceCounts = useMemo(() => {
    const opts = available?.experience_types ?? []
    let ranked = 0
    let unranked = 0
    let total = 0
    for (const o of opts) {
      // CONTRAT (GH5-2) : on matche sur o.VALUE (FR canonique), jamais o.label —
      // le backend localise désormais le LABEL (Ranked PvP / Unranked PvP sous EN)
      // mais garde la Value FR. Substring 'non classé' testé AVANT 'classé'.
      const v = o.value.toLowerCase()
      if (v.includes('non classé') || v.includes('non-classé') || v.includes('unranked')) {
        unranked += o.count
      } else if (v.includes('classé') || v.includes('ranked')) {
        ranked += o.count
      }
      total += o.count
    }
    return [
      { value: 'all' as const, count: total },
      { value: 'ranked' as const, count: ranked },
      { value: 'unranked' as const, count: unranked },
    ]
  }, [available?.experience_types])

  const playlistOptions: MultiSelectOption[] = useMemo(() => {
    return (available?.playlists ?? [])
      .map((p) => ({ value: p.value, label: p.label, count: p.count }))
      .filter((o) => o.count > 0 || pendingPlaylists.has(o.value))
  }, [available?.playlists, pendingPlaylists])

  const modeOptions: MultiSelectOption[] = useMemo(() => {
    return (available?.modes ?? [])
      .map((m) => ({ value: m.value, label: m.label, count: m.count }))
      .filter((o) => o.count > 0 || pendingModes.has(o.value))
  }, [available?.modes, pendingModes])

  // Les sessions disponibles sous le contexte EN COURS de réglage : elles viennent du
  // preview que le hook interroge déjà pour ses counts.
  const sessionOptions = previewData?.session_options?.all_sessions ?? []

  const hasActiveFilters =
    !!(committedPeriod.start_date || committedPeriod.end_date) ||
    committedExperience !== 'all' ||
    committedPlaylists.size > 0 ||
    committedModes.size > 0 ||
    committedView !== 'all' ||
    !!extrasActifs

  const isDirty =
    pendingPeriod.start_date !== committedPeriod.start_date ||
    pendingPeriod.end_date !== committedPeriod.end_date ||
    pendingExperience !== committedExperience ||
    pendingView !== committedView ||
    !setsEqual(pendingPlaylists, committedPlaylists) ||
    !setsEqual(pendingModes, committedModes)

  function handleAnalyser() {
    // L'ORDRE COMPTE EN MODE CONTRÔLÉ : `appliquer` peut être un `navigate` (URL), donc
    // asynchrone du point de vue du rendu. Fermer le calque d'abord aurait rendu un
    // cadre intermédiaire sur l'ANCIEN scope — les pills revenant à leur valeur
    // précédente le temps d'une frame. On applique, PUIS on ferme le calque : tant que
    // le scope commis n'est pas revenu, le calque continue d'afficher ce qui vient
    // d'être demandé (il porte exactement les mêmes valeurs).
    const demande = pending
    appliquer(demande)
    queueMicrotask(() => setCalque(null))
    closeAll()
  }

  function handleResetAll() {
    appliquer(LOCAL_FILTER_BAR_DEFAUT)
    setCalque(null)
    // Les filtres des `extras` sont remis à zéro dans le MÊME geste : « Réinitialiser
    // les filtres » qui n'en réinitialiserait qu'une partie serait un libellé faux.
    onResetExtras?.()
  }

  const bar = (
    <div className="sticky top-0 z-20 px-6" style={{ background: 'var(--background)' }}>
      <div className="flex min-h-10 items-center gap-1.5 border-b border-border py-1.5 flex-wrap">
        <ExperienceDropdown
          value={pendingExperience}
          onChange={setPendingExperience}
          counts={experienceCounts}
          dense
          labels={{
            placeholder: labels.experience,
            all: labels.experienceAll,
            ranked: labels.experienceRanked,
            unranked: labels.experienceUnranked,
          }}
        />
        {seasons.length > 0 && (
          <SaisonPill
            open={activePopover === 'saison'}
            onToggle={() => togglePopover('saison')}
            onClose={closeAll}
            seasons={seasons}
            activeSeason={activeSeason}
            onSelectSeason={(s) => setPendingPeriod(seasonToPeriod(s))}
            onClear={() => setPendingPeriod(DEFAULT_PERIOD)}
          />
        )}
        <PeriodePill
          open={activePopover === 'periode'}
          onToggle={() => togglePopover('periode')}
          onClose={closeAll}
          period={pendingPeriod}
          onSetPeriod={setPendingPeriod}
        />
        <MultiSelectFilter
          options={playlistOptions}
          selected={pendingPlaylists}
          toggle={(v) => regler({ playlists: basculer(pending.playlists, v) })}
          placeholder={labels.playlists}
          alwaysShow
          dense
          disabled={playlistOptions.length === 0 && pendingPlaylists.size === 0}
        />
        <MultiSelectFilter
          options={modeOptions}
          selected={pendingModes}
          toggle={(v) => regler({ modes: basculer(pending.modes, v) })}
          placeholder={labels.modes}
          alwaysShow
          dense
          disabled={modeOptions.length === 0 && pendingModes.size === 0}
        />
        {viewLabels && (
          <ViewDropdown value={pendingView} onChange={setPendingView} labels={viewLabels} />
        )}
        {extras?.({ sessionOptions })}
        <div className="flex-1" />
        <button
          type="button"
          onClick={handleAnalyser}
          className={[
            'shrink-0 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
            isDirty
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'border border-input bg-background text-muted-foreground hover:bg-muted',
          ].join(' ')}
        >
          {analyserLabel}
        </button>
        {hasActiveFilters && (
          <button
            type="button"
            onClick={handleResetAll}
            className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
            title={labels.reset}
          >
            ↺
          </button>
        )}
      </div>
    </div>
  )

  return {
    committedFilterContext,
    committedPeriod,
    committedHash,
    hasActiveFilters,
    sessionOptions,
    bar,
  }
}

/** basculer : ajoute ou retire une valeur d'une liste, sans la muter. */
function basculer(liste: string[], valeur: string): string[] {
  return liste.includes(valeur) ? liste.filter((v) => v !== valeur) : [...liste, valeur]
}
