/**
 * PersonalStatsLayout — layout partagé de la section Stats perso.
 *
 * Calque allégé de SquadLayout, sans la logique coéquipiers :
 *  - Pas de pill "joueur actif" (le joueur est implicite via /players/{slug}).
 *  - Pas de GamertagCombobox / CompareDrawer / AddFriendModal.
 *  - Une seule requête : useSynthesisPage → solo_kpis pour la SessionBriefing.
 *  - useFiltersPreview alimenté directement avec `pending` (pas de
 *    deriveSquadPending — solo n'a pas de match_context override).
 *
 * Barre de filtres unifiée (sticky top-0, NavL2 masqué sur /stats) :
 *   [Filtres▾] [Saison▾] [Période▾] [Sessions▾] [N matchs] [Analyser] [Réinitialiser]
 *
 * Route parente : /players/$playerSlug/stats (pathless layout `_personal`)
 * Routes enfants : /stats/{summary,maps-modes,distributions,progression,advanced}
 *
 * SessionBriefing : mode solo uniquement — props `squad` non passé. La grille
 * KPIs s'affiche, pas de bande verdict.
 *
 * Persistance des filtres + scope solo strict :
 *  - `globalFilterStore` (Zustand + persist middleware) garde les filtres
 *    en localStorage entre sessions navigateur.
 *  - `match_context: 'solo'` est forcé sur `useFiltersPreview` (cf. soloPending)
 *    : cascade, counts et session_options sont scoped solo. Les sessions
 *    squad sont aussi filtrées côté front (`is_squad === false`) pour rester
 *    cohérent même si le backend n'applique pas le scope.
 *  - Tracker `lastKnownLatestSoloSessionId` (store, persisté) maintenu par
 *    `useFiltersResolve`. Détection d'une nouvelle session solo cross-mount.
 *  - L'effet "Snap sur la dernière session solo" :
 *    - **Nouvelle session solo détectée** (latest !== `lastKnownLatestSoloSessionId`)
 *      → reset TOTAL inconditionnel : cascade + période + sessions wipées,
 *      snap sur la nouvelle session. Aucun garde-fou — même une période
 *      user-set est effacée. Justification : consultation typique = post-jeu,
 *      l'utilisateur veut une vue d'ensemble fraîche.
 *    - **Pas de nouvelle session** → respect intégral des filtres user
 *      (cascade, période, session). Si l'utilisateur a une période custom
 *      OU une session solo valide → no-op total, tous les filtres
 *      préservés. Si jamais hydraté OU sélection d'un autre kind (squad),
 *      fallback : snap sur la latest solo en **préservant la cascade**
 *      (seule la session est posée et la période vidée par exclusivité).
 *
 * Navigation vers un match depuis cette page :
 *  - Tout futur clic sur une tuile match (ex. dans SummaryTab) doit utiliser
 *    `useNavigateToMatch(playerSlug)` (cf. lib/match-nav/useNavigateToMatch.ts)
 *    avec un MatchNavContext correctement renseigné :
 *      source: 'history' | 'session' | …
 *      matchIds: liste DESC scope-courante
 *      contextDescriptor: { kind: 'session', startTimeUtc: … } ou similaire
 *      filterSpec: équivalent canonique du filterContext courant
 *    Ainsi MatchNavigationBar restera scoped et le bouton retour reviendra
 *    avec les filtres préservés (déjà persistés dans globalFilterStore).
 */
import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { Outlet, useParams, Link, useMatchRoute } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useFiltersPreview } from '@/features/filters/queries'
import { useSessionSnap } from '@/features/filters/useSessionSnap'
import { useSynthesisPage } from '@/features/synthesis/queries'
import { useAppShellStore } from '@/stores/appShellStore'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'
import { getPersonalStatsText } from './i18n'
import { PersonalStatsContext } from './PersonalStatsContext'
import type { FilterContextInput, LabelValue, SynthesisQueryRequest } from '@/lib/api/types'

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
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'

// ─── Helpers ───────────────────────────────────────────────────────────────

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

export function PersonalStatsLayout() {
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
  const t = getPersonalStatsText(locale)

  const matchRoute = useMatchRoute()

  // ── Filtre multi-sessions solo (persisté, appliqué immédiatement) ────────
  const sessionStorageKey = `personal-stats-sessions-${playerSlug}`
  const [pickedSessionLabels, setPickedSessionLabelsRaw] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(sessionStorageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch { return [] }
  })
  const applySessionLabels = useCallback(
    (labels: string[]) => {
      setPickedSessionLabelsRaw(labels)
      try {
        localStorage.setItem(sessionStorageKey, JSON.stringify(labels))
      } catch {
        /* ignore */
      }
      setSessions({
        picked_sessions: labels,
        gap_minutes: filterContext.sessions?.gap_minutes ?? 120,
      })
    },
    [sessionStorageKey, setSessions, filterContext.sessions?.gap_minutes],
  )

  // Au mount, restaurer la sélection localStorage dans le store global si
  // elle n'y est pas (cold reload de /stats).
  useEffect(() => {
    if (pickedSessionLabels.length === 0) return
    const current = filterContext.sessions?.picked_sessions ?? []
    const same =
      current.length === pickedSessionLabels.length &&
      current.every((id, i) => id === pickedSessionLabels[i])
    if (same) return
    setSessions({
      picked_sessions: pickedSessionLabels,
      gap_minutes: filterContext.sessions?.gap_minutes ?? 120,
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // mount-only

  // Post-mount : sync global → local quand picked_sessions change ailleurs
  // (rail prev/next, FilterOmnibar SessionPill, autoSnapToLatestSession).
  const mountedRef = useRef(false)
  useEffect(() => {
    if (!mountedRef.current) {
      mountedRef.current = true
      return
    }
    const globalPicked = filterContext.sessions?.picked_sessions ?? []
    const same =
      globalPicked.length === pickedSessionLabels.length &&
      globalPicked.every((v, i) => v === pickedSessionLabels[i])
    if (same) return
    setPickedSessionLabelsRaw(globalPicked)
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

  // Preview live : on force `match_context: 'solo'` pour scoper la cascade,
  // les counts et les session_options aux matchs solo uniquement (par
  // symétrie avec deriveSquadPending côté SquadLayout). Sans ce scope les
  // sessions squad pollueraient le SessionMultiSelect et les compteurs.
  const soloPending = useMemo<FilterContextInput>(
    () => ({ ...pending, match_context: 'solo' }),
    [pending],
  )
  const { data: previewResolve } = useFiltersPreview(playerSlug, soloPending)

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

  // Sessions disponibles : preview > resolved committed.
  // Filtre solo strict : `is_squad === false`. Le SessionMultiSelect, les
  // counts et le snap-to-latest n'opèrent QUE sur les sessions solo. Une
  // session squad qui arriverait dans la BDD ne fait PAS bouger le filtre
  // de cette page — par contre ça peut survenir transitoirement via le
  // auto-snap-on-sync universel de routes/players/$playerSlug.tsx (qui
  // opère sur la latest globale, peut être squad). L'effet snap-to-latest-
  // solo ci-dessous override ce cas en ramenant le filtre sur la dernière
  // solo.
  const sessionOptions = useMemo(
    () =>
      (previewResolve?.session_options?.all_sessions
        ?? resolvedContext?.session_options?.all_sessions
        ?? []).filter((s) => !s.is_squad),
    [previewResolve, resolvedContext],
  )
  const sessionLabels = useMemo(() => sessionOptions.map((s) => s.label), [sessionOptions])
  const sessionCounts = useMemo(() => {
    const map = new Map<string, number>()
    for (const s of sessionOptions) {
      map.set(s.label, s.match_count_filtered)
    }
    return map
  }, [sessionOptions])

  // Snap sur la dernière session solo — politique partagée avec /squad
  // (cf. features/filters/useSessionSnap.ts pour la sémantique complète).
  useSessionSnap({ sessions: sessionOptions, kind: 'solo' })
  const getSessionCount = useMemo(
    () => (label: string) => sessionCounts.get(label),
    [sessionCounts],
  )

  const presetCounts = previewResolve?.period_presets ?? resolvedContext?.period_presets

  // ── Saisons (cascade-aware counts + détection saison active) ─────────────
  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)
  const seasonCounts = previewResolve?.season_counts ?? resolvedContext?.season_counts

  // ── Requête synthesis (alimente SessionBriefing solo) ────────────────────
  // scopeHash = filterContextHash pour invalider le cache à chaque commit Analyser.
  const synthesisRequest: SynthesisQueryRequest = {
    filters: filterContext,
    period: 'all',
  }
  const { data: synthesisData, isLoading, isError, error } = useSynthesisPage(
    playerSlug,
    filterContextHash,
    synthesisRequest,
  )

  // ── Routes actives ───────────────────────────────────────────────────────
  const summaryRoute = '/players/$playerSlug/stats/summary' as const
  const mapsModesRoute = '/players/$playerSlug/stats/maps-modes' as const
  const distributionsRoute = '/players/$playerSlug/stats/distributions' as const
  const progressionRoute = '/players/$playerSlug/stats/progression' as const
  const advancedRoute = '/players/$playerSlug/stats/advanced' as const
  const isSummary = !!matchRoute({ to: summaryRoute, fuzzy: true })
  const isMapsModes = !!matchRoute({ to: mapsModesRoute, fuzzy: true })
  const isDistributions = !!matchRoute({ to: distributionsRoute, fuzzy: true })
  const isProgression = !!matchRoute({ to: progressionRoute, fuzzy: true })
  const isAdvanced = !!matchRoute({ to: advancedRoute, fuzzy: true })

  const tabClass = (active: boolean) =>
    `px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
      active
        ? 'border-primary text-primary'
        : 'border-transparent text-muted-foreground hover:text-foreground'
    }`

  return (
    <PersonalStatsContext.Provider
      value={{
        pageData: synthesisData ?? null,
        playerSlug,
      }}
    >
      {/* ─── Barre de filtres unifiée (sticky top-0) ─────────────────────────── */}
      <div className="sticky top-0 z-30 border-b border-border" style={{ background: 'var(--background)' }}>
        <div className="flex min-h-10 items-center gap-1.5 overflow-visible px-4 py-1.5">

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

          {/* Saison (catalog TOML kind="season") */}
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

          {/* Sessions solo (multi-select par label) */}
          {sessionLabels.length > 0 && (
            <SessionMultiSelect
              sessions={sessionLabels}
              selected={pickedSessionLabels}
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
            {t.filter.analyse}
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
            {t.filter.reset}
          </button>
        </div>
        {/* Rail de navigation période/session — sous les filtres au scroll. */}
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

      {!isLoading && !isError && !synthesisData && (
        <div className="p-6">
          <EmptyStateCard title={t.empty.noDataTitle} description={t.empty.noDataDescription} />
        </div>
      )}

      {!isLoading && !isError && synthesisData && (
        <div className="flex flex-col gap-6 p-6">
          {/* SessionBriefing — solo only (pas de prop squad). */}
          {synthesisData.solo_kpis && (
            <SessionBriefing kpis={synthesisData.solo_kpis} />
          )}

          {/* Navigation onglets */}
          <div className="border-b">
            <nav className="flex gap-0">
              <Link to={summaryRoute} params={{ playerSlug }} className={tabClass(isSummary)}>
                {t.nav.summary}
              </Link>
              <Link to={mapsModesRoute} params={{ playerSlug }} className={tabClass(isMapsModes)}>
                {t.nav.mapsModes}
              </Link>
              <Link to={distributionsRoute} params={{ playerSlug }} className={tabClass(isDistributions)}>
                {t.nav.distributions}
              </Link>
              <Link to={progressionRoute} params={{ playerSlug }} className={tabClass(isProgression)}>
                {t.nav.progression}
              </Link>
              <Link to={advancedRoute} params={{ playerSlug }} className={tabClass(isAdvanced)}>
                {t.nav.advanced}
              </Link>
            </nav>
          </div>

          <Outlet />
        </div>
      )}
    </PersonalStatsContext.Provider>
  )
}
