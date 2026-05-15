/**
 * useLocalFilterBar — barre de filtres locale réutilisable.
 *
 * Pattern factorisé depuis SynthesisPage : state pending → committed avec
 * cascade locale (expérience, playlists, modes), barre période/saison, et
 * useFiltersPreview pour les counts cascade-aware.
 *
 * Consommé par Citations, SessionDetail, SessionCompare. Chaque page consomme
 * le hook, l'utilise pour ses requêtes et rend `bar` au sommet de son layout.
 *
 * Le hook n'écrit dans AUCUN store global — l'état reste 100% local à la page,
 * cohérent avec le pattern « 1 page = 1 scope de filtres ».
 */
import { useMemo, useState, type ReactNode } from 'react'
import { useFiltersPreview } from '@/features/filters/queries'
import { PeriodePill, SaisonPill, DEFAULT_PERIOD } from '@/components/shell/FilterOmnibar'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import { MultiSelectFilter, type MultiSelectOption } from '@/features/explorer/MultiSelectFilter'
import { ExperienceDropdown, type Experience } from '@/features/_shared/ExperienceDropdown'
import type { CascadeInput, FilterContextInput, PeriodInput } from '@/lib/api/types'

// Mapping experience → cascade.experience_types (labels canoniques backend).
const EXPERIENCE_TO_CASCADE: Record<Experience, string[]> = {
  all: [],
  ranked: ['PVP classé'],
  unranked: ['PVP non classé'],
}

function setsEqual(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false
  for (const v of a) if (!b.has(v)) return false
  return true
}

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

interface UseLocalFilterBarOptions {
  playerSlug: string
  labels: LocalFilterBarLabels
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
  /** Élément JSX de la barre sticky à rendre dans le layout de la page. */
  bar: ReactNode
}

/** Construit un FilterContextInput minimal à partir d'une période et d'une cascade. */
function buildContext(period: PeriodInput, cascade: CascadeInput): FilterContextInput {
  return {
    filter_mode: 'period',
    period,
    sessions: { picked_sessions: [], gap_minutes: 120 },
    cascade,
  }
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

export function useLocalFilterBar({ playerSlug, labels }: UseLocalFilterBarOptions): UseLocalFilterBarResult {
  // States pending / committed
  const [pendingPeriod, setPendingPeriod] = useState<PeriodInput>(DEFAULT_PERIOD)
  const [pendingExperience, setPendingExperience] = useState<Experience>('all')
  const [pendingPlaylists, setPendingPlaylists] = useState<Set<string>>(() => new Set())
  const [pendingModes, setPendingModes] = useState<Set<string>>(() => new Set())

  const [committedPeriod, setCommittedPeriod] = useState<PeriodInput>(DEFAULT_PERIOD)
  const [committedExperience, setCommittedExperience] = useState<Experience>('all')
  const [committedPlaylists, setCommittedPlaylists] = useState<Set<string>>(() => new Set())
  const [committedModes, setCommittedModes] = useState<Set<string>>(() => new Set())

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
    () => buildContext(pendingPeriod, pendingCascade),
    [pendingPeriod, pendingCascade],
  )

  const committedFilterContext = useMemo(
    () => buildContext(committedPeriod, committedCascade),
    [committedPeriod, committedCascade],
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

  const hasActiveFilters =
    !!(committedPeriod.start_date || committedPeriod.end_date) ||
    committedExperience !== 'all' ||
    committedPlaylists.size > 0 ||
    committedModes.size > 0

  const isDirty =
    pendingPeriod.start_date !== committedPeriod.start_date ||
    pendingPeriod.end_date !== committedPeriod.end_date ||
    pendingExperience !== committedExperience ||
    !setsEqual(pendingPlaylists, committedPlaylists) ||
    !setsEqual(pendingModes, committedModes)

  function handleAnalyser() {
    setCommittedPeriod(pendingPeriod)
    setCommittedExperience(pendingExperience)
    setCommittedPlaylists(new Set(pendingPlaylists))
    setCommittedModes(new Set(pendingModes))
    closeAll()
  }

  function handleResetAll() {
    setPendingPeriod(DEFAULT_PERIOD)
    setCommittedPeriod(DEFAULT_PERIOD)
    setPendingExperience('all')
    setCommittedExperience('all')
    setPendingPlaylists(new Set())
    setCommittedPlaylists(new Set())
    setPendingModes(new Set())
    setCommittedModes(new Set())
  }

  const bar = (
    <div className="sticky top-0 z-20 border-b border-border" style={{ background: 'var(--background)' }}>
      <div className="flex min-h-10 items-center gap-1.5 px-4 py-1.5 flex-wrap">
        <ExperienceDropdown
          value={pendingExperience}
          onChange={setPendingExperience}
          counts={experienceCounts}
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
          toggle={(v) => {
            setPendingPlaylists((prev) => {
              const next = new Set(prev)
              if (next.has(v)) next.delete(v)
              else next.add(v)
              return next
            })
          }}
          placeholder={labels.playlists}
          alwaysShow
          disabled={playlistOptions.length === 0 && pendingPlaylists.size === 0}
        />
        <MultiSelectFilter
          options={modeOptions}
          selected={pendingModes}
          toggle={(v) => {
            setPendingModes((prev) => {
              const next = new Set(prev)
              if (next.has(v)) next.delete(v)
              else next.add(v)
              return next
            })
          }}
          placeholder={labels.modes}
          alwaysShow
          disabled={modeOptions.length === 0 && pendingModes.size === 0}
        />
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
          {labels.analyser ?? 'Analyser'}
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
    bar,
  }
}
