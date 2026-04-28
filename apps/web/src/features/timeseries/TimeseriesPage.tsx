/**
 * TimeseriesPage — page Séries temporelles (6 onglets).
 *
 * Phase 2 P2.E : migration partielle Plotly → ECharts pour les charts qui ont
 * un wrapper ECharts disponible (TimeseriesLineChart, Heatmap2DChart). Les
 * histogrammes, scatters, KDA bars et combat-yield restent sur Plotly en
 * attendant des wrappers ECharts dédiés (différé Phase 3). i18n via
 * `timeseriesManifest` + `formatMessage`.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { useTimeseriesPage, useCombatYieldHistory } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { DeltaCard } from '@/components/ui/delta-card'
import { TimeseriesKdaBars } from './TimeseriesKdaBars'
import { TimeseriesCombatYield } from './TimeseriesCombatYield'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import { HistogramChart } from '@/components/charts/HistogramChart'
import { TimeseriesCorrelationScatter } from './TimeseriesCorrelationScatter'
import type { TimeseriesKpiCard } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import {
  timeseriesManifest,
  type TimeseriesManifestKey,
} from '@/lib/i18n/generated/timeseries'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  cumulativePointsToSeries,
  heatmapCellsToSeries,
  distributionBucketsToSeries,
  DOW_LABELS_FR,
  DOW_LABELS_EN,
} from './seriesAdapters'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { EngagementTimeseriesSection } from '@/features/engagement/EngagementTimeseriesSection'

type TabId = 'summary' | 'cumul' | 'form' | 'intensity' | 'distributions' | 'combat'

const TAB_KEYS: { id: TabId; key: TimeseriesManifestKey }[] = [
  { id: 'summary', key: 'timeseries.tabs.summary' },
  { id: 'cumul', key: 'timeseries.tabs.cumul' },
  { id: 'form', key: 'timeseries.tabs.form' },
  { id: 'intensity', key: 'timeseries.tabs.intensity' },
  { id: 'distributions', key: 'timeseries.tabs.distributions' },
  { id: 'combat', key: 'timeseries.tabs.combat' },
]

export function TimeseriesPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const [activeTab, setActiveTab] = useState<TabId>('summary')
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: TimeseriesManifestKey) => formatMessage(timeseriesManifest, key, locale)
  const dowLabels = locale === 'en' ? DOW_LABELS_EN : DOW_LABELS_FR
  const { data: fieldMappings } = useFieldMappings()
  const outcomeLabels = {
    win: fieldMappings?.outcomes?.win?.label ?? t('timeseries.distributions.outcome_win_fallback'),
    loss:
      fieldMappings?.outcomes?.loss?.label ??
      t('timeseries.distributions.outcome_loss_fallback'),
    unknown:
      fieldMappings?.fields['outcome_unknown']?.label ??
      t('timeseries.distributions.outcome_unknown_fallback'),
  }

  const { data, isLoading, isError, refetch } = useTimeseriesPage(
    playerSlug,
    { filters: filterContext },
    filterContextHash,
  )

  const { data: combatData, isLoading: combatLoading } = useCombatYieldHistory(
    playerSlug,
    filterContextHash,
    filterContext,
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

  const { summary_tab, cumul_tab, form_tab, intensity_tab, distributions_tab } = data

  return (
    <div className="flex flex-col">
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
        {/* KPIs */}
        {activeTab === 'summary' && (
          <div className="space-y-6">
            {summary_tab.kpi_cards.length > 0 ? (
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
                {summary_tab.kpi_cards.map((card: TimeseriesKpiCard) => (
                  <Card key={card.key}>
                    <CardContent className="py-3 text-center">
                      <p className="text-xs text-muted-foreground">{card.label}</p>
                      <p
                        className="text-xl font-bold"
                        style={{ color: card.color ?? undefined }}
                      >
                        {card.value}
                      </p>
                      {card.delta && (
                        <p className="text-xs text-muted-foreground">{card.delta}</p>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title={t('timeseries.summary.empty_title')}
                description={t('timeseries.summary.empty_description')}
              />
            )}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.summary.kda_timeline_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesKdaBars
                  rows={data.match_rows ?? []}
                  labels={{
                    kills: t('timeseries.summary.kda_kills'),
                    deaths: t('timeseries.summary.kda_deaths'),
                    kdRatio: t('timeseries.summary.kda_ratio'),
                    yAxisLeft: t('timeseries.summary.kda_y_axis_left'),
                    yAxisRight: t('timeseries.summary.kda_y_axis_right'),
                    emptyTitle: t('timeseries.summary.kda_empty_title'),
                    emptyDescription: t('timeseries.summary.kda_empty_description'),
                  }}
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Cumul — ECharts */}
        {activeTab === 'cumul' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.cumul.kd_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={cumulativePointsToSeries(cumul_tab.cumulative_kd ?? [], {
                    key: 'timeseries.cumul.kd',
                    name: t('timeseries.cumul.kd_series_label'),
                  })}
                  xAxisType="time"
                  outcomeMarkers={false}
                  height={300}
                  emptyMessage={t('timeseries.empty.no_data_description')}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.cumul.net_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={cumulativePointsToSeries(cumul_tab.cumulative_net ?? [], {
                    key: 'timeseries.cumul.net',
                    name: t('timeseries.cumul.net_series_label'),
                  })}
                  xAxisType="time"
                  outcomeMarkers={false}
                  height={300}
                  emptyMessage={t('timeseries.empty.no_data_description')}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.cumul.rolling_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={cumulativePointsToSeries(cumul_tab.rolling_kd ?? [], {
                    key: 'timeseries.cumul.rolling',
                    name: t('timeseries.cumul.rolling_series_label'),
                  })}
                  xAxisType="time"
                  outcomeMarkers={false}
                  height={300}
                  emptyMessage={t('timeseries.empty.no_data_description')}
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Forme */}
        {activeTab === 'form' && (
          <div className="space-y-6">
            {form_tab.regression_stats.has_enough_for_trend ? (
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <DeltaCard
                  label={t('timeseries.form.kd_slope')}
                  value={form_tab.regression_stats.kd_slope?.toFixed(4) ?? '—'}
                  unit={t('timeseries.form.per_match')}
                  delta={form_tab.regression_stats.kd_slope}
                  warning={(form_tab.regression_stats.r_squared ?? 0) < 0.3}
                  warningText={t('timeseries.form.r_squared_warning')}
                />
                <DeltaCard
                  label={t('timeseries.form.winrate_slope')}
                  value={
                    form_tab.regression_stats.winrate_slope != null
                      ? `${(form_tab.regression_stats.winrate_slope * 100).toFixed(2)}%`
                      : '—'
                  }
                  unit={t('timeseries.form.per_match')}
                  delta={form_tab.regression_stats.winrate_slope}
                />
                <DeltaCard
                  label={t('timeseries.form.r_squared')}
                  value={form_tab.regression_stats.r_squared?.toFixed(3) ?? '—'}
                  warning={(form_tab.regression_stats.r_squared ?? 0) < 0.3}
                  warningText={t('timeseries.form.r_squared_unreliable')}
                />
              </div>
            ) : (
              <EmptyStateNotice
                title={t('timeseries.form.empty_title')}
                description={t('timeseries.form.empty_description')}
              />
            )}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.form.ewma_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={cumulativePointsToSeries(form_tab.ewma_kd_points ?? [], {
                    key: 'timeseries.form.ewma',
                    name: t('timeseries.form.ewma_series_label'),
                  })}
                  xAxisType="time"
                  outcomeMarkers={false}
                  height={300}
                  emptyMessage={t('timeseries.empty.no_data_description')}
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Intensité — Heatmap ECharts + line ECharts */}
        {activeTab === 'intensity' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.intensity.heatmap_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <Heatmap2DChart
                  series={heatmapCellsToSeries(intensity_tab.heatmap_data ?? [], {
                    key: 'timeseries.intensity.heatmap',
                    name: t('timeseries.intensity.heatmap_title'),
                    dowLabels,
                  })}
                  paletteMode="sequential"
                  height={320}
                  emptyMessage={t('timeseries.empty.no_data_description')}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.intensity.score_per_min_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={cumulativePointsToSeries(intensity_tab.score_per_min_data ?? [], {
                    key: 'timeseries.intensity.score_per_min',
                    name: t('timeseries.intensity.score_per_min_series_label'),
                  })}
                  xAxisType="time"
                  outcomeMarkers={false}
                  height={300}
                  emptyMessage={t('timeseries.empty.no_data_description')}
                />
              </CardContent>
            </Card>
            <EngagementTimeseriesSection playerSlug={playerSlug} limit={30} />
          </div>
        )}

        {/* Distributions */}
        {activeTab === 'distributions' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">{t('timeseries.distributions.kda_title')}</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <HistogramChart
                    series={distributionBucketsToSeries(distributions_tab.kda_buckets ?? [], {
                      key: 'timeseries.distributions.kda',
                      name: t('timeseries.distributions.kda_axis_x'),
                    })}
                    colorToken="perf-tier-2"
                    xAxisLabel={t('timeseries.distributions.kda_axis_x')}
                    height={280}
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">{t('timeseries.distributions.kills_title')}</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <HistogramChart
                    series={distributionBucketsToSeries(distributions_tab.kills_buckets ?? [], {
                      key: 'timeseries.distributions.kills',
                      name: t('timeseries.distributions.kills_axis_x'),
                    })}
                    colorToken="divergent-pos"
                    xAxisLabel={t('timeseries.distributions.kills_axis_x')}
                    height={280}
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">{t('timeseries.distributions.accuracy_title')}</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <HistogramChart
                    series={distributionBucketsToSeries(distributions_tab.accuracy_buckets ?? [], {
                      key: 'timeseries.distributions.accuracy',
                      name: t('timeseries.distributions.accuracy_axis_x'),
                    })}
                    colorToken="perf-tier-3"
                    xAxisLabel={t('timeseries.distributions.accuracy_axis_x')}
                    height={280}
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">{t('timeseries.distributions.score_per_min_title')}</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <HistogramChart
                    series={distributionBucketsToSeries(
                      distributions_tab.score_per_min_buckets ?? [],
                      {
                        key: 'timeseries.distributions.score_per_min',
                        name: t('timeseries.distributions.score_per_min_axis_x'),
                      },
                    )}
                    colorToken="narrative-dominant"
                    xAxisLabel={t('timeseries.distributions.score_per_min_axis_x')}
                    height={280}
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">{t('timeseries.distributions.rolling_wr_title')}</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <HistogramChart
                    series={distributionBucketsToSeries(
                      distributions_tab.rolling_wr_buckets ?? [],
                      {
                        key: 'timeseries.distributions.rolling_wr',
                        name: t('timeseries.distributions.rolling_wr_axis_x'),
                      },
                    )}
                    colorToken="divergent-neg"
                    xAxisLabel={t('timeseries.distributions.rolling_wr_axis_x')}
                    height={280}
                  />
                </CardContent>
              </Card>
            </div>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.distributions.correlations_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesCorrelationScatter
                  points={distributions_tab.correlation_points ?? []}
                  outcomeLabels={outcomeLabels}
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Combat */}
        {activeTab === 'combat' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">{t('timeseries.combat.yield_title')}</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                {combatLoading ? (
                  <div className="flex justify-center py-8">
                    <span className="text-muted-foreground text-sm">
                      {t('timeseries.combat.loading')}
                    </span>
                  </div>
                ) : (
                  <TimeseriesCombatYield
                    rows={combatData?.table.items ?? []}
                    labels={{
                      ocSeries: t('timeseries.combat.oc_series'),
                      drSeries: t('timeseries.combat.dr_series'),
                      ocReference: t('timeseries.combat.oc_reference'),
                      drReference: t('timeseries.combat.dr_reference'),
                      emptyTitle: t('timeseries.combat.empty_title'),
                      emptyDescription: t('timeseries.combat.empty_description'),
                    }}
                  />
                )}
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  )
}
