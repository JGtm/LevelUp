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
import type { FilterStore } from '@/stores/createFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFiltersPreview } from '@/features/filters/queries'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import type { CascadeInput, FilterContextInput, LabelValue, PeriodInput, SessionLabelEntry } from '@/lib/api/types'
import { FiltresPill } from './_filter_pills/FiltresPill'
import { PeriodePill } from './_filter_pills/PeriodePill'
import { SaisonPill } from './_filter_pills/SaisonPill'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import {
  DEFAULT_CASCADE,
  DEFAULT_PERIOD,
  DEFAULT_SESSIONS,
  computePendingHash,
  isUUIDLabel,
} from './_filter_pills/_hooks'

// ─── Re-exports pour les consommateurs externes (SquadLayout, SquadV2RouteHost…) ───

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
}

export function FilterOmnibar({ matchContext, filterStore = useSoloFilterStore }: FilterOmnibarProps = {}) {
  const filterContext = filterStore((s) => s.filterContext)
  const filterContextHash = filterStore((s) => s.filterContextHash)
  const resolvedContext = filterStore((s) => s.resolvedContext)
  const setFilterContext = filterStore((s) => s.setFilterContext)
  const resetFilters = filterStore((s) => s.resetFilters)
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  const locale = useAppShellStore((s) => s.locale)

  // État local (pending) — commité vers le store uniquement sur "Analyser".
  const [pending, setPending] = useState<FilterContextInput>(() => filterContext)
  // Preview live : résout les options disponibles pour le pending courant,
  // sans attendre "Analyser". Le matchContext de la page est injecté ici pour
  // que les counts et session_options soient scoped correctement.
  const previewPending = useMemo<FilterContextInput>(
    () => matchContext ? { ...pending, match_context: matchContext } : pending,
    [pending, matchContext],
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

  const allSessions =
    previewData?.session_options?.all_sessions ??
    resolvedContext?.session_options?.all_sessions ??
    []
  const sessionLabels = useMemo<SessionLabelEntry[]>(
    () => allSessions
      .filter((s) => !s.is_squad)
      .map((s) => ({ label: s.label, started_at: s.started_at_utc ?? '', ended_at: s.ended_at_utc ?? '' })),
    [allSessions],
  )
  const sessionMatchCount = useMemo(() => {
    const map = new Map<string, number>()
    for (const s of allSessions) {
      if (!s.is_squad) map.set(s.label, s.match_count_filtered)
    }
    return map
  }, [allSessions])
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

  // Priorité au preview (temps réel) ; fallback sur le contexte commité
  // tant que le premier fetch preview n'est pas revenu.
  const rawAvailable = previewData?.available_options ?? resolvedContext?.available_options
  const available = useMemo(() => {
    if (!rawAvailable) return undefined
    const filterUUIDs = (opts: LabelValue[]): LabelValue[] =>
      opts.filter((o) => !isUUIDLabel(o.label))
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
      const avSets = {
        playlists: new Set(av.playlists.map((o) => o.value)),
        modes: new Set(av.modes.map((o) => o.value)),
        maps: new Set(av.maps.map((o) => o.value)),
        experience_types: new Set(av.experience_types.map((o) => o.value)),
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
      aria-label="Filtres"
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

      {/* Séparateur données / scope temporel */}
      <div className="mx-1 h-4 w-px shrink-0 bg-border" aria-hidden />

      {seasons.length > 0 && (
        <SaisonPill
          open={active === 'saison'}
          onToggle={() => togglePopover('saison')}
          onClose={closeAll}
          seasons={seasons}
          activeSeason={activeSeason}
          seasonCounts={seasonCounts}
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
        presetCounts={presetCounts}
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
          title="Nombre de matchs correspondant aux filtres actifs"
        >
          {totalAfter} match{totalAfter > 1 ? 's' : ''}
        </span>
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
        {justAnalysed ? '✓ Appliqué' : isPreviewFetching ? '…' : 'Analyser'}
      </button>

      {hasActiveFilters && (
        <button
          type="button"
          onClick={() => {
            resetFilters()
          }}
          className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
          title="Réinitialiser tous les filtres"
        >
          ↺ Réinitialiser
        </button>
      )}
    </div>
  )
}
