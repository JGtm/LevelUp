/**
 * MatchCadenceChart — match_view.11.
 *
 * Cadence des kills par tranche de 60s, 1 série empilée par joueur du
 * scoreboard. Reconstruite côté front depuis `combat_tab.highlight_events`
 * (mêmes données que FragDiff/Antagonistes) afin de garantir la cohérence
 * d'affichage : si les autres charts ont des données, la cadence aussi.
 *
 * Couleurs alignées sur la matrice main/alliés/ennemis (cf. `colors.ts`).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import type { SemanticToken } from '@/lib/accessibility'
import type { MatchHighlightEvent, MatchScoreboardRow } from '@/lib/api/types'
import { cadenceSeriesFromEvents } from './_chartSeries'
import { buildMatchPlayerColors } from './colors'

const PHASE_SECONDS = 60

interface Props {
  events: MatchHighlightEvent[]
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
}

export function MatchCadenceChart({ events, scoreboard, meXUID }: Props) {
  const series = cadenceSeriesFromEvents(events, scoreboard, PHASE_SECONDS)
  if (series.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Pas de cadence disponible (events insuffisants pour ce match).
        </CardContent>
      </Card>
    )
  }
  const colors = buildMatchPlayerColors(scoreboard, meXUID)
  const componentColors: Record<string, SemanticToken> = {}
  for (const [gt, token] of colors.tokenByGamertag) {
    componentColors[gt] = token
  }
  return (
    <Card>
      <CardContent className="py-4">
        <BarStackedChart
          title={`Cadence des kills par phase de ${PHASE_SECONDS}s`}
          height={300}
          series={series}
          componentColors={componentColors}
          tooltipHideZero
        />
      </CardContent>
    </Card>
  )
}
