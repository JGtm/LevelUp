/**
 * squadPerMinuteChart — teammates.14 : K/D/A par minute, bars groupées.
 *
 * Spec : .ai/charts_specs/teammates/14_per_minute_stats.yaml
 *
 *   - 3 catégories X : Frags/min, Morts/min, Assists/min.
 *   - 1 série bar par joueur (main + coéquipiers, 1..4 séries).
 *   - Frags & Assists au-dessus de l'axe X (positifs), Morts en dessous (négatifs).
 *   - Couleur normale du joueur (opaque) pour Frags/Assists.
 *   - Couleur complémentaire (hue +180°, opaque) pour les Morts et stats négatives :
 *     identité joueur préservée (dérivée de la même source palette), contraste
 *     maximal avec la couleur positive, options d'accessibilité héritées
 *     automatiquement (hexComplement opère sur la couleur résolue par la palette).
 *   - Axe zéro en blanc gras (zerolinewidth=2 ↔ ECharts splitLine[0] custom).
 *   - Légende en pied de graphe, une pastille par joueur à sa couleur de BASE (celle des
 *     frags et des assistances ; les morts en portent l'opposé colorimétrique). Rétablie le
 *     2026-09-09 : « ce graphe n'a plus de légende ». La pill de la page dit bien qui est
 *     sélectionné, elle ne dit pas quelle barre est à qui — c'est le rôle de la légende.
 *   - Label sur barre = valeur ABSOLUE (frags positifs, morts négatifs vers le bas
 *     mais lus en valeur positive).
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from '@/components/charts/_utils'
import { axisTooltipFormatter } from '@/components/charts/axisTooltip'
import { tipNumber } from '@/components/charts/tipNumber'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerMinuteEntry } from '@/lib/api/types'
import { hexComplement } from '@/lib/accessibility'

export interface SquadPerMinuteOpts {
  /** Mapping gamertag → couleur hex. Doit couvrir tous les players de rows. */
  colorByPlayer: Record<string, string>
  metricLabels: { frags: string; deaths: string; assists: string }
  perMinuteSuffix: string // ex: " /min" pour tooltip
}

/** Repli quand aucune couleur n'est attribuée au joueur — gris structurel, pas une charte. */
const NO_PLAYER_COLOR = '#888' // color-allow: gris structurel pour joueur sans couleur attribuée

/** Deux décimales, dans la langue de l'application (axe, étiquettes et infobulle). */
function fmt(v: number): string {
  return tipNumber(v, 2)
}

export function buildSquadPerMinuteOption(
  series: ChartSeries<SquadPerMinuteEntry>[],
  opts: SquadPerMinuteOpts,
): EChartsCoreOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const xLabels = [opts.metricLabels.frags, opts.metricLabels.deaths, opts.metricLabels.assists]

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  // 1 série bar par joueur, 3 valeurs (frags, -deaths, assists).
  const echSeries = rows.map((r) => {
    const color = opts.colorByPlayer[r.player] ?? NO_PLAYER_COLOR
    const negColor = hexComplement(color) // hue +180°, opaque — accessibilité héritée de la palette active
    return {
      name: r.player,
      type: 'bar' as const,
      data: [
        { value: r.kills_per_minute, itemStyle: { color } },
        { value: -r.deaths_per_minute, itemStyle: { color: negColor } },
        { value: r.assists_per_minute, itemStyle: { color } },
      ],
      barMaxWidth: 22,
      label: {
        show: true,
        position: 'top' as const,
        color: tc.text,
        fontSize: 10,
        formatter: (p: { value: unknown; dataIndex: number }) => {
          const v = typeof p.value === 'number' ? p.value : 0
          if (p.dataIndex === 1) return fmt(Math.abs(v))
          return fmt(v)
        },
      },
    }
  })

  return {
    backgroundColor: CHART_BG,
    // `bottom: 56` (et non 36) : la légende se pose au ras du bas, il lui faut sa bande.
    grid: { top: 24, bottom: 56, left: 8, right: 24, containLabel: true },
    legend: {
      ...getLegendBase(tc),
      data: legendEntries(
        rows.map((r) => ({
          name: r.player,
          color: opts.colorByPlayer[r.player] ?? NO_PLAYER_COLOR,
        })),
      ),
    },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Un titre (frags, morts ou assistances), puis un joueur par ligne dans l'ordre des barres.
      formatter: axisTooltipFormatter({
        title: (p) => p.name ?? '',
        value: (r) => `${fmt(Math.abs(r.value))}${opts.perMinuteSuffix}`,
      }),
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLine: { lineStyle: { color: tc.text, width: 2 } }, // axe zéro accentué (foreground)
    },
    yAxis: {
      ...axis,
      type: 'value',
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmt(Math.abs(v)) },
    },
    series: echSeries,
  }
}
