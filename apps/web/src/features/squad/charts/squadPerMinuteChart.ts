/**
 * squadPerMinuteChart — teammates.14 : K/D/A par minute, bars groupées, et son graphe frère
 * « Dégâts infligés et subis par minute » (même construction, deux catégories).
 *
 *   - 3 catégories X : Frags/min, Morts/min, Assists/min (frère : Dégâts infligés/min, Dégâts subis/min).
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

export interface SquadPerMinuteOpts {
  /** Mapping gamertag → couleur hex. Doit couvrir tous les players de rows. */
  colorByPlayer: Record<string, string>
  metricLabels: { frags: string; deaths: string; assists: string }
  perMinuteSuffix: string // ex: " /min" pour tooltip
}

export interface SquadDamagePerMinuteOpts {
  colorByPlayer: Record<string, string>
  metricLabels: { dealt: string; taken: string }
  perMinuteSuffix: string
}

/** Repli quand aucune couleur n'est attribuée au joueur — gris structurel, pas une charte. */
const NO_PLAYER_COLOR = '#888' // color-allow(hex): gris structurel pour joueur sans couleur attribuée

/** Décimales des frags, morts et assistances par minute ; les dégâts sont des centaines. */
const COUNT_DECIMALS = 2
const DAMAGE_DECIMALS = 0

/** Une catégorie du graphe : sa valeur par joueur (absente = trou) et son sens (sous l'axe ou non). */
interface PerMinuteMetric {
  label: string
  value: (entry: SquadPerMinuteEntry) => number | null | undefined
  /** Tracée sous l'axe zéro, en couleur complémentaire du joueur (morts, dégâts subis). */
  negative: boolean
}

interface BarsOpts {
  colorByPlayer: Record<string, string>
  perMinuteSuffix: string
  decimals: number
}

/** Cœur commun : une série bar par joueur, une valeur par catégorie. */
function buildPerMinuteBars(
  series: ChartSeries<SquadPerMinuteEntry>[],
  opts: BarsOpts,
  metrics: PerMinuteMetric[],
): ChartOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0 || metrics.length === 0) return { backgroundColor: CHART_BG }

  // Deux décimales (comptes) ou aucune (dégâts), dans la langue de l'application (axe, étiquettes, infobulle).
  const fmt = (v: number) => tipNumber(v, opts.decimals)

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  // 1 série bar par joueur, une valeur par catégorie (négative sous l'axe).
  const echSeries = rows.map((r) => {
    const color = opts.colorByPlayer[r.player] ?? NO_PLAYER_COLOR
    const negColor = hexComplement(color) // hue +180°, opaque — accessibilité héritée de la palette active
    return {
      name: r.player,
      type: 'bar' as const,
      data: metrics.map((m) => {
        const v = m.value(r)
        if (v == null) return null
        return m.negative ? { value: -v, itemStyle: { color: negColor } } : { value: v, itemStyle: { color } }
      }),
      barMaxWidth: 22,
      label: {
        show: true,
        position: 'top' as const,
        color: tc.text,
        formatter: (p: { value: unknown }) => (typeof p.value === 'number' ? fmt(Math.abs(p.value)) : ''),
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
      // Un titre (une catégorie), puis un joueur par ligne dans l'ordre des barres.
      formatter: axisTooltipFormatter({
        title: (p) => p.name ?? '',
        value: (r) => `${fmt(Math.abs(r.value))}${opts.perMinuteSuffix}`,
      }),
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: metrics.map((m) => m.label),
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

export function buildSquadPerMinuteOption(
  series: ChartSeries<SquadPerMinuteEntry>[],
  opts: SquadPerMinuteOpts,
): ChartOption {
  return buildPerMinuteBars(
    series,
    { colorByPlayer: opts.colorByPlayer, perMinuteSuffix: opts.perMinuteSuffix, decimals: COUNT_DECIMALS },
    [
      { label: opts.metricLabels.frags, value: (r) => r.kills_per_minute, negative: false },
      { label: opts.metricLabels.deaths, value: (r) => r.deaths_per_minute, negative: true },
      { label: opts.metricLabels.assists, value: (r) => r.assists_per_minute, negative: false },
    ],
  )
}

/**
 * Dégâts infligés et subis par minute. Une catégorie n'existe que si au moins un joueur la porte
 * (les dégâts subis manquent à un titre qui ne les fournit pas) ; sans aucune des deux, option vide.
 */
export function buildSquadDamagePerMinuteOption(
  series: ChartSeries<SquadPerMinuteEntry>[],
  opts: SquadDamagePerMinuteOpts,
): ChartOption {
  const rows = series[0]?.datapoints ?? []
  const all: PerMinuteMetric[] = [
    { label: opts.metricLabels.dealt, value: (r) => r.damage_dealt_per_minute, negative: false },
    { label: opts.metricLabels.taken, value: (r) => r.damage_taken_per_minute, negative: true },
  ]
  const metrics = all.filter((m) => rows.some((r) => m.value(r) != null))
  return buildPerMinuteBars(
    series,
    { colorByPlayer: opts.colorByPlayer, perMinuteSuffix: opts.perMinuteSuffix, decimals: DAMAGE_DECIMALS },
    metrics,
  )
}

/** Au moins un joueur porte des dégâts par minute (sinon le graphe frère n'a rien à dire). */
export function squadHasDamagePerMinute(rows: readonly SquadPerMinuteEntry[]): boolean {
  return rows.some((r) => r.damage_dealt_per_minute != null || r.damage_taken_per_minute != null)
}
