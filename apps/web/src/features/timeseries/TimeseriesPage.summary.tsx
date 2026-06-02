/**
 * TimeseriesPage — onglet "Summary".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : outcome sequence + KDA trend + KDA density + avg life + assists +
 * top weapons + KDA trend value + perf session/week/month + map win-rate/perf.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeValue,
} from '@/components/charts/OutcomeSequenceTape'
import { TimeseriesKdaTrend } from './TimeseriesKdaTrend'
import { TimeseriesKdaDensity } from './TimeseriesKdaDensity'
import { TimeseriesTopWeapons } from './TimeseriesTopWeapons'
import {
  TimeseriesAssistsTrend,
  TimeseriesAvgLifeTrend,
  TimeseriesKdaValueTrend,
} from './TimeseriesFormCharts'
import { TimeseriesSessionPerformance } from './TimeseriesSquadAdapted'
import { ChartFrame } from './ChartFrame'
import { WinRateVsHistoryBulletChart } from '@/features/squad/WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from '@/features/squad/MapPerfVsHistoryChart'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'

function outcomeNumToValue(n: number | null): OutcomeValue {
  if (n === 2) return 'win'
  if (n === 3) return 'loss'
  if (n === 1) return 'tie'
  return 'dnf'
}

export interface OutcomeLabels {
  win: string
  loss: string
  tie: string
  dnf: string
  unknown: string
}

export interface TimeseriesSummaryTabProps {
  data: TimeseriesPageResponse
  t: (key: TimeseriesManifestKey) => string
  fieldMappings: FieldMappingsResponse | undefined
  outcomeLabels: OutcomeLabels
  mapLabelOf: (mapUI: string) => string
}

export function TimeseriesSummaryTab({
  data,
  t,
  fieldMappings,
  outcomeLabels,
  mapLabelOf,
}: TimeseriesSummaryTabProps) {
  return (
    <div className="space-y-6">
      {/* Séquence des résultats — en haut */}
      {(data.match_rows ?? []).length > 0 && (
        <div data-testid="timeseries-outcome-sequence">
          <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t('timeseries.summary.outcome_sequence')}
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
          title={t('timeseries.summary.kd_timeline_title')}
        >
          <TimeseriesKdaTrend
            rows={data.match_rows ?? []}
            labels={{
              kills: fieldMappings?.fields['kills']?.label ?? 'Frags',
              deaths: fieldMappings?.fields['deaths']?.label ?? 'Morts',
              yAxis: t('timeseries.summary.kd_yaxis'),
            }}
          />
        </ChartFrame>

        <ChartFrame title={t('timeseries.summary.kda_distribution_title')}>
          <TimeseriesKdaDensity
            buckets={data.distributions_tab.kda_buckets ?? []}
            rows={data.match_rows ?? []}
            labels={{
              density: t('timeseries.summary.density'),
              rug: fieldMappings?.fields['kda']?.label ?? 'FDA',
              xAxis: fieldMappings?.fields['kda']?.label ?? 'FDA',
              mean: t('timeseries.summary.mean'),
              median: t('timeseries.summary.median'),
            }}
          />
        </ChartFrame>
      </div>

      {/* Durée de vie moyenne (gauche) | Assistances (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <ChartFrame title={fieldMappings?.fields['avg_life_seconds']?.label ?? 'Durée de vie moyenne'}>
          <TimeseriesAvgLifeTrend
            rows={data.match_rows ?? []}
            lifeLabel={t('timeseries.summary.avg_life_axis')}
          />
        </ChartFrame>

        <ChartFrame title={fieldMappings?.fields['assists']?.label ?? 'Assistances'}>
          <TimeseriesAssistsTrend
            rows={data.match_rows ?? []}
            assistsLabel={
              fieldMappings?.fields['assists']?.label ??
              'Assistances'
            }
            smoothingLabel={t('timeseries.summary.trend')}
          />
        </ChartFrame>
      </div>

      {/* Outils de destruction (gauche) | Résultats par période (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <ChartFrame
          title={t('timeseries.summary.top_weapons_title')}
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
              smoothingLabel={t('timeseries.summary.trend')}
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
            if (g === 'week') return t('timeseries.summary.solo_perf_week')
            if (g === 'month') return t('timeseries.summary.solo_perf_month')
            return t('timeseries.summary.solo_perf_session')
          })()}
        >
          <TimeseriesSessionPerformance
            points={data.solo_session_perf.points}
            granularity={data.solo_session_perf.granularity}
            perfLabel={
              fieldMappings?.fields['performance_score']?.label ??
              t('timeseries.summary.perf_label')
            }
            winRateLabel={
              fieldMappings?.fields['win_rate']?.label ??
              t('timeseries.summary.win_rate_label')
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
            title={t('timeseries.summary.winrate_vs_history_title')}
            rows={data.map_breakdown ?? []}
            mapLabelOf={mapLabelOf}
            sessionLabel={t('timeseries.summary.session_label')}
            historyLabel={t('timeseries.summary.history_label')}
            parityLabel={t('timeseries.summary.parity_label')}
            zeroWinrateLabel={t('timeseries.summary.zero_winrate_label')}
          />
          {(data.map_breakdown ?? []).some(
            (r) =>
              r.performance_avg !== undefined &&
              r.historical_performance_avg !== undefined,
          ) && (
            <MapPerfVsHistoryChart
              title={t('timeseries.summary.mapperf_vs_history_title')}
              rows={data.map_breakdown ?? []}
              mapLabelOf={mapLabelOf}
              sessionLabel={t('timeseries.summary.session_label')}
              historyLabel={t('timeseries.summary.history_label')}
            />
          )}
        </div>
      )}
    </div>
  )
}
