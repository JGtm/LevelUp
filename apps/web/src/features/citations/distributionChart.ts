/**
 * distributionChart — helper builder pour la section Distribution de CitationsPage.
 *
 * Phase 2 P2.D : extrait du composant pour tester sans monter le React tree.
 */
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { MedalSummary } from '@/lib/api/types'

export const MAX_MEDALS_IN_CHART = 15

export interface DistributionLabels {
  filteredLabel: string
  totalLabel: string
}

/**
 * Construit la série stackable pour BarGroupedChart à partir des médailles
 * agrégées. Trie par count_total décroissant et limite à MAX_MEDALS_IN_CHART.
 */
export function buildDistributionSeries(
  medals: MedalSummary[],
  labels: DistributionLabels,
): ChartSeries<ChartPointStacked>[] {
  const top = [...medals]
    .sort((a, b) => b.count_total - a.count_total)
    .slice(0, MAX_MEDALS_IN_CHART)
  return [
    {
      key: 'citations.distribution.medals',
      meta: { gamertag: 'medals' },
      datapoints: top.map((m) => ({
        category: m.name,
        components: {
          [labels.filteredLabel]: m.count_filtered,
          [labels.totalLabel]: m.count_total,
        },
      })),
    },
  ]
}
