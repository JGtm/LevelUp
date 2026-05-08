/**
 * ExplorerPage — page Explorer (recherche + filtres).
 *
 * Mode Matchs : filtres cascade + tableau paginé (style MatchHistoryTable).
 * Mode Joueur : historique commun paginé (20/page) + badges encounter.
 *
 * URL params : ?mode=player&target=<gamertag> — auto-switch au chargement.
 */
import React, { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate, useSearch } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { GamertagSearchInput } from './GamertagSearchInput'
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import type { MatchEncounterBadge } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenVar, tokenCssVar } from '@/lib/accessibility'
import { mmrDeltaScale } from '@/lib/accessibility/scales'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

type SearchMode = 'matches' | 'player'

const OUTCOME_ROW_BG: Record<number, string> = {
  1: 'bg-info/20',
  2: 'bg-success/20',
  3: 'bg-destructive/20',
  4: 'bg-muted/20',
}

const OUTCOME_BADGE_VARIANT: Record<number, 'success' | 'destructive' | 'secondary' | 'outline'> = {
  1: 'secondary',
  2: 'success',
  3: 'destructive',
  4: 'outline',
}

function isSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
}

// ─── MultiSelectFilter ────────────────────────────────────────────────────────

interface MultiSelectFilterProps {
  options: string[]
  selected: Set<string>
  toggle: (v: string) => void
  placeholder: string
}

function MultiSelectFilter({ options, selected, toggle, placeholder }: MultiSelectFilterProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const label = selected.size === 0 ? placeholder : `${selected.size} sél.`

  if (options.length === 0) return null

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className={`rounded border px-2 py-1 text-sm bg-background flex items-center gap-1 whitespace-nowrap ${
          selected.size > 0
            ? 'border-primary text-primary'
            : 'border-input text-muted-foreground hover:border-foreground'
        }`}
      >
        {label}
        <span className="text-xs opacity-60">▾</span>
      </button>
      {open && (
        <div className="absolute z-50 mt-1 w-max min-w-full rounded-md border border-border bg-background shadow-lg max-h-52 overflow-y-auto">
          {options.map((opt) => (
            <label
              key={opt}
              className="flex items-center gap-2 px-3 py-1.5 text-sm cursor-pointer hover:bg-primary/10"
            >
              <input
                type="checkbox"
                checked={selected.has(opt)}
                onChange={() => toggle(opt)}
                className="rounded accent-primary"
              />
              {opt}
            </label>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── EncounterBadges ──────────────────────────────────────────────────────────

function EncounterBadges({ badges, locale }: { badges: MatchEncounterBadge[]; locale: string }) {
  if (!badges.length) return null
  const manifestLocale: 'fr' | 'en' = locale === 'en' ? 'en' : 'fr'
  const t = (key: SquadManifestKey, values?: Record<string, string | number>) =>
    formatMessage(squadManifest, key, manifestLocale, values)

  return (
    <div className="flex flex-wrap gap-1.5">
      {badges.map((badge, i) => {
        const labelKey = badge.label_key as SquadManifestKey
        const ordinal =
          badge.detail && typeof badge.detail['ordinal'] === 'number'
            ? (badge.detail['ordinal'] as number)
            : undefined
        const label = ordinal !== undefined ? t(labelKey, { ordinal }) : t(labelKey)
        const colorVar = isSemanticToken(badge.color_token)
          ? tokenVar(badge.color_token as SemanticToken)
          : undefined
        return <NarrativeBadge key={i} label={label} colorVar={colorVar} size="sm" />
      })}
    </div>
  )
}

// ─── ExplorerPage ─────────────────────────────────────────────────────────────

export function ExplorerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const search = useSearch({ from: '/players/$playerSlug/explorer/' }) as {
    mode?: SearchMode
    target?: string
  }

  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const locale = useAppShellStore((s) => s.locale)

  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'

  const [mode, setMode] = useState<SearchMode>(search.mode ?? 'matches')
  const [targetGamertag, setTargetGamertag] = useState(search.target ?? '')
  const [playerPage, setPlayerPage] = useState(1)
  const [compareOpen, setCompareOpen] = useState(false)
  const prefetchCompare = useComparePrefetch(playerSlug)

  // ─── Filtres principaux (ranked/outcome/tier/perf/sort) ────────────────────
  const [perfTiers, setPerfTiers] = useState<Set<number>>(new Set())
  const [skillTiers, setSkillTiers] = useState<Set<string>>(new Set())
  const [rankedContext, setRankedContext] = useState<'ranked' | 'unranked' | ''>('')
  const [outcomeFilter, setOutcomeFilter] = useState<Set<number>>(new Set())
  const [sortKey, setSortKey] = useState('start_time:desc')

  const [sortField, sortDir] = sortKey.split(':') as [string, string]

  function togglePerfTier(tier: number) {
    setPerfTiers((prev) => {
      const next = new Set(prev)
      next.has(tier) ? next.delete(tier) : next.add(tier)
      return next
    })
  }

  function toggleSkillTier(tier: string) {
    setSkillTiers((prev) => {
      const next = new Set(prev)
      next.has(tier) ? next.delete(tier) : next.add(tier)
      return next
    })
  }

  function toggleOutcome(code: number) {
    setOutcomeFilter((prev) => {
      const next = new Set(prev)
      next.has(code) ? next.delete(code) : next.add(code)
      return next
    })
  }

  function handleRankedContext(ctx: 'ranked' | 'unranked' | '') {
    setRankedContext(ctx)
    if (ctx !== rankedContext) setSkillTiers(new Set())
  }

  // ─── Filtres secondaires (date/squad/exp/playlist/mode/map/matchID) ────────
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [squadScope, setSquadScope] = useState<'' | 'solo' | 'squad'>('')
  const [expTypes, setExpTypes] = useState<Set<string>>(new Set())
  const [playlists, setPlaylists] = useState<Set<string>>(new Set())
  const [mapNames, setMapNames] = useState<Set<string>>(new Set())
  const [modeNames, setModeNames] = useState<Set<string>>(new Set())
  const [matchIDSearch, setMatchIDSearch] = useState('')

  function toggleSet<T>(setter: React.Dispatch<React.SetStateAction<Set<T>>>, value: T) {
    setter((prev) => {
      const next = new Set(prev)
      next.has(value) ? next.delete(value) : next.add(value)
      return next
    })
  }

  // ─── URL sync ──────────────────────────────────────────────────────────────
  useEffect(() => {
    if (search.mode) setMode(search.mode)
    if (search.target) setTargetGamertag(search.target)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  function setModeAndUrl(m: SearchMode) {
    setMode(m)
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: (prev) => ({ ...prev, mode: m }),
    })
  }

  function selectTarget(gamertag: string) {
    setTargetGamertag(gamertag)
    setPlayerPage(1)
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: (prev) => ({ ...prev, mode: 'player', target: gamertag }),
    })
  }

  const navigateToMatch = useNavigateToMatch(playerSlug)

  function goToMatch(matchId: string, matchIds: string[]) {
    const filterSpec = filterContextToMatchFilterSpec(filterContext)
    navigateToMatch(matchId, {
      source: 'history',
      matchIds,
      filterSpec: filterSpec ?? undefined,
    })
  }

  // ─── Queries ───────────────────────────────────────────────────────────────
  const matchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: filterContext,
      perf_tiers: perfTiers.size > 0 ? [...perfTiers] : undefined,
      skill_tiers: skillTiers.size > 0 ? [...skillTiers] : undefined,
      ranked_context: rankedContext || undefined,
      outcome_filter: outcomeFilter.size > 0 ? [...outcomeFilter] : undefined,
      sort_field: sortField,
      sort_dir: sortDir,
      match_start_date: startDate || null,
      match_end_date: endDate || null,
      experience_types: expTypes.size > 0 ? [...expTypes] : undefined,
      playlists: playlists.size > 0 ? [...playlists] : undefined,
      map_names: mapNames.size > 0 ? [...mapNames] : undefined,
      mode_names: modeNames.size > 0 ? [...modeNames] : undefined,
      squad_scope: squadScope || undefined,
      match_id_search: matchIDSearch || undefined,
    },
    filterContextHash,
  )

  const playerQuery = useExplorerPlayer(playerSlug, {
    target_gamertag: targetGamertag,
    page: playerPage,
  })

  const totalPages = playerQuery.data
    ? Math.ceil(playerQuery.data.total_count / (playerQuery.data.page_size || 20))
    : 0

  const summary = matchesQuery.data?.summary
  const hasActiveFilter =
    startDate ||
    endDate ||
    squadScope ||
    matchIDSearch ||
    expTypes.size > 0 ||
    playlists.size > 0 ||
    mapNames.size > 0 ||
    modeNames.size > 0 ||
    perfTiers.size > 0 ||
    skillTiers.size > 0 ||
    rankedContext !== '' ||
    outcomeFilter.size > 0 ||
    sortKey !== 'start_time:desc'

  function resetFilters() {
    setStartDate('')
    setEndDate('')
    setSquadScope('')
    setMatchIDSearch('')
    setExpTypes(new Set())
    setPlaylists(new Set())
    setMapNames(new Set())
    setModeNames(new Set())
    setPerfTiers(new Set())
    setSkillTiers(new Set())
    setRankedContext('')
    setOutcomeFilter(new Set())
    setSortKey('start_time:desc')
  }

  return (
    <div className="flex flex-col">
      <div className="p-6 space-y-6">
        {/* Onglets mode */}
        <div className="flex gap-2">
          <Button
            variant={mode === 'matches' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setModeAndUrl('matches')}
          >
            {t('explorer.mode.matches')}
          </Button>
          <Button
            variant={mode === 'player' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setModeAndUrl('player')}
          >
            {t('explorer.mode.player')}
          </Button>
        </div>

        {/* ─── Mode Joueur ─────────────────────────────────────────────────── */}
        {mode === 'player' && (
          <div className="space-y-4">
            <GamertagSearchInput onSelect={selectTarget} initialValue={targetGamertag} />

            {!targetGamertag && (
              <Card>
                <CardContent className="py-4 pt-4">
                  <EmptyStateNotice
                    title={t('explorer.player.no_selection_title')}
                    description={t('explorer.player.no_selection_description')}
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && playerQuery.isLoading && !playerQuery.data && (
              <div className="flex justify-center py-8">
                <Spinner label={t('explorer.player.searching')} />
              </div>
            )}

            {targetGamertag && playerQuery.isError && (
              <Card>
                <CardContent className="py-4 pt-4">
                  <EmptyStateNotice
                    title={t('explorer.player.error_title')}
                    description={t('explorer.player.error_description')}
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag &&
              !playerQuery.isLoading &&
              !playerQuery.isError &&
              !playerQuery.data && (
                <Card>
                  <CardContent className="py-4 pt-4">
                    <EmptyStateNotice
                      title={t('explorer.player.empty_title')}
                      description={t('explorer.player.empty_description')}
                    />
                  </CardContent>
                </Card>
              )}

            {targetGamertag && playerQuery.data && (
              <Card>
                <CardContent className="py-4 pt-4 space-y-4">
                  <div className="flex items-center justify-between">
                    <p className="font-semibold text-foreground">
                      {playerQuery.data.target_gamertag || targetGamertag}
                    </p>
                    <Button
                      size="sm"
                      variant="outline"
                      onMouseEnter={() =>
                        prefetchCompare(playerQuery.data.target_gamertag || targetGamertag)
                      }
                      onClick={() => setCompareOpen(true)}
                    >
                      {t('explorer.player.head_to_head')}
                    </Button>
                  </div>

                  {playerQuery.data.badges && playerQuery.data.badges.length > 0 && (
                    <EncounterBadges badges={playerQuery.data.badges} locale={locale} />
                  )}

                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <p className="text-xs text-muted-foreground">
                        {t('explorer.player.matches_together')}
                      </p>
                      <p className="font-bold text-foreground">{playerQuery.data.total_count}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">
                        {t('explorer.player.wins_together')}
                      </p>
                      <p className="font-bold" style={{ color: 'var(--ac-outcome-win)' }}>
                        {playerQuery.data.wins_together}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">
                        {t('explorer.player.losses_together')}
                      </p>
                      <p className="font-bold" style={{ color: 'var(--ac-outcome-loss)' }}>
                        {playerQuery.data.losses_together}
                      </p>
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-xs font-medium text-muted-foreground uppercase tracking-wide">
                      {t('explorer.player.history_title')}
                    </p>

                    {playerQuery.data.common_matches.length > 0 ? (
                      <>
                        <div className="overflow-x-auto rounded-lg border border-border bg-background">
                          <table className="w-full text-sm">
                            <thead>
                              <tr className="border-b border-border bg-muted text-xs font-medium text-muted-foreground">
                                <th className="px-3 py-2 text-left">
                                  {t('explorer.matches.col_date')}
                                </th>
                                <th className="px-3 py-2 text-left">
                                  {t('explorer.matches.col_map_mode')}
                                </th>
                                <th className="px-3 py-2 text-left">Rôle</th>
                                <th className="px-3 py-2 text-left">
                                  {t('explorer.matches.col_outcome')}
                                </th>
                                <th className="px-3 py-2 text-right">K/D</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                              {playerQuery.data.common_matches.map((m) => (
                                <tr
                                  key={m.match_id}
                                  className="hover:bg-primary/10 transition-colors cursor-pointer"
                                  onClick={() =>
                                    goToMatch(
                                      m.match_id,
                                      playerQuery.data.common_matches.map((x) => x.match_id),
                                    )
                                  }
                                  role="button"
                                  tabIndex={0}
                                  onKeyDown={(e) =>
                                    e.key === 'Enter' &&
                                    goToMatch(
                                      m.match_id,
                                      playerQuery.data.common_matches.map((x) => x.match_id),
                                    )
                                  }
                                >
                                  <td className="px-3 py-2 text-muted-foreground whitespace-nowrap">
                                    {new Date(m.start_time).toLocaleDateString(numberLocale)}
                                  </td>
                                  <td className="px-3 py-2">
                                    <span className="font-medium text-foreground">{m.map_ui}</span>
                                    <span className="ml-1 text-xs text-muted-foreground">
                                      · {m.mode_ui}
                                    </span>
                                  </td>
                                  <td className="px-3 py-2">
                                    <Badge variant="secondary" className="text-xs">
                                      {m.were_teammates
                                        ? t('explorer.player.were_teammates')
                                        : t('explorer.player.were_enemies')}
                                    </Badge>
                                  </td>
                                  <td className="px-3 py-2">
                                    <Badge
                                      variant={
                                        m.player_outcome === 2
                                          ? 'success'
                                          : m.player_outcome === 3
                                            ? 'destructive'
                                            : 'secondary'
                                      }
                                    >
                                      {m.outcome_label}
                                    </Badge>
                                  </td>
                                  <td className="px-3 py-2 text-right text-muted-foreground">
                                    {m.kills}/{m.deaths}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>

                        {totalPages > 1 && (
                          <div className="mt-3 flex items-center justify-between text-sm">
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={playerPage <= 1}
                              onClick={() => setPlayerPage((p) => p - 1)}
                            >
                              {t('explorer.player.prev_page')}
                            </Button>
                            <span className="text-muted-foreground">
                              {t('explorer.player.page_info', {
                                page: playerPage,
                                total: totalPages,
                              })}
                            </span>
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={playerPage >= totalPages}
                              onClick={() => setPlayerPage((p) => p + 1)}
                            >
                              {t('explorer.player.next_page')}
                            </Button>
                          </div>
                        )}
                      </>
                    ) : (
                      <EmptyStateNotice
                        title={t('explorer.player.no_common_matches_title')}
                        description={t('explorer.player.no_common_matches_description')}
                      />
                    )}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}

        {/* ─── Mode Matchs ─────────────────────────────────────────────────── */}
        {mode === 'matches' && (
          <div className="space-y-4">
            {/* Filtres */}
            <Card>
              <CardContent className="py-3 pt-3 space-y-2">
                {/* Ligne 1 : date range + match ID + contexte escouade + multi-selects */}
                <div className="flex flex-wrap gap-2 items-center">
                  <input
                    type="date"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                    title={t('explorer.filters.date_from')}
                    className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
                  />
                  <input
                    type="date"
                    value={endDate}
                    onChange={(e) => setEndDate(e.target.value)}
                    title={t('explorer.filters.date_to')}
                    className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
                  />
                  <input
                    type="text"
                    value={matchIDSearch}
                    onChange={(e) => setMatchIDSearch(e.target.value)}
                    placeholder={t('explorer.filters.match_id')}
                    className="rounded border border-input px-2 py-1 text-sm bg-background w-52"
                  />
                  <select
                    value={squadScope}
                    onChange={(e) => setSquadScope(e.target.value as '' | 'solo' | 'squad')}
                    className="rounded border border-input px-2 py-1 text-sm bg-background"
                  >
                    <option value="">{t('explorer.filters.context_all')}</option>
                    <option value="solo">{t('explorer.filters.context_solo')}</option>
                    <option value="squad">{t('explorer.filters.context_squad')}</option>
                  </select>
                  {/* Multi-selects dynamiques (options chargées depuis la réponse) */}
                  <MultiSelectFilter
                    options={summary?.available_experience_types ?? []}
                    selected={expTypes}
                    toggle={(v) => toggleSet(setExpTypes, v)}
                    placeholder={t('explorer.filters.experience_type')}
                  />
                  <MultiSelectFilter
                    options={summary?.available_playlists ?? []}
                    selected={playlists}
                    toggle={(v) => toggleSet(setPlaylists, v)}
                    placeholder={t('explorer.filters.playlist')}
                  />
                  <MultiSelectFilter
                    options={summary?.available_modes ?? []}
                    selected={modeNames}
                    toggle={(v) => toggleSet(setModeNames, v)}
                    placeholder={t('explorer.filters.mode')}
                  />
                  <MultiSelectFilter
                    options={summary?.available_maps ?? []}
                    selected={mapNames}
                    toggle={(v) => toggleSet(setMapNames, v)}
                    placeholder={t('explorer.filters.map')}
                  />
                </div>

                {/* Chips paliers de performance */}
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-muted-foreground">
                    {t('explorer.filters.perf_tier_label')} :
                  </span>
                  {(
                    [
                      { tier: 1, labelKey: 'explorer.filters.perf_tier_excellent' as const, token: 'perf-tier-1' },
                      { tier: 2, labelKey: 'explorer.filters.perf_tier_bon' as const, token: 'perf-tier-2' },
                      { tier: 3, labelKey: 'explorer.filters.perf_tier_correct' as const, token: 'perf-tier-3' },
                      { tier: 4, labelKey: 'explorer.filters.perf_tier_faible' as const, token: 'perf-tier-4' },
                      { tier: 5, labelKey: 'explorer.filters.perf_tier_mauvais' as const, token: 'perf-tier-5' },
                    ] as const
                  ).map(({ tier, labelKey, token }) => {
                    const active = perfTiers.has(tier)
                    const color = tokenVar(token as SemanticToken)
                    return (
                      <button
                        key={tier}
                        onClick={() => togglePerfTier(tier)}
                        className="rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors"
                        style={
                          active
                            ? { borderColor: color, backgroundColor: color, color: 'var(--background)' }
                            : { borderColor: color, color }
                        }
                      >
                        {t(labelKey)}
                      </button>
                    )
                  })}
                </div>

                {/* Contexte ranked + Chips résultat + Tier skill */}
                <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                  {/* Ranked */}
                  <div className="flex items-center gap-1.5">
                    {(
                      [
                        { value: '' as const, labelKey: 'explorer.filters.ranked_all' as const },
                        { value: 'ranked' as const, labelKey: 'explorer.filters.ranked_ranked' as const },
                        { value: 'unranked' as const, labelKey: 'explorer.filters.ranked_unranked' as const },
                      ]
                    ).map(({ value, labelKey }) => (
                      <button
                        key={value || 'all'}
                        onClick={() => handleRankedContext(value)}
                        className={`rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors ${
                          rankedContext === value
                            ? 'border-primary bg-primary text-primary-foreground'
                            : 'border-border text-muted-foreground hover:border-foreground hover:text-foreground'
                        }`}
                      >
                        {t(labelKey)}
                      </button>
                    ))}
                  </div>

                  {/* Outcome */}
                  <div className="flex items-center gap-1.5">
                    {(
                      [
                        { code: 2, labelKey: 'explorer.filters.outcome_win' as const, token: 'outcome-win' as SemanticToken },
                        { code: 3, labelKey: 'explorer.filters.outcome_loss' as const, token: 'outcome-loss' as SemanticToken },
                        { code: 1, labelKey: 'explorer.filters.outcome_tie' as const, token: 'outcome-draw' as SemanticToken },
                      ]
                    ).map(({ code, labelKey, token }) => {
                      const active = outcomeFilter.has(code)
                      const color = tokenVar(token)
                      return (
                        <button
                          key={code}
                          onClick={() => toggleOutcome(code)}
                          className="rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors"
                          style={
                            active
                              ? { borderColor: color, backgroundColor: color, color: 'var(--background)' }
                              : { borderColor: color, color }
                          }
                        >
                          {t(labelKey)}
                        </button>
                      )
                    })}
                  </div>

                  {/* Skill tier */}
                  <div className="flex items-center gap-1.5">
                    {(
                      [
                        { tier: 'Bronze', labelKey: 'explorer.filters.skill_tier_bronze' as const },
                        { tier: 'Silver', labelKey: 'explorer.filters.skill_tier_silver' as const },
                        { tier: 'Gold', labelKey: 'explorer.filters.skill_tier_gold' as const },
                        { tier: 'Platinum', labelKey: 'explorer.filters.skill_tier_platinum' as const },
                        { tier: 'Diamond', labelKey: 'explorer.filters.skill_tier_diamond' as const },
                        { tier: 'Onyx', labelKey: 'explorer.filters.skill_tier_onyx' as const },
                      ] as const
                    ).map(({ tier, labelKey }) => {
                      const active = skillTiers.has(tier)
                      const disabled = rankedContext === ''
                      return (
                        <button
                          key={tier}
                          onClick={() => !disabled && toggleSkillTier(tier)}
                          disabled={disabled}
                          title={disabled ? t('explorer.filters.skill_tier_disabled') : undefined}
                          className={`rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors ${
                            disabled
                              ? 'cursor-not-allowed opacity-40 border-border text-muted-foreground'
                              : active
                                ? 'border-primary bg-primary text-primary-foreground'
                                : 'border-border text-muted-foreground hover:border-foreground hover:text-foreground'
                          }`}
                        >
                          {t(labelKey)}
                        </button>
                      )
                    })}
                  </div>
                </div>

                {hasActiveFilter && (
                  <div className="flex justify-end">
                    <button className="text-xs text-primary hover:underline" onClick={resetFilters}>
                      {t('explorer.filters.reset')}
                    </button>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Résultats */}
            <div className="space-y-2">
              {matchesQuery.isLoading ? (
                <div className="flex justify-center py-16">
                  <Spinner label={t('explorer.matches.loading')} />
                </div>
              ) : matchesQuery.isError ? (
                <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-6 text-center">
                  <p className="text-destructive">{t('explorer.matches.error')}</p>
                  <button
                    onClick={() => matchesQuery.refetch()}
                    className="mt-2 text-sm text-primary underline"
                  >
                    {t('explorer.matches.retry')}
                  </button>
                </div>
              ) : matchesQuery.data ? (
                <>
                  {/* Barre résultats + tri */}
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm text-muted-foreground">
                      {t('explorer.matches.count_label', {
                        n: matchesQuery.data.summary?.total_matches ?? 0,
                      })}
                    </p>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <label className="text-xs text-muted-foreground whitespace-nowrap">
                        {t('explorer.sort.label')} :
                      </label>
                      <select
                        value={sortKey}
                        onChange={(e) => setSortKey(e.target.value)}
                        className="rounded border border-input px-2 py-1 text-xs bg-background"
                      >
                        <option value="start_time:desc">{t('explorer.sort.start_time_desc')}</option>
                        <option value="start_time:asc">{t('explorer.sort.start_time_asc')}</option>
                        <option value="performance_score_relative:desc">{t('explorer.sort.perf_desc')}</option>
                        <option value="performance_score_relative:asc">{t('explorer.sort.perf_asc')}</option>
                        <option value="kda:desc">{t('explorer.sort.kda_desc')}</option>
                        <option value="kills:desc">{t('explorer.sort.kills_desc')}</option>
                        <option value="delta_mmr:desc">{t('explorer.sort.delta_mmr_desc')}</option>
                        <option value="outcome:desc">{t('explorer.sort.outcome')}</option>
                      </select>
                    </div>
                  </div>

                  {/* Tableau résultats — style MatchHistoryTable */}
                  <div className="overflow-x-auto rounded-lg border border-border bg-card">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border bg-card/60 text-xs font-medium text-muted-foreground">
                          <th className="px-4 py-2.5 text-left whitespace-nowrap">
                            {t('explorer.matches.col_date')}
                          </th>
                          <th className="px-4 py-2.5 text-left whitespace-nowrap">
                            {t('explorer.matches.col_map_mode')}
                          </th>
                          <th className="px-4 py-2.5 text-left whitespace-nowrap">
                            {t('explorer.matches.col_outcome')}
                          </th>
                          <th className="px-4 py-2.5 text-left whitespace-nowrap">
                            {t('explorer.matches.col_score')}
                          </th>
                          <th className="px-4 py-2.5 text-right whitespace-nowrap">FDA</th>
                          <th className="px-4 py-2.5 text-right whitespace-nowrap">
                            {t('explorer.matches.col_perf')}
                          </th>
                          <th className="px-4 py-2.5 text-right whitespace-nowrap">ΔPerf</th>
                          <th className="px-4 py-2.5 text-left whitespace-nowrap">Rang</th>
                          <th className="px-4 py-2.5 text-right whitespace-nowrap">ΔRang</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border">
                        {matchesQuery.data.table.items.map((row) => {
                          const bg = OUTCOME_ROW_BG[row.outcome_code] ?? ''
                          return (
                            <tr
                              key={row.match_id}
                              className={`cursor-pointer transition-colors hover:brightness-125 ${bg}`}
                              onClick={() =>
                                goToMatch(
                                  row.match_id,
                                  matchesQuery.data!.table.items.map((r) => r.match_id),
                                )
                              }
                            >
                              {/* Date */}
                              <td className="px-4 py-2 text-muted-foreground whitespace-nowrap">
                                {row.start_time_label ??
                                  new Date(row.start_time).toLocaleDateString(numberLocale)}
                              </td>
                              {/* Carte / Mode */}
                              <td className="px-4 py-2">
                                <span className="font-medium text-foreground">{row.map_ui}</span>
                                <span className="ml-1 text-xs text-muted-foreground">
                                  · {row.mode_ui}
                                </span>
                              </td>
                              {/* Résultat */}
                              <td className="px-4 py-2">
                                <Badge variant={OUTCOME_BADGE_VARIANT[row.outcome_code] ?? 'secondary'}>
                                  {row.outcome_label}
                                </Badge>
                              </td>
                              {/* Score */}
                              <td className="px-4 py-2 text-muted-foreground">{row.score_label}</td>
                              {/* FDA — Kills / Deaths / Assists */}
                              <td className="px-4 py-2 text-right font-mono text-xs text-muted-foreground whitespace-nowrap">
                                {row.kills != null
                                  ? `${row.kills}/${row.deaths ?? '?'}/${row.assists ?? '?'}`
                                  : '—'}
                              </td>
                              {/* Perf (coloré par tier) */}
                              <td className="px-4 py-2 text-right">
                                {row.perf_score != null && row.perf_tier ? (
                                  <span
                                    className="font-semibold tabular-nums"
                                    style={{
                                      color: tokenVar(`perf-tier-${row.perf_tier}` as SemanticToken),
                                    }}
                                  >
                                    {row.perf_score}
                                  </span>
                                ) : (
                                  <span className="text-muted-foreground text-xs">—</span>
                                )}
                              </td>
                              {/* ΔPerf */}
                              <td className="px-4 py-2 text-right font-mono text-xs">
                                {row.delta_perf != null ? (
                                  <span
                                    style={{
                                      color:
                                        row.delta_perf > 0
                                          ? tokenVar('perf-tier-1' as SemanticToken)
                                          : row.delta_perf < 0
                                            ? tokenVar('perf-tier-5' as SemanticToken)
                                            : undefined,
                                    }}
                                  >
                                    {row.delta_perf >= 0 ? '+' : ''}
                                    {row.delta_perf}
                                  </span>
                                ) : (
                                  <span className="text-muted-foreground">—</span>
                                )}
                              </td>
                              {/* Rang (skill tier label) */}
                              <td className="px-4 py-2 text-xs text-muted-foreground">
                                {row.skill_tier_label ?? '—'}
                              </td>
                              {/* ΔRang (delta_mmr) */}
                              <td className="px-4 py-2 text-right font-mono text-sm">
                                {row.delta_mmr != null ? (
                                  <span
                                    style={{ color: tokenCssVar(mmrDeltaScale(row.delta_mmr)) }}
                                  >
                                    {row.delta_mmr >= 0 ? '+' : ''}
                                    {row.delta_mmr.toFixed(0)}
                                  </span>
                                ) : (
                                  <span className="text-muted-foreground">—</span>
                                )}
                              </td>
                            </tr>
                          )
                        })}
                        {matchesQuery.data.table.items.length === 0 && (
                          <tr>
                            <td
                              colSpan={9}
                              className="px-4 py-8 text-center text-muted-foreground"
                            >
                              {t('explorer.matches.empty_row')}
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>

                  {/* Pagination */}
                  {matchesQuery.data.table.pagination && (
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">
                        Page {matchesQuery.data.table.pagination.page} /{' '}
                        {Math.ceil(
                          matchesQuery.data.table.pagination.total /
                            matchesQuery.data.table.pagination.page_size,
                        )}
                      </span>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={!matchesQuery.data.table.pagination.has_prev}
                          onClick={() => {
                            /* pagination handled by server, not yet wired */
                          }}
                        >
                          ← {t('explorer.player.prev_page')}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={!matchesQuery.data.table.pagination.has_next}
                          onClick={() => {
                            /* pagination handled by server, not yet wired */
                          }}
                        >
                          {t('explorer.player.next_page')} →
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <Card>
                  <CardContent className="py-4 pt-4">
                    <EmptyStateNotice
                      title={t('explorer.matches.empty_title')}
                      description={t('explorer.matches.empty_description')}
                    />
                  </CardContent>
                </Card>
              )}
            </div>
          </div>
        )}
      </div>

      <CompareDrawer
        playerSlug={playerSlug}
        open={compareOpen}
        onClose={() => setCompareOpen(false)}
      />
    </div>
  )
}
