/**
 * squadPerMinuteChart — teammates.14 : K/D/A par minute, bars groupées, plus les dégâts par minute
 * sur un axe vertical SECONDAIRE à droite de la même grille.
 *
 *   - 3 catégories X : Frags/min, Morts/min, Assists/min ; quand un joueur porte des dégâts, deux
 *     catégories de plus : Dégâts infligés/min, Dégâts subis/min (axe de droite, zéro commun avec
 *     l'axe de gauche — `alignedZeroAxes`).
 *   - 1 série bar par joueur (main + coéquipiers, 1..4 séries) sur l'axe de gauche ; avec des
 *     dégâts, une seconde série par joueur sur l'axe de droite. Les deux séries d'un joueur portent
 *     le même `stack` : elles partagent UNE colonne (chaque catégorie n'a de valeur que dans l'une
 *     des deux, l'autre porte un trou), donc les joueurs restent côte à côte comme sans dégâts.
 *   - Frags & Assists au-dessus de l'axe X (positifs), Morts en dessous (négatifs) ; les dégâts
 *     infligés au-dessus, les dégâts subis en dessous.
 *   - Couleur normale du joueur (opaque) pour Frags/Assists/Dégâts infligés.
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
 *   - Un joueur sans valeur pour une catégorie (dégâts inconnus) laisse un trou : jamais un zéro.
 */
import type { ChartOption } from '@/components/charts/chartOption'
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
import { alignedZeroAxes } from '@/lib/charts/alignedZeroAxes'

export interface SquadPerMinuteOpts {
  /** Mapping gamertag → couleur hex. Doit couvrir tous les players de rows. */
  colorByPlayer: Record<string, string>
  metricLabels: { frags: string; deaths: string; assists: string; dealt: string; taken: string }
  /** Nom de l'axe de droite (dégâts par minute). */
  damageAxisName: string
  perMinuteSuffix: string // ex: " /min" pour tooltip
}

/** Repli quand aucune couleur n'est attribuée au joueur — gris structurel, pas une charte. */
const NO_PLAYER_COLOR = '#888' // color-allow(hex): gris structurel pour joueur sans couleur attribuée

/** Décimales des frags, morts et assistances par minute ; les dégâts sont des centaines. */
const COUNT_DECIMALS = 2
const DAMAGE_DECIMALS = 0

type MetricAxis = 'left' | 'right'

/** Une catégorie du graphe : sa valeur par joueur (absente = trou), son sens et son axe. */
interface PerMinuteMetric {
  label: string
  value: (entry: SquadPerMinuteEntry) => number | null | undefined
  /** Tracée sous l'axe zéro, en couleur complémentaire du joueur (morts, dégâts subis). */
  negative: boolean
  axis: MetricAxis
}

/** Les catégories à tracer : les trois comptes, puis les dégâts qu'au moins un joueur porte. */
function metricsOf(rows: readonly SquadPerMinuteEntry[], labels: SquadPerMinuteOpts['metricLabels']) {
  const counts: PerMinuteMetric[] = [
    { label: labels.frags, value: (r) => r.kills_per_minute, negative: false, axis: 'left' },
    { label: labels.deaths, value: (r) => r.deaths_per_minute, negative: true, axis: 'left' },
    { label: labels.assists, value: (r) => r.assists_per_minute, negative: false, axis: 'left' },
  ]
  const damage: PerMinuteMetric[] = [
    { label: labels.dealt, value: (r) => r.damage_dealt_per_minute, negative: false, axis: 'right' },
    { label: labels.taken, value: (r) => r.damage_taken_per_minute, negative: true, axis: 'right' },
  ]
  return [...counts, ...damage.filter((m) => rows.some((r) => m.value(r) != null))]
}

/** Au moins un joueur porte des dégâts par minute (sinon le graphe n'a ni catégorie ni axe de droite). */
export function squadHasDamagePerMinute(rows: readonly SquadPerMinuteEntry[]): boolean {
  return rows.some((r) => r.damage_dealt_per_minute != null || r.damage_taken_per_minute != null)
}

const peak = (rows: readonly SquadPerMinuteEntry[], pick: (r: SquadPerMinuteEntry) => number | null | undefined) =>
  Math.max(...rows.map((r) => pick(r) ?? 0), 0)

/** Bornes des deux axes, zéros alignés ; `null` sans dégâts (l'axe de gauche reste en auto-échelle). */
function alignedDomains(rows: readonly SquadPerMinuteEntry[]) {
  return alignedZeroAxes(
    {
      neg: peak(rows, (r) => r.deaths_per_minute),
      pos: Math.max(peak(rows, (r) => r.kills_per_minute), peak(rows, (r) => r.assists_per_minute)),
    },
    { neg: peak(rows, (r) => r.damage_taken_per_minute), pos: peak(rows, (r) => r.damage_dealt_per_minute) },
  )
}

export function buildSquadPerMinuteOption(
  series: ChartSeries<SquadPerMinuteEntry>[],
  opts: SquadPerMinuteOpts,
): ChartOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const metrics = metricsOf(rows, opts.metricLabels)
  const hasRight = metrics.some((m) => m.axis === 'right')
  const decimalsAt = (index: number | undefined) =>
    index != null && metrics[index]?.axis === 'right' ? DAMAGE_DECIMALS : COUNT_DECIMALS
  // Dans la langue de l'application (axe, étiquettes et infobulle).
  const fmt = (v: number, index?: number) => tipNumber(v, decimalsAt(index))

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  // Par joueur : une série bar sur l'axe de gauche et, avec des dégâts, une sur l'axe de droite.
  const seriesOf = (r: SquadPerMinuteEntry, side: MetricAxis) => {
    const color = opts.colorByPlayer[r.player] ?? NO_PLAYER_COLOR
    const negColor = hexComplement(color) // hue +180°, opaque — accessibilité héritée de la palette active
    return {
      name: r.player,
      type: 'bar' as const,
      yAxisIndex: side === 'left' ? 0 : 1,
      // Les deux séries d'un joueur partagent une colonne ; sans dégâts, une seule série, sans pile.
      ...(hasRight ? { stack: r.player } : {}),
      data: metrics.map((m) => {
        if (m.axis !== side) return null
        const v = m.value(r)
        if (v == null) return null
        return m.negative ? { value: -v, itemStyle: { color: negColor } } : { value: v, itemStyle: { color } }
      }),
      barMaxWidth: 22,
      label: {
        show: true,
        position: 'top' as const,
        color: tc.text,
        fontSize: 10,
        formatter: (p: { value: unknown; dataIndex: number }) =>
          typeof p.value === 'number' ? fmt(Math.abs(p.value), p.dataIndex) : '',
      },
    }
  }
  const echSeries = rows.flatMap((r) => (hasRight ? [seriesOf(r, 'left'), seriesOf(r, 'right')] : [seriesOf(r, 'left')]))

  const aligned = hasRight ? alignedDomains(rows) : null

  return {
    backgroundColor: CHART_BG,
    // `bottom: 56` (et non 36) : la légende se pose au ras du bas, il lui faut sa bande.
    grid: { top: 24, bottom: 56, left: 8, right: hasRight ? 44 : 24, containLabel: true },
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
      // Un titre (une catégorie), puis un joueur par ligne dans l'ordre des barres.
      formatter: axisTooltipFormatter({
        title: (p) => p.name ?? '',
        value: (r) => `${fmt(Math.abs(r.value), r.param.dataIndex)}${opts.perMinuteSuffix}`,
      }),
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: metrics.map((m) => m.label),
      axisLine: { lineStyle: { color: tc.text, width: 2 } }, // axe zéro accentué (foreground)
    },
    yAxis: [
      {
        ...axis,
        type: 'value',
        ...(aligned ? { min: aligned.left[0], max: aligned.left[1] } : {}),
        axisLabel: { ...axis.axisLabel, formatter: (v: number) => tipNumber(Math.abs(v), COUNT_DECIMALS) },
      },
      ...(aligned
        ? [
            {
              ...axis,
              type: 'value' as const,
              position: 'right' as const,
              min: aligned.right[0],
              max: aligned.right[1],
              name: opts.damageAxisName,
              nameTextStyle: { color: tc.axisLabel, fontSize: 10, align: 'right' as const },
              splitLine: { show: false }, // la grille est celle de l'axe de gauche
              axisLabel: { ...axis.axisLabel, formatter: (v: number) => tipNumber(Math.abs(v), DAMAGE_DECIMALS) },
            },
          ]
        : []),
    ],
    series: echSeries,
  }
}
