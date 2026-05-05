/**
 * MatchAntagonistChart — match_view.18.
 *
 * Graphe "qui a tué qui" sous forme de barres empilées horizontales :
 *   - 1 ligne par tueur (ordonnée par total kills décroissant)
 *   - segments empilés = victimes, colorées selon l'équipe (alliés / ennemis)
 *     pour offrir une lecture immédiate des duels intra/inter-équipes.
 *
 * Source : `combat_tab.killer_victim` (paires agrégées par le backend Go).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import type {
  MatchKillerVictimPair,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { antagonistStackedSeries } from './_chartSeries'
import { buildMatchPlayerColors } from './colors'

interface Props {
  pairs: MatchKillerVictimPair[] | undefined
  scoreboard: MatchScoreboardRow[]
  roster?: MatchRosterRow[]
  meXUID: string | null
  /** Gamertags amis (page Squad) — bonus visuel : couleurs squad pour les amis alliés. */
  friendGamertags?: readonly string[]
}

export function MatchAntagonistChart({ pairs, scoreboard, roster, meXUID, friendGamertags }: Props) {
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
  // Map gamertag → hex pré-résolu (allié vs ennemi, bonus squad pour amis
  // alliés). On passe le hex directement plutôt qu'un token : ça évite que
  // ECharts retombe sur sa palette interne (1ères couleurs en bleu) si la
  // CSS var d'un token n'a pas pu être lue à temps.
  const colors = buildMatchPlayerColors(scoreboard, meXUID, friendGamertags, roster)
  const componentHexColors: Record<string, string> = {}
  for (const [gt, hex] of colors.hexByGamertag) {
    if (hex) componentHexColors[gt] = hex
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
          componentHexColors={componentHexColors}
          tooltipHideZero
        />
      </CardContent>
    </Card>
  )
}
