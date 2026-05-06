/**
 * MatchCadenceChart — match_view.11 (Cadence des frags par tranche de temps).
 *
 * Bar stack vertical : kills par phase agrégés par équipe (mon équipe vs
 * adversaires). Le team_side est résolu via le scoreboard du match.
 *
 * Source : `combat_tab.cadence` (Go service.BuildMatchCadenceChart).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { resolveToken } from '@/lib/accessibility'
import type { MatchScoreboardRow, MatchViewCadence } from '@/lib/api/types'
import { cadenceTeamSeries } from './_chartSeries'
import type { MatchViewText } from './i18n'

interface Props {
  cadence: MatchViewCadence | null | undefined
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  t: MatchViewText
}

export function MatchCadenceChart({ cadence, scoreboard, meXUID, t }: Props) {
  const series = cadenceTeamSeries(cadence, scoreboard, meXUID, {
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
          title={t.combatCadenceTitle}
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
