/**
 * TimeseriesPage — onglet "Progression".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : first event, per minute, performance, spree/headshots, rank score,
 * skill rank perf, efficiency, engagement section, intensity heatmap + table.
 */
import { useMemo } from 'react'
import { type ColumnDef } from '@tanstack/react-table'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { EfficiencyTooltipText } from '@/components/charts/EfficiencyTooltipText'
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
import { EngagementTimeseriesSection } from '@/features/engagement/EngagementTimeseriesSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useCapability } from '@/lib/capabilities/capabilities'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { KillTypesDonutCard } from '@/components/charts/KillTypesDonutCard'
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
  const emptyMsg = t('timeseries.empty.no_data_description')
  // Charts dépendant du rating de skill (CSR/LUSR) : masqués pour un titre sans
  // système de rang. NO-OP halo_infinite (déclare 'ranked' + 'lusr'). RankScore
  // (score + placement de match) reste générique. Les deux hooks sont appelés
  // inconditionnellement (règles des hooks) avant la combinaison.
  const hasRanked = useCapability('ranked')
  const hasLusr = useCapability('lusr')
  const hasSkillRating = hasRanked || hasLusr
  // « Répartition des frags » : sur les titres à mécaniques natives (Halo 5), ce
  // donut est DÉJÀ rendu dans l'onglet Summary (TimeseriesKillTypesDonut). On évite
  // le doublon en ne le rendant ici (Progression) que pour les titres SANS cette
  // capability (Halo Infinite), où il n'apparaît pas ailleurs.
  const hasKillMechanics = useCapability('native_kill_mechanics')
  return (
    <div className="space-y-8">
      {/* timeseries.11 — Premier événement (gauche) | timeseries.14 — Par minute (droite) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <TimeseriesFirstEventDistribution
          title={t('timeseries.progression.first_event_title')}
          emptyMessage={emptyMsg}
          data={data.first_events ?? { buckets: [] }}
          killsLabel={t('timeseries.progression.first_kill')}
          deathsLabel={t('timeseries.progression.first_death')}
          meanLabel={t('timeseries.progression.avg')}
          xAxisLabel={t('timeseries.progression.time_axis')}
        />

        <TimeseriesPerMinuteTrend
          title={t('timeseries.progression.per_minute_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          killsLabel={fieldMappings?.fields['kills_per_minute']?.label ?? 'Frags / min'}
          deathsLabel={fieldMappings?.fields['deaths_per_minute']?.label ?? 'Morts / min'}
          assistsLabel={fieldMappings?.fields['assists_per_minute']?.label ?? 'Assistances / min'}
          perMinuteSuffix={t('timeseries.progression.per_minute_suffix')}
        />
      </div>

      {/* timeseries.12 (gauche) | timeseries.16 (droite) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <TimeseriesPerformanceTrend
          title={t('timeseries.summary.perf_label')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          smoothingLabel={t('timeseries.summary.trend')}
        />

        <TimeseriesSpreeHeadshots
          title={t('timeseries.progression.spree_headshots_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          spreeLabel={t('timeseries.progression.spree_label')}
          headshotsLabel={fieldMappings?.fields['headshot_kills']?.label ?? 'Tirs à la tête'}
          perfectLabel={
            fieldMappings?.fields['perfect_kills']?.label ??
            t('timeseries.progression.perfect_kills')
          }
        />
      </div>

      {/* Progression CSR (classé) ou LUSR (non classé) — masquée pour un titre
          sans système de rang (CSR/LUSR). Quand la ventilation des frags est
          disponible : donut (gauche) | progression (droite), colonne donut plus
          étroite pour laisser respirer la time-series ; sinon la progression
          reprend toute la largeur. */}
      {hasSkillRating &&
        (data.detailed_stats && !hasKillMechanics ? (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,20rem)_minmax(0,1fr)]">
            <KillTypesDonutCard
              title={t('timeseries.progression.kill_types_title')}
              otherLabel={t('timeseries.progression.kill_type_other')}
              melee={data.detailed_stats.total_melee_kills}
              powerWeapon={data.detailed_stats.total_power_weapon_kills}
              grenade={data.detailed_stats.total_grenade_kills}
              totalKills={(data.match_rows ?? []).reduce((acc, r) => acc + r.kills, 0)}
            />
            <TimeseriesSkillProgression rows={data.match_rows ?? []} locale={locale} emptyMessage={emptyMsg} />
          </div>
        ) : (
          <TimeseriesSkillProgression rows={data.match_rows ?? []} locale={locale} emptyMessage={emptyMsg} />
        ))}

      {/* timeseries.19 — Score & rang de match (générique : personal_score +
          placement). | Skill rank + Performance (CSR/LUSR, masqué sans rang).
          Sans rang, Score & rang passe pleine largeur (pas de colonne vide). */}
      <div className={hasSkillRating ? 'grid grid-cols-1 gap-4 lg:grid-cols-2' : ''}>
        <TimeseriesRankScore
          title={t('timeseries.progression.rank_score_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          scoreLabel={fieldMappings?.fields['personal_score']?.label ?? 'Score personnel'}
          rankLabel={
            fieldMappings?.fields['rank']?.label ??
            t('timeseries.progression.rank')
          }
        />

        {hasSkillRating && (
          <TimeseriesSkillRankPerformance
            title={t('timeseries.progression.rank_perf_title')}
            emptyMessage={emptyMsg}
            rows={data.match_rows ?? []}
            ratingLabel={t('timeseries.progression.rank')}
            perfLabel={
              fieldMappings?.fields['performance_score']?.label ??
              t('timeseries.summary.perf_label')
            }
          />
        )}
      </div>

      {/* Rendement & Résistance — pleine largeur. */}
      <TimeseriesEfficiency
        title={
          <span className="flex items-center gap-1.5">
            {t('timeseries.progression.efficiency_title')}
            <InfoTooltip content={<EfficiencyTooltipText locale={locale} />} />
          </span>
        }
        emptyMessage={emptyMsg}
        rows={data.match_rows ?? []}
        rendementLabel={t('timeseries.progression.dmg_per_kill')}
        resistanceLabel={t('timeseries.progression.dmg_per_death')}
        refLabel={t('timeseries.progression.ref_one_life')}
      />

      {/* Engagement — pleine largeur. EngagementTimeseriesSection
          rend déjà sa propre ChartCard avec titre interne, donc pas de
          wrapper supplémentaire (sinon double titre). Gaté sur `engagement`. */}
      <FeatureGate capability="engagement">
        <EngagementTimeseriesSection
          playerSlug={playerSlug}
          filters={soloFilterContext}
          filterHash={filterContextHash}
          limit={30}
        />
      </FeatureGate>

      {/* Intensité — frags par phase de match (pleine largeur). */}
      <TimeseriesIntensityHeatmap
        title={t('timeseries.progression.intensity_title')}
        emptyMessage={emptyMsg}
        rows={data.intensity_rows ?? []}
        zLabel={t('timeseries.progression.intensity_z')}
        height={Math.max(200, Math.min(640, (data.intensity_rows ?? []).length * 18 + 80))}
      />

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
