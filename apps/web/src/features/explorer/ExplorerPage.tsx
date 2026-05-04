/**
 * ExplorerPage — page Explorer (recherche + filtres).
 *
 * Mode Matchs : filtres cascade + tableau paginé.
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
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'
import type { ExplorerMatchFilters, MatchEncounterBadge } from '@/lib/api/types'
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

function EncounterBadges({ badges, locale }: { badges: MatchEncounterBadge[]; locale: string }) {
  if (!badges.length) return null
  const t = (key: SquadManifestKey, values?: Record<string, string | number>) =>
    formatMessage(squadManifest, key, locale, values)

  return (
    <div className="flex flex-wrap gap-1.5">
      {badges.map((badge, i) => {
        const labelKey = badge.label_key as SquadManifestKey
        const ordinal = badge.detail && typeof badge.detail['ordinal'] === 'number'
          ? (badge.detail['ordinal'] as number)
          : undefined
        const label = ordinal !== undefined
          ? t(labelKey, { ordinal })
          : t(labelKey)
        const colorVar = isSemanticToken(badge.color_token)
          ? tokenVar(badge.color_token as SemanticToken)
          : undefined
        return (
          <NarrativeBadge
            key={i}
            label={label}
            colorVar={colorVar}
            size="sm"
          />
        )
      })}
    </div>
  )
}

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

  // Sync URL → state once on initial load (not on every URL change)
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

  function goToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  // Filtres cascade mode Matchs
  const [dateFilter, setDateFilter] = useState('')
  const [squadScope, setSquadScope] = useState<'' | 'all' | 'solo' | 'squad'>('')
  const [expType, setExpType] = useState('')
  const [playlistFilter, setPlaylistFilter] = useState('')
  const [modeFilter, setModeFilter] = useState('')
  const [mapFilter, setMapFilter] = useState('')

  const matchFilters: ExplorerMatchFilters = {
    selected_date: dateFilter || null,
    squad_scope: squadScope || undefined,
    experience_type: expType || null,
    playlist: playlistFilter || null,
    mode: modeFilter || null,
    map: mapFilter || null,
  }

  const matchesQuery = useExplorerMatches(
    playerSlug,
    { filters: filterContext, match_filters: matchFilters },
    filterContextHash,
  )

  const playerQuery = useExplorerPlayer(playerSlug, {
    target_gamertag: targetGamertag,
    page: playerPage,
  })

  const totalPages = playerQuery.data
    ? Math.ceil(playerQuery.data.total_count / (playerQuery.data.page_size || 20))
    : 0

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

        {/* Mode Joueur */}
        {mode === 'player' && (
          <div className="space-y-4">
            <GamertagSearchInput
              onSelect={selectTarget}
              initialValue={targetGamertag}
            />

            {!targetGamertag && (
              <Card>
                <CardContent className="py-4">
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
                <CardContent className="py-4">
                  <EmptyStateNotice
                    title={t('explorer.player.error_title')}
                    description={t('explorer.player.error_description')}
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && !playerQuery.isLoading && !playerQuery.isError && !playerQuery.data && (
              <Card>
                <CardContent className="py-4">
                  <EmptyStateNotice
                    title={t('explorer.player.empty_title')}
                    description={t('explorer.player.empty_description')}
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && playerQuery.data && (
              <Card>
                <CardContent className="py-4 space-y-4">
                  {/* En-tête joueur + bouton face-à-face */}
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

                  {/* Badges encounter */}
                  {playerQuery.data.badges && playerQuery.data.badges.length > 0 && (
                    <EncounterBadges badges={playerQuery.data.badges} locale={locale} />
                  )}

                  {/* Compteurs */}
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

                  {/* Historique complet paginé */}
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
                                  onClick={() => goToMatch(m.match_id)}
                                  role="button"
                                  tabIndex={0}
                                  onKeyDown={(e) => e.key === 'Enter' && goToMatch(m.match_id)}
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

                        {/* Pagination */}
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

        {/* Mode Matchs */}
        {mode === 'matches' && (
          <div className="space-y-4">
            {/* Filtres cascade */}
            <Card>
              <CardContent className="py-3">
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-6">
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">
                      {t('explorer.filters.date')}
                    </label>
                    <input
                      type="date"
                      value={dateFilter}
                      onChange={(e) => setDateFilter(e.target.value)}
                      className="w-full rounded border border-input px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">
                      {t('explorer.filters.context')}
                    </label>
                    <select
                      value={squadScope}
                      onChange={(e) =>
                        setSquadScope(e.target.value as '' | 'all' | 'solo' | 'squad')
                      }
                      className="w-full rounded border border-input px-2 py-1 text-sm"
                    >
                      <option value="">{t('explorer.filters.context_all')}</option>
                      <option value="solo">{t('explorer.filters.context_solo')}</option>
                      <option value="squad">{t('explorer.filters.context_squad')}</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">
                      {t('explorer.filters.experience_type')}
                    </label>
                    <input
                      type="text"
                      placeholder={t('explorer.filters.experience_placeholder')}
                      value={expType}
                      onChange={(e) => setExpType(e.target.value)}
                      className="w-full rounded border border-input px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">
                      {t('explorer.filters.playlist')}
                    </label>
                    <input
                      type="text"
                      placeholder={t('explorer.filters.playlist_placeholder')}
                      value={playlistFilter}
                      onChange={(e) => setPlaylistFilter(e.target.value)}
                      className="w-full rounded border border-input px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">
                      {t('explorer.filters.mode')}
                    </label>
                    <input
                      type="text"
                      placeholder={t('explorer.filters.mode_placeholder')}
                      value={modeFilter}
                      onChange={(e) => setModeFilter(e.target.value)}
                      className="w-full rounded border border-input px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-muted-foreground mb-1">
                      {t('explorer.filters.map')}
                    </label>
                    <input
                      type="text"
                      placeholder={t('explorer.filters.map_placeholder')}
                      value={mapFilter}
                      onChange={(e) => setMapFilter(e.target.value)}
                      className="w-full rounded border border-input px-2 py-1 text-sm"
                    />
                  </div>
                </div>
                {(dateFilter || squadScope || expType || playlistFilter || modeFilter || mapFilter) && (
                  <div className="mt-2 flex justify-end">
                    <button
                      className="text-xs text-primary hover:underline"
                      onClick={() => {
                        setDateFilter('')
                        setSquadScope('')
                        setExpType('')
                        setPlaylistFilter('')
                        setModeFilter('')
                        setMapFilter('')
                      }}
                    >
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
                  <p className="text-sm text-muted-foreground">
                    {t('explorer.matches.count_label', {
                      n: matchesQuery.data.summary?.total_matches ?? 0,
                    })}
                  </p>
                  <div className="overflow-x-auto rounded-lg border border-border bg-background">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border bg-muted text-xs font-medium text-muted-foreground">
                          <th className="px-4 py-2.5 text-left">
                            {t('explorer.matches.col_date')}
                          </th>
                          <th className="px-4 py-2.5 text-left">
                            {t('explorer.matches.col_map_mode')}
                          </th>
                          <th className="px-4 py-2.5 text-left">
                            {t('explorer.matches.col_outcome')}
                          </th>
                          <th className="px-4 py-2.5 text-left">
                            {t('explorer.matches.col_score')}
                          </th>
                          <th className="px-4 py-2.5 text-left">
                            {t('explorer.matches.col_type')}
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border">
                        {matchesQuery.data.table.items.map((row) => (
                          <tr
                            key={row.match_id}
                            className="hover:bg-primary/10 transition-colors cursor-pointer"
                            onClick={() => goToMatch(row.match_id)}
                          >
                            <td className="px-4 py-2 text-muted-foreground">
                              {new Date(row.start_time).toLocaleDateString(numberLocale)}
                            </td>
                            <td className="px-4 py-2">
                              <span className="font-medium text-foreground">{row.map_ui}</span>
                              <span className="ml-1 text-xs text-muted-foreground">
                                · {row.mode_ui}
                              </span>
                            </td>
                            <td className="px-4 py-2">
                              <Badge
                                variant={
                                  row.outcome_label.toLowerCase().includes('victoire')
                                    ? 'success'
                                    : row.outcome_label.toLowerCase().includes('défaite')
                                    ? 'destructive'
                                    : 'secondary'
                                }
                              >
                                {row.outcome_label}
                              </Badge>
                            </td>
                            <td className="px-4 py-2 text-foreground">{row.score_label}</td>
                            <td className="px-4 py-2 text-muted-foreground">
                              {row.experience_type_label}
                            </td>
                          </tr>
                        ))}
                        {matchesQuery.data.table.items.length === 0 && (
                          <tr>
                            <td
                              colSpan={5}
                              className="px-4 py-8 text-center text-muted-foreground"
                            >
                              {t('explorer.matches.empty_row')}
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </>
              ) : (
                <Card>
                  <CardContent className="py-4">
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
