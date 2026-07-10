/**
 * MatchAntagonistChart — match_view.18.
 *
 * Graphe "qui a tué qui" sous forme de barres empilées horizontales :
 *   - 1 ligne par tueur (groupées par équipe via antagonistStackedSeries)
 *   - segments empilés = victimes, colorées par chart-series-1..8 (distance
 *     perceptuelle maximale) — le groupement Y-axis suffit pour la lecture équipe.
 *
 * Source : `combat_tab.killer_victim` (paires agrégées par le backend Go).
 */
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { MatchKillerVictimPair, MatchScoreboardRow } from '@/lib/api/types'
import { antagonistStackedSeries } from './_chartSeries'
import type { MatchViewText } from './i18n'

// 11 tokens à distance perceptuelle maximale sur la roue des teintes.
// chart-series-1..5 évités : trop proches (nuances d'indigo quasi-identiques).
const ANTAGONIST_TOKENS: SemanticToken[] = [
  'outcome-loss',
  'chart-series-7',
  'chart-series-6',
  'chart-series-8',
  'perf-tier-2',
  'compare-a',
  'narrative-humiliation',
  'narrative-debacle',
  'narrative-dominant',
  'narrative-contre-remontada',
  'perf-tier-4',
]

interface Props {
  pairs: MatchKillerVictimPair[] | undefined
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  t: MatchViewText
}

export function MatchAntagonistChart({ pairs, scoreboard, meXUID, t }: Props) {
  const series = antagonistStackedSeries(pairs ?? [], scoreboard, meXUID)

  // Couleurs par victime — cycle ANTAGONIST_TOKENS (11 teintes distinctes).
  const victimSet = new Set<string>()
  for (const s of series) {
    for (const dp of s.datapoints) {
      for (const key of Object.keys(dp.components)) victimSet.add(key)
    }
  }
  const componentHexColors: Record<string, string> = {}
  Array.from(victimSet).forEach((gt, idx) => {
    componentHexColors[gt] = resolveToken(ANTAGONIST_TOKENS[idx % ANTAGONIST_TOKENS.length])
  })

  // series vide => BarStackedChart (ChartCard) rend son emptyMessage dans le
  // bloc titré au lieu de faire disparaître le graphe.
  const killerCount = series.length > 0 ? series[0].datapoints.length : 0
  const height = Math.max(240, 80 + 24 * killerCount)
  return (
    <BarStackedChart
      title={t.antagonistTitle}
      height={height}
      orientation="horizontal"
      series={series}
      emptyMessage={t.antagonistNoData}
      componentHexColors={componentHexColors}
      tooltipHideZero
    />
  )
}
