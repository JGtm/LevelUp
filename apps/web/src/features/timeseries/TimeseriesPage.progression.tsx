/**
 * TimeseriesPage — onglet "Progression".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : first event, per minute, performance, spree/headshots, rank score,
 * skill rank perf, efficiency, engagement section, intensity heatmap + table.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
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
  return (
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
          title={locale === 'en' ? 'Rank and performance' : 'Rang et performance'}
        >
          <TimeseriesSkillRankPerformance
            rows={data.match_rows ?? []}
            ratingLabel={locale === 'en' ? 'Rank' : 'Rang'}
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
      <EngagementTimeseriesSection
        playerSlug={playerSlug}
        filters={soloFilterContext}
        filterHash={filterContextHash}
        limit={30}
      />

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
      {explorerMatchRows && explorerMatchRows.length > 0 && (
        <ChartFrame
          title={locale === 'en' ? 'Match history' : 'Historique des matchs'}
        >
          <ExplorerMatchesTable
            rows={explorerMatchRows}
            playerSlug={playerSlug}
            alwaysShowPagination
          />
        </ChartFrame>
      )}

    </div>
  )
}
