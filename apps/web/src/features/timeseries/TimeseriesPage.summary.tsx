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
  locale: 'fr' | 'en'
  t: (key: TimeseriesManifestKey) => string
  fieldMappings: FieldMappingsResponse | undefined
  outcomeLabels: OutcomeLabels
  mapLabelOf: (mapUI: string) => string
}

export function TimeseriesSummaryTab({
  data,
  locale,
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
            buckets={data.distributions_tab.kda_buckets ?? []}
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
  )
}
