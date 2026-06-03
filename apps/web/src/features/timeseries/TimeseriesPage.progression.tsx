/**
 * TimeseriesPage — onglet "Progression".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : first event, per minute, performance, spree/headshots, rank score,
 * skill rank perf, efficiency, engagement section, intensity heatmap + table.
 */
import { useMemo } from 'react'
import { type ColumnDef } from '@tanstack/react-table'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility'
import { formatWinProb } from '@/lib/winProbCategory'
import { TimeseriesFirstEventDistribution } from './TimeseriesFirstEventDistribution'
import {
  TimeseriesPerformanceTrend,
  TimeseriesPerMinuteTrend,
  TimeseriesSpreeHeadshots,
  TimeseriesRankScore,
  TimeseriesSkillRankPerformance,
} from './TimeseriesFormCharts'
import {
  TimeseriesEfficiency,
  TimeseriesIntensityHeatmap,
} from './TimeseriesSquadAdapted'
import { ChartFrame } from './ChartFrame'
import { EngagementTimeseriesSection } from '@/features/engagement/EngagementTimeseriesSection'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { TimeseriesSkillProgression } from './TimeseriesSkillProgression'
import type { FilterContextInput, TimeseriesPageResponse, ExplorerMatchRow } from '@/lib/api/types'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'

export interface TimeseriesProgressionTabProps {
  data: TimeseriesPageResponse
  playerSlug: string
  locale: 'fr' | 'en'
  t: (key: TimeseriesManifestKey) => string
  fieldMappings: FieldMappingsResponse | undefined
  soloFilterContext: FilterContextInput
  filterContextHash: string
  explorerMatchRows: ExplorerMatchRow[] | undefined
}

export function TimeseriesProgressionTab({
  data,
  playerSlug,
  locale,
  t,
  fieldMappings,
  soloFilterContext,
  filterContextHash,
  explorerMatchRows,
}: TimeseriesProgressionTabProps) {
  // Colonne « Prob. vic. » (expected_win_prob, LUSR v2) injectée après « Résultat »
  // dans le tableau historique — spécifique à cette vue (pas sur la page Explorer).
  const winProbColumns = useMemo<ColumnDef<ExplorerMatchRow>[]>(
    () => [
      {
        id: 'expected_win_prob',
        header: t('timeseries.progression.col_win_prob'),
        cell: (ctx) => {
          const v = ctx.row.original.expected_win_prob
          if (v == null) return <span className="text-muted-foreground">-</span>
          const color = v >= 0.5 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
          return (
            <span className="font-mono tabular-nums" style={{ color }}>
              {formatWinProb(v)}
            </span>
          )
        },
      },
    ],
    [t],
  )
  return (
    <div className="space-y-8">
      {/* timeseries.11 — Premier événement (gauche) | timeseries.14 — Par minute (droite) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ChartFrame
          title={t('timeseries.progression.first_event_title')}
        >
          {data.first_events && data.first_events.buckets.length > 0 ? (
            <TimeseriesFirstEventDistribution
              data={data.first_events}
              killsLabel={t('timeseries.progression.first_kill')}
              deathsLabel={t('timeseries.progression.first_death')}
              meanLabel={t('timeseries.progression.avg')}
              xAxisLabel={t('timeseries.progression.time_axis')}
            />
          ) : (
            <EmptyStateNotice
              title={t('timeseries.empty.page_title')}
              description={t('timeseries.empty.no_data_description')}
            />
          )}
        </ChartFrame>

        <ChartFrame
          title={t('timeseries.progression.per_minute_title')}
        >
          <TimeseriesPerMinuteTrend
            rows={data.match_rows ?? []}
            killsLabel={fieldMappings?.fields['kills_per_minute']?.label ?? 'Frags / min'}
            deathsLabel={fieldMappings?.fields['deaths_per_minute']?.label ?? 'Morts / min'}
            assistsLabel={fieldMappings?.fields['assists_per_minute']?.label ?? 'Assistances / min'}
            perMinuteSuffix={t('timeseries.progression.per_minute_suffix')}
          />
        </ChartFrame>
      </div>

      {/* timeseries.12 (gauche) | timeseries.16 (droite) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ChartFrame title={t('timeseries.summary.perf_label')}>
          <TimeseriesPerformanceTrend
            rows={data.match_rows ?? []}
            smoothingLabel={t('timeseries.summary.trend')}
          />
        </ChartFrame>

        <ChartFrame
          title={t('timeseries.progression.spree_headshots_title')}
        >
          <TimeseriesSpreeHeadshots
            rows={data.match_rows ?? []}
            spreeLabel={t('timeseries.progression.spree_label')}
            headshotsLabel={fieldMappings?.fields['headshot_kills']?.label ?? 'Tirs à la tête'}
            perfectLabel={
              fieldMappings?.fields['perfect_kills']?.label ??
              t('timeseries.progression.perfect_kills')
            }
          />
        </ChartFrame>
      </div>

      {/* Progression CSR (classé) ou LUSR (non classé) — pleine largeur, avant le bloc rank+perf. */}
      <TimeseriesSkillProgression rows={data.match_rows ?? []} locale={locale} />

      {/* timeseries.19 (gauche) | Skill rank + Performance (droite) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ChartFrame
          title={t('timeseries.progression.rank_score_title')}
        >
          <TimeseriesRankScore
            rows={data.match_rows ?? []}
            scoreLabel={fieldMappings?.fields['personal_score']?.label ?? 'Score personnel'}
            rankLabel={
              fieldMappings?.fields['rank']?.label ??
              t('timeseries.progression.rank')
            }
          />
        </ChartFrame>

        <ChartFrame
          title={t('timeseries.progression.rank_perf_title')}
        >
          <TimeseriesSkillRankPerformance
            rows={data.match_rows ?? []}
            ratingLabel={t('timeseries.progression.rank')}
            perfLabel={
              fieldMappings?.fields['performance_score']?.label ??
              t('timeseries.summary.perf_label')
            }
          />
        </ChartFrame>
      </div>

      {/* Rendement & Résistance — pleine largeur. */}
      <ChartFrame
        title={t('timeseries.progression.efficiency_title')}
      >
        <TimeseriesEfficiency
          rows={data.match_rows ?? []}
          rendementLabel={fieldMappings?.fields['offensive_conversion']?.label ?? 'Rendement'}
          resistanceLabel={fieldMappings?.fields['defensive_resistance']?.label ?? 'Résistance'}
          refLabel={t('timeseries.progression.ref_100')}
        />
      </ChartFrame>

      {/* Engagement — pleine largeur. EngagementTimeseriesSection
          rend déjà sa propre ChartCard avec titre interne, donc pas de
          wrapper supplémentaire (sinon double titre). */}
      <EngagementTimeseriesSection
        playerSlug={playerSlug}
        filters={soloFilterContext}
        filterHash={filterContextHash}
        limit={30}
      />

      {/* Intensité — frags par phase de match (pleine largeur). */}
      {(data.intensity_rows ?? []).length > 0 && (
        <ChartFrame
          title={t('timeseries.progression.intensity_title')}
        >
          <TimeseriesIntensityHeatmap
            rows={data.intensity_rows ?? []}
            zLabel={t('timeseries.progression.intensity_z')}
            height={Math.max(200, Math.min(640, (data.intensity_rows ?? []).length * 18 + 80))}
          />
        </ChartFrame>
      )}

      {/* Historique des matchs — tableau Explorer standalone (sans bloc ni titre)
          en bas de Progression. Reflète le scope solo du filtre global (mêmes
          matchs que les charts ci-dessus). Colonne « Prob. vic. » injectée après
          « Résultat ». */}
      {explorerMatchRows && explorerMatchRows.length > 0 && (
        <ExplorerMatchesTable
          rows={explorerMatchRows}
          playerSlug={playerSlug}
          alwaysShowPagination
          extraColumns={winProbColumns}
          extraColumnsAfterId="outcome_code"
        />
      )}

    </div>
  )
}
