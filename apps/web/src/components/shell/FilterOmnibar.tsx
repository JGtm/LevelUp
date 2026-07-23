/**
 * FilterOmnibar — bandeau de filtres en pills (h-9) avec popovers ciblés.
 *
 * Ordre d'affichage :
 *   [Filtres ▾]  |  [Toutes les périodes ▾]  [Toutes les sessions ▾]  [Analyser]
 *   ← données →      ←————————— scope temporel —————————→
 *
 * Les changements sont "pending" : les pills opèrent sur un état local.
 * Le bouton "Analyser" commit l'état local vers le store (un seul re-fetch).
 * Les changements externes (auto-snap, reset) resynchronisent l'état local.
 */
import { useEffect, useMemo, useRef, useState } from 'react'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useSessionContextStore } from '@/stores/sessionContextStore'
import type { FilterStore } from '@/stores/createFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFiltersPreview } from '@/features/filters/queries'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import type { CascadeInput, FilterContextInput, LabelValue, PeriodInput, SessionLabelEntry } from '@/lib/api/types'
import { FiltresPill } from './_filter_pills/FiltresPill'
import { PeriodePill } from './_filter_pills/PeriodePill'
import { SaisonPill } from './_filter_pills/SaisonPill'
import { ShareLinkButton } from './_filter_pills/ShareLinkButton'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import { api } from '@/lib/api/client'
import type { FilterMatchIdsResponse } from '@/lib/api/types'
import {
  DEFAULT_CASCADE,
  DEFAULT_PERIOD,
  DEFAULT_SESSIONS,
  computePendingHash,
  isUUIDLabel,
} from './_filter_pills/_hooks'
import type { Locale } from '@/lib/i18n/locale'

// ─── Re-exports pour les consommateurs externes (SquadLayout, SquadV2RouteHost…) ───
// Ces re-exports cohabitent intentionnellement avec FilterOmnibar pour servir
// de point d'entrée unique aux consommateurs. Le coût HMR (fast refresh sur
// les composants) est accepté tant que ce module reste petit.
/* eslint-disable react-refresh/only-export-components */

export { FiltresPill, type FiltresPillProps } from './_filter_pills/FiltresPill'
export { PeriodePill } from './_filter_pills/PeriodePill'
export { SessionPill } from './_filter_pills/SessionPill'
export { SaisonPill } from './_filter_pills/SaisonPill'
export { CheckboxGroup } from './_filter_pills/CheckboxGroup'
export {
  DEFAULT_CASCADE,
  DEFAULT_PERIOD,
  DEFAULT_SESSIONS,
  PERIOD_PRESETS,
  computePendingHash,
  detectActivePreset,
  isoDate,
  presetPeriod,
  useDismissable,
} from './_filter_pills/_hooks'
export type { PresetId } from './_filter_pills/_hooks'

/* eslint-enable react-refresh/only-export-components */

// ─── Composant principal ──────────────────────────────────────────────────────

type ActivePopover = 'session' | 'periode' | 'saison' | 'filtres' | null

interface FilterOmnibarProps {
  /** Restreint la population de matchs au contexte de la page (solo/squad/all).
   *  Injecté sur le preview et sur le filterContext commité, mais PAS persisté
   *  dans le store global (contexte page, pas contexte utilisateur). */
  matchContext?: 'solo' | 'squad' | 'all'
  /** Store de filtres à utiliser. Défaut : `useSoloFilterStore` (compat
   *  rétroactive avec NavL2/Stats Solo). SquadLayout injecte
   *  `useSquadFilterStore` quand il consomme ce composant. */
  filterStore?: FilterStore
  /** Affiche le sélecteur de contexte (Solo / Escouade / Mixte) et fait piloter
   *  le contexte par `useSessionContextStore` au lieu du prop `matchContext`.
   *  Activé UNIQUEMENT sur la page Sessions — ailleurs, le contexte reste figé
   *  sur `matchContext` (aucune régression sur Timeseries/History). */
  contextSelectable?: boolean
}

// Libellés du sélecteur de contexte (FR/EN). Module-level pour éviter une
// recréation par rendu.
const CONTEXT_LABELS: Record<Locale, Record<'solo' | 'squad' | 'all', string>> = {
  fr: { solo: 'Solo', squad: 'Escouade', all: 'Mixte' },
  en: { solo: 'Solo', squad: 'Squad', all: 'Mixed' },
}
const CONTEXT_ORDER = ['solo', 'squad', 'all'] as const

export function FilterOmnibar({ matchContext, filterStore = useSoloFilterStore, contextSelectable = false }: FilterOmnibarProps = {}) {
  const filterContext = filterStore((s) => s.filterContext)
  const filterContextHash = filterStore((s) => s.filterContextHash)
  const resolvedContext = filterStore((s) => s.resolvedContext)
  const setFilterContext = filterStore((s) => s.setFilterContext)
  const resetFilters = filterStore((s) => s.resetFilters)
  const buildShareUrl = filterStore((s) => s.buildShareUrl)
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  const locale = useAppShellStore((s) => s.locale)
  const tCommon = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // État local (pending) — commité vers le store uniquement sur "Analyser".
  const [pending, setPending] = useState<FilterContextInput>(() => filterContext)

  // Contexte de match effectif : piloté par useSessionContextStore UNIQUEMENT
  // quand le sélecteur est actif (page Sessions). Sinon figé sur le prop
  // `matchContext` → comportement Timeseries/History inchangé.
  const sessionMatchContext = useSessionContextStore((s) => s.matchContext)
  const setSessionMatchContext = useSessionContextStore((s) => s.setMatchContext)
  const effectiveContext = contextSelectable ? sessionMatchContext : matchContext

  // Preview live : résout les options disponibles pour le pending courant,
  // sans attendre "Analyser". Le contexte effectif est injecté ici pour
  // que les counts et session_options soient scoped correctement.
  const previewPending = useMemo<FilterContextInput>(
    () => effectiveContext ? { ...pending, match_context: effectiveContext } : pending,
    [pending, effectiveContext],
  )
  const { data: previewData, isFetching: isPreviewFetching } = useFiltersPreview(playerSlug, previewPending)

  // Feedback visuel après clic sur Analyser.
  const [justAnalysed, setJustAnalysed] = useState(false)

  // Sync depuis le store quand un changement externe arrive (auto-snap, reset).
  const lastSyncedHash = useRef(filterContextHash)
  useEffect(() => {
    if (filterContextHash !== lastSyncedHash.current) {
      lastSyncedHash.current = filterContextHash
      setPending(filterContext)
    }
  }, [filterContextHash, filterContext])

  const [active, setActive] = useState<ActivePopover>(null)
  const togglePopover = (which: ActivePopover) =>
    setActive((cur) => (cur === which ? null : which))
  const closeAll = () => setActive(null)

  const allSessions = useMemo(
    () =>
      previewData?.session_options?.all_sessions ??
      resolvedContext?.session_options?.all_sessions ??
      [],
    [previewData?.session_options?.all_sessions, resolvedContext?.session_options?.all_sessions],
  )
  // Filtre solo/squad du sélecteur de sessions, piloté par le contexte effectif.
  // Défaut 'solo' (ou contexte non défini) → on n'expose que les sessions solo,
  // comportement historique. 'squad' → sessions escouade ; 'all' → toutes.
  const sessionMatchesContext = (isSquad: boolean): boolean => {
    if (effectiveContext === 'squad') return isSquad
    if (effectiveContext === 'all') return true
    return !isSquad
  }
  const sessionLabels = useMemo<SessionLabelEntry[]>(
    () => allSessions
      .filter((s) => sessionMatchesContext(s.is_squad))
      .map((s) => ({ label: s.label, started_at: s.started_at_utc ?? '', ended_at: s.ended_at_utc ?? '' })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [allSessions, effectiveContext],
  )
  const sessionMatchCount = useMemo(() => {
    const map = new Map<string, number>()
    for (const s of allSessions) {
      if (sessionMatchesContext(s.is_squad)) map.set(s.label, s.match_count_filtered)
    }
    return map
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allSessions, effectiveContext])
  const getSessionCount = useMemo(
    () => (label: string) => sessionMatchCount.get(label),
    [sessionMatchCount],
  )
  const presetCounts = previewData?.period_presets ?? resolvedContext?.period_presets
  const pendingCascade = (pending.cascade ?? DEFAULT_CASCADE) as CascadeInput
  const pendingPeriod = pending.period
  const pendingPickedLabels = pending.sessions?.picked_sessions ?? []

  // Saisons (catalog TOML kind="season") — détection saison active + counts
  // cascade-aware. Symétrie avec SquadLayout.
  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)
  const seasonCounts = previewData?.season_counts ?? resolvedContext?.season_counts

  const isDirty = filterContextHash !== computePendingHash(pending)

  const cascadeCount = (['playlists', 'modes', 'maps', 'experience_types'] as const)
    .reduce((n, k) => n + ((pendingCascade[k] as string[] | undefined)?.length ?? 0), 0)
  const hasPeriod = !!(pendingPeriod?.start_date || pendingPeriod?.end_date)
  const hasActiveFilters = pendingPickedLabels.length > 0 || hasPeriod || cascadeCount > 0
  // Priorité au preview (live) pour que le compteur reflète le pending —
  // sinon il reste figé sur le dernier commit jusqu'au clic Analyser.
  const totalAfter =
    (previewData?.counts ?? resolvedContext?.counts)?.total_matches_after_filters ?? null

  // Bouton "Voir les matchs" : on récupère à la volée la liste ORDONNÉE des
  // match_id de la sélection (POST /filters/match-ids — même pipeline que le
  // compteur, donc match_context/sessions/cascade respectés), puis on ouvre le
  // 1er match en passant la liste explicite. Le parcours prev/next reste ainsi
  // exact, là où /neighbors (shared-only) ne sait pas filtrer solo/squad.
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const [isBrowsing, setIsBrowsing] = useState(false)
  const canBrowse = (totalAfter ?? 0) > 0
  const handleBrowseMatches = async () => {
    if (isBrowsing || !canBrowse) return
    // Même contexte effectif que le compteur affiché (pending si preview dispo),
    // avec match_context injecté comme le fait previewPending.
    const base = previewData ? previewPending : filterContext
    const sourceCtx = effectiveContext ? { ...base, match_context: effectiveContext } : base
    setIsBrowsing(true)
    try {
      const { match_ids } = await api.post<FilterMatchIdsResponse>(
        `/players/${playerSlug}/filters/match-ids`,
        sourceCtx,
      )
      if (match_ids.length === 0) return
      const filterSpec = filterContextToMatchFilterSpec(sourceCtx)
      navigateToMatch(match_ids[0], {
        source: 'history',
        matchIds: match_ids,
        filterSpec: filterSpec ?? undefined,
      })
    } catch {
      // Fail-open : pas de navigation si l'appel échoue (l'utilisateur réessaie).
      // Un 401 déclenche déjà le flux de réauth global via l'event auth-required.
    } finally {
      setIsBrowsing(false)
    }
  }

  // Priorité au preview (temps réel) ; fallback sur le contexte commité
  // tant que le premier fetch preview n'est pas revenu.
  const rawAvailable = previewData?.available_options ?? resolvedContext?.available_options
  const available = useMemo(() => {
    if (!rawAvailable) return undefined
    // Défense : un slice nil Go sérialise en JSON null. `?? []` garantit qu'on
    // ne crashe pas si le backend dérape sur le contrat. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
    const filterUUIDs = (opts: LabelValue[] | null | undefined): LabelValue[] =>
      (opts ?? []).filter((o) => !isUUIDLabel(o.label))
    return {
      playlists: filterUUIDs(rawAvailable.playlists),
      modes: filterUUIDs(rawAvailable.modes),
      maps: filterUUIDs(rawAvailable.maps),
      experience_types: filterUUIDs(rawAvailable.experience_types),
    }
  }, [rawAvailable])

  function setPendingPeriod(p: PeriodInput) {
    const isPeriodSet = !!(p?.start_date || p?.end_date)
    setPending((prev) => ({
      ...prev,
      period: p,
      filter_mode: isPeriodSet ? 'period' : 'sessions',
      sessions: isPeriodSet ? DEFAULT_SESSIONS : prev.sessions,
    }))
  }

  function setPendingSessionLabels(labels: string[]) {
    setPending((prev) => ({
      ...prev,
      sessions: { ...(prev.sessions ?? DEFAULT_SESSIONS), picked_sessions: labels },
      filter_mode: labels.length > 0 ? 'sessions' : 'period',
      period: labels.length > 0 ? DEFAULT_PERIOD : prev.period,
    }))
  }

  function setPendingCascade(c: CascadeInput) {
    setPending((prev) => ({ ...prev, cascade: c }))
  }

  function handleAnalyser() {
    // Purger les zombies avant de committer : évite de committer une combinaison
    // incompatible quand le preview a déjà identifié les conflits.
    let committed = pending
    if (previewData) {
      const av = previewData.available_options
      // Défense `?? []` : même raison que `filterUUIDs` plus haut.
      const avSets = {
        playlists: new Set((av.playlists ?? []).map((o) => o.value)),
        modes: new Set((av.modes ?? []).map((o) => o.value)),
        maps: new Set((av.maps ?? []).map((o) => o.value)),
        experience_types: new Set((av.experience_types ?? []).map((o) => o.value)),
      }
      const c = (pending.cascade ?? DEFAULT_CASCADE) as CascadeInput
      const cleanCascade: CascadeInput = {
        playlists: ((c.playlists ?? []) as string[]).filter((v) => avSets.playlists.has(v)),
        modes: ((c.modes ?? []) as string[]).filter((v) => avSets.modes.has(v)),
        maps: ((c.maps ?? []) as string[]).filter((v) => avSets.maps.has(v)),
        experience_types: ((c.experience_types ?? []) as string[]).filter((v) =>
          avSets.experience_types.has(v),
        ),
      }
      committed = { ...pending, cascade: cleanCascade }
      setPending(committed)
    }
    setFilterContext(committed)
    lastSyncedHash.current = computePendingHash(committed)
    setJustAnalysed(true)
    setTimeout(() => setJustAnalysed(false), 1800)
  }

  return (
    <div
      className="flex h-9 items-center gap-2 overflow-visible px-4"
      role="toolbar"
      aria-label={tCommon('common.filters.pill_label')}
    >
      {available && (
        <FiltresPill
          open={active === 'filtres'}
          onToggle={() => togglePopover('filtres')}
          onClose={closeAll}
          available={available}
          cascade={pendingCascade}
          cascadeCount={cascadeCount}
          onSetCascade={setPendingCascade}
          isFetching={isPreviewFetching}
        />
      )}

      {/* Sélecteur de contexte (Solo / Escouade / Mixte) — page Sessions uniquement.
          Pilote useSessionContextStore ; n'affecte pas le filterContext partagé. */}
      {contextSelectable && (
        <div
          className="flex shrink-0 items-center gap-0.5 rounded-md border border-input bg-background p-0.5 text-xs"
          role="group"
          aria-label={tCommon('common.filters.context_aria')}
        >
          {CONTEXT_ORDER.map((ctx) => (
            <button
              key={ctx}
              type="button"
              onClick={() => setSessionMatchContext(ctx)}
              aria-pressed={(effectiveContext ?? 'solo') === ctx}
              className={[
                'rounded px-2 py-0.5 transition-colors',
                (effectiveContext ?? 'solo') === ctx
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              ].join(' ')}
            >
              {CONTEXT_LABELS[locale === 'en' ? 'en' : 'fr'][ctx]}
            </button>
          ))}
        </div>
      )}

      {/* Séparateur données / scope temporel */}
      <div className="mx-1 h-4 w-px shrink-0 bg-border" aria-hidden />

      {seasons.length > 0 && (
        <SaisonPill
          open={active === 'saison'}
          onToggle={() => togglePopover('saison')}
          onClose={closeAll}
          seasons={seasons}
          activeSeason={activeSeason}
          seasonCounts={seasonCounts ?? undefined}
          onSelectSeason={(s) => setPendingPeriod(seasonToPeriod(s))}
          onClear={() => setPendingPeriod(DEFAULT_PERIOD)}
        />
      )}

      <PeriodePill
        open={active === 'periode'}
        onToggle={() => togglePopover('periode')}
        onClose={closeAll}
        period={pendingPeriod}
        onSetPeriod={setPendingPeriod}
        presetCounts={presetCounts ?? undefined}
      />

      {sessionLabels.length > 0 && (
        <SessionMultiSelect
          sessions={sessionLabels}
          selected={pendingPickedLabels}
          onChange={setPendingSessionLabels}
          locale={locale}
          triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
          getMatchCount={getSessionCount}
        />
      )}

      <div className="flex-1" />

      {totalAfter !== null && (
        <span
          className="shrink-0 text-xs text-muted-foreground"
          aria-live="polite"
          title={tCommon('common.filters.matches_count_title')}
        >
          {formatMessage(commonManifest, 'common.filters.matches_count', locale, { count: totalAfter })}
        </span>
      )}

      {canBrowse && (
        <button
          type="button"
          onClick={handleBrowseMatches}
          disabled={isBrowsing}
          className="shrink-0 inline-flex items-center gap-1 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-60"
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
      )}

      <button
        type="button"
        onClick={handleAnalyser}
        disabled={isPreviewFetching}
        className={[
          'shrink-0 rounded-md px-3 py-1 text-xs font-medium transition-colors',
          justAnalysed
            ? 'border border-input bg-background text-foreground'
            : isDirty
              ? 'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-60'
              : 'border border-input bg-background text-muted-foreground hover:bg-muted',
        ].join(' ')}
      >
        {justAnalysed ? tCommon('common.filter.applied') : isPreviewFetching ? '…' : tCommon('common.filter.analyser')}
      </button>

      {hasActiveFilters && (
        <button
          type="button"
          onClick={() => {
            resetFilters()
          }}
          className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
          title={tCommon('common.filters.reset_title')}
        >
          {tCommon('common.filters.reset_label')}
        </button>
      )}

      {/* Copier le lien avec les filtres — à la demande (le share-link n'est plus
          écrit automatiquement). Masqué si le store n'a pas de share-link (escouade). */}
      <ShareLinkButton buildShareUrl={buildShareUrl} />
    </div>
  )
}
