/**
 * FirstBloodLanes — « Premier frag / première mort » en bandes par joueur.
 *
 * Remplace l'histogramme par tranches de 15 s (illisible dès 3 joueurs) par une
 * lecture directe du TIMING : une bande horizontale par joueur, axe X = secondes
 * depuis le début du match.
 *
 *   1. Nuage de fond      — 1 point par match : premiers frags au-dessus de la
 *                           ligne (vert), premières morts en dessous (rouge).
 *   2. Barre d'avance     — rectangle arrondi entre médiane(frag) et médiane(mort),
 *                           centré sur la ligne. VERT si le joueur frappe avant de
 *                           tomber, ROUGE sinon. C'est l'élément principal.
 *   3. Marqueurs médiane  — deux gros points (frag / mort) posés sur la ligne.
 *   4. Colonne de gauche  — pseudo, « méd. 50s → 1m04 », « +14s d'avance ».
 *
 * Choix de rendu :
 *   - La barre d'avance est une série `custom` : elle s'étend entre DEUX valeurs X
 *     arbitraires avec une hauteur en PIXELS (8 px) et des bouts arrondis — ni
 *     `bar` ni `scatter` ne savent le faire sans dupliquer les catégories.
 *   - Les nuages et les marqueurs de médiane sont des séries `scatter` :
 *     `symbolOffset` porte le décalage vertical (±14 px) et le tooltip `item`
 *     natif évite de re-router les survols à la main dans un `renderItem`.
 *   - Le redimensionnement suit le conteneur : `echarts-for-react` monte un
 *     size-sensor (`autoResize` par défaut) — aucun ResizeObserver à câbler ici.
 *
 * Couleurs : `outcome-win` (premier frag) / `outcome-loss` (première mort) —
 * mêmes tokens que l'histogramme remplacé, aucune valeur hex.
 * Modèle pur (médianes, tri, formats) : `./firstBloodLanesModel` — nommé ainsi (et
 * pas `firstBloodLanes.ts`) pour ne pas différer de ce fichier par la seule casse :
 * sur un FS insensible à la casse, Vite résoudrait `./firstBloodLanes` vers ce
 * module-ci et le composant importé serait `undefined`.
 */
import { useCallback, useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import { formatDate } from '@/lib/formatters/date'
import { formatMessage } from '@/lib/i18n/format'
import { firstBloodManifest, type FirstBloodManifestKey } from '@/lib/i18n/generated/first_blood'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors, getTooltipBase } from './_utils'
import {
  DEFAULT_MAX_SEC,
  GAP_BAR_HEIGHT,
  GRID_BOTTOM,
  GRID_TOP,
  LABEL_WIDTH,
  POINT_OFFSET,
  buildFirstBloodLanes,
  firstBloodLanesHeight,
  formatAxisTick,
  formatGapSeconds,
  formatLaneSeconds,
  sanitizeRichText,
  type FirstBloodLane,
  type FirstBloodPlayerSeries,
} from './firstBloodLanesModel'

/** Placeholder d'une médiane ou d'un libellé (carte/mode) manquant. */
const NO_VALUE = '—'
/** Taille des points de nuage / des marqueurs de médiane. */
const CLOUD_SYMBOL_SIZE = 8
const MEDIAN_SYMBOL_SIZE = 16
/** Opacité des points de nuage. */
const CLOUD_OPACITY = 0.55
/**
 * Sous ce nombre de matchs, une lane n'a pas de distribution à montrer (1-2
 * points valent la médiane elle-même) : le nuage est supprimé pour elle,
 * seuls la médiane et la barre d'avance restent dessinées. Retour utilisateur
 * (2026-08-29) : à faible N, le nuage (6 px/0.4 d'opacité) se confondait avec
 * le marqueur de médiane (16 px).
 */
const MIN_MATCHES_FOR_CLOUD = 3

export type { FirstBloodMatch, FirstBloodPlayerSeries } from './firstBloodLanesModel'

/** Libellés localisés consommés par le builder (le builder reste sans locale). */
export interface FirstBloodLanesLabels {
  /** Préfixe de la 2e ligne de la colonne de gauche (« méd. »). */
  medianPrefix: string
  /** 3e ligne — reçoit l'écart DÉJÀ signé et formaté (« +14s »). */
  advance: (gap: string) => string
  /**
   * Tooltip d'un point de nuage (déjà échappé HTML). `map`/`mode` sont déjà
   * résolus en placeholder (NO_VALUE) si absents ; `startTime` est l'ISO brut,
   * formaté en date locale ICI (le builder pur ne connaît pas la locale) —
   * DEC-4 : carte · mode · date, plus jamais l'uuid du match.
   */
  matchEvent: (
    kind: 'kill' | 'death',
    v: { player: string; map: string; mode: string; startTime: string; time: string },
  ) => string
  /** Tooltip d'un marqueur de médiane. */
  medianEvent: (kind: 'kill' | 'death', v: { time: string; n: number; total: number }) => string
  /** Tooltip de la barre d'avance (écart déjà signé et formaté). */
  gap: (gap: string) => string
}

export interface FirstBloodLanesProps {
  data: FirstBloodPlayerSeries[]
  /** Fenêtre de l'axe X en secondes (défaut 300). */
  maxSec?: number
  /**
   * Titre de la carte. Absent → titre du manifest ; chaîne vide → carte sans
   * en-tête (le graphe est alors titré par son conteneur).
   */
  title?: ReactNode
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  /** Force la locale (défaut : celle du shell). */
  locale?: Locale
}

export function FirstBloodLanes({
  data,
  maxSec = DEFAULT_MAX_SEC,
  title,
  loading,
  error,
  emptyMessage,
  locale: localeProp,
}: FirstBloodLanesProps) {
  const shellLocale = useAppShellStore((s) => s.locale)
  const locale = localeProp ?? shellLocale

  const lanes = useMemo(() => buildFirstBloodLanes(data), [data])
  const labels = useMemo(() => makeLabels(locale), [locale])

  const buildOption = useCallback(
    (series: ChartSeries<FirstBloodLane>[]) =>
      buildFirstBloodLanesOption(series[0]?.datapoints ?? [], { maxSec, labels }),
    [maxSec, labels],
  )

  // Une lane sans aucun événement exploitable ne dessine rien : si AUCUNE lane
  // n'en a, on rend l'état vide de ChartCard plutôt qu'une grille nue.
  const hasEvents = lanes.some((l) => l.kills.length > 0 || l.deaths.length > 0)
  const series: ChartSeries<FirstBloodLane>[] = hasEvents
    ? [{ key: 'first-blood', datapoints: lanes }]
    : []

  return (
    <ChartCard
      title={title ?? formatMessage(firstBloodManifest, 'first_blood.title', locale)}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage ?? formatMessage(firstBloodManifest, 'first_blood.empty', locale)}
      height={firstBloodLanesHeight(lanes.length)}
      buildOption={buildOption}
    />
  )
}

/** Résout les libellés du manifest pour une locale donnée. */
function makeLabels(locale: Locale): FirstBloodLanesLabels {
  const t = (key: FirstBloodManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(firstBloodManifest, key, locale, vars)
  return {
    medianPrefix: t('first_blood.label.median_prefix'),
    advance: (gap) => t('first_blood.label.advance', { gap }),
    matchEvent: (kind, v) =>
      t(kind === 'kill' ? 'first_blood.tooltip.first_kill' : 'first_blood.tooltip.first_death', {
        player: v.player,
        map: v.map || NO_VALUE,
        mode: v.mode || NO_VALUE,
        date: formatDate(v.startTime, locale),
        time: v.time,
      }),
    medianEvent: (kind, v) =>
      t(kind === 'kill' ? 'first_blood.tooltip.median_kill' : 'first_blood.tooltip.median_death', {
        time: v.time,
        n: v.n,
        total: v.total,
      }),
    gap: (gap) => t('first_blood.tooltip.gap', { gap }),
  }
}

// ── Builder d'option (pur, testable sans React) ───────────────────────────────

interface BuildOpts {
  maxSec: number
  labels: FirstBloodLanesLabels
}

/** Point de nuage porté par les séries scatter (tooltip item). */
interface CloudPoint {
  value: [number, number]
  player: string
  matchId: string
  /** Carte/mode/date du match — dégradent en NO_VALUE / omis côté tooltip
   *  si absents (DEC-4, jamais l'uuid). */
  mapUI?: string
  modeUI?: string
  startTime?: string
}

/** Marqueur de médiane (tooltip item : temps + couverture n/total). */
interface MedianPoint {
  value: [number, number]
  player: string
  n: number
  total: number
}

/** Barre d'avance d'une lane, résolue en amont du renderItem. */
interface GapBar {
  value: [number, number, number]
  player: string
  gapSec: number
  positive: boolean
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildFirstBloodLanesOption(
  lanes: FirstBloodLane[],
  opts: BuildOpts,
): EChartsCoreOption {
  const { maxSec, labels } = opts
  if (lanes.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const killColor = resolveToken('outcome-win')
  const deathColor = resolveToken('outcome-loss')

  const bars: GapBar[] = lanes.flatMap((l, i) =>
    l.medianKillSec == null || l.medianDeathSec == null || l.gapSec == null
      ? []
      : [
          {
            value: [l.medianKillSec, i, l.medianDeathSec] as [number, number, number],
            player: l.player,
            gapSec: l.gapSec,
            positive: l.gapSec >= 0,
          },
        ],
  )

  const cloud = (kind: 'kill' | 'death'): CloudPoint[] =>
    lanes.flatMap((l, i) => {
      // Sous MIN_MATCHES_FOR_CLOUD, la lane n'a pas de distribution à montrer
      // (médianes et barre d'avance restent dessinées, cf. buildGapSeries /
      // buildMedianSeries — seul LE NUAGE est concerné par ce seuil).
      if (l.totalMatches < MIN_MATCHES_FOR_CLOUD) return []
      return (kind === 'kill' ? l.kills : l.deaths).map((p) => ({
        value: [p.sec, i] as [number, number],
        player: l.player,
        matchId: p.matchId,
        mapUI: p.mapUI,
        modeUI: p.modeUI,
        startTime: p.startTime,
      }))
    })

  const medians = (kind: 'kill' | 'death'): MedianPoint[] =>
    lanes.flatMap((l, i) => {
      const sec = kind === 'kill' ? l.medianKillSec : l.medianDeathSec
      if (sec == null) return []
      return [
        {
          value: [sec, i] as [number, number],
          player: l.player,
          n: (kind === 'kill' ? l.kills : l.deaths).length,
          total: l.totalMatches,
        },
      ]
    })

  return {
    backgroundColor: CHART_BG,
    grid: { top: GRID_TOP, bottom: GRID_BOTTOM, left: LABEL_WIDTH + 8, right: 16 },
    tooltip: { ...getTooltipBase(tc), trigger: 'item' },
    xAxis: {
      type: 'value',
      min: 0,
      max: maxSec,
      interval: 60,
      axisLine: { lineStyle: { color: tc.axisLine } },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: tc.splitLine, type: 'dashed' } },
      axisLabel: { color: tc.axisLabel, fontSize: 10, formatter: formatAxisTick },
    },
    yAxis: buildLaneAxis(lanes, { killColor, deathColor, textColor: tc.text, mutedColor: tc.axisLabel, labels }),
    series: [
      buildGapSeries(bars, { killColor, deathColor, labels }),
      ...buildCloudSeries(cloud, { killColor, deathColor, labels }),
      ...buildMedianSeries(medians, { killColor, deathColor, borderColor: tc.tooltipBg, labels }),
    ],
  }
}

interface AxisColors {
  killColor: string
  deathColor: string
  textColor: string
  mutedColor: string
  labels: FirstBloodLanesLabels
}

/**
 * Axe des lanes : catégorie inversée (index 0 = ligne du haut = joueur le plus
 * rapide) et colonne de libellés en texte enrichi. `margin: LABEL_WIDTH` + `align:
 * 'left'` ancrent le bloc à gauche du grid : RIEN n'empiète sur la zone de tracé.
 */
function buildLaneAxis(lanes: FirstBloodLane[], c: AxisColors) {
  return {
    type: 'category' as const,
    data: lanes.map((l) => sanitizeRichText(l.player)),
    inverse: true,
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { show: false },
    axisLabel: {
      align: 'left' as const,
      margin: LABEL_WIDTH,
      width: LABEL_WIDTH,
      overflow: 'truncate' as const,
      formatter: (_value: string, index: number) => richLaneLabel(lanes[index], c.labels),
      rich: {
        gt: { color: c.textColor, fontSize: 12, fontWeight: 'bold' as const, lineHeight: 17 },
        med: { color: c.mutedColor, fontSize: 10, lineHeight: 15 },
        kill: { color: c.killColor, fontSize: 10, fontWeight: 'bold' as const, lineHeight: 15 },
        death: { color: c.deathColor, fontSize: 10, fontWeight: 'bold' as const, lineHeight: 15 },
        gapPos: { color: c.killColor, fontSize: 10, fontWeight: 'bold' as const, lineHeight: 15 },
        gapNeg: { color: c.deathColor, fontSize: 10, fontWeight: 'bold' as const, lineHeight: 15 },
      },
    },
  }
}

/** 3 lignes : pseudo · « méd. 50s → 1m04 » · « +14s d'avance ». */
function richLaneLabel(lane: FirstBloodLane | undefined, labels: FirstBloodLanesLabels): string {
  if (!lane) return ''
  const kill = lane.medianKillSec == null ? NO_VALUE : formatLaneSeconds(lane.medianKillSec)
  const death = lane.medianDeathSec == null ? NO_VALUE : formatLaneSeconds(lane.medianDeathSec)
  const lines = [
    `{gt|${sanitizeRichText(lane.player)}}`,
    `{med|${labels.medianPrefix} }{kill|${kill}}{med| → }{death|${death}}`,
  ]
  if (lane.gapSec != null) {
    const style = lane.gapSec >= 0 ? 'gapPos' : 'gapNeg'
    lines.push(`{${style}|${sanitizeRichText(labels.advance(formatGapSeconds(lane.gapSec)))}}`)
  }
  return lines.join('\n')
}

interface SeriesColors {
  killColor: string
  deathColor: string
  labels: FirstBloodLanesLabels
}

/**
 * Barre « fenêtre d'avance » : rectangle arrondi de 8 px entre les deux médianes,
 * centré sur la ligne. Série `custom` car les deux bornes sont des valeurs X et la
 * hauteur est en pixels (indépendante de l'échelle Y catégorielle).
 */
function buildGapSeries(bars: GapBar[], c: SeriesColors) {
  return {
    type: 'custom' as const,
    z: 1,
    data: bars,
    renderItem: (params: unknown, api: unknown) => {
      const bar = bars[(params as { dataIndex: number }).dataIndex]
      if (!bar) return { type: 'group', children: [] }
      const a = api as { coord: (v: number[]) => number[] }
      const [x0, y] = a.coord([bar.value[0], bar.value[1]])
      const [x1] = a.coord([bar.value[2], bar.value[1]])
      const left = Math.min(x0, x1)
      const width = Math.max(2, Math.abs(x1 - x0))
      return {
        type: 'rect',
        shape: {
          x: left,
          y: y - GAP_BAR_HEIGHT / 2,
          width,
          height: GAP_BAR_HEIGHT,
          r: GAP_BAR_HEIGHT / 2,
        },
        style: { fill: bar.positive ? c.killColor : c.deathColor, opacity: 0.32 },
      }
    },
    tooltip: {
      formatter: (p: unknown) => {
        const bar = (p as { data?: GapBar }).data
        if (!bar) return ''
        return `<b>${escapeHtml(bar.player)}</b><br/>${c.labels.gap(formatGapSeconds(bar.gapSec))}`
      },
    },
  }
}

/**
 * Nuages de fond : 1 point par match, décalés de ±14 px autour de la ligne.
 * Supprimé lane par lane sous MIN_MATCHES_FOR_CLOUD (cf. `cloud()` dans
 * buildFirstBloodLanesOption) ; taille/opacité relevées (8 px/0.55) pour que
 * les points restants se distinguent plutôt que de faire tache.
 */
function buildCloudSeries(cloud: (kind: 'kill' | 'death') => CloudPoint[], c: SeriesColors) {
  const one = (kind: 'kill' | 'death') => ({
    type: 'scatter' as const,
    z: 2,
    symbolSize: CLOUD_SYMBOL_SIZE,
    symbolOffset: [0, kind === 'kill' ? -POINT_OFFSET : POINT_OFFSET] as [number, number],
    itemStyle: { color: kind === 'kill' ? c.killColor : c.deathColor, opacity: CLOUD_OPACITY },
    data: cloud(kind),
    tooltip: {
      formatter: (p: unknown) => {
        const d = (p as { data?: CloudPoint }).data
        if (!d) return ''
        // Échappement des données non constantes (pseudo, carte, mode — des
        // libellés résolus serveur, pas des constantes) — le gabarit i18n,
        // lui, doit garder ses apostrophes intactes. DEC-4 : plus jamais
        // l'identifiant de match dans le tooltip.
        return c.labels.matchEvent(kind, {
          player: escapeHtml(d.player),
          map: escapeHtml(d.mapUI ?? ''),
          mode: escapeHtml(d.modeUI ?? ''),
          startTime: d.startTime ?? '',
          time: formatLaneSeconds(d.value[0]),
        })
      },
    },
  })
  return [one('kill'), one('death')]
}

/**
 * Marqueurs de médiane : gros points sur la ligne, cerclés de la couleur de fond
 * de carte (`--popover`) — CHART_BG vaut 'transparent' et ne détacherait rien du
 * nuage sous-jacent (même « découpe » que les losanges d'OutcomeSequenceTape).
 */
function buildMedianSeries(
  medians: (kind: 'kill' | 'death') => MedianPoint[],
  c: SeriesColors & { borderColor: string },
) {
  const one = (kind: 'kill' | 'death') => ({
    type: 'scatter' as const,
    z: 3,
    symbolSize: MEDIAN_SYMBOL_SIZE,
    itemStyle: {
      color: kind === 'kill' ? c.killColor : c.deathColor,
      borderColor: c.borderColor,
      borderWidth: 2,
    },
    data: medians(kind),
    tooltip: {
      formatter: (p: unknown) => {
        const d = (p as { data?: MedianPoint }).data
        if (!d) return ''
        const line = c.labels.medianEvent(kind, {
          time: formatLaneSeconds(d.value[0]),
          n: d.n,
          total: d.total,
        })
        // `line` ne contient que du gabarit i18n + des nombres : seul le pseudo
        // (donnée non constante) est échappé.
        return `<b>${escapeHtml(d.player)}</b><br/>${line}`
      },
    },
  })
  return [one('kill'), one('death')]
}
