/**
 * squadEfficiencyChart — Rendement & Résistance de la page Dynamique escouade, en
 * PISTES EMPILÉES PAR JOUEUR (décision d'artefact « C » du lot v7.3-2).
 *
 * Une piste = un joueur. On y trace, match par match, l'ÉCART SIGNÉ EN % à la
 * frontière élite du titre (80e percentile) :
 *   - metric `offensive` → indicateur = `rendement_offensif` (OC servi par l'API),
 *     frontière = `offensive_conversion_p80` du titre (0.90 Infinite, 1.264 h5) ;
 *   - metric `defensive` → indicateur = `resistance_defensive` (DR servie par l'API),
 *     frontière = `defensive_resistance_p80` du titre (1.65 Infinite).
 *
 * Les DEUX indicateurs montent quand le joueur va mieux (OC = PV convertis en frags
 * effectifs par dégât infligé ; DR = dégâts encaissés par mort rapportés aux PV), donc
 * un écart positif = au-dessus de l'élite dans les deux cartes : remplissage
 * `divergent-pos` au-dessus de zéro, `divergent-neg` en dessous, trait et étiquette de
 * piste à la couleur du joueur. Axe symétrique PARTAGÉ par toutes les pistes (±40 %
 * de départ, élargi par pas de 10 jusqu'à ±150 si les données débordent) : une bosse
 * de même hauteur vaut la même chose chez tous les joueurs.
 *
 * Pourquoi l'indicateur du PAYLOAD et pas un recalcul local : la frontière P80 est
 * définie dans l'espace OC/DR côté back (games.OffensiveConversionP80 /
 * games.DefensiveResistanceP80, tous deux servis par TitleSummary). Recalculer
 * `dégâts / frags` sans les assistances — ce que faisait la version précédente —
 * comparait deux échelles différentes et contredisait l'ADR 0006 (frag effectif =
 * frags + assistances / 3). Le survol montre les valeurs BRUTES (dégâts du match,
 * dégâts par frag effectif / par mort) en plus de l'indicateur normalisé.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getEChartsThemeColors,
  getTooltipBase,
  hoverRevealSymbol,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { damagePerDeath } from '@/lib/charts/oneLifeDamageGradient'
import { effectiveDmgPerFrag, formatNumberFixed, formatSignedFixed } from '@/lib/formatters'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

/** Métrique rendue : rendement offensif (OC) ou résistance défensive (DR). */
export type EfficiencyMetric = 'offensive' | 'defensive'

/** Demi-amplitude minimale de l'axe d'écart (%) — cadre de lecture par défaut. */
export const GAP_AXIS_MIN_PCT = 40
/** Demi-amplitude maximale (%) : au-delà, les points sont écrêtés par l'axe plutôt
 *  que d'aplatir toutes les pistes (un match à 2 frags pour 30 dégâts sort à +1000 %). */
export const GAP_AXIS_MAX_PCT = 150
/** Pas d'élargissement de l'axe. */
const GAP_AXIS_STEP_PCT = 10

/** Géométrie d'une piste (px) — la hauteur de carte en découle, cf. tracksChartHeight. */
const TRACK_H = 56
const TRACK_GAP = 8
const TOP_PAD = 20
const BOTTOM_PAD = 34
/** Gouttière gauche réservée à l'étiquette de piste (nom du joueur). */
const LABEL_W = 96
/** Gouttière droite réservée aux libellés ±bornes de l'axe. */
const RIGHT_W = 46
/** Longueur max d'un gamertag en étiquette de piste. */
const PLAYER_LABEL_MAX = 13

/** Libellés du chart (FR/EN fournis par la feature). */
export interface EfficiencyTrackLabels {
  /** Nom de l'indicateur offensif (« Rendement »). */
  offensiveMetric: string
  /** Nom de l'indicateur défensif (« Résistance »). */
  defensiveMetric: string
  /** « frontière élite ». */
  eliteBoundary: string
  /** « Dégâts infligés ». */
  damageDealt: string
  /** « Dégâts subis ». */
  damageTaken: string
  /** Unité du ratio offensif (« / frag effectif »). */
  perFrag: string
  /** Unité du ratio défensif (« / mort »). */
  perDeath: string
}

export interface EfficiencyTracksOpts {
  metric: EfficiencyMetric
  /** Frontière élite du titre : OC P80 en offensif, DR P80 en défensif. */
  eliteP80: number
  /** gamertag → couleur hex résolue (trait + étiquette de piste). */
  colorByPlayer: Record<string, string>
  labels: EfficiencyTrackLabels
}

/** Point de piste : écart tracé + valeurs brutes servies au survol. */
export interface EfficiencyTrackPoint {
  /** Index de match partagé de l'escouade (`match_order`). */
  order: number
  /** Écart signé à la frontière élite, en % (positif = au-dessus de l'élite). */
  gap: number
  /** Indicateur normalisé du titre (OC en offensif, DR en défensif). */
  normalized: number
  /** Dégâts bruts du match (infligés en offensif, subis en défensif). */
  rawDamage: number | null
  /** Dégâts par frag effectif (offensif) ou par mort (défensif). */
  perEvent: number | null
}

/**
 * eliteGapPercent — écart signé en % entre un indicateur et la frontière élite du
 * titre : `(valeur / p80 − 1) × 100`. 0 % = pile sur la frontière. Renvoie null si
 * l'indicateur est absent/non fini ou si la frontière est invalide (jamais de
 * division par zéro silencieuse).
 */
export function eliteGapPercent(value: number | null | undefined, p80: number): number | null {
  if (value == null || !Number.isFinite(value)) return null
  if (!Number.isFinite(p80) || p80 <= 0) return null
  return (value / p80 - 1) * 100
}

/**
 * symmetricGapBound — demi-amplitude de l'axe PARTAGÉ par toutes les pistes :
 * l'écart absolu maximal arrondi au pas supérieur, borné à [40, 150] %. Les valeurs
 * au-delà de 150 % sont écrêtées par l'axe (le survol garde la valeur vraie).
 */
export function symmetricGapBound(gaps: Array<number | null>): number {
  let maxAbs = 0
  for (const g of gaps) {
    if (g != null && Number.isFinite(g)) maxAbs = Math.max(maxAbs, Math.abs(g))
  }
  const rounded = Math.ceil(maxAbs / GAP_AXIS_STEP_PCT) * GAP_AXIS_STEP_PCT
  return Math.min(GAP_AXIS_MAX_PCT, Math.max(GAP_AXIS_MIN_PCT, rounded))
}

/** Paire [x, y] d'une aire de remplissage (y null = rupture de piste). */
type AreaPoint = [number, number | null]

/**
 * splitSignedAtZero — découpe une suite d'écarts signés en DEUX polylignes (partie
 * positive écrêtée à 0, partie négative écrêtée à 0) en INSÉRANT le point de
 * croisement exact quand deux écarts consécutifs changent de signe. Sans cette
 * insertion, le remplissage vert commencerait au match précédent au lieu du vrai
 * passage par zéro. Un trou (null) rompt les deux polylignes et interdit un
 * croisement à cheval sur le trou.
 */
export function splitSignedAtZero(gaps: Array<number | null>): {
  positive: AreaPoint[]
  negative: AreaPoint[]
} {
  const positive: AreaPoint[] = []
  const negative: AreaPoint[] = []
  let prev: number | null = null
  let prevX = -1
  for (let i = 0; i < gaps.length; i++) {
    const v = gaps[i]
    if (v == null || !Number.isFinite(v)) {
      positive.push([i, null])
      negative.push([i, null])
      prev = null
      continue
    }
    if (prev != null && ((prev < 0 && v > 0) || (prev > 0 && v < 0))) {
      const t = -prev / (v - prev)
      const xCross = prevX + t * (i - prevX)
      positive.push([xCross, 0])
      negative.push([xCross, 0])
    }
    positive.push([i, Math.max(v, 0)])
    negative.push([i, Math.min(v, 0)])
    prev = v
    prevX = i
  }
  return { positive, negative }
}

/**
 * buildTrackPoints — projette les matchs d'UN joueur sur l'axe de match partagé
 * (`match_order`), en écart à la frontière élite. Index sans donnée = null (le
 * joueur n'était pas sur ce match, ou l'indicateur n'est pas calculable).
 *
 * Garde défensive : une DR de 0 SANS dégât subi n'est pas une résistance nulle,
 * c'est une absence d'engagement défensif mesuré — on ne trace rien plutôt qu'un
 * −100 % faux. (Le backend, lui, écrête déjà le cas « 0 mort avec dégâts subis ».)
 */
export function buildTrackPoints(
  rows: SquadPerformanceSeriesPoint[],
  length: number,
  opts: Pick<EfficiencyTracksOpts, 'metric' | 'eliteP80'>,
): Array<EfficiencyTrackPoint | null> {
  const out = new Array<EfficiencyTrackPoint | null>(length).fill(null)
  for (const row of rows) {
    if (row.match_order < 0 || row.match_order >= length) continue
    const offensive = opts.metric === 'offensive'
    const normalized = offensive ? row.rendement_offensif : row.resistance_defensive
    const rawDamage = (offensive ? row.damage_dealt : row.damage_taken) ?? null
    if (!offensive && normalized === 0 && (rawDamage ?? 0) === 0) continue
    const gap = eliteGapPercent(normalized, opts.eliteP80)
    if (gap == null || normalized == null) continue
    out[row.match_order] = {
      order: row.match_order,
      gap,
      normalized,
      rawDamage,
      perEvent: offensive
        ? effectiveDmgPerFrag(row.damage_dealt, row.kills, row.assists)
        : damagePerDeath(row.damage_taken, row.deaths),
    }
  }
  return out
}

/** Hauteur totale (px) du chart pour `n` pistes — consommée par la carte hôte. */
export function tracksChartHeight(n: number): number {
  const tracks = Math.max(1, n)
  return TOP_PAD + (tracks - 1) * (TRACK_H + TRACK_GAP) + TRACK_H + BOTTOM_PAD
}

/** Étiquette d'axe X d'un match : « #3 » ou « #3 · Streets ». */
function matchLabel(order: number, map: string | undefined): string {
  return map ? `#${order + 1} · ${truncateMap(map)}` : `#${order + 1}`
}

/** Séries d'une piste : aire positive, aire négative, puis le trait du joueur. */
function trackSeries(
  index: number,
  player: string,
  points: Array<EfficiencyTrackPoint | null>,
  color: string,
  tc: EChartsThemeColors,
): unknown[] {
  const { positive, negative } = splitSignedAtZero(points.map((p) => p?.gap ?? null))
  const area = (data: AreaPoint[], token: 'divergent-pos' | 'divergent-neg') => ({
    type: 'line' as const,
    xAxisIndex: index,
    yAxisIndex: index,
    data,
    silent: true,
    symbol: 'none' as const,
    lineStyle: { width: 0 },
    areaStyle: { color: resolveToken(token), opacity: 0.32, origin: 0 },
    connectNulls: false,
  })
  return [
    area(positive, 'divergent-pos'),
    area(negative, 'divergent-neg'),
    {
      name: player,
      type: 'line' as const,
      xAxisIndex: index,
      yAxisIndex: index,
      // Le point porte ses valeurs brutes : le formatter de tooltip n'a rien à
      // ré-agréger et ignore les deux aires (qui portent des x de croisement).
      data: points.map((p, i) => ({ value: [i, p?.gap ?? null], point: p, player })),
      ...hoverRevealSymbol(color, 6),
      // Une piste d'un seul match n'a aucun segment à dessiner : on montre alors
      // les symboles en permanence plutôt qu'un cadre vide.
      showSymbol: points.length <= 2,
      lineStyle: { color, width: 2 },
      connectNulls: false,
      markLine: {
        silent: true,
        symbol: 'none' as const,
        data: [{ yAxis: 0 }],
        lineStyle: { color: tc.axisLine, width: 1, type: 'solid' as const },
        label: { show: false },
      },
    },
  ]
}

/** Formatter de tooltip : écart signé + valeurs brutes du match survolé. */
function tooltipFormatter(
  opts: EfficiencyTracksOpts,
  xLabels: string[],
  colorOf: (p: string) => string,
) {
  const l = opts.labels
  const offensive = opts.metric === 'offensive'
  return (params: unknown): string => {
    const items = params as Array<{ data?: { point?: EfficiencyTrackPoint | null; player?: string } }>
    if (!Array.isArray(items)) return ''
    const hit = items.find((it) => it.data?.point != null)
    const pt = hit?.data?.point
    if (!pt || !hit?.data?.player) return ''
    const player = hit.data.player
    const head = escapeHtml(xLabels[pt.order] ?? `#${pt.order + 1}`)
    const dot = `<span style="color:${colorOf(player)}">●</span>`
    const gap = `${formatSignedFixed(pt.gap, 0)} %`
    const metric = offensive ? l.offensiveMetric : l.defensiveMetric
    const damage = offensive ? l.damageDealt : l.damageTaken
    const unit = offensive ? l.perFrag : l.perDeath
    return [
      `<div style="font-size:11px">${head}`,
      `${dot} ${escapeHtml(player)} — <b>${gap}</b>`,
      `${escapeHtml(metric)} ${formatNumberFixed(pt.normalized, 2)} · ${escapeHtml(l.eliteBoundary)} ${formatNumberFixed(opts.eliteP80, 2)}`,
      `${escapeHtml(damage)} ${formatNumberFixed(pt.rawDamage, 0)} · ${formatNumberFixed(pt.perEvent, 0)} ${escapeHtml(unit)}</div>`,
    ].join('<br/>')
  }
}

/**
 * buildSquadEfficiencyTracksOption — pistes empilées (1 par joueur) de l'écart à la
 * frontière élite du titre, axe symétrique partagé. Ordre des pistes = `players`.
 */
export function buildSquadEfficiencyTracksOption(
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>,
  players: string[],
  opts: EfficiencyTracksOpts,
): EChartsCoreOption {
  let n = 0
  const mapByOrder = new Map<number, string>()
  for (const p of players) {
    for (const pt of rowsByPlayer[p] ?? []) {
      n = Math.max(n, pt.match_order + 1)
      if (pt.map_name && !mapByOrder.has(pt.match_order)) mapByOrder.set(pt.match_order, pt.map_name)
    }
  }
  if (n === 0 || players.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const xLabels = Array.from({ length: n }, (_, i) => matchLabel(i, mapByOrder.get(i)))
  const colorOf = (p: string) => opts.colorByPlayer[p] ?? tc.axisLabel
  const pointsByPlayer = players.map((p) => buildTrackPoints(rowsByPlayer[p] ?? [], n, opts))
  const bound = symmetricGapBound(pointsByPlayer.flat().map((p) => p?.gap ?? null))
  const lastTrack = players.length - 1
  const trackTop = (i: number) => TOP_PAD + i * (TRACK_H + TRACK_GAP)

  return {
    backgroundColor: CHART_BG,
    grid: players.map((_, i) => ({
      left: LABEL_W,
      right: RIGHT_W,
      top: trackTop(i),
      height: TRACK_H,
    })),
    // Étiquette de piste (nom du joueur, couleur du joueur) dans la gouttière gauche :
    // identité portée par le texte ET la couleur, jamais par la couleur seule.
    graphic: players.map((p, i) => ({
      type: 'text' as const,
      left: 8,
      top: trackTop(i) + TRACK_H / 2,
      silent: true,
      style: {
        text: truncateMap(p, PLAYER_LABEL_MAX),
        fill: colorOf(p),
        fontSize: 11,
        fontWeight: 600,
        textVerticalAlign: 'middle' as const,
      },
    })),
    xAxis: players.map((_, i) => ({
      type: 'value' as const,
      gridIndex: i,
      min: -0.5,
      max: n - 0.5,
      minInterval: 1,
      axisLine: { show: false, onZero: false },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: {
        show: i === lastTrack,
        color: tc.axisLabel,
        fontSize: 9,
        formatter: (v: number) => (Number.isInteger(v) ? (xLabels[v] ?? '') : ''),
      },
    })),
    yAxis: players.map((_, i) => ({
      type: 'value' as const,
      gridIndex: i,
      min: -bound,
      max: bound,
      interval: bound,
      position: 'right' as const,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: {
        color: tc.axisLabel,
        fontSize: 9,
        formatter: (v: number) => (v === 0 ? '' : `${formatSignedFixed(v, 0)} %`),
      },
    })),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: tooltipFormatter(opts, xLabels, colorOf),
    },
    series: players.flatMap((p, i) => trackSeries(i, p, pointsByPlayer[i], colorOf(p), tc)),
  }
}
