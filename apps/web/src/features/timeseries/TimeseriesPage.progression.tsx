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
  TimeseriesCareerXP,
} from './TimeseriesFormCharts'
import {
  TimeseriesEfficiency,
  TimeseriesIntensityProfile,
} from './TimeseriesSquadAdapted'
import { EngagementTimeseriesSection } from '@/features/engagement/EngagementTimeseriesSection'
import { TimeseriesEngagementGapTrend } from './TimeseriesEngagementGapTrend'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useCapability } from '@/lib/capabilities/capabilities'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { NUMERIC_SORT } from '@/features/explorer/explorerMatchesClientSort'
import { TimeseriesSkillProgression } from './TimeseriesSkillProgression'
import type { FilterContextInput, TimeseriesPageResponse, ExplorerMatchRow } from '@/lib/api/types'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'

export interface TimeseriesProgressionTabProps {
  data: TimeseriesPageResponse
  playerSlug: string
  locale: Locale
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
        // I16 : colonne triable (tri client, cf. `sortable` sur ExplorerMatchesTable
        // ci-dessous) — valeur brute nullable, nuls rangés en bas (cf. NUMERIC_SORT).
        accessorFn: (r) => r.expected_win_prob ?? undefined,
        ...NUMERIC_SORT,
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
  // XP de carrière estimée : gate DATA-DRIVEN. Le backend ne renseigne
  // career_xp_estimated que si le titre porte la capability analytics.career_xp_estimate
  // (Infinite) ; un titre sans capability → tous nuls → chart masqué (pas de slug côté front).
  const hasCareerXP = (data.match_rows ?? []).some((r) => r.career_xp_estimated != null)
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
          sans système de rang (CSR/LUSR). La « Répartition des frags » vit désormais
          sur l'onglet Résumé (sunburst v2 unifié, D7) : plus de donut kill-type ici. */}
      {hasSkillRating && (
        <TimeseriesSkillProgression rows={data.match_rows ?? []} locale={locale} emptyMessage={emptyMsg} />
      )}

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

      {/* XP de carrière (estimée) — pleine largeur, gaté data-driven (capability
          analytics.career_xp_estimate côté backend). Cumul (ligne) + XP par match
          (barres) ; méthodologie dans l'InfoTooltip du titre. */}
      {hasCareerXP && (
        <TimeseriesCareerXP
          title={
            <span className="flex items-center gap-1.5">
              {t('timeseries.progression.career_xp_title')}
              <InfoTooltip content={t('timeseries.progression.career_xp_tooltip')} />
            </span>
          }
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          cumulativeLabel={t('timeseries.progression.career_xp_cumulative')}
          perMatchLabel={t('timeseries.progression.career_xp_per_match')}
        />
      )}

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
        {/* Écart d'engagement cumulé (P4) — adjacent, réutilise la même query
            d'engagement (dédup cache TanStack Query). */}
        <TimeseriesEngagementGapTrend
          playerSlug={playerSlug}
          filters={soloFilterContext}
          filterHash={filterContextHash}
          limit={30}
        />
      </FeatureGate>

      {/* Intensité — profil médian des parts de frags par phase + enveloppe
          P25–P75 (panneau solo pleine largeur). */}
      <TimeseriesIntensityProfile
        title={
          <div className="flex flex-col gap-0.5">
            <span className="flex items-center gap-1.5">
              {t('timeseries.progression.intensity_title')}
              <InfoTooltip content={t('timeseries.progression.intensity_tooltip')} />
            </span>
            <span className="text-xs font-normal text-muted-foreground">
              {t('timeseries.progression.intensity_subtitle')}
            </span>
          </div>
        }
        emptyMessage={emptyMsg}
        rows={data.intensity_rows ?? []}
        medianLabel={t('timeseries.progression.intensity_median')}
        envelopeLabel={t('timeseries.progression.intensity_envelope')}
        refLabel={t('timeseries.progression.intensity_ref')}
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
          // I16 : même endpoint/tri backend (date DESC) que le mode Matchs →
          // le défaut interne de ExplorerMatchesTable reproduit déjà l'ordre
          // serveur, aucun `defaultSort` à surcharger (cf. ExplorerPage.playerMode).
          sortable
        />
      )}

    </div>
  )
}
