/**
 * MatchFragDiffChart — match_view.13.
 *
 * Frags différentiel cumulé pour tous les joueurs du match.
 * Chaque kill = +1, chaque mort = -1. Une courbe par joueur, lignes lisses
 * sans marqueurs. Couleurs : main player en `compare-a`, alliés sur palette
 * cool, ennemis sur palette warm (cf. `colors.ts`).
 *
 * Source : `combat_tab.highlight_events` (kill+death horodatés).
 */
import { Card, CardContent } from '@/components/ui/card'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type {
  MatchHighlightEvent,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { allPlayersFragDiffSeries, formatBinSeconds } from './_chartSeries'
import { buildMatchPlayerColors } from './colors'

interface Props {
  events: MatchHighlightEvent[]
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
}

export function MatchFragDiffChart({ events, scoreboard, meXUID }: Props) {
  const colors = buildMatchPlayerColors(scoreboard, meXUID)
  const series = allPlayersFragDiffSeries(events, scoreboard, meXUID, colors.tokenByXUID)
  if (series.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Pas d'events kill/death disponibles pour tracer le différentiel cumulé.
        </CardContent>
      </Card>
    )
  }
  return (
    <Card>
      <CardContent className="py-4">
        <TimeseriesLineChart
          title="Frags différentiel cumulé — tous les joueurs"
          height={360}
          xAxisType="value"
          timeAxis={false}
          outcomeMarkers={false}
          showSymbol={false}
          smooth
          xAxisLabelFormatter={(v) => formatBinSeconds(Number(v))}
          series={series}
        />
      </CardContent>
    </Card>
  )
}
