/**
 * placementCharts.ts — les deux graphes du bloc « Groupés ou isolés » de l'onglet Emprise (lot
 * V4 du plan PLAN_EMPRISE_VIES_2026-09-28, spécification §2, maquette validée « Écart à
 * l'équipe », proposition 2, graphes `c2b` et `c2c`) :
 *
 *   - `buildPlacementLifeOption` : « Placement et rendement de chaque vie » — un point par vie
 *     (X = distance médiane au coéquipier le plus proche en portées de radar, Y = frags dans la
 *     vie), un gros point par joueur (médianes), repère du radar, frontière « au moins un frag »,
 *     quatre quarts nommés dont le seul « isolé et coûteux » est teinté ;
 *   - `buildPlacementQuartsOption` : « Part des vies par placement » — une barre empilée à 100 %
 *     par joueur, les quatre quarts aux teintes de l'échelle de niveaux.
 *
 * RENDU SEULEMENT. Quarts, médianes, parts et seuils viennent du contrat (Go, `analysis/
 * squademprise/placement.go`) : le seuil d'isolement et le nombre de frags d'une vie rentable
 * sont ceux du bloc, jamais recopiés ici. Les bornes des axes (0 à 2 portées, −0,5 à 5,5 frags),
 * les plafonds d'affichage (2 et 5 : l'infobulle garde la vraie valeur) et le décalage vertical
 * (déterministe, dérivé de la clé de la vie : aucun hasard) sont ceux de la spécification.
 * Couleurs : jetons seulement (joueurs d'escouade, `perf-tier-1/2/4/5`, neutres du thème).
 */
import type { EChartsCoreOption } from 'echarts/core'

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
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type {
  SquadEmprisePlacement,
  SquadEmprisePlacementLife,
  SquadEmprisePlacementPlayer,
  SquadEmprisePlacementQuadrant,
} from '@/lib/api/types'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import type { PlacementText } from './placementStrings'

/** Les quatre quarts, dans l'ordre des segments de la barre et des textes. */
export const QUADRANT_ORDER: readonly SquadEmprisePlacementQuadrant[] = [
  'in_range_productive',
  'isolated_productive',
  'in_range_costly',
  'isolated_costly',
]

/** Jeton de chaque quart (`perf-tier-3` inutilisé) : mêmes teintes sur le nuage et sur la barre. */
export const QUADRANT_TOKENS: Record<SquadEmprisePlacementQuadrant, SemanticToken> = {
  in_range_productive: 'perf-tier-1',
  isolated_productive: 'perf-tier-2',
  in_range_costly: 'perf-tier-4',
  isolated_costly: 'perf-tier-5',
}

/** Axe X : 0 à 2 portées de radar, pas de 0,25 ; au-delà de 2, le point est posé à 2. */
export const X_MAX = 2
export const X_STEP = 0.25
/**
 * Axe Y : étendue −0,5 à 5,5 (marge du décalage vertical) ; graduations, grille et libellés posés
 * sur `Y_TICKS` (calculés, ils tomberaient sur les demi-unités), « 5+ » au plafond de 5 frags.
 */
export const Y_MIN = -0.5
export const Y_MAX = 5.5
export const KILLS_CAP = 5
const Y_TICKS: readonly number[] = Array.from({ length: KILLS_CAP + 1 }, (_, i) => i)
const yTickLabel = (v: number) => (v >= KILLS_CAP ? `${KILLS_CAP}+` : String(v))
/** Décalage vertical maximal (± 0,25) qui décolle les points de même compte. */
export const JITTER_MAX = 0.25
/**
 * Deux gros points de joueur se recouvrent sous 0,06 portée d'écart en X et une demi-frag en Y
 * (diamètre de 18 à 38 px sur un tracé de ~1 200 × 330 px) : leurs médianes sont souvent égales à
 * l'unité près (médiane de frags entière), le dernier tracé cachait les autres. Un tel groupe est
 * écarté verticalement, pas de 0,55 frag, centré sur sa valeur ; l'infobulle garde les vraies.
 */
export const MEDIAN_OVERLAP_X = 0.06
export const MEDIAN_OVERLAP_Y = 0.5
export const MEDIAN_SPREAD = 0.55
/** Une valeur s'écrit dans son segment à partir de 8 %. */
export const SEGMENT_LABEL_MIN_PCT = 8

const LIFE_SIZE_BASE = 6
const LIFE_SIZE_CAP_S = 90
const LIFE_SIZE_DIV = 9
const MEDIAN_SIZE_BASE = 18
const MEDIAN_SIZE_CAP_LIVES = 200
const MEDIAN_SIZE_DIV = 10
const BAR_WIDTH = 22
/** Opacité du fond du seul quart teinté. */
const COSTLY_FILL_ALPHA = 0.1

export type PlacementBlock = SquadEmprisePlacement
type Player = SquadEmprisePlacementPlayer

export interface PlacementColors {
  /** Couleur d'escouade d'un joueur, par gamertag. */
  player: (gamertag: string) => string
  quadrant: Record<SquadEmprisePlacementQuadrant, string>
  /** Repère de la portée du radar (« accent »). */
  radar: string
  /** Encre sombre des valeurs écrites DANS les segments. */
  ink: string
  theme: EChartsThemeColors
}

export interface PlacementFormats {
  /** Deux décimales, séparateur de la langue (« 0,25 » / « 0.25 »). */
  ratio: Intl.NumberFormat
  /** Sans zéro inutile (médiane de frags « 1 », « 1,5 » ; seuil d'isolement « 1 », « 1,25 »). */
  count: Intl.NumberFormat
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
  }
}

/**
 * Les couleurs du graphe, résolues au rendu (thème, palette). Joueurs : le jeton de la palette
 * de la page (`useSquadPlayerPalette`, ordre de la sélection), jamais l'ordre du bloc. Repère du radar : le violet d'accent de la maquette n'a pas de jeton
 * homonyme ; `extreme` est la teinte violette du thème, sans usage dans ces deux graphes.
 * Encre sombre : `--warning-foreground`, sombre dans les deux thèmes.
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
    ink: darkInk(theme),
    theme,
  }
}

function darkInk(theme: EChartsThemeColors): string {
  if (typeof document === 'undefined') return theme.text
  const v = getComputedStyle(document.documentElement).getPropertyValue('--warning-foreground').trim()
  return v || theme.text
}

/**
 * Décalage vertical d'une vie, dans [−0,25 ; +0,25], DÉTERMINISTE : un hachage (FNV-1a) de sa
 * clé `(match_id, xuid, start_ms)`. Affichage seulement ; la même vie tombe au même endroit
 * à chaque rendu.
 */
export function lifeJitter(matchId: string, xuid: string, startMs: number): number {
  const key = `${matchId}|${xuid}|${startMs}`
  let h = 0x811c9dc5
  for (let i = 0; i < key.length; i++) {
    h ^= key.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  // Brassage final (fmix32 de MurmurHash3) : FNV-1a seul disperse mal des clés qui ne diffèrent
  // que par leur dernier caractère (débuts de vie consécutifs).
  h ^= h >>> 16
  h = Math.imul(h, 0x85ebca6b)
  h ^= h >>> 13
  h = Math.imul(h, 0xc2b2ae35)
  h ^= h >>> 16
  return ((h >>> 0) / 0x100000000 - 0.5) * 2 * JITTER_MAX
}

/** Taille d'un point de vie : `6 + min(durée_s, 90) / 9`. */
export function lifeSymbolSize(durationMs: number): number {
  return LIFE_SIZE_BASE + Math.min(durationMs / 1000, LIFE_SIZE_CAP_S) / LIFE_SIZE_DIV
}

/** Taille du gros point d'un joueur : `18 + min(nombre de vies, 200) / 10`. */
export function medianSymbolSize(lives: number): number {
  return MEDIAN_SIZE_BASE + Math.min(lives, MEDIAN_SIZE_CAP_LIVES) / MEDIAN_SIZE_DIV
}

const mmss = (ms: number) => {
  const s = Math.round(ms / 1000)
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
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

function lifeDatum(p: Player, l: SquadEmprisePlacementLife, o: PlacementChartOpts): TipDatum {
  const { t, formats } = o
  return {
    value: [
      Math.min(l.radar_ratio, X_MAX),
      Math.min(l.kills, KILLS_CAP) + lifeJitter(l.match_id, p.xuid, l.start_ms),
    ],
    symbolSize: lifeSymbolSize(l.duration_ms),
    tip: [
      `<b>${escapeHtml(p.gamertag)}</b> · ${t.life.lifeHead(mmss(l.duration_ms))}`,
      t.life.lifeBody(formats.ratio.format(l.radar_ratio), Math.round(l.out_of_radar_share * 100), l.kills),
    ].join('<br>'),
  }
}

/** La position affichée du gros point d'un joueur (plafonnée comme les vies), avant écartement. */
function medianSpot(p: Player): [number, number] | null {
  if (p.median_radar_ratio == null || p.median_kills == null) return null
  return [Math.min(p.median_radar_ratio, X_MAX), Math.min(p.median_kills, KILLS_CAP)]
}

/**
 * medianOffsets — le décalage vertical d'affichage du gros point de chaque joueur (par xuid) : les
 * gros points qui se recouvrent (`MEDIAN_OVERLAP_X` / `MEDIAN_OVERLAP_Y`, de proche en proche)
 * forment un groupe, écarté de `MEDIAN_SPREAD` dans l'ordre des joueurs du bloc, centré sur sa
 * valeur et gardé dans l'axe. Déterministe ; 0 pour un gros point seul.
 */
export function medianOffsets(players: readonly Player[]): Map<string, number> {
  const spots = players.flatMap((p) => {
    const s = medianSpot(p)
    return s ? [{ xuid: p.xuid, x: s[0], y: s[1] }] : []
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
    const centre = g.reduce((acc, m) => acc + m.y, 0) / g.length
    const half = ((g.length - 1) * MEDIAN_SPREAD) / 2
    const lo = Math.min(Math.max(centre - half, Y_MIN + JITTER_MAX), Y_MAX - JITTER_MAX - 2 * half)
    g.forEach((m, i) => out.set(m.xuid, g.length > 1 ? lo + i * MEDIAN_SPREAD - m.y : 0))
  }
  return out
}

/** Le gros point d'un joueur : médiane X × médiane des frags de ses vies mesurées. */
function medianDatum(p: Player, ratio: number, kills: number, offset: number, o: PlacementChartOpts): TipDatum {
  const { t, formats } = o
  return {
    value: [Math.min(ratio, X_MAX), Math.min(kills, KILLS_CAP) + offset],
    symbolSize: medianSymbolSize(p.lives_measured),
    tip: [
      `<b>${escapeHtml(p.gamertag)}</b> · ${t.life.medianHead(p.lives_measured)}`,
      t.life.medianMid(formats.ratio.format(ratio), formats.count.format(kills), kills),
      t.life.medianLast(Math.round(quadrantPct(p, 'isolated_costly'))),
    ].join('<br>'),
  }
}

/**
 * Les deux séries d'un joueur : le semis de ses vies, puis son gros point au-dessus (`z`), décalé
 * de `offset` à l'affichage (`medianOffsets`). MÊME NOM pour les deux : un clic sur la légende
 * isole ou masque les deux ensemble.
 */
function playerSeries(p: Player, offset: number, c: PlacementColors, o: PlacementChartOpts): unknown[] {
  const color = c.player(p.gamertag)
  const series: unknown[] = [
    {
      name: p.gamertag,
      type: 'scatter',
      data: (p.lives ?? []).map((l) => lifeDatum(p, l, o)),
      itemStyle: { color, borderColor: c.theme.card, borderWidth: 1 },
    },
  ]
  if (p.median_radar_ratio != null && p.median_kills != null) {
    series.push({
      name: p.gamertag,
      type: 'scatter',
      z: 6,
      data: [medianDatum(p, p.median_radar_ratio, p.median_kills, offset, o)],
      itemStyle: { color, borderColor: c.theme.text, borderWidth: 2 },
    })
  }
  return series
}

/** Repère de la portée du radar : trait vertical pointillé au seuil d'isolement, PAS de zone teintée. */
function radarMarker(isolatedFrom: number, c: PlacementColors, t: PlacementText): unknown {
  return {
    type: 'line',
    silent: true,
    data: [],
    z: 0,
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: c.radar, type: 'dashed', width: 1.5 },
      label: { show: true, formatter: t.life.radarLine, color: c.radar, position: 'insideEndTop', fontSize: 11 },
      data: [{ xAxis: isolatedFrom }],
    },
  }
}

/** Une zone nommée : titre en capitales + sous-titre, dans les coins (x0, y0) (x1, y1). */
function quadrantArea(
  name: string,
  from: [number, number],
  to: [number, number],
  extra?: { fill: string; ink: string },
) {
  return [
    {
      name,
      xAxis: from[0],
      yAxis: from[1],
      ...(extra ? { itemStyle: { color: extra.fill }, label: { color: extra.ink } } : {}),
    },
    { xAxis: to[0], yAxis: to[1] },
  ]
}

/**
 * La frontière horizontale pointillée grise (au moins un frag dans la vie) et les quatre quarts
 * nommés, texte gris discret 12 px. Seul « isolé et coûteux » est teinté (`perf-tier-5` à 10 %,
 * son texte dans cette couleur) : les quarts ne se classent pas du bon au mauvais.
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
    z: 0,
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
      label: { color: c.theme.axisLabel, fontSize: 12, lineHeight: 16, position: 'inside' },
      data: [
        quadrantArea(name('in_range_productive'), [0, Y_MAX], [iso, front]),
        quadrantArea(name('isolated_productive'), [iso, Y_MAX], [X_MAX, front]),
        quadrantArea(name('in_range_costly'), [0, front], [iso, Y_MIN]),
        quadrantArea(name('isolated_costly'), [iso, front], [X_MAX, Y_MIN], {
          fill: hexToRgba(c.quadrant.isolated_costly, COSTLY_FILL_ALPHA),
          ink: c.quadrant.isolated_costly,
        }),
      ],
    },
  }
}

const tipOf = (p: { data?: { tip?: string } }) => p.data?.tip ?? ''

/** « Placement et rendement de chaque vie » : le nuage (hauteur 420 côté carte). */
export function buildPlacementLifeOption(block: PlacementBlock, c: PlacementColors, o: PlacementChartOpts): EChartsCoreOption {
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
      ...players.flatMap((p) => playerSeries(p, offsets.get(p.xuid) ?? 0, c, o)),
      radarMarker(block.isolated_from_ratio, c, t),
      quadrantsMarker(block, c, t),
    ],
  }
}

/** « Part des vies par placement » : une barre à 100 % par joueur, le premier en haut (hauteur 230). */
export function buildPlacementQuartsOption(block: PlacementBlock, c: PlacementColors, o: PlacementChartOpts): EChartsCoreOption {
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
      label: {
        show: true,
        position: 'inside',
        color: c.ink,
        fontSize: 11,
        fontWeight: 600,
        formatter: (p: { value: number }) => {
          const v = Math.round(p.value)
          return v >= SEGMENT_LABEL_MIN_PCT ? t.quarts.value(v) : ''
        },
      },
    })),
  }
}
