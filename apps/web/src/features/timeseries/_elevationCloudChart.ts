/**
 * _elevationCloudChart — LE NUAGE « DISTANCE × DÉNIVELÉ » (décision D25, proposition T5).
 *
 * Il REMPLACE les deux barres empilées par arme : la question n'est pas « avec quelle arme »
 * mais « d'où je frague, d'où je meurs ». Un point par frag mesuré, rien d'agrégé — les cas
 * extrêmes (le frag à +9 m, la mort à 35 m) survivent là où une case les noierait. Le halo du
 * 1er au 3e quartile donne la masse sans binning, et les deux médianes posent le résumé.
 *
 * LE SIGNE NE SE RECALCULE PAS ICI. `delta_z_m` arrive DÉJÀ au point de vue du joueur
 * (positif = j'étais au-dessus, pour un frag comme pour une mort) : la convention est posée
 * une fois côté Go (`analysis.signedElevation`). La rejouer ici dirait le contraire de la
 * vérité — c'est le piège que l'ancien `_weaponElevationChart.ts` documentait déjà.
 *
 * PUR : aucune couleur résolue, aucune locale, aucun accès au thème. L'appelant passe ses
 * encres et ses formateurs, ce module rend une option et des nombres.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getTooltipBase,
  hexToRgba,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import type { ElevationCloudBlock, ElevationPoint, TimeseriesMatchRow } from '@/lib/api/types'

/**
 * ELEVATION_LEVEL_BAND_M — MIROIR de `analysis.WeaponRangeLevelBandM` (Go) : la demi-largeur
 * de la bande « à niveau ». Un mètre sépare une marche, un rebord, un étage — un avantage de
 * position — du simple décalage de capsule entre deux joueurs debout sur le même sol.
 *
 * MIROIR ET NON FILTRE : la bande n'est qu'un REPÈRE dessiné, aucun point n'est classé ni
 * écarté ici. Si le seuil Go bouge, le repère devient imprécis, jamais le nuage faux.
 */
export const ELEVATION_LEVEL_BAND_M = 1

/** Marge, en mètres, laissée au-delà de la donnée la plus extrême de chaque axe. */
const AXIS_PAD_M = 1

/**
 * elevationMatchLabels — la table `match_id` -> « #N · Carte », pour l'infobulle des points.
 *
 * LA MÊME NUMÉROTATION QUE TOUS LES GRAPHES PAR MATCH DE LA PAGE : elle vient de `index` des
 * lignes de match, jamais d'un compteur local qui aurait divergé au premier filtre. Un match
 * dont le nuage porte des points sans ligne de match (cas résiduel) n'a pas d'entrée, et
 * l'infobulle omet alors la ligne du match plutôt que d'inventer un numéro.
 */
export function elevationMatchLabels(rows: readonly TimeseriesMatchRow[]): Record<string, string> {
  const out: Record<string, string> = {}
  rows.forEach((r, i) => {
    const map = r.map_name_fr || r.map_name
    const numero = `#${r.index ?? i + 1}`
    out[r.match_id] = map ? `${numero} · ${map}` : numero
  })
  return out
}

/**
 * elevationAxisBounds — les bornes des deux axes.
 *
 * L'AXE DES DÉNIVELÉS EST SYMÉTRIQUE, et c'est le point : « au-dessus » et « en dessous »
 * doivent se comparer à l'œil. Un axe cadré sur les données mettrait la ligne zéro n'importe
 * où et ferait paraître un joueur « toujours en hauteur » parce que sa pire chute est courte.
 * La bande « à niveau » est toujours contenue, même sur un nuage entièrement plat.
 */
export function elevationAxisBounds(block: ElevationCloudBlock): { xMax: number; yAbs: number } {
  let xMax = 0
  let yAbs = ELEVATION_LEVEL_BAND_M
  for (const p of [...(block.kills ?? []), ...(block.deaths ?? [])]) {
    if (p.distance_m > xMax) xMax = p.distance_m
    const abs = Math.abs(p.delta_z_m)
    if (abs > yAbs) yAbs = abs
  }
  return { xMax: Math.ceil(xMax + AXIS_PAD_M), yAbs: Math.ceil(yAbs + AXIS_PAD_M) }
}

/** Le rectangle p25-p75 d'un côté, ou null si le côté n'a pas de quartile (côté vide). */
export function elevationHalo(
  summary: ElevationCloudBlock['kills_summary'],
): { x0: number; x1: number; y0: number; y1: number } | null {
  const { distance_p25: x0, distance_p75: x1, delta_z_p25: y0, delta_z_p75: y1 } = summary
  if (x0 == null || x1 == null || y0 == null || y1 == null) return null
  return { x0, x1, y0, y1 }
}

/** Un côté du nuage : ses points, son encre, ses libellés. */
interface SideInput {
  key: 'kills' | 'deaths'
  points: readonly ElevationPoint[]
  summary: ElevationCloudBlock['kills_summary']
  color: string
  /** Nom du côté dans l'infobulle et sur la légende (« Mes frags »). */
  name: string
  /** Étiquette directe de la médiane, déjà écrite (« médiane frags +2,4 m »). */
  medianLabel: string
}

export interface ElevationCloudOptionInput {
  block: ElevationCloudBlock
  tc: EChartsThemeColors
  /** Encres résolues par l'appelant : un côté, l'autre, et le fond des marqueurs. */
  colors: { kills: string; deaths: string; card: string }
  /** `match_id` -> « #N · Carte » (cf. `elevationMatchLabels`). */
  matchLabels: Record<string, string>
  /** Clé d'arme -> libellé déjà choisi dans la langue de la page. */
  weaponLabels: Record<string, string>
  fmt: { distance: (m: number) => string; signedDistance: (m: number) => string }
  labels: {
    kills: string
    deaths: string
    medianKills: string
    medianDeaths: string
    xAxis: string
    yAxis: string
    levelBand: string
  }
}

export function buildElevationCloudOption(input: ElevationCloudOptionInput): EChartsCoreOption {
  const { block, tc, colors, labels } = input
  const { xMax, yAbs } = elevationAxisBounds(block)
  const sides: SideInput[] = [
    {
      key: 'kills',
      points: block.kills ?? [],
      summary: block.kills_summary,
      color: colors.kills,
      name: labels.kills,
      medianLabel: labels.medianKills,
    },
    {
      key: 'deaths',
      points: block.deaths ?? [],
      summary: block.deaths_summary,
      color: colors.deaths,
      name: labels.deaths,
      medianLabel: labels.medianDeaths,
    },
  ]
  const axis = getAxisBase(tc)
  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ top: 16, bottom: 48, left: 52, right: 20 }),
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: tooltipFormatter(input) },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: xMax,
      name: labels.xAxis,
      nameLocation: 'middle',
      nameGap: 28,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: -yAbs,
      max: yAbs,
      name: labels.yAxis,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      levelBandSeries(tc, labels.levelBand),
      ...sides.flatMap((s) => [pointsSeries(s), medianSeries(s, colors.card)]),
    ],
  }
}

/**
 * levelBandSeries — la bande « à niveau » (±1 m) et la ligne zéro, DERRIÈRE les points.
 *
 * Série muette et sans donnée : elle ne code aucune grandeur, elle situe. `silent` la retire
 * du survol — une bande de fond qui déclenche une infobulle vole le point qu'elle entoure.
 */
function levelBandSeries(tc: EChartsThemeColors, bandLabel: string) {
  return {
    type: 'scatter' as const,
    name: bandLabel,
    silent: true,
    data: [] as number[][],
    markArea: {
      silent: true,
      itemStyle: { color: tc.splitLine, opacity: 0.55 },
      data: [
        [
          { yAxis: -ELEVATION_LEVEL_BAND_M },
          { yAxis: ELEVATION_LEVEL_BAND_M },
        ],
      ],
    },
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: tc.axisLabel, width: 1, type: 'solid' as const, opacity: 0.6 },
      label: { show: false },
      data: [{ yAxis: 0 }],
    },
  }
}

/**
 * pointsSeries — les points bruts d'un côté, et le halo p25-p75 du MÊME côté.
 *
 * POINTS PETITS ET PEU OPAQUES : c'est ce qui laisse lire la densité par superposition sans
 * qu'un amas devienne une tache pleine. Le halo est porté par la même série pour que masquer
 * un côté dans la légende emporte son halo avec lui — deux séries l'auraient dissocié.
 */
function pointsSeries(side: SideInput) {
  const halo = elevationHalo(side.summary)
  return {
    type: 'scatter' as const,
    name: side.name,
    symbolSize: 5,
    itemStyle: { color: side.color, opacity: 0.35 },
    data: side.points.map((p) => [p.distance_m, p.delta_z_m, side.key, p.match_id, p.weapon]),
    markArea: halo
      ? {
          silent: true,
          itemStyle: {
            color: hexToRgba(side.color, 0.1),
            borderColor: side.color,
            borderWidth: 1,
            borderType: 'dashed' as const,
          },
          data: [
            [
              { xAxis: halo.x0, yAxis: halo.y0 },
              { xAxis: halo.x1, yAxis: halo.y1 },
            ],
          ],
        }
      : undefined,
  }
}

/**
 * medianSeries — LE point médian d'un côté, en gros, avec son étiquette directe.
 *
 * ANNEAU DE LA COULEUR DE CARTE : sans lui, le gros point se confond avec l'amas qu'il
 * résume. L'étiquette est posée À CÔTÉ (jamais dans une légende) — c'est elle qui fait dire
 * au graphe « médiane frags +2,4 m » sans une phrase de lecteur sous la carte (loi D22).
 */
function medianSeries(side: SideInput, cardColor: string) {
  const x = side.summary.distance_p50
  const y = side.summary.delta_z_p50
  const data = x == null || y == null ? [] : [[x, y]]
  return {
    type: 'scatter' as const,
    name: side.medianLabel,
    symbolSize: 14,
    z: 5,
    itemStyle: { color: side.color, borderColor: cardColor, borderWidth: 2 },
    label: {
      show: data.length > 0,
      position: 'right' as const,
      distance: 8,
      color: side.color,
      fontSize: 11,
      fontWeight: 600 as const,
      formatter: side.medianLabel,
    },
    data,
  }
}

/**
 * tooltipFormatter — ce qu'un point dit quand on le survole : son côté, sa distance, son
 * dénivelé SIGNÉ, son arme, son match. Jamais une clé d'arme brute : le dictionnaire du bloc
 * la traduit, et une clé absente laisse la ligne de côté plutôt que d'afficher un identifiant.
 */
function tooltipFormatter(input: ElevationCloudOptionInput) {
  const { matchLabels, weaponLabels, fmt, labels } = input
  return (params: unknown): string => {
    const p = params as { value?: unknown[] }
    const v = p.value
    if (!Array.isArray(v) || v.length < 5) return ''
    const [dist, dz, key, matchID, weapon] = v as [number, number, string, string, string]
    const lignes = [
      `<b>${escapeHtml(key === 'deaths' ? labels.deaths : labels.kills)}</b>`,
      `${escapeHtml(labels.xAxis)} : <b>${escapeHtml(fmt.distance(dist))}</b>`,
      `${escapeHtml(labels.yAxis)} : <b>${escapeHtml(fmt.signedDistance(dz))}</b>`,
    ]
    const arme = weaponLabels[weapon]
    if (arme) lignes.push(escapeHtml(arme))
    const match = matchLabels[matchID]
    if (match) lignes.push(escapeHtml(match))
    return lignes.join('<br>')
  }
}
