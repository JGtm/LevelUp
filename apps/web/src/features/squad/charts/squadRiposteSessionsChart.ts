/**
 * squadRiposteSessionsChart — la frise de la riposte, soirée par soirée (carte « Riposte »
 * de la section Coordination, D19 du 2026-09-21).
 *
 * CE MODULE N'EST PLUS QU'UN ADAPTATEUR (2026-09-22, lot Q). La grammaire de la frise —
 * bâtons 18 px, courbe pleine non lissée, repère d'habituel tireté, rang de volumes sous
 * l'axe — a été HISSÉE dans `components/charts/sessionBarsTrendChart.ts` parce que les
 * Séries temporelles en demandent deux instances de plus (Riposte, Appui reçu) : la règle
 * des deux copies interdit de la recopier. Ici ne reste que la TRADUCTION de la matière de
 * l'Escouade (`FriseRiposte`) en séries génériques, et les deux choix qui lui appartiennent :
 *
 *   1. LA COULEUR DU BÂTON PORTE UN VERDICT, jamais l'identité d'une soirée : au-dessus de
 *      l'habituel (`success`) ou en dessous (`warning`). Les deux verdicts sont DEUX SÉRIES
 *      EMPILÉES À TROUS — c'est ce qui leur donne une entrée de légende chacun, là qu'une
 *      colorisation par `itemStyle` n'aurait nommée nulle part.
 *   2. LE VOLUME DE CHAQUE SOIRÉE (morts mesurées) est un second rang d'étiquettes sous
 *      l'axe des dates, pas un second graphe : c'est le dénominateur, il n'a pas d'échelle
 *      propre. La courbe que cette frise remplace n'en montrait aucun — une soirée à
 *      3 morts mesurées s'y lisait comme une soirée à 80.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import { seriesColor } from '@/components/charts/_utils'
import {
  buildSessionBarsTrendOption,
  type SessionBarsSeriesSpec,
} from '@/components/charts/sessionBarsTrendChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { FriseRiposte } from '../squadRiposte.logic'

export interface SquadRiposteSessionsOpts {
  aboveLabel: string
  belowLabel: string
  trendLabel: string
  /** Libellé du repère d'habituel, déjà formaté avec son taux. */
  usualLabel: string
  yAxisLabel: string
  /** Rang d'étiquettes sous l'axe : son nom (« morts mesurées »). */
  volumeAxisLabel: string
  /** Infobulle du volume d'une soirée, déjà pluralisée. */
  volumeTooltip: (n: number) => string
}

/**
 * buildSquadRiposteSessionsOption — l'option ECharts complète de la frise.
 *
 * La série porte UN SEUL datapoint : la frise entière (`FriseRiposte`). C'est la forme
 * qu'attend `ChartCard`, dont le contrat est « une liste de séries de points ».
 */
export function buildSquadRiposteSessionsOption(
  series: ChartSeries<FriseRiposte>[],
  opts: SquadRiposteSessionsOpts,
): EChartsCoreOption {
  const frise = series[0]?.datapoints[0]
  if (!frise || frise.soirees.length === 0) return buildSessionBarsTrendOption({ labels: [], series: [], yAxisLabel: opts.yAxisLabel })

  const above = resolveToken('success')
  const below = resolveToken('warning')
  // Deux séries à TROUS, empilées : chaque soirée n'alimente que celle de son verdict.
  const specs: SessionBarsSeriesSpec[] = [
    {
      name: opts.aboveLabel,
      color: above,
      stack: 'soiree',
      valuesPct: frise.soirees.map((s) => (s.auDessus ? s.tauxPct : null)),
      usual: { valuePct: frise.habituelPct, label: opts.usualLabel },
      trend: {
        valuesPct: frise.tendancePct,
        label: opts.trendLabel,
        color: seriesColor(2),
      },
    },
    {
      name: opts.belowLabel,
      color: below,
      stack: 'soiree',
      valuesPct: frise.soirees.map((s) => (s.auDessus ? null : s.tauxPct)),
    },
  ]

  return buildSessionBarsTrendOption({
    labels: frise.soirees.map((s) => s.label),
    series: specs,
    yAxisLabel: opts.yAxisLabel,
    volumeAxis: {
      label: opts.volumeAxisLabel,
      values: frise.soirees.map((s) => String(s.morts)),
    },
    tooltipLines: (i) => [opts.volumeTooltip(frise.soirees[i]?.morts ?? 0)],
  })
}
