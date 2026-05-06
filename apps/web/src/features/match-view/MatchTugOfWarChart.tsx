/**
 * MatchTugOfWarChart — match_view.10 (Tug-of-war dominance par tranche de
 * temps).
 *
 * Bar stack par tranche : `team_kills` (mon équipe) vs `enemy_kills`
 * (adversaires). Catégorie = mm:ss du milieu de tranche.
 *
 * Source : `combat_tab.tug_of_war` (port Go analysis.ComputeTugOfWarBins).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { resolveToken } from '@/lib/accessibility'
import type { MatchTugOfWarBin } from '@/lib/api/types'
import { tugOfWarStackedSeries } from './_chartSeries'
import type { MatchViewText } from './i18n'

interface Props {
  bins: MatchTugOfWarBin[]
  t: MatchViewText
}

export function MatchTugOfWarChart({ bins, t }: Props) {
  const series = tugOfWarStackedSeries(bins, {
    team: t.combatTeamLabel,
    enemy: t.combatEnemyLabel,
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
  const componentHexColors: Record<string, string> = {
    [t.combatTeamLabel]: resolveToken('compare-a'),
    [t.combatEnemyLabel]: resolveToken('outcome-loss'),
  }
  return (
    <Card>
      <CardContent className="py-4">
        <BarStackedChart
          title={t.combatTugOfWarTitle}
          height={300}
          orientation="vertical"
          series={series}
          componentOrder={[t.combatTeamLabel, t.combatEnemyLabel]}
          componentHexColors={componentHexColors}
          tooltipHideZero
        />
      </CardContent>
    </Card>
  )
}
