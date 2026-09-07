/**
 * _weaponElevationChart — LE DÉNIVELÉ SIGNÉ DES ENGAGEMENTS, arme par arme.
 *
 * Décision D4 du plan `.ai/PLAN_DUELS_PORTEE_2026-09-06.md` : `killer_z − victim_z` par frag
 * mesuré, en trois classes à ±1 m près (une marche, un rebord — pas un décalage de capsule).
 * Le point de vue est CELUI DU JOUEUR des deux côtés : côté morts, « d'en haut » veut dire
 * que le tueur était au-dessus de moi. L'inversion de signe est faite UNE fois, côté Go
 * (`WeaponRangeAggregate`) ; la refaire ici dirait l'exact contraire de la vérité.
 *
 * Deux barres empilées à 100 % par ligne (frags puis morts), MÊMES catégories et MÊME ordre
 * que le graphe de portée : les deux se lisent ensemble, ligne à ligne. La couleur NE JUGE
 * PAS — une seule teinte, du clair (d'en bas) au foncé (d'en haut), et le gris des libellés
 * d'axe pour « à niveau ».
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getTooltipBase,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import type { WeaponRangeSide } from '@/lib/api/types'

import { weaponRangeCategoryLabel, type WeaponRangeLine } from './_weaponRangeChart'

/** Les trois classes, dans l'ordre de lecture de la pile (haut → bas). */
export const ELEVATION_KEYS = ['above', 'level', 'below'] as const
export type ElevationKey = (typeof ELEVATION_KEYS)[number]

/**
 * Part inscrite DANS le segment à partir de ce pourcentage — en dessous, le nombre déborde
 * sur ses voisins et rend la barre illisible. La valeur reste dans l'infobulle.
 */
const LABEL_MIN_PCT = 18

/** La part d'un côté pour une classe donnée ; côté absent = segment nul (barre vide). */
function share(side: WeaponRangeSide | null, key: ElevationKey): number {
  if (!side) return 0
  if (key === 'above') return side.above_pct
  if (key === 'level') return side.level_pct
  return side.below_pct
}

export interface WeaponElevationOptionInput {
  /** Lignes DANS L'ORDRE DU BACKEND — l'inversion de l'axe Y se fait ici, comme la portée. */
  lines: readonly WeaponRangeLine[]
  tc: EChartsThemeColors
  /** Encres résolues par l'appelant, une par classe. */
  colors: Record<ElevationKey, string>
  /** Séparateur de segments = fond de carte : deux classes voisines ne se confondent pas. */
  cardColor: string
  /** Formate un pourcentage (« 49 % ») — la locale vit chez l'appelant. */
  fmtPercent: (v: number) => string
  /** Libellés déjà localisés : les deux côtés, les trois classes, l'absence de mesure. */
  labels: {
    kills: string
    deaths: string
    noMeasure: string
    segments: Record<ElevationKey, string>
  }
}

export function buildWeaponElevationOption({
  lines,
  tc,
  colors,
  cardColor,
  fmtPercent,
  labels,
}: WeaponElevationOptionInput): EChartsCoreOption {
  const ordered = [...lines].reverse()
  const axis = getAxisBase(tc)

  const segment = (key: ElevationKey, side: 'kills' | 'deaths', stack: string) => ({
    name: labels.segments[key],
    type: 'bar',
    stack,
    barWidth: 7,
    barGap: '55%',
    itemStyle: { color: colors[key], borderColor: cardColor, borderWidth: 1 },
    label: {
      show: true,
      color: tc.text,
      fontSize: 9,
      formatter: (p: { value: number }) => (p.value >= LABEL_MIN_PCT ? fmtPercent(p.value) : ''),
    },
    data: ordered.map((line) => share(line[side], key)),
  })

  const sideLine = (name: string, side: WeaponRangeSide | null) => {
    if (!side) return `${escapeHtml(name)} : ${escapeHtml(labels.noMeasure)}`
    const parts = ELEVATION_KEYS.map(
      (k) => `${escapeHtml(fmtPercent(share(side, k)))} ${escapeHtml(labels.segments[k])}`,
    )
    return `${escapeHtml(name)} : ${parts.join(' · ')}`
  }

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ top: 8, bottom: 24, left: 8 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const list = Array.isArray(params) ? params : [params]
        const first = list[0] as { dataIndex?: number } | undefined
        const line = first?.dataIndex != null ? ordered[first.dataIndex] : undefined
        if (!line) return ''
        return [
          `<b>${escapeHtml(line.label)}</b>`,
          sideLine(labels.kills, line.kills),
          sideLine(labels.deaths, line.deaths),
        ].join('<br/>')
      },
    },
    xAxis: {
      type: 'value',
      max: 100,
      ...axis,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmtPercent(v) },
    },
    yAxis: {
      type: 'category',
      data: ordered.map(weaponRangeCategoryLabel),
      ...axis,
      splitLine: { show: false },
    },
    // Deux piles côte à côte : ECharts les dispose dans l'ordre des séries, frags AU-DESSUS
    // — le même ordre que les deux bâtons du graphe de portée.
    series: [
      ...ELEVATION_KEYS.map((k) => segment(k, 'kills', 'kills')),
      ...ELEVATION_KEYS.map((k) => segment(k, 'deaths', 'deaths')),
    ],
  }
}
