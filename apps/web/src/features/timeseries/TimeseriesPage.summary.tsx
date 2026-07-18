/**
 * TimeseriesPage — onglet "Summary".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : outcome sequence + KDA trend + KDA density + avg life + assists +
 * top weapons + KDA trend value + perf session/week/month + map win-rate/perf.
 */
import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { outcomeCodeToTapeValue } from '@/lib/outcome'
import { TimeseriesKdaTrend } from './TimeseriesKdaTrend'
import { TimeseriesKdaDensity } from './TimeseriesKdaDensity'
import { TimeseriesTopWeapons } from './TimeseriesTopWeapons'
import { TimeseriesKillTypesDonut } from './TimeseriesKillTypesDonut'
import {
  TimeseriesAssistsTrend,
  TimeseriesAvgLifeTrend,
  TimeseriesKdaValueTrend,
} from './TimeseriesFormCharts'
import { TimeseriesSessionPerformance } from './TimeseriesSquadAdapted'
import { WinRateVsHistoryBulletChart } from '@/features/squad/WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from '@/features/squad/MapPerfVsHistoryChart'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'

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
  const emptyMsg = t('timeseries.empty.no_data_description')
  // MMR par match indisponible (titre sans `team_mmr`, ex. Halo 5) → la série MMR
  // de « Performance par session » est retirée (data + axe + légende).
  const hasTeamMmr = useCapability('team_mmr')
  const soloPerf = data.solo_session_perf
  const soloGranularity: 'session' | 'week' | 'month' =
    soloPerf?.granularity === 'week' || soloPerf?.granularity === 'month'
      ? soloPerf.granularity
      : 'session'
  const soloPerfTitle =
    soloGranularity === 'week'
      ? t('timeseries.summary.solo_perf_week')
      : soloGranularity === 'month'
        ? t('timeseries.summary.solo_perf_month')
        : t('timeseries.summary.solo_perf_session')
  return (
    <div className="space-y-6">
      {/* Séquence des résultats — en haut. Libellé conservé + message court
          quand aucun match, au lieu de masquer le bloc. */}
      <div data-testid="timeseries-outcome-sequence">
        <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t('timeseries.summary.outcome_sequence')}
        </p>
        {(data.match_rows ?? []).length > 0 ? (
          <OutcomeSequenceTape
            matches={(data.match_rows ?? []).map<OutcomePoint>((r) => ({
              outcome: outcomeCodeToTapeValue(r.outcome),
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
        ) : (
          <p className="text-sm text-muted-foreground">{emptyMsg}</p>
        )}
      </div>

      {/* Évolution Frags / Morts (gauche) | Distribution FDA (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <TimeseriesKdaTrend
          title={t('timeseries.summary.kd_timeline_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          labels={{
            kills: fieldMappings?.fields['kills']?.label ?? 'Frags',
            deaths: fieldMappings?.fields['deaths']?.label ?? 'Morts',
            yAxis: t('timeseries.summary.kd_yaxis'),
          }}
        />

        <TimeseriesKdaDensity
          title={t('timeseries.summary.kda_distribution_title')}
          emptyMessage={emptyMsg}
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
      </div>

      {/* Durée de vie moyenne (gauche) | Assistances (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <TimeseriesAvgLifeTrend
          title={fieldMappings?.fields['avg_life_seconds']?.label ?? 'Durée de vie moyenne'}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          lifeLabel={t('timeseries.summary.avg_life_axis')}
        />

        <TimeseriesAssistsTrend
          title={fieldMappings?.fields['assists']?.label ?? 'Assistances'}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          assistsLabel={fieldMappings?.fields['assists']?.label ?? 'Assistances'}
          smoothingLabel={t('timeseries.summary.trend')}
        />
      </div>

      {/* Halo 5 : « répartition des frags » du viewer sur la période (mécaniques
          natives incl. assassinats + compétences spartiate). Null hors h5. */}
      <TimeseriesKillTypesDonut killTypes={data.kill_types} t={t} />

      {/* Outils de destruction (gauche) | FDA (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <TimeseriesTopWeapons
          title={t('timeseries.summary.top_weapons_title')}
          emptyMessage={emptyMsg}
          weapons={data.top_weapons ?? []}
          labels={{
            seriesName: fieldMappings?.fields['kills']?.label ?? 'Frags',
            fallbackLabel: (id) => `#${id}`,
          }}
        />

        <TimeseriesKdaValueTrend
          title={fieldMappings?.fields['kda']?.label ?? 'FDA'}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          fdaLabel={fieldMappings?.fields['kda']?.label ?? 'FDA'}
          smoothingLabel={t('timeseries.summary.trend')}
        />
      </div>

      {/* Performance solo par session/semaine/mois — agrégat backend
          sur tous les matchs solo (cross-session, granularité auto). */}
      <TimeseriesSessionPerformance
        title={soloPerfTitle}
        emptyMessage={emptyMsg}
        points={soloPerf?.points ?? []}
        granularity={soloGranularity}
        perfLabel={
          fieldMappings?.fields['performance_score']?.label ??
          t('timeseries.summary.perf_label')
        }
        winRateLabel={
          fieldMappings?.fields['win_rate']?.label ??
          t('timeseries.summary.win_rate_label')
        }
        mmrLabel={fieldMappings?.fields['team_mmr']?.label ?? 'MMR équipe'}
        showMmr={hasTeamMmr}
      />

      {/* Taux de victoire (gauche) | Performance par carte (droite) —
          réutilisation des wrappers Escouade (teammates.02 + .13). */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <WinRateVsHistoryBulletChart
          title={t('timeseries.summary.winrate_vs_history_title')}
          emptyMessage={emptyMsg}
          rows={data.map_breakdown ?? []}
          mapLabelOf={mapLabelOf}
          sessionLabel={t('timeseries.summary.session_label')}
          historyLabel={t('timeseries.summary.history_label')}
          parityLabel={t('timeseries.summary.parity_label')}
          zeroWinrateLabel={t('timeseries.summary.zero_winrate_label')}
        />
        <MapPerfVsHistoryChart
          title={t('timeseries.summary.mapperf_vs_history_title')}
          emptyMessage={emptyMsg}
          rows={data.map_breakdown ?? []}
          mapLabelOf={mapLabelOf}
          sessionLabel={t('timeseries.summary.session_label')}
          historyLabel={t('timeseries.summary.history_label')}
        />
      </div>
    </div>
  )
}
