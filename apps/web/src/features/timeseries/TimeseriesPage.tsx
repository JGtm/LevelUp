/**
 * TimeseriesPage — page Séries temporelles (6 onglets).
 *
 * Phase 2 P2.E : migration partielle Plotly → ECharts pour les charts qui ont
 * un wrapper ECharts disponible (TimeseriesLineChart, Heatmap2DChart). Les
 * histogrammes, scatters, KDA bars et combat-yield restent sur Plotly en
 * attendant des wrappers ECharts dédiés (différé Phase 3). i18n via
 * `timeseriesManifest` + `formatMessage`.
 */
import { useMemo } from 'react'
import { useParams, useSearch, useNavigate } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { useTimeseriesPage } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeValue,
} from '@/components/charts/OutcomeSequenceTape'
import { TimeseriesKdaTrend } from './TimeseriesKdaTrend'
import { TimeseriesKdaDensity } from './TimeseriesKdaDensity'
import { TimeseriesTopWeapons } from './TimeseriesTopWeapons'
import { TimeseriesDistributionHistogram } from './TimeseriesDistributionHistogram'
import { TimeseriesScatterWithTrend } from './TimeseriesScatterWithTrend'
import {
  TimeseriesPerformanceTrend,
  TimeseriesAssistsTrend,
  TimeseriesPerMinuteTrend,
  TimeseriesAvgLifeTrend,
  TimeseriesSpreeHeadshots,
  TimeseriesRankScore,
  TimeseriesKdaValueTrend,
  TimeseriesSkillRankPerformance,
} from './TimeseriesFormCharts'
import { TimeseriesFirstEventDistribution } from './TimeseriesFirstEventDistribution'
import {
  TimeseriesSessionPerformance,
  TimeseriesEfficiency,
  TimeseriesIntensityHeatmap,
} from './TimeseriesSquadAdapted'
import { EngagementTimeseriesSection } from '@/features/engagement/EngagementTimeseriesSection'
import { ChartFrame } from './ChartFrame'
import { WinRateVsHistoryBulletChart } from '@/features/squad/WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from '@/features/squad/MapPerfVsHistoryChart'
import { useExplorerMatches } from '@/features/explorer/queries'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { formatMessage } from '@/lib/i18n/format'
import {
  timeseriesManifest,
  type TimeseriesManifestKey,
} from '@/lib/i18n/generated/timeseries'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'

function outcomeNumToValue(n: number | null): OutcomeValue {
  if (n === 2) return 'win'
  if (n === 3) return 'loss'
  if (n === 1) return 'tie'
  return 'dnf'
}

type TabId = 'summary' | 'distributions' | 'progression'

const TAB_KEYS: { id: TabId; key: TimeseriesManifestKey }[] = [
  { id: 'summary', key: 'timeseries.tabs.summary' },
  { id: 'distributions', key: 'timeseries.tabs.distributions' },
  { id: 'progression', key: 'timeseries.tabs.progression' },
]

export function TimeseriesPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const { tab } = useSearch({
    from: '/players/$playerSlug/stats/timeseries',
  })
  const activeTab: TabId = tab ?? 'summary'
  const navigate = useNavigate({ from: '/players/$playerSlug/stats/timeseries' })
  const setActiveTab = (next: TabId) => {
    navigate({ search: (prev) => ({ ...prev, tab: next }), replace: true }).catch(() => {})
  }
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: TimeseriesManifestKey) => formatMessage(timeseriesManifest, key, locale)
  const { data: fieldMappings } = useFieldMappings()
  const outcomeLabels = {
    win: fieldMappings?.outcomes?.win?.label ?? t('timeseries.distributions.outcome_win_fallback'),
    loss:
      fieldMappings?.outcomes?.loss?.label ??
      t('timeseries.distributions.outcome_loss_fallback'),
    tie: fieldMappings?.outcomes?.tie?.label ?? 'Égalité',
    dnf: fieldMappings?.outcomes?.dnf?.label ?? 'Abandon',
    unknown:
      fieldMappings?.fields['outcome_unknown']?.label ??
      t('timeseries.distributions.outcome_unknown_fallback'),
  }
  // Resolver de label de carte aligné sur la page Escouade.
  const mapAssets = fieldMappings?.assets?.['map']
  const mapLabelOf = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI

  // Injecter match_context='solo' : cette page n'affiche que les matchs solo.
  const soloFilterContext = useMemo(
    () => ({ ...filterContext, match_context: 'solo' as const }),
    [filterContext],
  )

  const { data, isLoading, isError, refetch } = useTimeseriesPage(
    playerSlug,
    { filters: soloFilterContext },
    filterContextHash,
  )

  // Tableau historique de matchs (reproduit l'Explorer mode "Matchs") affiché
  // en bas de l'onglet Progression. Suit le scope du filtre global solo pour
  // matcher les matchs des charts. Pas de filtres avancés (perf/skill/etc.).
  const explorerMatchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: soloFilterContext,
      pagination: { page: 1, page_size: 10000 },
      sort_field: 'start_time',
      sort_dir: 'desc',
    },
    filterContextHash,
    activeTab === 'progression',
  )

  if (isLoading) return null

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('timeseries.errors.load_failed')}</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              {t('timeseries.errors.retry')}
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={t('timeseries.empty.page_title')}
          description={t('timeseries.empty.page_description')}
          actionLabel={t('timeseries.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const { distributions_tab } = data

  return (
    <div className="flex flex-col">
      {/* Briefing — KPI bar mode solo (rail + grid 7 cards, pas de verdict) */}
      {data.briefing_kpis && (
        <div className="px-6 pt-6 pb-6">
          <SessionBriefing kpis={data.briefing_kpis} />
        </div>
      )}

      {/* Onglets */}
      <div className="flex gap-0 border-b bg-background px-6">
        {TAB_KEYS.map((tab) => (
          <Button
            key={tab.id}
            variant="ghost"
            size="sm"
            onClick={() => setActiveTab(tab.id)}
            className={`rounded-none border-b-2 px-4 py-3 text-sm ${
              activeTab === tab.id
                ? 'border-primary font-semibold text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t(tab.key)}
          </Button>
        ))}
      </div>

      <div className="p-6 space-y-6">
        {/* KPIs — outcome sequence + 4 charts (timeseries.02 → .05) */}
        {activeTab === 'summary' && (
          <div className="space-y-6">
            {/* Séquence des résultats — en haut */}
            {(data.match_rows ?? []).length > 0 && (
              <div data-testid="timeseries-outcome-sequence">
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  {locale === 'en' ? 'Outcome sequence' : 'Séquence des résultats'}
                </p>
                <OutcomeSequenceTape
                  matches={(data.match_rows ?? []).map<OutcomePoint>((r) => ({
                    outcome: outcomeNumToValue(r.outcome),
                    matchId: r.match_id,
                    mode: r.playlist_name || undefined,
                  }))}
                  labels={{
                    win: outcomeLabels.win,
                    loss: outcomeLabels.loss,
                    tie: fieldMappings?.outcomes?.tie?.label ?? 'Égalité',
                    dnf: outcomeLabels.unknown,
                  }}
                />
              </div>
            )}

            {/* Évolution Frags / Morts (gauche) | Assistances (droite) */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <ChartFrame
                title={
                  locale === 'en' ? 'Kills / Deaths timeline' : 'Évolution Frags / Morts'
                }
              >
                <TimeseriesKdaTrend
                  rows={data.match_rows ?? []}
                  labels={{
                    kills: fieldMappings?.fields['kills']?.label ?? 'Frags',
                    deaths: fieldMappings?.fields['deaths']?.label ?? 'Morts',
                    yAxis: locale === 'en' ? 'Kills / Deaths' : 'Frags / Morts',
                  }}
                />
              </ChartFrame>

              <ChartFrame title={locale === 'en' ? 'KDA distribution' : 'Distribution FDA'}>
                <TimeseriesKdaDensity
                  buckets={distributions_tab.kda_buckets ?? []}
                  rows={data.match_rows ?? []}
                  labels={{
                    density: locale === 'en' ? 'Density' : 'Densité',
                    rug: fieldMappings?.fields['kda']?.label ?? 'FDA',
                    xAxis: fieldMappings?.fields['kda']?.label ?? 'FDA',
                    mean: locale === 'en' ? 'Mean' : 'Moyenne',
                    median: locale === 'en' ? 'Median' : 'Médiane',
                  }}
                />
              </ChartFrame>
            </div>

            {/* Durée de vie moyenne (gauche) | Assistances (droite) */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <ChartFrame title={fieldMappings?.fields['avg_life_seconds']?.label ?? 'Durée de vie moyenne'}>
                <TimeseriesAvgLifeTrend
                  rows={data.match_rows ?? []}
                  lifeLabel={locale === 'en' ? 'Average life (s)' : 'Durée de vie (s)'}
                />
              </ChartFrame>

              <ChartFrame title={fieldMappings?.fields['assists']?.label ?? 'Assistances'}>
                <TimeseriesAssistsTrend
                  rows={data.match_rows ?? []}
                  assistsLabel={
                    fieldMappings?.fields['assists']?.label ??
                    'Assistances'
                  }
                  smoothingLabel={locale === 'en' ? 'Trend' : 'Tendance'}
                />
              </ChartFrame>
            </div>

            {/* Outils de destruction (gauche) | Résultats par période (droite) */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <ChartFrame
                title={locale === 'en' ? 'Tools of destruction' : 'Outils de destruction'}
              >
                {(data.top_weapons ?? []).length > 0 ? (
                  <TimeseriesTopWeapons
                    weapons={data.top_weapons ?? []}
                    labels={{
                      seriesName: fieldMappings?.fields['kills']?.label ?? 'Frags',
                      fallbackLabel: (id) => `#${id}`,
                    }}
                  />
                ) : (
                  <EmptyStateNotice
                    title={t('timeseries.empty.page_title')}
                    description={t('timeseries.empty.no_data_description')}
                  />
                )}
              </ChartFrame>

              <ChartFrame title={fieldMappings?.fields['kda']?.label ?? 'FDA'}>
                {(data.match_rows ?? []).length > 0 ? (
                  <TimeseriesKdaValueTrend
                    rows={data.match_rows ?? []}
                    fdaLabel={fieldMappings?.fields['kda']?.label ?? 'FDA'}
                    smoothingLabel={locale === 'en' ? 'Trend' : 'Tendance'}
                  />
                ) : (
                  <EmptyStateNotice
                    title={t('timeseries.empty.page_title')}
                    description={t('timeseries.empty.no_data_description')}
                  />
                )}
              </ChartFrame>
            </div>

            {/* Performance solo par session/semaine/mois — agrégat backend
                sur tous les matchs solo (cross-session, granularité auto). */}
            {data.solo_session_perf && data.solo_session_perf.points.length > 0 && (
              <ChartFrame
                title={(() => {
                  const g = data.solo_session_perf?.granularity ?? 'session'
                  if (locale === 'en') {
                    if (g === 'week') return 'Solo performance per week'
                    if (g === 'month') return 'Solo performance per month'
                    return 'Solo performance per session'
                  }
                  if (g === 'week') return 'Performance solo par semaine'
                  if (g === 'month') return 'Performance solo par mois'
                  return 'Performance solo par session'
                })()}
              >
                <TimeseriesSessionPerformance
                  points={data.solo_session_perf.points}
                  granularity={data.solo_session_perf.granularity}
                  perfLabel={
                    fieldMappings?.fields['performance_score']?.label ??
                    (locale === 'en' ? 'Performance' : 'Performance')
                  }
                  winRateLabel={
                    fieldMappings?.fields['win_rate']?.label ??
                    (locale === 'en' ? 'Win rate' : 'Win rate')
                  }
                  mmrLabel={fieldMappings?.fields['team_mmr']?.label ?? 'MMR équipe'}
                />
              </ChartFrame>
            )}

            {/* Taux de victoire (gauche) | Performance par carte (droite) —
                réutilisation des wrappers Escouade (teammates.02 + .13). */}
            {(data.map_breakdown ?? []).length > 0 && (
              <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                <WinRateVsHistoryBulletChart
                  title={
                    locale === 'en'
                      ? 'Win rate — Session vs History'
                      : 'Taux de victoire — Session vs Historique'
                  }
                  rows={data.map_breakdown ?? []}
                  mapLabelOf={mapLabelOf}
                  sessionLabel={locale === 'en' ? 'Session' : 'Session'}
                  historyLabel={locale === 'en' ? 'History' : 'Historique'}
                  parityLabel={locale === 'en' ? 'Parity' : 'Parité'}
                  zeroWinrateLabel={locale === 'en' ? '0 % win rate' : '0 % de victoires'}
                />
                {(data.map_breakdown ?? []).some(
                  (r) =>
                    r.performance_avg !== undefined &&
                    r.historical_performance_avg !== undefined,
                ) && (
                  <MapPerfVsHistoryChart
                    title={
                      locale === 'en'
                        ? 'Performance per map — Session vs History'
                        : 'Performance par carte — Session vs Historique'
                    }
                    rows={data.map_breakdown ?? []}
                    mapLabelOf={mapLabelOf}
                    sessionLabel={locale === 'en' ? 'Session' : 'Session'}
                    historyLabel={locale === 'en' ? 'History' : 'Historique'}
                  />
                )}
              </div>
            )}
          </div>
        )}

        {/* Distributions — 6 histos (timeseries.09) + 5 scatters (timeseries.10) */}
        {activeTab === 'distributions' && (
          <div className="space-y-8">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              {locale === 'en' ? 'Distributions' : 'Distributions'}
            </h3>
            {/* 6 histogrammes en grille 3×2, chacun avec médiane verticale.
                Performance utilise un coloring par tier (perf-tier-1..5). */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {([
                {
                  buckets: distributions_tab.accuracy_buckets ?? [],
                  title:
                    fieldMappings?.fields['accuracy']?.label ?? 'Précision',
                  colorToken: 'chart-series-2' as const,
                  xAxisLabel: locale === 'en' ? 'Accuracy (%)' : 'Précision (%)',
                  colorTokenByBucket: undefined,
                },
                {
                  buckets: distributions_tab.kills_buckets ?? [],
                  title:
                    fieldMappings?.fields['kills']?.label ?? 'Frags',
                  colorToken: 'chart-series-1' as const,
                  xAxisLabel: fieldMappings?.fields['kills']?.label ?? 'Frags',
                  colorTokenByBucket: undefined,
                },
                {
                  buckets: distributions_tab.life_buckets ?? [],
                  title: locale === 'en' ? 'Average life (s)' : 'Durée de vie moyenne (s)',
                  colorToken: 'chart-series-3' as const,
                  xAxisLabel: locale === 'en' ? 'Seconds' : 'Secondes',
                  colorTokenByBucket: undefined,
                },
                {
                  buckets: distributions_tab.personal_score_buckets ?? [],
                  title:
                    fieldMappings?.fields['personal_score']?.label ?? 'Score personnel',
                  colorToken: 'chart-series-5' as const,
                  xAxisLabel: fieldMappings?.fields['personal_score']?.label ?? 'Score personnel',
                  colorTokenByBucket: undefined,
                },
                {
                  buckets: distributions_tab.perf_score_buckets ?? [],
                  title:
                    fieldMappings?.fields['performance_score']?.label ??
                    (locale === 'en' ? 'Performance' : 'Performance'),
                  colorToken: 'perf-tier-3' as const,
                  xAxisLabel: fieldMappings?.fields['performance_score']?.label ?? 'Score de performance',
                  // Grading color : perf-tier-1..5 selon le bucket midpoint sur [0,100].
                  colorTokenByBucket: ((b: { bucket_lower: number; bucket_upper: number }) => {
                    const mid = (b.bucket_lower + b.bucket_upper) / 2
                    if (mid < 20) return 'perf-tier-1' as const
                    if (mid < 40) return 'perf-tier-2' as const
                    if (mid < 60) return 'perf-tier-3' as const
                    if (mid < 80) return 'perf-tier-4' as const
                    return 'perf-tier-5' as const
                  }),
                },
                {
                  buckets: distributions_tab.max_killing_spree_buckets ?? [],
                  title:
                    locale === 'en' ? 'Killing spree (max)' : 'Folie meurtrière (max)',
                  colorToken: 'chart-series-6' as const,
                  xAxisLabel:
                    locale === 'en' ? 'Killing spree' : 'Folie meurtrière',
                  colorTokenByBucket: undefined,
                },
              ]).map((cfg, i) => (
                <ChartFrame key={i} title={cfg.title}>
                  {cfg.buckets.length > 0 ? (
                    <TimeseriesDistributionHistogram
                      buckets={cfg.buckets}
                      colorToken={cfg.colorToken}
                      xAxisLabel={cfg.xAxisLabel}
                      medianLabel={locale === 'en' ? 'Median' : 'Médiane'}
                      colorTokenByBucket={cfg.colorTokenByBucket}
                    />
                  ) : (
                    <EmptyStateNotice
                      title={t('timeseries.empty.page_title')}
                      description={t('timeseries.empty.no_data_description')}
                    />
                  )}
                </ChartFrame>
              ))}
            </div>

            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              {locale === 'en' ? 'Correlations' : 'Corrélations'}
            </h3>
            {/* 4 scatters en grille 2×2 + MMR seul en bas (pleine largeur). */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {([
                {
                  metricXKey: 'lifespan',
                  metricYKey: 'kills',
                  title: locale === 'en' ? 'Life vs Kills' : 'Vie vs Frags',
                  xLabel: locale === 'en' ? 'Life (s)' : 'Vie (s)',
                  yLabel:
                    fieldMappings?.fields['kills']?.label ?? 'Frags',
                },
                {
                  metricXKey: 'accuracy',
                  metricYKey: 'kda',
                  title: locale === 'en' ? 'Accuracy vs KDA' : 'Précision vs FDA',
                  xLabel: `${fieldMappings?.fields['accuracy']?.label ?? 'Précision'} (%)`,
                  yLabel: fieldMappings?.fields['kda']?.label ?? 'FDA',
                },
                {
                  metricXKey: 'lifespan',
                  metricYKey: 'deaths',
                  title: locale === 'en' ? 'Life vs Deaths' : 'Vie vs Morts',
                  xLabel: locale === 'en' ? 'Life (s)' : 'Vie (s)',
                  yLabel:
                    fieldMappings?.fields['deaths']?.label ?? 'Morts',
                },
                {
                  metricXKey: 'kills',
                  metricYKey: 'deaths',
                  title: locale === 'en' ? 'Kills vs Deaths' : 'Frags vs Morts',
                  xLabel:
                    fieldMappings?.fields['kills']?.label ?? 'Frags',
                  yLabel:
                    fieldMappings?.fields['deaths']?.label ?? 'Morts',
                },
              ] as const).map((cfg) => (
                <ChartFrame key={`${cfg.metricXKey}_${cfg.metricYKey}`} title={cfg.title}>
                  <TimeseriesScatterWithTrend
                    points={distributions_tab.correlation_points ?? []}
                    metricXKey={cfg.metricXKey}
                    metricYKey={cfg.metricYKey}
                    xAxisLabel={cfg.xLabel}
                    yAxisLabel={cfg.yLabel}
                    outcomeLabels={outcomeLabels}
                    trendLabel={locale === 'en' ? 'Trend' : 'Tendance'}
                    emptyTitle={t('timeseries.empty.page_title')}
                    emptyDescription={t('timeseries.empty.no_data_description')}
                    height={240}
                  />
                </ChartFrame>
              ))}
            </div>

            {/* MMR équipe / adverse — seul sur sa propre ligne. */}
            <ChartFrame
              title={
                locale === 'en'
                  ? 'Team MMR vs Enemy MMR'
                  : 'MMR équipe vs MMR adverse'
              }
            >
              <TimeseriesScatterWithTrend
                points={distributions_tab.correlation_points ?? []}
                metricXKey="mmr_team"
                metricYKey="mmr_enemy"
                xAxisLabel={fieldMappings?.fields['team_mmr']?.label ?? 'MMR équipe'}
                yAxisLabel={fieldMappings?.fields['enemy_mmr']?.label ?? 'MMR adverse'}
                outcomeLabels={outcomeLabels}
                trendLabel={locale === 'en' ? 'Trend' : 'Tendance'}
                emptyTitle={t('timeseries.empty.page_title')}
                emptyDescription={t('timeseries.empty.no_data_description')}
                height={320}
              />
            </ChartFrame>
          </div>
        )}

        {/* Progression — charts timeseries.11..19 (mock spec) */}
        {activeTab === 'progression' && (
          <div className="space-y-8">
            {/* timeseries.11 — Premier événement (gauche) | timeseries.14 — Par minute (droite) */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <ChartFrame
                title={
                  locale === 'en'
                    ? 'First kill / first death timing'
                    : 'Temps du premier frag / première mort'
                }
              >
                {data.first_events && data.first_events.buckets.length > 0 ? (
                  <TimeseriesFirstEventDistribution
                    data={data.first_events}
                    killsLabel={locale === 'en' ? '1st kill' : '1er frag'}
                    deathsLabel={locale === 'en' ? '1st death' : '1ère mort'}
                    meanLabel={locale === 'en' ? 'Avg' : 'Moy.'}
                    xAxisLabel={locale === 'en' ? 'Time (m s)' : 'Temps (m s)'}
                  />
                ) : (
                  <EmptyStateNotice
                    title={t('timeseries.empty.page_title')}
                    description={t('timeseries.empty.no_data_description')}
                  />
                )}
              </ChartFrame>

              <ChartFrame
                title={locale === 'en' ? 'Stats per minute' : 'Stats par minute'}
              >
                <TimeseriesPerMinuteTrend
                  rows={data.match_rows ?? []}
                  killsLabel={fieldMappings?.fields['kills_per_minute']?.label ?? 'Frags / min'}
                  deathsLabel={fieldMappings?.fields['deaths_per_minute']?.label ?? 'Morts / min'}
                  assistsLabel={fieldMappings?.fields['assists_per_minute']?.label ?? 'Assistances / min'}
                  perMinuteSuffix={locale === 'en' ? ' /min' : ' /min'}
                />
              </ChartFrame>
            </div>

            {/* timeseries.12 (gauche) | timeseries.16 (droite) */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <ChartFrame title={locale === 'en' ? 'Performance' : 'Performance'}>
                <TimeseriesPerformanceTrend
                  rows={data.match_rows ?? []}
                  smoothingLabel={locale === 'en' ? 'Trend' : 'Tendance'}
                />
              </ChartFrame>

              <ChartFrame
                title={
                  locale === 'en'
                    ? 'Killing spree / Headshots / Perfect kills'
                    : 'Folie meurtrière / Tirs à la tête / Frags parfaits'
                }
              >
                <TimeseriesSpreeHeadshots
                  rows={data.match_rows ?? []}
                  spreeLabel={locale === 'en' ? 'Killing spree (max)' : 'Folie meurtrière (max)'}
                  headshotsLabel={fieldMappings?.fields['headshot_kills']?.label ?? 'Tirs à la tête'}
                  perfectLabel={
                    fieldMappings?.fields['perfect_kills']?.label ??
                    (locale === 'en' ? 'Perfect kills' : 'Kills parfaits')
                  }
                />
              </ChartFrame>
            </div>

            {/* timeseries.19 (gauche) | Skill rank + Performance (droite) */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <ChartFrame
                title={locale === 'en' ? 'Rank and personal score' : 'Rang et score personnel'}
              >
                <TimeseriesRankScore
                  rows={data.match_rows ?? []}
                  scoreLabel={fieldMappings?.fields['personal_score']?.label ?? 'Score personnel'}
                  rankLabel={
                    fieldMappings?.fields['rank']?.label ??
                    (locale === 'en' ? 'Rank' : 'Rang')
                  }
                />
              </ChartFrame>

              <ChartFrame
                title={
                  locale === 'en'
                    ? 'Skill rank and performance'
                    : 'Skill rank et performance'
                }
              >
                <TimeseriesSkillRankPerformance
                  rows={data.match_rows ?? []}
                  ratingLabel={locale === 'en' ? 'Skill rating' : 'Skill rating'}
                  perfLabel={
                    fieldMappings?.fields['performance_score']?.label ??
                    (locale === 'en' ? 'Performance' : 'Performance')
                  }
                />
              </ChartFrame>
            </div>

            {/* Rendement & Résistance — pleine largeur. */}
            <ChartFrame
              title={locale === 'en' ? 'Output & resistance' : 'Rendement & Résistance'}
            >
              <TimeseriesEfficiency
                rows={data.match_rows ?? []}
                rendementLabel={locale === 'en' ? 'Output' : 'Rendement'}
                resistanceLabel={locale === 'en' ? 'Resistance' : 'Résistance'}
                refLabel={locale === 'en' ? 'Ref. 1.0' : 'Réf. 1.0'}
              />
            </ChartFrame>

            {/* Engagement — pleine largeur. EngagementTimeseriesSection
                rend déjà sa propre ChartCard avec titre interne, donc pas de
                wrapper supplémentaire (sinon double titre). */}
            <EngagementTimeseriesSection playerSlug={playerSlug} limit={30} />

            {/* Intensité — frags par phase de match (pleine largeur). */}
            {(data.intensity_rows ?? []).length > 0 && (
              <ChartFrame
                title={
                  locale === 'en'
                    ? 'Intensity'
                    : 'Intensité'
                }
              >
                <TimeseriesIntensityHeatmap
                  rows={data.intensity_rows ?? []}
                  zLabel={locale === 'en' ? 'kills' : 'frags'}
                  height={Math.max(200, Math.min(640, (data.intensity_rows ?? []).length * 18 + 80))}
                />
              </ChartFrame>
            )}

            {/* Historique des matchs — tableau Explorer en bas de Progression.
                Reflète le scope solo du filtre global (mêmes matchs que les
                charts ci-dessus). */}
            {explorerMatchesQuery.data?.table?.items &&
              explorerMatchesQuery.data.table.items.length > 0 && (
                <ChartFrame
                  title={locale === 'en' ? 'Match history' : 'Historique des matchs'}
                >
                  <ExplorerMatchesTable
                    rows={explorerMatchesQuery.data.table.items}
                    playerSlug={playerSlug}
                    alwaysShowPagination
                  />
                </ChartFrame>
              )}
          </div>
        )}

      </div>
    </div>
  )
}
