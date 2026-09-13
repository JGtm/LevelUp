/**
 * squadMapHeatmapChart — teammates.03 : heatmap perf joueur × carte.
 *
 * Spec : .ai/charts_specs/teammates/03_squad_heatmap.yaml
 *
 * visualMap discret (5 paliers) sur les seuils SCORE_THRESHOLDS (75/60/45/30).
 * yAxis = joueurs (moi en haut). xAxis = toutes les cartes jouées en escouade,
 * triées par fréquence décroissante.
 *
 * Accessibilité (CVD) : ce heatmap reste volontairement HORS du helper
 * heatmapRampTokens — ce dernier ne couvre que les rampes CONTINUES
 * (sequential/frequency/divergent). Ici l'échelle est ORDINALE discrète à 5
 * paliers, encodée par les tokens sémantiques perf-tier-* (palette-aware et
 * CVD-safe par construction : axe bleu→jaune→vermillon en palette daltonienne,
 * cf. palettes/okabe-ito) ET chaque palier porte un label texte (opts.pieceLabels)
 * qui désambiguïse même à teintes proches. Migration non pertinente.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadMapHeatmap, SquadMapHeatmapCell } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

/**
 * Combien d'étiquettes de carte la bande sous le graphe peut porter SANS que deux voisines
 * se chevauchent.
 *
 * MESURÉ, PAS DEVINÉ (capture du 2026-09-13, Escouade sur 55 sessions) : à 56 cartes les 56
 * étiquettes obliques se superposaient en un pâté illisible. Une étiquette tronquée
 * (`truncateMap`, 9 caractères + points de suspension) occupe ~45 px d'emprise horizontale
 * une fois tournée ; sur la largeur utile d'une carte pleine (~1 200 px) cela fait 18
 * étiquettes qui respirent. Au-delà de ce compte, on n'en écrit plus qu'UNE SUR K — la
 * numérotation « #N » de chaque étiquette dit combien de colonnes le saut a passées, et
 * l'infobulle de chaque case nomme toujours sa carte en entier.
 */
const MAX_ETIQUETTES_X = 18

/**
 * `axisLabel.interval` d'ECharts : nombre d'étiquettes SAUTÉES entre deux écrites (0 = toutes).
 * Exporté pour être vérifié hors rendu.
 */
export function xLabelInterval(mapCount: number): number {
  if (mapCount <= MAX_ETIQUETTES_X) return 0
  return Math.ceil(mapCount / MAX_ETIQUETTES_X) - 1
}

export interface SquadMapHeatmapOpts {
  mapLabelOf: (mapUI: string) => string
  pieceLabels: { tier1: string; tier2: string; tier3: string; tier4: string; tier5: string }
  noScoreLabel: string
  /** Nom de l'axe X (« Carte ») — localise par l'appelant. */
  xAxisName: string
  /** Nom de l'axe Y (« Joueur ») — localise par l'appelant. */
  yAxisName: string
}

export function buildSquadMapHeatmapOption(
  series: ChartSeries<SquadMapHeatmap>[],
  opts: SquadMapHeatmapOpts,
): EChartsCoreOption {
  const heatmap = series[0]?.datapoints[0]
  const players = heatmap?.players ?? []
  const mapsTopn = heatmap?.maps_topn ?? []
  const cells = heatmap?.cells ?? []
  if (!heatmap || players.length === 0 || mapsTopn.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  // Étiquettes X « #N\nCarte » (format compact 2 lignes, comme les autres charts).
  // Nom complet conservé dans `mapNames` pour le tooltip.
  const mapNames = mapsTopn.map(opts.mapLabelOf)
  const xLabels = mapNames.map((name, i) => `#${i + 1}\n${truncateMap(name)}`)
  const yLabels = players

  // Map (player, map) → cell pour lookup O(1).
  const cellByKey = new Map<string, SquadMapHeatmapCell>()
  for (const c of cells) {
    cellByKey.set(`${c.player}|${c.map_ui}`, c)
  }

  const data: Array<[number, number, number | null]> = []
  for (let yi = 0; yi < players.length; yi += 1) {
    for (let xi = 0; xi < mapsTopn.length; xi += 1) {
      const c = cellByKey.get(`${players[yi]}|${mapsTopn[xi]}`)
      const v = c?.perf_avg !== undefined ? Number(c.perf_avg.toFixed(1)) : null
      data.push([xi, yi, v])
    }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    // `top: 30` : la place du nom d'axe Y, pose en TETE d'axe (cf. yAxis.nameLocation).
    // `bottom: 104` : etiquettes rotees + nom d'axe X. La reglette du visualMap est
    // masquee (cf. plus bas) : sa bande revient au trace.
    grid: { top: 30, bottom: 104, left: 8, right: 8, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: unknown) => {
        const point = p as { data?: [number, number, number | null] }
        const d = point?.data
        if (!d) return ''
        const [xi, yi, v] = d
        const cell = cellByKey.get(`${players[yi]}|${mapsTopn[xi]}`)
        const perf = v === null ? opts.noScoreLabel : v.toFixed(1)
        const n = cell?.match_count ?? 0
        return `${escapeHtml(players[yi] ?? '')} — ${escapeHtml(mapNames[xi] ?? '')}<br/>Perf: ${perf}<br/>N: ${n}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      // margin : décolle les étiquettes (2 lignes « #N\nCarte ») du bas du graphe.
      // interval : toutes les cartes tant qu'elles tiennent, une sur K au-delà (cf.
      // `xLabelInterval`) — 56 cartes, c'était 56 étiquettes l'une sur l'autre.
      axisLabel: { ...axis.axisLabel, rotate: -35, interval: xLabelInterval(mapsTopn.length), margin: 14 },
      // AXES NOMMES (2026-09-13) : sans eux, une grille de gamertags par cartes laisse
      // deviner ce qui est en ligne et ce qui est en colonne. `nameGap` passe SOUS les
      // etiquettes rotees, dont la place est prise dans `grid.bottom`.
      name: opts.xAxisName,
      nameLocation: 'middle',
      nameGap: 86,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: yLabels,
      inverse: true,
      // NOM POSE EN TETE D'AXE, pas au milieu : les etiquettes de cet axe sont des
      // gamertags (jusqu'a une centaine de pixels), et `containLabel` reserve leur place
      // sans reserver celle du NOM — au milieu, « Joueur » tombait hors du canvas et ne
      // s'affichait pas du tout (mesure sur capture le 2026-09-13).
      name: opts.yAxisName,
      // `start` et pas `end` : l'axe est INVERSE (`inverse: true`), donc son « end » est
      // EN BAS, ou le nom retombait sur les etiquettes de cartes.
      nameLocation: 'start',
      nameGap: 12,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10, align: 'left' },
    },
    visualMap: {
      // RÉGLETTE MASQUÉE, MAPPING CONSERVÉ (`show: false` ne coupe que l'affichage du
      // composant). Les cinq paliers sont déjà nommés par la légende DOM du pied de carte,
      // qui se lit mieux : deux rangées identiques sous le même graphe, c'est une de trop.
      show: false,
      type: 'piecewise',
      pieces: [
        { lt: 30, color: resolveToken('perf-tier-5'), label: opts.pieceLabels.tier5 },
        { gte: 30, lt: 45, color: resolveToken('perf-tier-4'), label: opts.pieceLabels.tier4 },
        { gte: 45, lt: 60, color: resolveToken('perf-tier-3'), label: opts.pieceLabels.tier3 },
        { gte: 60, lt: 75, color: resolveToken('perf-tier-2'), label: opts.pieceLabels.tier2 },
        { gte: 75, color: resolveToken('perf-tier-1'), label: opts.pieceLabels.tier1 },
      ],
      orient: 'horizontal',
      left: 'center',
      bottom: 4,
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        type: 'heatmap',
        data,
        label: { show: false },
        emphasis: { itemStyle: { borderColor: tc.text, borderWidth: 1 } },
      },
    ],
  }
}
