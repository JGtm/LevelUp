/**
 * TimeseriesPage — onglet "Distributions".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : 6 histogrammes + 4 scatters de corrélations + MMR team/enemy.
 */
import { TimeseriesDistributionHistogram } from './TimeseriesDistributionHistogram'
import { TimeseriesScatterWithTrend } from './TimeseriesScatterWithTrend'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesDistributionsTab } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { OutcomeLabels } from './TimeseriesPage.summary'

export interface TimeseriesDistributionsTabProps {
  distributions_tab: TimeseriesDistributionsTab
  t: (key: TimeseriesManifestKey) => string
  fieldMappings: FieldMappingsResponse | undefined
  outcomeLabels: OutcomeLabels
}

export function TimeseriesDistributionsTabView({
  distributions_tab,
  t,
  fieldMappings,
  outcomeLabels,
}: TimeseriesDistributionsTabProps) {
  const emptyMsg = t('timeseries.empty.no_data_description')
  // MMR par match indisponible (titre sans `team_mmr`, ex. Halo 5) → on ne rend
  // pas la carte scatter MMR équipe/adverse du tout (pas de carte titrée vide).
  const hasTeamMmr = useCapability('team_mmr')
  // DATA-GATE précision : aucune capability dédiée. Pour Halo 5 l'accuracy est
  // nil → buckets vides ET aucun correlation point `accuracy`. On retire alors
  // l'histogramme « Précision » et le scatter accuracy↔FDA (pas de carte vide).
  const hasAccuracyBuckets =
    (distributions_tab.accuracy_buckets ?? []).reduce((s, b) => s + b.count, 0) > 0
  const hasAccuracyPoints = (distributions_tab.correlation_points ?? []).some(
    (p) => p.metric_x_key === 'accuracy',
  )
  return (
    <div className="space-y-8">
      <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {t('timeseries.tabs.distributions')}
      </h3>
      {/* 6 histogrammes en grille 3×2, chacun avec médiane verticale.
          Performance utilise un coloring par tier (perf-tier-1..5). */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {([
          // `hidden: true` (accuracy nil, ex. Halo 5) → entrée filtrée plus bas,
          // l'histogramme « Précision » n'est jamais monté (pas de carte vide).
          {
            buckets: distributions_tab.accuracy_buckets ?? [],
            title:
              fieldMappings?.fields['accuracy']?.label ?? 'Précision',
            colorToken: 'chart-series-2' as const,
            xAxisLabel: t('timeseries.distributions.accuracy_axis_x'),
            colorTokenByBucket: undefined,
            hidden: !hasAccuracyBuckets,
          },
          {
            buckets: distributions_tab.kills_buckets ?? [],
            title:
              fieldMappings?.fields['kills']?.label ?? 'Frags',
            colorToken: 'chart-series-1' as const,
            xAxisLabel: fieldMappings?.fields['kills']?.label ?? 'Frags',
            colorTokenByBucket: undefined,
            hidden: false,
          },
          {
            buckets: distributions_tab.life_buckets ?? [],
            title: t('timeseries.distributions.life_title'),
            colorToken: 'chart-series-3' as const,
            xAxisLabel: t('timeseries.distributions.seconds'),
            colorTokenByBucket: undefined,
            hidden: false,
          },
          {
            buckets: distributions_tab.personal_score_buckets ?? [],
            title:
              fieldMappings?.fields['personal_score']?.label ?? 'Score personnel',
            colorToken: 'chart-series-5' as const,
            xAxisLabel: fieldMappings?.fields['personal_score']?.label ?? 'Score personnel',
            colorTokenByBucket: undefined,
            hidden: false,
          },
          {
            buckets: distributions_tab.perf_score_buckets ?? [],
            title:
              fieldMappings?.fields['performance_score']?.label ??
              t('timeseries.summary.perf_label'),
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
            hidden: false,
          },
          {
            buckets: distributions_tab.max_killing_spree_buckets ?? [],
            title:
              t('timeseries.progression.spree_label'),
            colorToken: 'chart-series-6' as const,
            xAxisLabel: t('timeseries.distributions.spree_axis'),
            colorTokenByBucket: undefined,
            hidden: false,
          },
        ])
          .filter((cfg) => !cfg.hidden)
          .map((cfg, i) => (
          <TimeseriesDistributionHistogram
            key={i}
            title={cfg.title}
            emptyMessage={emptyMsg}
            buckets={cfg.buckets}
            colorToken={cfg.colorToken}
            xAxisLabel={cfg.xAxisLabel}
            medianLabel={t('timeseries.summary.median')}
            colorTokenByBucket={cfg.colorTokenByBucket}
          />
        ))}
      </div>

      <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {t('timeseries.distributions.correlations_title')}
      </h3>
      {/* 4 scatters en grille 2×2 + MMR seul en bas (pleine largeur). */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {([
          {
            metricXKey: 'lifespan',
            metricYKey: 'kills',
            title: t('timeseries.distributions.life_vs_kills'),
            xLabel: t('timeseries.distributions.life_axis'),
            yLabel:
              fieldMappings?.fields['kills']?.label ?? 'Frags',
          },
          {
            metricXKey: 'accuracy',
            metricYKey: 'kda',
            title: t('timeseries.distributions.accuracy_vs_kda'),
            xLabel: `${fieldMappings?.fields['accuracy']?.label ?? 'Précision'} (%)`,
            yLabel: fieldMappings?.fields['kda']?.label ?? 'FDA',
          },
          {
            metricXKey: 'lifespan',
            metricYKey: 'deaths',
            title: t('timeseries.distributions.life_vs_deaths'),
            xLabel: t('timeseries.distributions.life_axis'),
            yLabel:
              fieldMappings?.fields['deaths']?.label ?? 'Morts',
          },
          {
            metricXKey: 'kills',
            metricYKey: 'deaths',
            title: t('timeseries.distributions.kills_vs_deaths'),
            xLabel:
              fieldMappings?.fields['kills']?.label ?? 'Frags',
            yLabel:
              fieldMappings?.fields['deaths']?.label ?? 'Morts',
          },
        ] as const)
          // DATA-GATE : le scatter accuracy↔FDA est retiré quand aucun point
          // accuracy n'existe (Halo 5, accuracy nil) — pas de carte vide.
          .filter((cfg) => cfg.metricXKey !== 'accuracy' || hasAccuracyPoints)
          .map((cfg) => (
          <TimeseriesScatterWithTrend
            key={`${cfg.metricXKey}_${cfg.metricYKey}`}
            title={cfg.title}
            emptyMessage={emptyMsg}
            points={distributions_tab.correlation_points ?? []}
            metricXKey={cfg.metricXKey}
            metricYKey={cfg.metricYKey}
            xAxisLabel={cfg.xLabel}
            yAxisLabel={cfg.yLabel}
            outcomeLabels={outcomeLabels}
            trendLabel={t('timeseries.summary.trend')}
            height={240}
          />
        ))}
      </div>

      {/* MMR équipe / adverse — seul sur sa propre ligne. Retiré entièrement
          quand le titre ne fournit pas de MMR par match (Halo 5). */}
      {hasTeamMmr && (
        <TimeseriesScatterWithTrend
          title={t('timeseries.distributions.mmr_title')}
          emptyMessage={emptyMsg}
          points={distributions_tab.correlation_points ?? []}
          metricXKey="mmr_team"
          metricYKey="mmr_enemy"
          xAxisLabel={fieldMappings?.fields['team_mmr']?.label ?? 'MMR équipe'}
          yAxisLabel={fieldMappings?.fields['enemy_mmr']?.label ?? 'MMR adverse'}
          outcomeLabels={outcomeLabels}
          trendLabel={t('timeseries.summary.trend')}
          height={320}
        />
      )}
    </div>
  )
}
