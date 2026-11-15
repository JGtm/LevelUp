/**
 * placementCharts.ts — les deux graphes du bloc « Groupé ou isolé » de l'onglet Emprise
 * (spécification §2, maquette validée « Écart à l'équipe », graphes `c2b` et `c2c`) :
 *
 *   - `buildPlacementLifeOption` : « Distance au coéquipier le plus proche et frags, par vie » —
 *     une carte de chaleur (X = distance médiane au coéquipier le plus proche en portées de radar,
 *     huit cases de 0,25 ; Y = frags dans la vie, 0 à « 5+ » ; valeur d'une case = part des vies de
 *     l'escouade, `placementHeatmap.logic.ts` et `placementHeatmapChart.ts`), un gros point par
 *     joueur (distance médiane, frags moyens par vie), repère du radar, frontière « au moins un
 *     frag », quatre quarts nommés dont le seul « isolé et coûteux » est teinté ;
 *   - `buildPlacementQuartsOption` : « Part des vies par placement » — une barre empilée à 100 %
 *     par joueur, les quatre quarts aux teintes de l'échelle de niveaux.
 *
 * RENDU SEULEMENT. Quarts, médianes, parts et seuils viennent du contrat (Go, `analysis/
 * squademprise/placement.go`) : le seuil d'isolement et le nombre de frags d'une vie rentable
 * sont ceux du bloc, jamais recopiés ici. Les bornes des axes (0 à 2 portées, −0,5 à 5,5 frags) et
 * les plafonds d'affichage (2 et 5 : l'infobulle garde la vraie valeur) sont ceux de la
 * spécification. Couleurs : jetons seulement (joueurs d'escouade, `perf-tier-1/2/4/5`, rampe
 * séquentielle neutre, neutres du thème).
 */
import type { ChartOption } from '@/components/charts/chartOption'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getLegendBase,
  getTooltipBase,
  hexToRgba,
  legendEntries,
} from '@/components/charts/_utils'
import { barLabel } from '@/components/charts/barLabel'
import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { SquadEmprisePlacement, SquadEmprisePlacementPlayer, SquadEmprisePlacementQuadrant } from '@/lib/api/types'
import { meanOfValid } from '@/lib/charts/stats'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import { binLives, KILLS_CAP, playerSpot, X_MAX, X_STEP } from './placementHeatmap.logic'
import { buildHeatCellsSeries } from './placementHeatmapChart'
import type { PlacementText } from './placementStrings'

/** Les quatre quarts, dans l'ordre des segments de la barre et des textes. */
export const QUADRANT_ORDER: readonly SquadEmprisePlacementQuadrant[] = [
  'in_range_productive',
  'isolated_productive',
  'in_range_costly',
  'isolated_costly',
]

/** Jeton de chaque quart (`perf-tier-3` inutilisé) : mêmes teintes sur la carte et sur la barre. */
export const QUADRANT_TOKENS: Record<SquadEmprisePlacementQuadrant, SemanticToken> = {
  in_range_productive: 'perf-tier-1',
  isolated_productive: 'perf-tier-2',
  in_range_costly: 'perf-tier-4',
  isolated_costly: 'perf-tier-5',
}

/**
 * Axe Y : étendue −0,5 à 5,5 (une ligne de frags est centrée sur son entier) ; graduations, grille
 * et libellés posés sur `Y_TICKS`, « 5+ » à la dernière ligne.
 */
export const Y_MIN = -0.5
export const Y_MAX = KILLS_CAP + 0.5
const Y_TICKS: readonly number[] = Array.from({ length: KILLS_CAP + 1 }, (_, i) => i)
const yTickLabel = (v: number) => (v >= KILLS_CAP ? `${KILLS_CAP}+` : String(v))
/** Marge, dans l'axe, que garde un gros point écarté (en bas comme en haut). */
export const MEDIAN_EDGE_MARGIN = 0.25
/**
 * Deux gros points de joueur se recouvrent sous 0,06 portée d'écart en X et une demi-frag en Y
 * (diamètre de 18 à 38 px) : leurs positions sont parfois voisines, le dernier tracé cachait les
 * autres. Un tel groupe est écarté verticalement, pas de 0,55 frag, centré sur sa valeur ;
 * l'infobulle garde les vraies.
 */
export const MEDIAN_OVERLAP_X = 0.06
export const MEDIAN_OVERLAP_Y = 0.5
export const MEDIAN_SPREAD = 0.55
/** Une valeur s'écrit dans son segment à partir de 8 %. */
export const SEGMENT_LABEL_MIN_PCT = 8

const MEDIAN_SIZE_BASE = 18
const MEDIAN_SIZE_CAP_LIVES = 200
const MEDIAN_SIZE_DIV = 10
const BAR_WIDTH = 22
/** Opacité du fond du seul quart teinté. */
const COSTLY_FILL_ALPHA = 0.1
/** Z des repères : au-dessus des cases (1), sous les gros points (6). */
const MARKERS_Z = 3

export type PlacementBlock = SquadEmprisePlacement
type Player = SquadEmprisePlacementPlayer

export interface PlacementColors {
  /** Couleur d'escouade d'un joueur, par gamertag. */
  player: (gamertag: string) => string
  quadrant: Record<SquadEmprisePlacementQuadrant, string>
  /** Repère de la portée du radar (« accent »). */
  radar: string
  /** Teinte des cases : le haut de la rampe séquentielle neutre. */
  heat: string
  theme: EChartsThemeColors
}

export interface PlacementFormats {
  /** Deux décimales, séparateur de la langue (« 0,25 » / « 0.25 »). */
  ratio: Intl.NumberFormat
  /** Sans zéro inutile (frags moyens « 1 », « 1,5 » ; seuil d'isolement « 1 », « 1,25 »). */
  count: Intl.NumberFormat
  /** Un chiffre après la virgule au plus (part d'une case dans l'infobulle). */
  pct: Intl.NumberFormat
}

export interface PlacementChartOpts {
  t: PlacementText
  formats: PlacementFormats
}

export function placementFormats(locale: Locale): PlacementFormats {
  const loc = intlLocale(locale)
  return {
    ratio: new Intl.NumberFormat(loc, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    count: new Intl.NumberFormat(loc, { minimumFractionDigits: 0, maximumFractionDigits: 2 }),
    pct: new Intl.NumberFormat(loc, { minimumFractionDigits: 0, maximumFractionDigits: 1 }),
  }
}

/**
 * Les couleurs du graphe, résolues au rendu (thème, palette). Joueurs : le jeton de la palette
 * de la page (`useSquadPlayerPalette`, ordre de la sélection), jamais l'ordre du bloc. Repère du
 * radar : le violet d'accent de la maquette n'a pas de jeton homonyme ; `extreme` est la teinte
 * violette du thème, sans usage dans ces deux graphes. Cases : le haut de la rampe de fréquence
 * (`heatmapRampTokens`, source unique des rampes).
 */
export function resolvePlacementColors(tokenOf: (gamertag: string) => SemanticToken | null): PlacementColors {
  const theme = getEChartsThemeColors()
  return {
    player: (gamertag) => {
      const token = tokenOf(gamertag)
      return token ? resolveToken(token) : theme.text
    },
    quadrant: {
      in_range_productive: resolveToken(QUADRANT_TOKENS.in_range_productive),
      isolated_productive: resolveToken(QUADRANT_TOKENS.isolated_productive),
      in_range_costly: resolveToken(QUADRANT_TOKENS.in_range_costly),
      isolated_costly: resolveToken(QUADRANT_TOKENS.isolated_costly),
    },
    radar: resolveToken('extreme'),
    heat: resolveToken(heatmapRampTokens('frequency').at(-1)!),
    theme,
  }
}

/** Taille du gros point d'un joueur : `18 + min(nombre de vies, 200) / 10`. */
export function medianSymbolSize(lives: number): number {
  return MEDIAN_SIZE_BASE + Math.min(lives, MEDIAN_SIZE_CAP_LIVES) / MEDIAN_SIZE_DIV
}

/** Part d'un joueur dans un quart, en pourcentage (0 sans vie mesurée). */
function quadrantPct(p: Player, q: SquadEmprisePlacementQuadrant): number {
  return (p.quadrants?.find((x) => x.quadrant === q)?.share ?? 0) * 100
}

interface TipDatum {
  value: [number, number]
  symbolSize: number
  tip: string
}

/**
 * medianOffsets — le décalage vertical d'affichage du gros point de chaque joueur (par xuid) : les
 * gros points qui se recouvrent (`MEDIAN_OVERLAP_X` / `MEDIAN_OVERLAP_Y`, de proche en proche)
 * forment un groupe, écarté de `MEDIAN_SPREAD` dans l'ordre des joueurs du bloc, centré sur sa
 * valeur et gardé dans l'axe. Déterministe ; 0 pour un gros point seul.
 */
export function medianOffsets(players: readonly Player[]): Map<string, number> {
  const spots = players.flatMap((p) => {
    const s = playerSpot(p)
    return s ? [{ xuid: p.xuid, x: s.x, y: s.y }] : []
  })
  const near = (a: (typeof spots)[number], b: (typeof spots)[number]) =>
    Math.abs(a.x - b.x) < MEDIAN_OVERLAP_X && Math.abs(a.y - b.y) < MEDIAN_OVERLAP_Y
  const groups: (typeof spots)[] = []
  for (const s of spots) {
    const hits = groups.filter((g) => g.some((m) => near(m, s)))
    const merged = [...hits.flat(), s]
    for (const h of hits) groups.splice(groups.indexOf(h), 1)
    groups.push(merged)
  }
  const out = new Map<string, number>()
  const order = new Map(spots.map((s, i) => [s.xuid, i]))
  for (const g of groups) {
    g.sort((a, b) => (order.get(a.xuid) ?? 0) - (order.get(b.xuid) ?? 0))
    const centre = meanOfValid(g.map((m) => m.y)) ?? 0
    const half = ((g.length - 1) * MEDIAN_SPREAD) / 2
    const lo = Math.min(Math.max(centre - half, Y_MIN + MEDIAN_EDGE_MARGIN), Y_MAX - MEDIAN_EDGE_MARGIN - 2 * half)
    g.forEach((m, i) => out.set(m.xuid, g.length > 1 ? lo + i * MEDIAN_SPREAD - m.y : 0))
  }
  return out
}

/** Le gros point d'un joueur : distance médiane × frags moyens par vie, décalé de `offset` à l'affichage. */
function medianDatum(p: Player, offset: number, o: PlacementChartOpts): TipDatum | null {
  const spot = playerSpot(p)
  if (!spot || p.median_radar_ratio == null) return null
  const { t, formats } = o
  return {
    value: [spot.x, spot.y + offset],
    symbolSize: medianSymbolSize(p.lives_measured),
    tip: [
      `<b>${escapeHtml(p.gamertag)}</b> · ${t.life.medianHead(p.lives_measured)}`,
      t.life.medianMid(formats.ratio.format(p.median_radar_ratio), formats.count.format(spot.meanKills), spot.meanKills),
      t.life.medianLast(Math.round(quadrantPct(p, 'isolated_costly'))),
    ].join('<br>'),
  }
}

/** Le gros point d'un joueur, au-dessus de tout (`z`), à la couleur du joueur ; une série par joueur (légende native). */
function playerSeries(p: Player, offset: number, c: PlacementColors, o: PlacementChartOpts): unknown {
  const datum = medianDatum(p, offset, o)
  return {
    name: p.gamertag,
    type: 'scatter',
    z: 6,
    data: datum ? [datum] : [],
    itemStyle: { color: c.player(p.gamertag), borderColor: c.theme.text, borderWidth: 2 },
  }
}

/** Repère de la portée du radar : trait vertical pointillé au seuil d'isolement, PAS de zone teintée. */
function radarMarker(isolatedFrom: number, c: PlacementColors, t: PlacementText): unknown {
  return {
    type: 'line',
    silent: true,
    data: [],
    z: MARKERS_Z,
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: c.radar, type: 'dashed', width: 1.5 },
      label: { show: true, formatter: t.life.radarLine, color: c.radar, position: 'insideEndTop', fontSize: 11 },
      data: [{ xAxis: isolatedFrom }],
    },
  }
}

/**
 * Une zone nommée : titre en capitales + sous-titre, dans les coins (x0, y0) (x1, y1), écrit au
 * coin extérieur de la zone (`position`) pour ne pas recouvrir le centre des cases.
 */
function quadrantArea(
  name: string,
  position: string,
  from: [number, number],
  to: [number, number],
  extra?: { fill: string; ink: string },
) {
  return [
    {
      name,
      xAxis: from[0],
      yAxis: from[1],
      label: { position, ...(extra ? { color: extra.ink } : {}) },
      ...(extra ? { itemStyle: { color: extra.fill } } : {}),
    },
    { xAxis: to[0], yAxis: to[1] },
  ]
}

/**
 * La frontière horizontale pointillée grise (au moins un frag dans la vie) et les quatre quarts
 * nommés, texte gris discret 12 px aux coins extérieurs de la carte. Seul « isolé et coûteux » est
 * teinté (`perf-tier-5` à 10 %, son texte dans cette couleur) : les quarts ne se classent pas du
 * bon au mauvais.
 */
function quadrantsMarker(block: PlacementBlock, c: PlacementColors, t: PlacementText): unknown {
  const iso = block.isolated_from_ratio
  const front = block.productive_from_kills - 0.5
  const q = t.life.quadrants
  const name = (k: SquadEmprisePlacementQuadrant) => `${q[k].title}\n${q[k].sub}`
  return {
    type: 'line',
    silent: true,
    data: [],
    z: MARKERS_Z,
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: c.theme.axisLabel, type: 'dashed' },
      label: { show: false },
      data: [{ yAxis: front }],
    },
    markArea: {
      silent: true,
      itemStyle: { color: 'transparent' },
      label: {
        color: c.theme.axisLabel,
        fontSize: 12,
        lineHeight: 16,
        textBorderColor: c.theme.card,
        textBorderWidth: 2,
      },
      data: [
        quadrantArea(name('in_range_productive'), 'insideTopLeft', [0, Y_MAX], [iso, front]),
        quadrantArea(name('isolated_productive'), 'insideTopRight', [iso, Y_MAX], [X_MAX, front]),
        quadrantArea(name('in_range_costly'), 'insideBottomLeft', [0, front], [iso, Y_MIN]),
        quadrantArea(name('isolated_costly'), 'insideBottomRight', [iso, front], [X_MAX, Y_MIN], {
          fill: hexToRgba(c.quadrant.isolated_costly, COSTLY_FILL_ALPHA),
          ink: c.quadrant.isolated_costly,
        }),
      ],
    },
  }
}

const tipOf = (p: { data?: { tip?: string } }) => p.data?.tip ?? ''

/** « Distance au coéquipier le plus proche et frags, par vie » : la carte de chaleur (hauteur 520 côté carte). */
export function buildPlacementLifeOption(block: PlacementBlock, c: PlacementColors, o: PlacementChartOpts): ChartOption {
  const players = block.players ?? []
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const { t, formats } = o
  const tc = c.theme
  const axis = getAxisBase(tc)
  const offsets = medianOffsets(players)
  return {
    backgroundColor: CHART_BG,
    animation: false,
    // Pied : titre de l'axe X (nameGap 30) et légende ; gauche : libellés Y et titre vertical.
    grid: { left: 64, right: 20, top: 16, bottom: 78 },
    legend: {
      ...getLegendBase(tc),
      bottom: 4,
      left: 'center',
      data: legendEntries(players.map((p) => ({ name: p.gamertag, color: c.player(p.gamertag) }))),
    },
    xAxis: {
      ...axis,
      type: 'value',
      name: t.life.xAxis,
      nameLocation: 'middle',
      nameGap: 30,
      nameTextStyle: { color: tc.axisLabel },
      min: 0,
      max: X_MAX,
      interval: X_STEP,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => formats.ratio.format(v) },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: t.life.yAxis,
      nameLocation: 'middle',
      nameGap: 34,
      nameTextStyle: { color: tc.axisLabel },
      min: Y_MIN,
      max: Y_MAX,
      axisTick: { ...axis.axisTick, customValues: [...Y_TICKS] },
      axisLabel: { ...axis.axisLabel, customValues: [...Y_TICKS], formatter: yTickLabel },
    },
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: tipOf },
    series: [
      buildHeatCellsSeries(binLives(players), c.heat, t, formats),
      ...players.map((p) => playerSeries(p, offsets.get(p.xuid) ?? 0, c, o)),
      radarMarker(block.isolated_from_ratio, c, t),
      quadrantsMarker(block, c, t),
    ],
  }
}

/** « Part des vies par placement » : une barre à 100 % par joueur, le premier en haut (hauteur 230). */
export function buildPlacementQuartsOption(block: PlacementBlock, c: PlacementColors, o: PlacementChartOpts): ChartOption {
  const players = block.players ?? []
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const { t } = o
  const tc = c.theme
  const axis = getAxisBase(tc)
  // Un axe catégoriel se lit du bas vers le haut : on inverse pour que le premier joueur soit en haut.
  const rows = [...players].reverse()
  const byName = new Map(players.map((p) => [p.gamertag, p]))
  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: getGridBase({ left: 16, right: 16, top: 12, bottom: 56 }),
    legend: {
      ...getLegendBase(tc),
      bottom: 4,
      left: 'center',
      data: legendEntries(QUADRANT_ORDER.map((q) => ({ name: t.quarts.names[q], color: c.quadrant[q] }))),
    },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => t.quarts.value(v) },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: rows.map((p) => p.gamertag),
      axisLabel: { ...axis.axisLabel, color: tc.text, fontWeight: 600 },
      splitLine: { show: false },
    },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const ps = params as { axisValue: string; marker: string; seriesName: string; value: number }[]
        const p = byName.get(ps[0]?.axisValue ?? '')
        if (!p) return ''
        return [
          `<b>${escapeHtml(p.gamertag)}</b> · ${t.quarts.tipHead(p.lives_measured)}`,
          ...ps.map((s) => `${s.marker} ${t.quarts.tipLine(s.seriesName, Math.round(s.value))}`),
        ].join('<br>')
      },
    },
    series: QUADRANT_ORDER.map((q) => ({
      name: t.quarts.names[q],
      type: 'bar',
      stack: 'quarts',
      barWidth: BAR_WIDTH,
      data: rows.map((p) => quadrantPct(p, q)),
      itemStyle: { color: c.quadrant[q], borderColor: tc.card, borderWidth: 1 },
      label: barLabel({
        fontSize: 11,
        fontWeight: 600,
        formatter: (p: { value: number }) => {
          const v = Math.round(p.value)
          return v >= SEGMENT_LABEL_MIN_PCT ? t.quarts.value(v) : ''
        },
      }),
    })),
  }
}
