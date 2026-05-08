/**
 * ExplorerPage — page Explorer (recherche + filtres).
 *
 * Mode Matchs : filtres (dropdowns + date range) + tableau paginé
 *               (ExplorerMatchesTable, repris du SquadMatchHistoryTable).
 * Mode Joueur : historique commun paginé (20/page) + badges encounter.
 *
 * URL params : ?mode=player&target=<gamertag> — auto-switch au chargement.
 */
import { useState, useEffect } from 'react'
import { useParams, useNavigate, useSearch } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { GamertagSearchInput } from './GamertagSearchInput'
import { MultiSelectFilter, type MultiSelectOption } from './MultiSelectFilter'
import { ExplorerMatchesTable } from './ExplorerMatchesTable'
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import type { MatchEncounterBadge, LabelValue } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

type SearchMode = 'matches' | 'player'

function isSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
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

  // ─── Filtres ───────────────────────────────────────────────────────────────
  const [perfTiers, setPerfTiers] = useState<Set<string>>(new Set())
  const [skillTiers, setSkillTiers] = useState<Set<string>>(new Set())
  const [rankedContext, setRankedContext] = useState<'ranked' | 'unranked' | ''>('')
  const [outcomeFilter, setOutcomeFilter] = useState<Set<string>>(new Set())
  const [sortKey, setSortKey] = useState('start_time:desc')
  const [sortField, sortDir] = sortKey.split(':') as [string, string]

  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [squadScope, setSquadScope] = useState<'' | 'solo' | 'squad'>('')
  const [expTypes, setExpTypes] = useState<Set<string>>(new Set())
  const [playlists, setPlaylists] = useState<Set<string>>(new Set())
  const [mapNames, setMapNames] = useState<Set<string>>(new Set())
  const [modeNames, setModeNames] = useState<Set<string>>(new Set())
  const [matchIDSearch, setMatchIDSearch] = useState('')

  function toggleSet(setter: React.Dispatch<React.SetStateAction<Set<string>>>, value: string) {
    setter((prev) => {
      const next = new Set(prev)
      if (next.has(value)) next.delete(value)
      else next.add(value)
      return next
    })
  }

  function handleStartDate(v: string) {
    setStartDate(v)
    if (endDate && v && endDate < v) setEndDate('')
  }

  function handleRankedContext(v: 'ranked' | 'unranked' | '') {
    setRankedContext(v)
    if (v !== rankedContext) setSkillTiers(new Set())
  }

  // ─── URL sync ──────────────────────────────────────────────────────────────
  // Init unique depuis l'URL au mount — légitime pour hydrater l'état initial.
  useEffect(() => {
    if (search.mode) setMode(search.mode)
    if (search.target) setTargetGamertag(search.target)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
  function goToPlayerMatch(matchId: string, matchIds: string[]) {
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
      perf_tiers: perfTiers.size > 0 ? [...perfTiers].map(Number) : undefined,
      skill_tiers: skillTiers.size > 0 ? [...skillTiers] : undefined,
      ranked_context: rankedContext || undefined,
      outcome_filter: outcomeFilter.size > 0 ? [...outcomeFilter].map(Number) : undefined,
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

  // ─── Options pour les MultiSelectFilter ───────────────────────────────────
  const expTypeOptions: MultiSelectOption[] = (summary?.available_experience_types ?? []).map(
    (v) => ({ value: v, label: v }),
  )
  const playlistOptions: MultiSelectOption[] = (summary?.available_playlists ?? []).map((v) => ({
    value: v,
    label: v,
  }))
  const modeOptions: MultiSelectOption[] = (summary?.available_modes ?? []).map((v) => ({
    value: v,
    label: v,
  }))
  const mapOptions: MultiSelectOption[] = (summary?.available_maps ?? []).map((v) => ({
    value: v,
    label: v,
  }))

  // Mappe les counts backend (LabelValue[]) sur les options frontend (qui portent
  // labels i18n + swatch). Si pas de backend data, count reste undefined (pas de
  // grayout). Match par `value` exact.
  function withCounts(opts: MultiSelectOption[], backend?: LabelValue[]): MultiSelectOption[] {
    if (!backend) return opts
    const map = new Map(backend.map((b) => [b.value, b.count]))
    return opts.map((o) => ({ ...o, count: map.get(o.value) ?? 0 }))
  }

  const perfTierOptions: MultiSelectOption[] = withCounts(
    [
      { value: '1', label: t('explorer.filters.perf_tier_excellent'), swatch: tokenVar('perf-tier-1' as SemanticToken) },
      { value: '2', label: t('explorer.filters.perf_tier_bon'), swatch: tokenVar('perf-tier-2' as SemanticToken) },
      { value: '3', label: t('explorer.filters.perf_tier_correct'), swatch: tokenVar('perf-tier-3' as SemanticToken) },
      { value: '4', label: t('explorer.filters.perf_tier_faible'), swatch: tokenVar('perf-tier-4' as SemanticToken) },
      { value: '5', label: t('explorer.filters.perf_tier_mauvais'), swatch: tokenVar('perf-tier-5' as SemanticToken) },
    ],
    summary?.available_perf_tiers,
  )

  const outcomeOptions: MultiSelectOption[] = withCounts(
    [
      { value: '2', label: t('explorer.filters.outcome_win'), swatch: tokenVar('outcome-win' as SemanticToken) },
      { value: '3', label: t('explorer.filters.outcome_loss'), swatch: tokenVar('outcome-loss' as SemanticToken) },
      { value: '1', label: t('explorer.filters.outcome_tie'), swatch: tokenVar('outcome-draw' as SemanticToken) },
    ],
    summary?.available_outcomes,
  )

  const skillTierOptions: MultiSelectOption[] = withCounts(
    [
      { value: 'Bronze', label: t('explorer.filters.skill_tier_bronze') },
      { value: 'Silver', label: t('explorer.filters.skill_tier_silver') },
      { value: 'Gold', label: t('explorer.filters.skill_tier_gold') },
      { value: 'Platinum', label: t('explorer.filters.skill_tier_platinum') },
      { value: 'Diamond', label: t('explorer.filters.skill_tier_diamond') },
      { value: 'Onyx', label: t('explorer.filters.skill_tier_onyx') },
    ],
    summary?.available_skill_tiers,
  )

  // Counts pour les single-selects (ranked context, squad scope) — on les
  // interpole dans les <option> labels, et on désactive celles à count=0.
  const rankedCountByValue = new Map(
    (summary?.available_ranked_contexts ?? []).map((b) => [b.value, b.count]),
  )
  const squadCountByValue = new Map(
    (summary?.available_squad_scopes ?? []).map((b) => [b.value, b.count]),
  )

  const hasActiveFilter =
    !!startDate ||
    !!endDate ||
    !!squadScope ||
    !!matchIDSearch ||
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
                                <th className="px-3 py-2 text-left">
                                  {t('explorer.player.col_role')}
                                </th>
                                <th className="px-3 py-2 text-left">
                                  {t('explorer.matches.col_outcome')}
                                </th>
                                <th className="px-3 py-2 text-right">
                                  {t('explorer.player.col_kd')}
                                </th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                              {playerQuery.data.common_matches.map((m) => (
                                <tr
                                  key={m.match_id}
                                  className="hover:bg-primary/10 transition-colors cursor-pointer"
                                  onClick={() =>
                                    goToPlayerMatch(
                                      m.match_id,
                                      playerQuery.data.common_matches.map((x) => x.match_id),
                                    )
                                  }
                                  role="button"
                                  tabIndex={0}
                                  onKeyDown={(e) =>
                                    e.key === 'Enter' &&
                                    goToPlayerMatch(
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
              <CardContent className="py-3 pt-3 space-y-3">
                {/* Ligne 1 : période + ID + escouade */}
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                  <div className="flex items-center gap-1.5">
                    <label className="text-xs text-muted-foreground whitespace-nowrap">
                      {t('explorer.filters.date_from_label')}
                    </label>
                    <input
                      type="date"
                      value={startDate}
                      onChange={(e) => handleStartDate(e.target.value)}
                      className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
                    />
                  </div>
                  <div className="flex items-center gap-1.5">
                    <label className="text-xs text-muted-foreground whitespace-nowrap">
                      {t('explorer.filters.date_to_label')}
                    </label>
                    <input
                      type="date"
                      value={endDate}
                      min={startDate || undefined}
                      onChange={(e) => setEndDate(e.target.value)}
                      className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
                    />
                  </div>
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
                    {(['', 'solo', 'squad'] as const).map((v) => {
                      const labelKey =
                        v === '' ? 'explorer.filters.context_all'
                        : v === 'solo' ? 'explorer.filters.context_solo'
                        : 'explorer.filters.context_squad'
                      const c = squadCountByValue.get(v)
                      const isCurrent = v === squadScope
                      return (
                        <option key={v} value={v} disabled={c === 0 && !isCurrent}>
                          {t(labelKey)}{c !== undefined ? ` (${c})` : ''}
                        </option>
                      )
                    })}
                  </select>
                </div>

                {/* Ligne 2 : tous les multi-selects */}
                <div className="flex flex-wrap items-center gap-2">
                  <MultiSelectFilter
                    options={expTypeOptions}
                    selected={expTypes}
                    toggle={(v) => toggleSet(setExpTypes, v)}
                    placeholder={t('explorer.filters.experience_type')}
                  />
                  <MultiSelectFilter
                    options={playlistOptions}
                    selected={playlists}
                    toggle={(v) => toggleSet(setPlaylists, v)}
                    placeholder={t('explorer.filters.playlist')}
                  />
                  <MultiSelectFilter
                    options={modeOptions}
                    selected={modeNames}
                    toggle={(v) => toggleSet(setModeNames, v)}
                    placeholder={t('explorer.filters.mode')}
                  />
                  <MultiSelectFilter
                    options={mapOptions}
                    selected={mapNames}
                    toggle={(v) => toggleSet(setMapNames, v)}
                    placeholder={t('explorer.filters.map')}
                  />
                  <MultiSelectFilter
                    options={outcomeOptions}
                    selected={outcomeFilter}
                    toggle={(v) => toggleSet(setOutcomeFilter, v)}
                    placeholder={t('explorer.filters.outcome_label')}
                    alwaysShow
                  />
                  <MultiSelectFilter
                    options={perfTierOptions}
                    selected={perfTiers}
                    toggle={(v) => toggleSet(setPerfTiers, v)}
                    placeholder={t('explorer.filters.perf_tier_label')}
                    alwaysShow
                  />
                  <select
                    value={rankedContext}
                    onChange={(e) =>
                      handleRankedContext(e.target.value as 'ranked' | 'unranked' | '')
                    }
                    className="rounded border border-input px-2 py-1 text-sm bg-background"
                  >
                    {(['', 'ranked', 'unranked'] as const).map((v) => {
                      const labelKey =
                        v === '' ? 'explorer.filters.ranked_all'
                        : v === 'ranked' ? 'explorer.filters.ranked_ranked'
                        : 'explorer.filters.ranked_unranked'
                      const c = rankedCountByValue.get(v)
                      const isCurrent = v === rankedContext
                      const prefix = v === '' ? `${t('explorer.filters.ranked_label')} : ` : ''
                      return (
                        <option key={v || 'all'} value={v} disabled={c === 0 && !isCurrent}>
                          {prefix}{t(labelKey)}{c !== undefined ? ` (${c})` : ''}
                        </option>
                      )
                    })}
                  </select>
                  <MultiSelectFilter
                    options={skillTierOptions}
                    selected={skillTiers}
                    toggle={(v) => toggleSet(setSkillTiers, v)}
                    placeholder={t('explorer.filters.skill_tier_label')}
                    alwaysShow
                    disabled={rankedContext === ''}
                    title={rankedContext === '' ? t('explorer.filters.skill_tier_disabled') : undefined}
                  />
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

                  {/* Tableau résultats — composant repris depuis Squad */}
                  <ExplorerMatchesTable
                    rows={matchesQuery.data.table.items}
                    playerSlug={playerSlug}
                  />
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
