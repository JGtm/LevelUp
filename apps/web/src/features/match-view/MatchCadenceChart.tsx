/**
 * MatchCadenceChart — match_view.11.
 *
 * Cadence des kills par tranche de 60s (1 série par joueur du scoreboard,
 * empilée sur le même axe). Couleurs cyclées via la palette par défaut du
 * BarStackedChart. Source : `combat_tab.cadence` (build côté Go via
 * narrative.ComputeCadenceProfiles).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import type { MatchScoreboardRow, MatchViewCadence } from '@/lib/api/types'
import { cadenceSeriesWithGamertags } from './_chartSeries'

interface Props {
  cadence: MatchViewCadence | null | undefined
  scoreboard: MatchScoreboardRow[]
}

export function MatchCadenceChart({ cadence, scoreboard }: Props) {
  const series = cadenceSeriesWithGamertags(cadence, scoreboard)
  if (series.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Pas de cadence disponible (events insuffisants pour ce match).
        </CardContent>
      </Card>
    )
  }
  const phaseSeconds =
    typeof cadence?.meta?.phase_seconds === 'number'
      ? (cadence.meta.phase_seconds as number)
      : 60
  return (
    <Card>
      <CardContent className="py-4">
        <BarStackedChart
          title={`Cadence des kills par phase de ${phaseSeconds}s`}
          height={300}
          series={series}
        />
      </CardContent>
    </Card>
  )
}
