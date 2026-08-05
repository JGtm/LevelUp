/**
 * squadEfficiencyChart — cartes « Rendement » et « Résistance » de la page
 * Dynamique escouade, en COURBES SUPERPOSÉES (une grille par carte, une courbe
 * par joueur, couleurs `squad-player-1..4` via `features/squad/colors.ts`).
 *
 * La donnée tracée est un TAUX rapporté à UNE VIE :
 *   - metric `offensive` → `rendement_offensif × 100` ;
 *   - metric `defensive` → `resistance_defensive × 100`.
 * Les deux indicateurs sont canoniques (ADR 0006) et servis tels quels par
 * l'API : le pivot « une vie » (90 PV + 135 bouclier sur Infinite) est appliqué
 * CÔTÉ GO dans l'indicateur — aucun barème de dégâts n'est recodé ici. 100 % =
 * exactement une vie ; au-dessus le joueur fait mieux, en dessous moins bien,
 * dans les DEUX cartes.
 *
 * Cadre de lecture commun et FIXE : la fenêtre 50…200 %
 * ({@link ONE_LIFE_RATE_BOUNDS}) + les deux zones
 * ({@link oneLifeZonesMarkArea}) + le repère « 1 vie » à 100. Rien n'est
 * recalculé depuis les données : deux sessions se comparent à l'œil par
 * construction. L'identité des courbes est portée par une ÉTIQUETTE DE FIN
 * (nom du joueur, à sa couleur), pas par une légende.
 *
 * Asymétrie assumée : la résistance vit généralement au-dessus de 100 % et le
 * rendement en dessous — c'est une information, pas un défaut d'axe.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
  hoverRevealSymbol,
} from '@/components/charts/_utils'
import {
  ONE_LIFE_RATE_BOUNDS,
  ONE_LIFE_RATE_PCT,
  damagePerDeath,
  oneLifeZonesMarkArea,
} from '@/lib/charts/oneLifeWindow'
import { effectiveDmgPerFrag, formatNumberFixed } from '@/lib/formatters'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

/** Métrique rendue : rendement offensif (OC) ou résistance défensive (DR). */
export type EfficiencyMetric = 'offensive' | 'defensive'

const RATE_BOUNDS = ONE_LIFE_RATE_BOUNDS
/** Plancher FIXE de l'axe (%) — jamais dérivé des données. */
export const RATE_AXIS_MIN_PCT = RATE_BOUNDS.min
/** Plafond FIXE de l'axe (%) — jamais dérivé des données. */
export const RATE_AXIS_MAX_PCT = RATE_BOUNDS.max

/** Hauteur (px) d'une carte : une seule grille, indépendante du nombre de joueurs. */
export const EFFICIENCY_CHART_HEIGHT = 300

/** Gouttière droite réservée aux étiquettes de fin de courbe. */
const END_LABEL_GUTTER = 84
/** Longueur max d'un gamertag en étiquette de fin. */
const PLAYER_LABEL_MAX = 13

/** Libellés du chart (FR/EN fournis par la feature). */
export interface EfficiencyChartLabels {
  /** Nom de l'indicateur offensif (« Rendement »). */
  offensiveMetric: string
  /** Nom de l'indicateur défensif (« Résistance »). */
  defensiveMetric: string
  /** Libellé du repère 100 % (« 1 vie »). */
  oneLife: string
  /** « Dégâts infligés ». */
  damageDealt: string
  /** « Dégâts subis ». */
  damageTaken: string
  /** Unité du ratio offensif (« / frag effectif »). */
  perFrag: string
  /** Unité du ratio défensif (« / mort »). */
  perDeath: string
}

export interface EfficiencyChartOpts {
  metric: EfficiencyMetric
  /** gamertag → couleur hex résolue (trait + étiquette de fin). */
  colorByPlayer: Record<string, string>
  labels: EfficiencyChartLabels
}

/** Point tracé : taux en % + valeurs brutes servies au survol. */
export interface EfficiencyPoint {
  /** Index de match partagé de l'escouade (`match_order`). */
  order: number
  /** Taux rapporté à une vie, en % (100 = une vie). */
  rate: number
  /** Dégâts bruts du match (infligés en offensif, subis en défensif). */
  rawDamage: number | null
  /** Dégâts par frag effectif (offensif) ou par mort (défensif). */
  perEvent: number | null
}

/**
 * buildEfficiencyPoints — projette les matchs d'UN joueur sur l'axe de match
 * partagé (`match_order`), en taux « une vie » (%). Index sans donnée = null
 * (le joueur n'était pas sur ce match, ou l'indicateur n'est pas calculable).
 *
 * Garde défensive : une DR de 0 SANS dégât subi n'est pas une résistance nulle,
 * c'est une absence d'engagement défensif mesuré — on ne trace rien plutôt qu'un
 * 0 % faux. (Le backend, lui, écrête déjà le cas « 0 mort avec dégâts subis ».)
 */
export function buildEfficiencyPoints(
  rows: SquadPerformanceSeriesPoint[],
  length: number,
  metric: EfficiencyMetric,
): Array<EfficiencyPoint | null> {
  const out = new Array<EfficiencyPoint | null>(length).fill(null)
  const offensive = metric === 'offensive'
  for (const row of rows) {
    if (row.match_order < 0 || row.match_order >= length) continue
    const normalized = offensive ? row.rendement_offensif : row.resistance_defensive
    const rawDamage = (offensive ? row.damage_dealt : row.damage_taken) ?? null
    if (normalized == null || !Number.isFinite(normalized)) continue
    if (!offensive && normalized === 0 && (rawDamage ?? 0) === 0) continue
    out[row.match_order] = {
      order: row.match_order,
      rate: normalized * 100,
      rawDamage,
      perEvent: offensive
        ? effectiveDmgPerFrag(row.damage_dealt, row.kills, row.assists)
        : damagePerDeath(row.damage_taken, row.deaths),
    }
  }
  return out
}

/** Étiquette d'axe X d'un match : « #3 » ou « #3 · Streets ». */
function matchLabel(order: number, map: string | undefined): string {
  return map ? `#${order + 1} · ${truncateMap(map)}` : `#${order + 1}`
}

/** Épaisseur du trait joueur. */
const LINE_W = 2

/** Donnée d'un point de courbe : la valeur tracée porte ses valeurs brutes. */
type SeriesDatum = { value: number; point: EfficiencyPoint } | null

/** Formatter de tooltip : taux + valeurs brutes du match survolé, tous joueurs. */
function tooltipFormatter(opts: EfficiencyChartOpts, xLabels: string[]) {
  const l = opts.labels
  const offensive = opts.metric === 'offensive'
  const metric = offensive ? l.offensiveMetric : l.defensiveMetric
  const damage = offensive ? l.damageDealt : l.damageTaken
  const unit = offensive ? l.perFrag : l.perDeath
  return (params: unknown): string => {
    const items = params as Array<{
      seriesName?: string
      color?: string
      dataIndex?: number
      data?: SeriesDatum
    }>
    if (!Array.isArray(items)) return ''
    const hits = items.filter((it) => it.data?.point != null)
    if (hits.length === 0) return ''
    const order = hits[0].data?.point?.order ?? hits[0].dataIndex ?? 0
    const rows = hits.map((it) => {
      const pt = it.data?.point as EfficiencyPoint
      const dot = `<span style="color:${it.color ?? ''}">●</span>`
      return [
        `${dot} ${escapeHtml(it.seriesName ?? '')} — <b>${formatNumberFixed(pt.rate, 0)} %</b>`,
        `${escapeHtml(damage)} ${formatNumberFixed(pt.rawDamage, 0)}`,
        `${formatNumberFixed(pt.perEvent, 0)} ${escapeHtml(unit)}`,
      ].join(' · ')
    })
    return [
      `<div style="font-size:11px">${escapeHtml(xLabels[order] ?? `#${order + 1}`)}`,
      ...rows,
      // Définition compacte : le pivot est la clé de lecture de la carte.
      `<span style="opacity:.7">${escapeHtml(metric)} — 100 % = ${escapeHtml(l.oneLife)}</span></div>`,
    ].join('<br/>')
  }
}

/**
 * buildSquadEfficiencyOption — une grille, N courbes joueurs superposées, taux
 * « une vie » en %, fenêtre FIXE 50…200 %, deux zones de lecture, repère 100 %
 * et étiquettes de fin de courbe. Ordre des courbes = `players`.
 */
export function buildSquadEfficiencyOption(
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>,
  players: string[],
  opts: EfficiencyChartOpts,
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
  const axis = getAxisBase(tc)
  const xLabels = Array.from({ length: n }, (_, i) => matchLabel(i, mapByOrder.get(i)))
  const colorOf = (p: string) => opts.colorByPlayer[p] ?? tc.axisLabel

  const series = players.map((player, idx) => {
    const color = colorOf(player)
    const points = buildEfficiencyPoints(rowsByPlayer[player] ?? [], n, opts.metric)
    const drawn = points.filter((p) => p != null).length
    return {
      name: player,
      type: 'line' as const,
      data: points.map((p): SeriesDatum => (p == null ? null : { value: p.rate, point: p })),
      ...hoverRevealSymbol(color, 6),
      // Un joueur d'un seul match n'a aucun segment à dessiner : on montre alors
      // les symboles en permanence plutôt qu'une courbe invisible.
      showSymbol: drawn <= 1,
      lineStyle: { color, width: LINE_W },
      connectNulls: false,
      // Canal d'identité qui remplace la légende : nom du joueur, à sa couleur,
      // au bout de sa courbe.
      endLabel: {
        show: true,
        formatter: truncateMap(player, PLAYER_LABEL_MAX),
        color,
        fontSize: 10,
        fontWeight: 600 as const,
        distance: 6,
      },
      // Repère et zones rendus UNE SEULE fois (portés par la première courbe) :
      // ils décrivent la grille, pas un joueur.
      ...(idx === 0
        ? {
            // Polarité unique du cadre « une vie » : les indicateurs canoniques
            // montent quand le joueur va mieux (taux rapportés à une vie), donc
            // le vert est toujours au-dessus du repère.
            markArea: oneLifeZonesMarkArea(ONE_LIFE_RATE_PCT, RATE_BOUNDS),
            markLine: {
              silent: true,
              symbol: 'none' as const,
              lineStyle: { color: tc.axisLine, width: 1, type: 'solid' as const },
              label: {
                show: true,
                formatter: opts.labels.oneLife,
                color: tc.axisLabel,
                fontSize: 10,
                position: 'insideStartTop' as const,
              },
              data: [{ yAxis: ONE_LIFE_RATE_PCT }],
            },
          }
        : {}),
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, right: END_LABEL_GUTTER, bottom: 32, left: 8, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: tooltipFormatter(opts, xLabels),
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: { ...axis.axisLabel, fontSize: 9, interval: n > 12 ? Math.floor(n / 8) : 0 },
    },
    // Fenêtre FIXE : bornes CONSTANTES, jamais dérivées des données (deux
    // sessions se superposent à l'œil). Un point hors fenêtre est écrêté par
    // l'axe ; le survol en garde la valeur vraie.
    yAxis: {
      ...axis,
      type: 'value',
      min: RATE_AXIS_MIN_PCT,
      max: RATE_AXIS_MAX_PCT,
      interval: (RATE_AXIS_MAX_PCT - RATE_AXIS_MIN_PCT) / 3,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => `${Math.round(v)} %` },
    },
    series,
  }
}
