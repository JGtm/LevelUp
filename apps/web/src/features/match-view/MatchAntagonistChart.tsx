/**
 * MatchAntagonistChart — match_view.18.
 *
 * Graphe "qui a tué qui" sous forme de barres empilées horizontales :
 *   - 1 ligne par tueur (ordonnée par total kills décroissant)
 *   - segments empilés = victimes (chaque victime a sa couleur cyclée)
 *
 * Source : `combat_tab.killer_victim` (paires agrégées par le backend Go).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import type { MatchKillerVictimPair } from '@/lib/api/types'
import { antagonistStackedSeries } from './_chartSeries'

interface Props {
  pairs: MatchKillerVictimPair[] | undefined
}

export function MatchAntagonistChart({ pairs }: Props) {
  const series = antagonistStackedSeries(pairs ?? [])
  if (series.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Pas de paires killer→victim disponibles pour ce match.
        </CardContent>
      </Card>
    )
  }
  // Hauteur dynamique : 80px de marge + 24px par tueur (min 240px).
  const killerCount = series[0].datapoints.length
  const height = Math.max(240, 80 + 24 * killerCount)
  return (
    <Card>
      <CardContent className="py-4">
        <BarStackedChart
          title="Antagonistes — qui a tué qui"
          height={height}
          orientation="horizontal"
          series={series}
        />
      </CardContent>
    </Card>
  )
}
