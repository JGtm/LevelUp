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
import { Outlet, useParams, Link, useMatchRoute, useSearch } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useTeammates } from './queries'
import { useSettings } from '@/features/settings/queries'
import { useFiltersPreview, useFiltersResolve } from '@/features/filters/queries'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { AddFriendModal } from '@/features/friends/AddFriendFlow'
import { tokenCssVar } from '@/lib/accessibility'
import { getSquadText } from './i18n'
import { SquadObjectiveStatsPanel } from './SquadObjectiveStatsPanel'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { log } from './_logger'
import { SquadContext } from './SquadContext'
import { SquadFocusStrip } from './SquadFocusStrip'
import { useSquadPresets } from './useSquadPresets'
import { getSquadTeammateColors, SQUAD_MAIN_PLAYER_TOKEN } from './colors'
import type { KPIStats, LabelValue, TeammateRow, TeammatesQueryRequest } from '@/lib/api/types'
import type { KPIStats as V2KPIStats } from './v2/types'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'
import {
  deriveSquadPending,
  reconcileSquadSessionLabels,
  decideCompositionReanchor,
  stripSessionCountSuffix,
  mergeSessionCounts,
} from './squadPending'
import { formatDataIssues } from './squadDataIssues'
import { exactCompositionDefault } from './exactComposition'

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
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'

// ─── Constantes ───────────────────────────────────────────────────────────────

const MAX_SELECTION = 3
const CHART_COLORS = getSquadTeammateColors(MAX_SELECTION)

/**
 * Le DTO local v2/types.ts::KPIStats déclare avg_offensive_conversion /
 * avg_defensive_resistance / combat_profile en `T | null` (le Go peut renvoyer
 * null), alors que le contrat OpenAPI (KPIStats) les expose en `T | undefined`.
 * SessionBriefing consomme le type du contrat → on normalise null→undefined au
 * passage (le consommateur traite déjà null et undefined comme « absent »).
 */
function toContractKpis(k: V2KPIStats): KPIStats {
  return {
    ...k,
    avg_offensive_conversion: k.avg_offensive_conversion ?? undefined,
    avg_defensive_resistance: k.avg_defensive_resistance ?? undefined,
    combat_profile: k.combat_profile ?? undefined,
  }
}

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
  const titleSlug = useTitleSlug()
  // Deep-link depuis l'accueil (card session escouade) : capturé UNE fois au montage
  // via un ref-initializer, AVANT le redirect index → /squad/synergies qui drop la
  // query. Consommé plus bas (compose = amis de la session + session pinnée).
  const squadDeepLink = useSearch({ strict: false }) as { session?: string; teammates?: string }
  const deepLinkRef = useRef<{ session: string; teammates: string[] } | null>(
    squadDeepLink.session
      ? {
          session: squadDeepLink.session,
          teammates: (squadDeepLink.teammates ?? '')
            .split(',')
            .map((g) => g.trim())
            .filter(Boolean),
        }
      : null,
  )
  const {
    filterContext,
    filterContextHash,
    resolvedContext,
    setFilterContext,
    setSessions,
    resetFilters,
    autoSnapToLatestSession,
  } = useSquadFilterStore()
  // Résout le filterContext squad côté backend → alimente `resolvedContext`
  // (session_options, available_options, period_presets) pour les pills et le rail.
  useFiltersResolve(playerSlug, useSquadFilterStore)
  // L'ancrage de session est piloté par la COMPOSITION (cf. effet de ré-ancrage
  // plus bas) — pas via useFollowLatestSession, qui snappait sur la dernière
  // session squad du joueur principal (composition-agnostique → ajoutait un
  // coéquipier à une session qu'il n'avait pas jouée).
  const locale = useAppShellStore((s) => s.locale)
  const hasLinkedIdentity = useAppShellStore((s) => !!s.linkedHaloIdentity)
  const t = getSquadText(locale)
  const tCommon = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
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

  // ── Option « composition stricte » (cochée par défaut) ───────────────────
  // Par défaut, seuls les matchs joués avec exactement cette composition sont
  // comptés. La décocher élargit à la règle « matchs commencés ensemble »
  // (intersection du roster), même si un autre joueur connu accompagnait
  // l'équipe. Choix persisté par joueur (cf. exactCompositionDefault),
  // appliqué en direct (pas de passage par Analyser, comme les
  // coéquipiers/sessions).
  const exactCompositionStorageKey = `squad-exact-composition-${playerSlug}`
  const [exactComposition, setExactCompositionRaw] = useState<boolean>(() => {
    try {
      return exactCompositionDefault(localStorage.getItem(exactCompositionStorageKey))
    } catch { return exactCompositionDefault(null) }
  })
  const setExactComposition = (value: boolean) => {
    setExactCompositionRaw(value)
    try { localStorage.setItem(exactCompositionStorageKey, String(value)) } catch { /* ignore */ }
  }

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
    // eslint-disable-next-line react-hooks/set-state-in-effect -- resync sur changement async du store de filtres global + écriture localStorage (2026-07-22)
    setPickedSquadSessionLabelsRaw(globalPicked)
    try {
      localStorage.setItem(sessionStorageKey, JSON.stringify(globalPicked))
    } catch {
      /* ignore */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterContext.sessions?.picked_sessions])

  // ── Deep-link accueil (card session escouade) ────────────────────────────
  // Une seule fois au montage : pose la composition = amis de la session puis pin
  // la session. Le suffixe « (N) » volatil est réconcilié par l'effet dédié ;
  // l'init-coéquipiers depuis settings est neutralisée (cf. deepLinkRef plus bas).
  useEffect(() => {
    const dl = deepLinkRef.current
    if (!dl) return
    setSelectedGts(dl.teammates)
    applySessionLabels([dl.session])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
    // Défense : un slice Go nil sérialise en JSON null. `?? []` empêche un
    // crash si le contrat est violé. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
    const filterUUIDs = (opts: LabelValue[] | null | undefined): LabelValue[] =>
      (opts ?? []).filter((o) => !UUID_RE.test(o.label.trim()))
    return {
      playlists: filterUUIDs(rawAvailable.playlists),
      modes: filterUUIDs(rawAvailable.modes),
      maps: filterUUIDs(rawAvailable.maps),
      experience_types: filterUUIDs(rawAvailable.experience_types),
    }
  }, [rawAvailable])

  const presetCounts = previewResolve?.period_presets ?? resolvedContext?.period_presets

  // ── Saisons (cascade-aware counts + détection saison active) ─────────────
  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)
  const seasonCounts = previewResolve?.season_counts ?? resolvedContext?.season_counts

  // ── Init coéquipiers depuis settings ────────────────────────────────────
  // Neutralisée en arrivée par deep-link (card session escouade) : la composition
  // est alors imposée par la session, pas par les amis configurés par défaut.
  useEffect(() => {
    if (deepLinkRef.current) return
    if (settings?.friend_gamertags?.length && selectedGts.length === 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- init de la composition à l'arrivée async des settings (garde deep-link + sélection vide) (2026-07-22)
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
    filter_exact_composition: exactComposition,
  }
  const { data, isLoading, isError, error, isPlaceholderData } = useTeammates(
    playerSlug,
    request,
    filterContextHash,
    confirmedGts,
    pickedSquadSessionLabels,
  )

  // Sessions à afficher (multi-select) + référence pour le ré-ancrage.
  // Avec coéquipier(s) : sessions de la COMPOSITION EXACTE (intersection back-end,
  // historique complet). Sans coéquipier : sessions squad du joueur principal.
  // On ne retombe PAS sur les sessions du main quand l'intersection est vide
  // (sinon on afficherait des sessions non jouées par la composition).
  const hasTeammates = confirmedGts.length > 0
  const compositionSessions = useMemo(
    () => (hasTeammates ? (data?.composition_sessions ?? []) : (data?.session_labels?.squad ?? [])),
    [hasTeammates, data],
  )
  const latestCompositionSession = data?.latest_composition_session ?? ''

  // Counts par session label — alimente SessionMultiSelect (masque les sessions
  // vides + affiche le compte). SOURCE UNIQUE en contexte escouade : le compte
  // « commencés ensemble » servi par teammates (composition_sessions.match_count),
  // c'est-à-dire exactement la population des tableaux et graphes de la page.
  // Les counts de /filters/resolve (population du joueur principal, cascade
  // seule) ne servent plus que de repli tant que la réponse teammates n'est pas
  // arrivée — c'est cette double source qui donnait 11/8/6/5 sur une même session.
  const sessionCounts = useMemo(
    () =>
      mergeSessionCounts(
        previewResolve?.session_options?.all_sessions ?? resolvedContext?.session_options?.all_sessions ?? [],
        compositionSessions,
      ),
    [previewResolve, resolvedContext, compositionSessions],
  )
  const getSessionCount = useMemo(
    () => (label: string) => sessionCounts.get(label),
    [sessionCounts],
  )

  // Dégradations remontées par l'API (chargements best-effort en échec) :
  // affichées telles quelles — un chiffre partiel doit se voir, pas se deviner.
  const dataIssueMessages = useMemo(() => formatDataIssues(data?.data_issues, t), [data?.data_issues, t])

  // Bouton "Voir les matchs" (L2) : la liste de matchs de la composition est
  // déjà chargée ici (data.match_history, DESC), donc zéro aller-retour serveur.
  // On reproduit exactement la logique de SquadMatchHistoryTable : matchIds en
  // ordre chronologique (oldest-first), point d'entrée = match le plus récent.
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const squadMatchIds = useMemo(
    () => [...(data?.match_history ?? [])].reverse().map((m) => m.match_id),
    [data?.match_history],
  )
  const squadEntryMatchId = data?.match_history?.[0]?.match_id
  const handleBrowseMatches = () => {
    if (!squadEntryMatchId) return
    const filterSpec = filterContextToMatchFilterSpec(squadFilterContext)
    navigateToMatch(squadEntryMatchId, {
      source: 'session',
      matchIds: squadMatchIds,
      filterSpec: filterSpec ?? undefined,
    })
  }
  // Bouton « Voir les matchs » — déplacé dans le rail (zone centrale, après le
  // compteur de matchs) pour décharger la barre de filtres.
  const browseButton =
    squadEntryMatchId && (totalAfter ?? 0) > 0 ? (
      <button
        type="button"
        onClick={handleBrowseMatches}
        className="shrink-0 inline-flex items-center gap-1 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted"
        title={tCommon('common.filters.browse_title')}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 16 16"
          fill="currentColor"
          className="h-3.5 w-3.5 opacity-70"
          aria-hidden="true"
        >
          <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
          <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
        </svg>
        {tCommon('common.filters.browse_label')}
      </button>
    ) : null

  // Réconciliation anti-zombie des sessions pickées (suffixe " (N)" volatil, cf.
  // buildSessionLabel côté Go). On remappe chaque label pické vers sa forme
  // courante dans compositionSessions et on droppe les doublons. Si TOUS les
  // labels sont des zombies pour la composition courante, on ne fait rien : le
  // ré-ancrage composition (effet suivant) reprend la main proprement.
  useEffect(() => {
    if (compositionSessions.length === 0 || pickedSquadSessionLabels.length === 0) return
    const reconciled = reconcileSquadSessionLabels(pickedSquadSessionLabels, compositionSessions)
    if (reconciled.length === 0) return
    const unchanged =
      reconciled.length === pickedSquadSessionLabels.length &&
      reconciled.every((l, i) => l === pickedSquadSessionLabels[i])
    // eslint-disable-next-line react-hooks/set-state-in-effect -- réconciliation des labels de session sur arrivée async de compositionSessions (2026-07-22)
    if (!unchanged) applySessionLabels(reconciled)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [compositionSessions])

  // ── Ré-ancrage composition-aware ─────────────────────────────────────────
  // Quand la composition change (ajout/retrait d'un coéquipier), on retombe sur
  // la dernière session où exactement cette composition a joué ensemble
  // (data.latest_composition_session, calculé back-end sur l'intersection), ou un
  // état vide si elle n'a jamais joué ensemble.
  //
  // Anti-boucle : on n'agit qu'UNE fois par couple (composition, dernière session)
  // — ref — et seulement sur des données FRAÎCHES (!isPlaceholderData = data
  // correspond à la sélection courante, pas au placeholder keepPreviousData). La
  // navigation rail/manuelle et les changements de filtre ne changent ni la
  // composition ni la dernière session → pas de ré-ancrage, donc pas de conflit
  // avec une sélection de session délibérée.
  //
  // La clé de garde inclut la DERNIÈRE SESSION (clé sans le suffixe « (N) »
  // volatil) : avec la composition seule, l'arrivée d'une nouvelle soirée pendant
  // que la page est montée (refetch post-sync) ne rouvrait jamais la décision —
  // l'escouade ne monte pas useFollowLatestSession (seul le solo l'a).
  const lastAnchoredCompositionRef = useRef<string | null>(null)
  useEffect(() => {
    if (!data || isPlaceholderData) return
    const compositionKey = hasTeammates ? [...confirmedGts].sort().join(',') : ''
    const anchorKey = `${compositionKey}|${stripSessionCountSuffix(latestCompositionSession)}`
    if (anchorKey === lastAnchoredCompositionRef.current) return
    lastAnchoredCompositionRef.current = anchorKey

    const {
      filterContext: fc,
      isAutoSnappingToLatest,
      lastKnownLatestSessionId,
      setLastKnownLatestSessionId,
    } = useSquadFilterStore.getState()
    const picked = fc.sessions?.picked_sessions ?? []
    const hasPeriod = !!(fc.period?.start_date || fc.period?.end_date)
    // « follow-latest » : pas de sélection manuelle épinglée (cf. useFollowLatestSession).
    const followLatest = isAutoSnappingToLatest || (!hasPeriod && picked.length === 0)
    const action = decideCompositionReanchor({
      hasTeammates,
      followLatest,
      latestCompositionSession,
      pickedSessions: picked,
      compositionSessionLabels: compositionSessions.map((s) => s.label),
      // Ancrage déjà posé (persisté) : distingue « l'utilisateur a épinglé cette
      // session » de « une nouvelle session est arrivée depuis ».
      lastAnchoredLatestSession: lastKnownLatestSessionId ?? '',
    })
    if (action.kind === 'clear') {
      // Composition sans session commune → on vide et on affiche l'état vide
      // (le backend logge déjà composition_resolved avec composition_sessions=0).
      // eslint-disable-next-line react-hooks/set-state-in-effect -- ré-ancrage composition sur arrivée async de data (dispatch store), pas un dérivé synchrone (2026-07-22)
      applySessionLabels([])
    } else if (action.kind === 'snap') {
      autoSnapToLatestSession({ session_id: action.label, label: action.label }, true)
    } else if (latestCompositionSession && latestCompositionSession !== lastKnownLatestSessionId) {
      // Pas de snap (déjà dessus, ou sélection délibérée respectée) : on mémorise
      // quand même la dernière session vue, sinon elle resterait « jamais ancrée »
      // et re-déclencherait un snap à chaque montage.
      setLastKnownLatestSessionId(latestCompositionSession)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, isPlaceholderData, confirmedGts, hasTeammates, latestCompositionSession])

  // ── Routes actives ───────────────────────────────────────────────────────
  const synergiesRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/contributions' as const
  const dynamiqueRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })
  const isDynamique = !!matchRoute({ to: dynamiqueRoute, fuzzy: true })

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

  // Labels des playlists/modes du filtre courant → tri-en-tête des escouades dont
  // les contextes habituels matchent (indice souple).
  const activeContextLabels = useMemo(() => {
    const labels: string[] = []
    const collect = (opts: LabelValue[] | undefined, sel: string[] | undefined) => {
      if (!opts || !sel || sel.length === 0) return
      const selSet = new Set(sel)
      for (const o of opts) if (selSet.has(o.value)) labels.push(o.label)
    }
    collect(available?.playlists, pendingCascade.playlists as string[] | undefined)
    collect(available?.modes, pendingCascade.modes as string[] | undefined)
    return labels
  }, [available, pendingCascade])

  // XUID absolu du joueur courant — résolu depuis la card "moi" du header
  // (player_cards filtré sur main_player). Sert à exclure le viewer du roster de
  // façon player-agnostic (jamais par le slug d'URL). Vide tant que le header
  // squad n'est pas chargé (dégradation gracieuse transitoire).
  const currentPlayerXuid = useMemo(() => {
    const mainGT = data?.main_player ?? ''
    return data?.header?.player_cards?.find((c) => c.gamertag === mainGT)?.xuid ?? ''
  }, [data])

  // Presets du combobox : escouades sauvegardées + groupes d'accès (charger un
  // roster), + footer de gestion (enregistrer / renommer / supprimer).
  const {
    presetGroups: squadPresetGroups,
    footer: squadPresetFooter,
    onClose: squadPresetOnClose,
  } = useSquadPresets({
    playerSlug,
    currentPlayerXuid,
    hasLinkedIdentity,
    locale,
    selectedRows,
    activeContextLabels,
  })

  return (
    <SquadContext.Provider
      value={{
        selectedRows,
        confirmedGamertags: confirmedGts,
        pageData: data ?? null,
        playerSlug,
        currentPlayerXuid,
      }}
    >
      {/* ─── Barre de filtres unifiée (sticky top-12, remplace NavL2/FilterOmnibar) ─── */}
      <div className="sticky top-0 z-30 px-6" style={{ background: 'var(--background)' }}>
        <div className="flex min-h-10 items-center gap-1.5 overflow-visible border-b border-border py-1.5">

          {/* Joueur actif (pill de tête non-supprimable) + coéquipiers (multi-select
              compact inline, jusqu'à 3). La pill du joueur actif est rendue DANS le
              combobox (leadingPill) → même ligne flex que les pills coéquipiers, donc
              alignement vertical garanti. Le popover intègre les presets « Mes
              escouades » (charger/gérer une compo) et « Mes groupes ». */}
          <GamertagCombobox
            compact
            leadingPill={{ label: playerSlug, color: tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN) }}
            selected={selectedGts}
            onChange={setSelectedGts}
            max={MAX_SELECTION}
            frequentOptions={availableOptions}
            colors={CHART_COLORS}
            excludeGamertag={playerSlug}
            placeholder={t.selection.placeholder(availableOptions.length)}
            onAddAsFriend={setAddFriendGamertag}
            presetGroups={squadPresetGroups}
            onLoadPreset={(gts) =>
              // Dédup vs la pill de tête (leadingPill = joueur courant) : un roster
              // legacy peut encore contenir le viewer → on le retire avant sélection.
              setSelectedGts(
                gts.filter((g) => g.toLowerCase() !== playerSlug.toLowerCase()).slice(0, MAX_SELECTION),
              )
            }
            footer={squadPresetFooter}
            onClose={squadPresetOnClose}
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
              seasonCounts={seasonCounts ?? undefined}
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
            presetCounts={presetCounts ?? undefined}
          />

          {/* Sessions escouade (multi-select par label) — composition-aware */}
          {compositionSessions.length > 0 && (
            <SessionMultiSelect
              sessions={compositionSessions}
              selected={pickedSquadSessionLabels}
              onChange={applySessionLabels}
              locale={locale}
              triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
              getMatchCount={getSessionCount}
            />
          )}

          {/* Composition stricte — option cochée par défaut : la règle affichée est
              « exactement cette composition » ; la décocher élargit aux matchs
              commencés ensemble. Appliquée en direct (pas d'Analyser). */}
          {hasTeammates && (
            <label
              className="shrink-0 inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted"
              title={t.filter.exactCompositionTitle}
            >
              <input
                type="checkbox"
                className="h-3 w-3 rounded border-input text-primary focus:ring-1 focus:ring-ring"
                checked={exactComposition}
                onChange={(e) => setExactComposition(e.target.checked)}
              />
              {t.filter.exactComposition}
            </label>
          )}

          <div className="flex-1" />

          {/* Compteur de matchs + « Voir les matchs » : déplacés dans le rail
              ci-dessous (zone centrale) pour éviter le doublon avec son compteur
              et décharger la barre. */}

          {/* Analyser — applique les filtres en attente (cascade + période).
              Les coéquipiers et sessions, eux, s'appliquent en direct. */}
          <button
            type="button"
            onClick={handleAnalyser}
            title={tCommon('common.filters.apply_pending_title')}
            className={[
              'shrink-0 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
              isDirty
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'border border-input bg-background text-muted-foreground hover:bg-muted',
            ].join(' ')}
          >
            Analyser
          </button>

          {/* Réinitialiser — bouton (aligné visuellement sur Analyser). */}
          <button
            type="button"
            onClick={() => {
              resetFilters()
              applySessionLabels([])
            }}
            className="shrink-0 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-destructive"
            title={tCommon('common.filters.reset_title')}
          >
            {tCommon('common.filters.reset_label')}
          </button>
        </div>
        {/* Rail de navigation période/session — placé DANS la barre sticky pour
            apparaître toujours juste sous les filtres Squad au scroll. Reçoit le
            compteur de matchs (tous modes) + le bouton « Voir les matchs ». */}
        <PeriodSessionRail filterStore={useSquadFilterStore} matchCount={totalAfter} trailing={browseButton} />
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

      {/* Données partielles : un chargement best-effort a échoué côté serveur.
          Visible plutôt que silencieux — sinon les compteurs varient d'une
          requête à l'autre sans explication (cf. domain.DataIssue).
          text-warning (et non text-warning-foreground, pensé pour un fond plein) :
          sur le tint bg-warning/10 ce dernier est illisible en thème sombre —
          piège documenté dans components/ui/privacy-banner.tsx. */}
      {dataIssueMessages.length > 0 && (
        <div
          role="status"
          className="mx-4 mt-3 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning"
        >
          <p className="font-medium">{t.dataIssues.title}</p>
          <ul className="mt-1 list-disc pl-4">
            {dataIssueMessages.map((msg) => (
              <li key={msg}>{msg}</li>
            ))}
          </ul>
        </div>
      )}

      {!isLoading && !isError && !data && (
        <div className="p-6">
          <EmptyStateCard title={t.empty.noDataTitle} description={t.empty.noDataDescription} />
        </div>
      )}

      {/* Coéquipier(s) sélectionné(s) mais aucune session commune : la composition
          exacte n'a jamais joué ensemble (sur le scope filtré) → état vide clair,
          pas les stats du joueur principal. */}
      {!isLoading && !isError && data && hasTeammates && compositionSessions.length === 0 && (
        <div className="p-6">
          <EmptyStateCard
            title={t.empty.invalidSelectionTitle}
            description={t.empty.invalidSelectionDescription}
          />
        </div>
      )}

      {!isLoading && !isError && data && !(hasTeammates && compositionSessions.length === 0) && (
        <div className="flex flex-col gap-6 p-6">
          {/* SessionBriefing — KPIs + verdict squad + drill-down click */}
          {/* Remplace l'ancienne section "Synergies avec les coéquipiers sélectionnés"
             (KPIBlock par teammate) — meme info accessible via le drill-down click sur
             la card du joueur dans la bande verdict. */}
          {data?.header?.solo_kpis && (() => {
            const header = data.header
            const soloKpis = header.solo_kpis
            if (!soloKpis) return null
            // Réutilise le XUID courant déjà mémoïsé (source unique, cf.
            // currentPlayerXuid) plutôt que de le recalculer inline.
            const briefingSquad =
              header?.squad_score &&
              header?.player_cards &&
              header?.team_avg_kpis &&
              header?.kpis_by_xuid &&
              currentPlayerXuid
                ? {
                    score: header.squad_score,
                    players: header.player_cards,
                    kpisByXuid: Object.fromEntries(
                      Object.entries(header.kpis_by_xuid).map(([x, k]) => [x, toContractKpis(k)]),
                    ),
                    teamAvgKpis: toContractKpis(header.team_avg_kpis),
                    activeXuid: currentPlayerXuid,
                  }
                : undefined
            return <SessionBriefing kpis={toContractKpis(soloKpis)} squad={briefingSquad} />
          })()}

          {/* Objectifs de l'escouade (CTF/Zones/Oddball) — capability-gated + data-driven. */}
          <SquadObjectiveStatsPanel
            statsByXuid={data?.header?.objective_stats_by_xuid}
            texts={t}
            numLoc={t.intlLocale}
          />

          {/* « Cap d'escouade » (Enregistrer cette compo) — remonté AU-DESSUS de la
              barre d'onglets L3, commun aux deux onglets (Synergies / Contributions). */}
          <SquadFocusStrip />

          {/* Navigation onglets */}
          <div className="border-b">
            <nav className="flex gap-0">
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isSynergies ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.synergies}
              </Link>
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/contributions"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isContributions ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.contributions}
              </Link>
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isDynamique ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.dynamique}
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
