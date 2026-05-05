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
import type { SemanticToken } from '@/lib/accessibility'
import type { MatchKillerVictimPair, MatchScoreboardRow } from '@/lib/api/types'
import { antagonistStackedSeries } from './_chartSeries'
import { buildMatchPlayerColors } from './colors'

interface Props {
  pairs: MatchKillerVictimPair[] | undefined
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
}

export function MatchAntagonistChart({ pairs, scoreboard, meXUID }: Props) {
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
  // Map gamertag (clé des composants empilés) → token sémantique du joueur
  // dans le match (allié vs ennemi). Le BarStacked utilise `componentColors`
  // pour figer la couleur de chaque sous-clé indépendamment de son ordre.
  const colors = buildMatchPlayerColors(scoreboard, meXUID)
  const componentColors: Record<string, SemanticToken> = {}
  for (const [gt, token] of colors.tokenByGamertag) {
    componentColors[gt] = token
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
          componentColors={componentColors}
          tooltipHideZero
        />
      </CardContent>
    </Card>
  )
}
