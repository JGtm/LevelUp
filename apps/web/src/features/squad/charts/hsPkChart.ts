/**
 * hsPkChart — Headshot kills / Perfect kills par partie.
 *
 * Produit un ChartSeries<ChartPointStacked>[], consommé par BarGroupedChart.
 * Remplace l'ancien builder Plotly barmode='overlay'.
 *
 * Multi-titres : aucun libellé hardcodé. Noms de composants et extracteurs
 * passés en argument par le caller.
 */
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { TeammateRow } from '@/lib/api/types'
import type { SquadMetric } from '../metrics'

export interface HsPkSeriesArgs {
  rows: TeammateRow[]
  hsMetric: SquadMetric
  pkMetric: SquadMetric
  hsLabel: string
  pkLabel: string
}

export function buildHsPkSeries({
  rows,
  hsMetric,
  pkMetric,
  hsLabel,
  pkLabel,
}: HsPkSeriesArgs): ChartSeries<ChartPointStacked>[] {
  if (rows.length === 0) return []

  const datapoints: ChartPointStacked[] = rows.map((row) => ({
    category: row.gamertag,
    components: {
      [hsLabel]: hsMetric.extract(row.with_kpis) ?? 0,
      [pkLabel]: pkMetric.extract(row.with_kpis) ?? 0,
    },
  }))

  return [{ key: 'hspk', datapoints }]
}
