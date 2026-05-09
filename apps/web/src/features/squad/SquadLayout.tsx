/**
 * SquadLayout — layout partagé de la section Escouade.
 *
 * Gère la sélection des coéquipiers (via data.options), les KPI cards et la
 * navigation par onglets (Synergies / Contributions). Expose les données
 * sélectionnées via SquadContext pour les onglets enfants.
 *
 * Multi-titres : tous les libellés métier passent par useFieldMappings
 * (fields.toml du titre courant) ; les strings UI passent par getSquadText
 * (FR/EN). La liste des KPIs (SQUAD_KPI_METRICS) dégrade gracefully quand
 * un FieldKey est absent du titre courant.
 *
 * Barre de filtres unifiée (sticky top-12, NavL2 masqué sur /squad) :
 *   [joueur actif] [coéquipiers▾] [Filtres▾] [Période▾] [Sessions escouade▾]
 *   [N matchs] [Analyser] [Réinitialiser]
 *
 * Route parente : /players/$playerSlug/squad
 * Routes enfants : /squad/synergies · /squad/contributions
 */
import { useState, useEffect, useMemo, useRef } from 'react'
import { Outlet, useParams, Link, useMatchRoute } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useTeammates } from './queries'
import { useSettings } from '@/features/settings/queries'
import { useFiltersPreview } from '@/features/filters/queries'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { AddFriendModal } from '@/features/friends/AddFriendFlow'
import { tokenCssVar } from '@/lib/accessibility'
import { getSquadText } from './i18n'
import { log } from './_logger'
import { SquadContext } from './SquadContext'
import { getSquadTeammateColors } from './colors'
import type { LabelValue, TeammateRow, TeammatesQueryRequest } from '@/lib/api/types'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'
import { deriveSquadPending } from './squadPending'

import {
  FiltresPill,
  PeriodePill,
  SaisonPill,
  DEFAULT_CASCADE,
  DEFAULT_PERIOD,
  DEFAULT_SESSIONS,
  computePendingHash,
} from '@/components/shell/FilterOmnibar'
import { PeriodSessionRail } from '@/components/shell/PeriodSessionRail'
import { useActiveSeason, seasonToPeriod } from './useActiveSeason'

// ─── Constantes ───────────────────────────────────────────────────────────────

const MAX_SELECTION = 3
const CHART_COLORS = getSquadTeammateColors(MAX_SELECTION)

function formatError(err: unknown): string {
  if (err == null) return 'Erreur inconnue'
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  if (typeof err === 'object') {
    const e = err as { message?: unknown; status?: unknown; statusText?: unknown }
    if (typeof e.message === 'string') return e.message
    if (typeof e.statusText === 'string') {
      return typeof e.status === 'number' ? `${e.status} ${e.statusText}` : e.statusText
    }
    try { return JSON.stringify(err) } catch { return 'Erreur non sérialisable' }
  }
  return String(err)
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function SquadLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const {
    filterContext,
    filterContextHash,
    resolvedContext,
    setFilterContext,
    setSessions,
    resetFilters,
  } = useGlobalFilterStore()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const storageKey = `squad-teammates-${playerSlug}`

  // ── Sélection coéquipiers ────────────────────────────────────────────────
  const [selectedGts, setSelectedGtsRaw] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(storageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch { return [] }
  })
  const setSelectedGts = (next: string[] | ((prev: string[]) => string[])) => {
    setSelectedGtsRaw((prev) => {
      const value = typeof next === 'function' ? next(prev) : next
      try { localStorage.setItem(storageKey, JSON.stringify(value)) } catch { /* ignore */ }
      return value
    })
  }
  const confirmedGts = selectedGts
  const [addFriendGamertag, setAddFriendGamertag] = useState<string | null>(null)

  const matchRoute = useMatchRoute()
  const { data: settings } = useSettings()

  // ── Filtre multi-sessions escouade (persisté, appliqué immédiatement) ────
  const sessionStorageKey = `squad-sessions-${playerSlug}`
  const [pickedSquadSessionLabels, setPickedSquadSessionLabelsRaw] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(sessionStorageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch { return [] }
  })
  const applySessionLabels = (labels: string[]) => {
    setPickedSquadSessionLabelsRaw(labels)
    try { localStorage.setItem(sessionStorageKey, JSON.stringify(labels)) } catch { /* ignore */ }
    // Synchroniser avec le globalFilterStore pour que PeriodSessionRail voie
    // la sélection (le rail lit le store global). Squad utilise des labels
    // (pas des session_id) — getRailMode matche par label OU session_id.
    setSessions({
      picked_sessions: labels,
      gap_minutes: filterContext.sessions?.gap_minutes ?? 120,
    })
  }

  // Au mount, si des labels sont restaurés du localStorage mais que le store
  // global n'a pas la même sélection, on les push (cold reload Squad).
  useEffect(() => {
    if (pickedSquadSessionLabels.length === 0) return
    const current = filterContext.sessions?.picked_sessions ?? []
    const same =
      current.length === pickedSquadSessionLabels.length &&
      current.every((id, i) => id === pickedSquadSessionLabels[i])
    if (same) return
    setSessions({
      picked_sessions: pickedSquadSessionLabels,
      gap_minutes: filterContext.sessions?.gap_minutes ?? 120,
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // mount-only

  // Post-mount : sync global -> local quand picked_sessions change ailleurs
  // (rail nav prev/next, FilterOmnibar SessionPill, autoSnapToLatestSession).
  // Sans ce sync, le rail navigue le filterContext mais le SessionMultiSelect
  // garde son ancienne sélection ; teammates_service reçoit alors deux états
  // contradictoires (filters.sessions vs picked_squad_session_labels).
  const mountedRef = useRef(false)
  useEffect(() => {
    if (!mountedRef.current) {
      mountedRef.current = true
      return
    }
    const globalPicked = filterContext.sessions?.picked_sessions ?? []
    const same =
      globalPicked.length === pickedSquadSessionLabels.length &&
      globalPicked.every((v, i) => v === pickedSquadSessionLabels[i])
    if (same) return
    setPickedSquadSessionLabelsRaw(globalPicked)
    try {
      localStorage.setItem(sessionStorageKey, JSON.stringify(globalPicked))
    } catch {
      /* ignore */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterContext.sessions?.picked_sessions])

  // ── Filtres global pending (période + cascade) — commités via Analyser ──
  const [pending, setPending] = useState(() => filterContext)
  const lastSyncedHash = useRef(filterContextHash)
  useEffect(() => {
    if (filterContextHash !== lastSyncedHash.current) {
      lastSyncedHash.current = filterContextHash
      setPending(filterContext)
    }
  }, [filterContextHash, filterContext])

  const [activePopover, setActivePopover] = useState<'filtres' | 'periode' | 'saison' | null>(null)
  const togglePopover = (which: 'filtres' | 'periode' | 'saison') =>
    setActivePopover((cur) => (cur === which ? null : which))
  const closeAll = () => setActivePopover(null)

  const pendingCascade = pending.cascade ?? DEFAULT_CASCADE
  const pendingPeriod = pending.period

  function setPendingPeriod(p: typeof DEFAULT_PERIOD) {
    const isPeriodSet = !!(p?.start_date || p?.end_date)
    setPending((prev) => ({
      ...prev,
      period: p,
      filter_mode: isPeriodSet ? 'period' : 'sessions',
      sessions: isPeriodSet ? DEFAULT_SESSIONS : prev.sessions,
    }))
  }
  function setPendingCascade(c: typeof DEFAULT_CASCADE) {
    setPending((prev) => ({ ...prev, cascade: c }))
  }

  const isDirty = filterContextHash !== computePendingHash(pending)
  const cascadeCount = (['playlists', 'modes', 'maps', 'experience_types'] as const)
    .reduce((n, k) => n + ((pendingCascade[k] as string[] | undefined)?.length ?? 0), 0)

  function handleAnalyser() {
    setFilterContext(pending)
    lastSyncedHash.current = computePendingHash(pending)
  }

  // Preview live : dérive un FilterContextInput depuis pending +
  // pickedSquadSessionLabels (cf. deriveSquadPending pour la sémantique).
  const squadPending = useMemo(
    () => deriveSquadPending(pending, pickedSquadSessionLabels),
    [pending, pickedSquadSessionLabels],
  )
  const { data: previewResolve } = useFiltersPreview(playerSlug, squadPending)

  // Compteur dynamique : préférer les counts du preview (mis à jour à la volée)
  // plutôt que ceux du resolvedContext commité (figé jusqu'au clic Analyser).
  const totalAfter = (previewResolve?.counts ?? resolvedContext?.counts)?.total_matches_after_filters ?? null
  const rawAvailable = previewResolve?.available_options ?? resolvedContext?.available_options
  const available = useMemo(() => {
    if (!rawAvailable) return undefined
    const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
    const filterUUIDs = (opts: LabelValue[]): LabelValue[] =>
      opts.filter((o) => !UUID_RE.test(o.label.trim()))
    return {
      playlists: filterUUIDs(rawAvailable.playlists),
      modes: filterUUIDs(rawAvailable.modes),
      maps: filterUUIDs(rawAvailable.maps),
      experience_types: filterUUIDs(rawAvailable.experience_types),
    }
  }, [rawAvailable])

  // Counts par session label (post-cascade en mode escouade) — alimente
  // SessionMultiSelect pour masquer les sessions vides + afficher les counts.
  const sessionCounts = useMemo(() => {
    const src = previewResolve?.session_options?.all_sessions ?? resolvedContext?.session_options?.all_sessions ?? []
    const map = new Map<string, number>()
    for (const s of src) {
      map.set(s.label, s.match_count_filtered)
    }
    return map
  }, [previewResolve, resolvedContext])
  const getSessionCount = useMemo(
    () => (label: string) => sessionCounts.get(label),
    [sessionCounts],
  )

  const presetCounts = previewResolve?.period_presets ?? resolvedContext?.period_presets

  // ── Saisons (cascade-aware counts + détection saison active) ─────────────
  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)
  const seasonCounts = previewResolve?.season_counts ?? resolvedContext?.season_counts

  // ── Init coéquipiers depuis settings ────────────────────────────────────
  useEffect(() => {
    if (settings?.friend_gamertags?.length && selectedGts.length === 0) {
      setSelectedGts(settings.friend_gamertags.slice(0, MAX_SELECTION))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings?.friend_gamertags])

  // ── Requête TeammatesService ─────────────────────────────────────────────
  // match_context="squad" : le backend ne considère que les matchs is_with_friends=true.
  const squadFilterContext = useMemo(() => ({ ...filterContext, match_context: 'squad' as const }), [filterContext])
  const request: TeammatesQueryRequest = {
    filters: squadFilterContext,
    selected_gamertags: confirmedGts.length > 0 ? confirmedGts : undefined,
    picked_squad_session_labels: pickedSquadSessionLabels.length > 0 ? pickedSquadSessionLabels : undefined,
    locale,
  }
  const { data, isLoading, isError, error } = useTeammates(
    playerSlug,
    request,
    filterContextHash,
    confirmedGts,
    pickedSquadSessionLabels,
  )

  // Sessions escouade (stables : LoadSynthesisMatches charge TOUT l'historique,
  // indépendamment de la période filtrée).
  const squadSessions = data?.session_labels?.squad ?? []

  // ── Routes actives ───────────────────────────────────────────────────────
  const synergiesRoute = '/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/players/$playerSlug/squad/contributions' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })

  // ── Gestion chargement / erreur ──────────────────────────────────────────
  // La barre de filtres (sticky) est toujours rendue; seul le contenu est
  // conditionnel pour ne pas faire disparaître les contrôles lors d'un refetch.
  const availableOptions = data?.options ?? []
  const teammates = data?.teammates ?? []
  const selectedRows = confirmedGts
    .map((gt) => teammates.find((r) => r.gamertag.toLowerCase() === gt.toLowerCase()))
    .filter(Boolean) as TeammateRow[]

  if (confirmedGts.length > 0 && !isLoading && selectedRows.length === 0) {
    log.warn(
      `invalid_selection:${playerSlug}`,
      `Aucun gamertag confirmé n'a matché un teammate côté backend (player=${playerSlug}).`,
      { confirmedGts },
    )
  }

  return (
    <SquadContext.Provider
      value={{
        selectedRows,
        confirmedGamertags: confirmedGts,
        pageData: data ?? null,
        playerSlug,
      }}
    >
      {/* ─── Barre de filtres unifiée (sticky top-12, remplace NavL2/FilterOmnibar) ─── */}
      <div className="sticky top-0 z-30 border-b border-border" style={{ background: 'var(--background)' }}>
        <div className="flex min-h-10 items-center gap-1.5 overflow-visible px-4 py-1.5">

          {/* Joueur actif — pill colorée fixe, non supprimable */}
          <span
            className="inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-sm font-medium"
            style={{ backgroundColor: tokenCssVar('compare-a'), color: '#fff' }} // color-allow: blanc structurel pour contraste sur fond compare-a
            title={playerSlug}
          >
            <span className="max-w-[7rem] truncate">{playerSlug}</span>
          </span>

          {/* Coéquipiers (multi-select compact inline, jusqu'à 3) */}
          <GamertagCombobox
            compact
            selected={selectedGts}
            onChange={setSelectedGts}
            max={MAX_SELECTION}
            frequentOptions={availableOptions}
            colors={CHART_COLORS}
            excludeGamertag={playerSlug}
            placeholder={t.selection.placeholder(availableOptions.length)}
            onAddAsFriend={setAddFriendGamertag}
          />

          {/* Séparateur */}
          <div className="mx-0.5 h-5 w-px shrink-0 bg-border" aria-hidden />

          {/* Filtres cascade (playlists / modes / cartes / expérience) */}
          <FiltresPill
            open={activePopover === 'filtres'}
            onToggle={() => togglePopover('filtres')}
            onClose={closeAll}
            available={available ?? { playlists: [], modes: [], maps: [], experience_types: [] }}
            cascade={pendingCascade}
            cascadeCount={cascadeCount}
            onSetCascade={setPendingCascade}
          />

          {/* Saison (catalog TOML kind="season" — applique la fenêtre via setPendingPeriod) */}
          {seasons.length > 0 && (
            <SaisonPill
              open={activePopover === 'saison'}
              onToggle={() => togglePopover('saison')}
              onClose={closeAll}
              seasons={seasons}
              activeSeason={activeSeason}
              seasonCounts={seasonCounts}
              onSelectSeason={(s) => setPendingPeriod(seasonToPeriod(s))}
              onClear={() => setPendingPeriod(DEFAULT_PERIOD)}
            />
          )}

          {/* Période */}
          <PeriodePill
            open={activePopover === 'periode'}
            onToggle={() => togglePopover('periode')}
            onClose={closeAll}
            period={pendingPeriod}
            onSetPeriod={setPendingPeriod}
            presetCounts={presetCounts}
          />

          {/* Sessions escouade (multi-select par label) */}
          {squadSessions.length > 0 && (
            <SessionMultiSelect
              sessions={squadSessions}
              selected={pickedSquadSessionLabels}
              onChange={applySessionLabels}
              locale={locale}
              triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
              getMatchCount={getSessionCount}
            />
          )}

          <div className="flex-1" />

          {/* Compteur dynamique */}
          {totalAfter !== null && (
            <span className="shrink-0 text-xs text-muted-foreground" aria-live="polite">
              {totalAfter} match{totalAfter > 1 ? 's' : ''}
            </span>
          )}

          {/* Analyser */}
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
            Analyser
          </button>

          {/* Réinitialiser */}
          <button
            type="button"
            onClick={() => {
              resetFilters()
              applySessionLabels([])
            }}
            className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
            title="Réinitialiser tous les filtres"
          >
            ↺ Réinitialiser
          </button>
        </div>
        {/* Rail de navigation période/session — placé DANS la barre sticky pour
            apparaître toujours juste sous les filtres Squad au scroll. */}
        <PeriodSessionRail />
      </div>

      {/* ─── Contenu ─────────────────────────────────────────────────────────── */}
      {isLoading && (
        <div className="flex items-center justify-center p-12 text-sm text-muted-foreground">
          Chargement…
        </div>
      )}

      {!isLoading && isError && (
        <div className="p-6 text-center text-destructive">
          {t.errors.loadError(formatError(error))}
        </div>
      )}

      {!isLoading && !isError && !data && (
        <div className="p-6">
          <EmptyStateCard title={t.empty.noDataTitle} description={t.empty.noDataDescription} />
        </div>
      )}

      {!isLoading && !isError && data && (
        <div className="flex flex-col gap-6 p-6">
          {/* SessionBriefing — KPIs + verdict squad + drill-down click */}
          {/* Remplace l'ancienne section "Synergies avec les coéquipiers sélectionnés"
             (KPIBlock par teammate) — meme info accessible via le drill-down click sur
             la card du joueur dans la bande verdict. */}
          {data?.header?.solo_kpis && (() => {
            const header = data.header
            const soloKpis = header.solo_kpis
            if (!soloKpis) return null
            const mainGT = data.main_player ?? ''
            const mainXuid = header?.player_cards?.find((c) => c.gamertag === mainGT)?.xuid ?? ''
            const briefingSquad =
              header?.squad_score && header?.player_cards && header?.team_avg_kpis && header?.kpis_by_xuid && mainXuid
                ? {
                    score: header.squad_score,
                    players: header.player_cards,
                    kpisByXuid: header.kpis_by_xuid,
                    teamAvgKpis: header.team_avg_kpis,
                    activeXuid: mainXuid,
                  }
                : undefined
            return <SessionBriefing kpis={soloKpis} squad={briefingSquad} />
          })()}

          {/* Navigation onglets */}
          <div className="border-b">
            <nav className="flex gap-0">
              <Link
                to="/players/$playerSlug/squad/synergies"
                params={{ playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isSynergies ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.synergies}
              </Link>
              <Link
                to="/players/$playerSlug/squad/contributions"
                params={{ playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isContributions ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.contributions}
              </Link>
            </nav>
          </div>

          <Outlet />
        </div>
      )}

      {addFriendGamertag && (
        <AddFriendModal
          gamertag={addFriendGamertag}
          open={!!addFriendGamertag}
          onClose={() => setAddFriendGamertag(null)}
          locale={locale}
        />
      )}
    </SquadContext.Provider>
  )
}
