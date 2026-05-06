/**
 * MatchKDCumulChart — match_view.09 (Kill/Death cumulés du joueur courant).
 *
 * Deux courbes en escalier (Frags vs Morts) sur un axe X temporel (mm:ss).
 * Source : `combat_tab.kd_timeline` (port Go analysis.ComputeKDTimeline).
 *
 * Volontairement plus simple que le mock ECharts d'origine — on ne réplique
 * pas les markPoints/markLine d'annotation par badge : les badges sont
 * affichés au-dessus dans <MatchImpactBadgesBar>, ce qui évite la
 * superposition coûteuse à maintenir dans ECharts.
 */
import { Card, CardContent } from '@/components/ui/card'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { resolveToken } from '@/lib/accessibility'
import type { MatchKDTimelinePoint } from '@/lib/api/types'
import { kdCumulSeries, formatBinSeconds } from './_chartSeries'
import type { MatchViewText } from './i18n'

interface Props {
  points: MatchKDTimelinePoint[]
  t: MatchViewText
}

export function MatchKDCumulChart({ points, t }: Props) {
  const series = kdCumulSeries(points, {
    kills: t.combatKillsLabel,
    deaths: t.combatDeathsLabel,
  })

  if (series.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-sm text-muted-foreground">
          {t.combatNoData}
        </CardContent>
      </Card>
    )
  }

  const seriesHex: Record<string, string> = {
    [series[0].key]: resolveToken('compare-a'),
    [series[1].key]: resolveToken('outcome-loss'),
  }

  return (
    <Card>
      <CardContent className="py-4">
        <TimeseriesLineChart
          title={t.combatKdCumulTitle}
          height={320}
          xAxisType="value"
          timeAxis={false}
          outcomeMarkers={false}
          showSymbol
          xAxisLabelFormatter={(v) => formatBinSeconds(Number(v))}
          series={series}
          seriesColorResolver={(s) => seriesHex[s.key]}
        />
      </CardContent>
    </Card>
  )
}
