/**
 * squadIntensityProfileChart — « Intensité » refondue : profil d'activité par
 * phase de match (remplace la heatmap matchs × phases).
 *
 * Un panneau par joueur (grille 2 colonnes : 1 joueur = pleine largeur ; 2 = une
 * rangée ; 3 = 2+1 ; 4 = 2×2). Chaque panneau trace :
 *   - la MÉDIANE des parts de frags par phase (trait épais, couleur joueur) ;
 *   - l'ENVELOPPE interquartile P25–P75 (aplat même teinte, opacité faible) —
 *     paire de séries empilées (base P25 transparente + (P75−P25) en areaStyle) ;
 *   - un repère pointillé à 10 % (activité uniforme sur 10 phases).
 *
 * Échelle Y PARTAGÉE entre panneaux (max commun) pour comparer les joueurs. Un
 * joueur sans manche exploitable n'a pas de panneau. En-deçà de
 * `MIN_MATCHES_FOR_ENVELOPE` manches, le panneau n'affiche que la médiane
 * (enveloppe dégénérée). Le calcul agrégé vit dans `lib/charts/phaseProfile`.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
  hexToRgba,
} from '@/components/charts/_utils'
import {
  MIN_MATCHES_FOR_ENVELOPE,
  PHASE_COUNT,
  phaseProfile,
  type PhaseProfileResult,
} from '@/lib/charts/phaseProfile'
import { SQUAD_INTENSITY_PHASE_LABELS } from './squadIntensityHeatmapChart'

/** Repère « activité uniforme » : 1/PHASE_COUNT des frags par phase. */
export const UNIFORM_SHARE = 1 / PHASE_COUNT

/** Opacité de l'aplat d'enveloppe (même teinte que la médiane du joueur). */
const ENVELOPE_OPACITY = 0.16

/** Un panneau = un joueur : gamertag, couleur résolue, manches du scope. */
export interface IntensityPanelInput {
  key: string
  /** Titre du panneau (gamertag). */
  label: string
  /** Couleur hex résolue (semantic token via colorByPlayer). */
  color: string
  rows: Array<{ phases: number[] | null }>
}

export interface IntensityProfileOpts {
  panels: IntensityPanelInput[]
  /** Libellés tooltip (i18n caller). */
  medianLabel: string
  envelopeLabel: string
  /** Libellé du repère 10 %. */
  refLabel: string
}

/** Panneau retenu (au moins une manche exploitable) + son profil agrégé. */
interface ResolvedPanel extends IntensityPanelInput {
  profile: PhaseProfileResult
}

interface GridBox {
  left: string
  width: string
  top: string
  height: string
  /** Centre horizontal (%) — ancre du titre de panneau. */
  centerX: number
  /** Haut du titre de panneau (%). */
  titleTop: number
}

// Marges de layout (pourcentages du conteneur). Le conteneur grandit avec le
// nombre de rangées (hauteur pilotée par le composant), ces ratios restent donc
// lisibles jusqu'à 2 rangées.
const LAYOUT = { top: 9, bottom: 7, vGap: 13, left: 6, right: 3, hGap: 7 } as const

/** Positions des grilles (2 colonnes, N panneaux). N=1 → pleine largeur. */
export function computeGrids(n: number): GridBox[] {
  const cols = n === 1 ? 1 : 2
  const rows = Math.ceil(n / cols)
  const usableH = 100 - LAYOUT.top - LAYOUT.bottom - (rows - 1) * LAYOUT.vGap
  const rowH = usableH / rows
  const usableW = 100 - LAYOUT.left - LAYOUT.right - (cols - 1) * LAYOUT.hGap
  const colW = usableW / cols
  const boxes: GridBox[] = []
  for (let i = 0; i < n; i += 1) {
    const r = Math.floor(i / cols)
    const c = i % cols
    const leftPct = LAYOUT.left + c * (colW + LAYOUT.hGap)
    const topPct = LAYOUT.top + r * (rowH + LAYOUT.vGap)
    boxes.push({
      left: `${leftPct}%`,
      width: `${colW}%`,
      top: `${topPct}%`,
      height: `${rowH}%`,
      centerX: leftPct + colW / 2,
      titleTop: Math.max(1, topPct - 6),
    })
  }
  return boxes
}

/** Borne Y partagée : max des P75 (ou médianes si enveloppe omise) + le repère. */
function sharedYMax(panels: ResolvedPanel[]): number {
  let max = UNIFORM_SHARE
  for (const p of panels) {
    const useEnvelope = p.profile.nMatches >= MIN_MATCHES_FOR_ENVELOPE
    const top = useEnvelope ? p.profile.p75 : p.profile.median
    for (const v of top) if (v > max) max = v
  }
  // Marge de tête ~12 %, plafonnée à 1 (100 % des frags).
  return Math.min(1, max * 1.12)
}

/** Séries d'un panneau (base P25 + bande + médiane + repère), liées à la grille i. */
function buildPanelSeries(panel: ResolvedPanel, gi: number, refColor: string, refLabel: string) {
  const { profile, color } = panel
  const axisBinding = { xAxisIndex: gi, yAxisIndex: gi } as const
  const noSymbol = { showSymbol: false, symbol: 'none' as const }
  const series: Record<string, unknown>[] = []

  if (profile.nMatches >= MIN_MATCHES_FOR_ENVELOPE) {
    const band = profile.p75.map((hi, i) => hi - profile.p25[i])
    series.push({
      id: `base-${gi}`,
      name: panel.label,
      type: 'line',
      stack: `env-${gi}`,
      data: profile.p25,
      lineStyle: { opacity: 0 },
      areaStyle: { color: 'transparent', opacity: 0 },
      ...noSymbol,
      silent: true,
      ...axisBinding,
      z: 2,
    })
    series.push({
      id: `band-${gi}`,
      name: panel.label,
      type: 'line',
      stack: `env-${gi}`,
      data: band,
      lineStyle: { opacity: 0 },
      areaStyle: { color: hexToRgba(color, ENVELOPE_OPACITY) },
      ...noSymbol,
      silent: true,
      ...axisBinding,
      z: 2,
    })
  }

  series.push({
    id: `median-${gi}`,
    name: panel.label,
    type: 'line',
    data: profile.median,
    lineStyle: { color, width: 2.5 },
    itemStyle: { color },
    ...noSymbol,
    ...axisBinding,
    z: 4,
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: refColor, type: 'dashed', width: 1, opacity: 0.7 },
      data: [{ yAxis: UNIFORM_SHARE }],
      label: { formatter: refLabel, position: 'end', color: refColor, fontSize: 8 },
    },
  })
  return series
}

const asPct = (v: number): string => `${Math.round(v * 100)}%`

/** Tooltip axis : phase + médiane + fourchette P25–P75 du panneau survolé. */
function buildTooltipFormatter(medianLabel: string, envelopeLabel: string) {
  return (params: unknown): string => {
    const arr = (Array.isArray(params) ? params : [params]) as Array<{
      seriesId?: string
      seriesName?: string
      axisValueLabel?: string
      axisValue?: string
      value?: number
    }>
    if (arr.length === 0) return ''
    const phase = arr[0].axisValueLabel ?? arr[0].axisValue ?? ''
    const median = arr.find((p) => String(p.seriesId ?? '').startsWith('median-'))
    const base = arr.find((p) => String(p.seriesId ?? '').startsWith('base-'))
    const band = arr.find((p) => String(p.seriesId ?? '').startsWith('band-'))
    const lines = [`<b>${escapeHtml(String(phase))}</b>`]
    if (median?.seriesName) lines.push(`<b>${escapeHtml(median.seriesName)}</b>`)
    if (median && Number.isFinite(median.value)) {
      lines.push(`${escapeHtml(medianLabel)} : <b>${asPct(median.value as number)}</b>`)
    }
    if (base && band && Number.isFinite(base.value) && Number.isFinite(band.value)) {
      const lo = base.value as number
      const hi = lo + (band.value as number)
      lines.push(`${escapeHtml(envelopeLabel)} : ${asPct(lo)} – ${asPct(hi)}`)
    }
    return lines.join('<br/>')
  }
}

/**
 * buildSquadIntensityProfileOption — option ECharts multi-panneaux. Retourne une
 * option minimale (fond seul) si aucun joueur n'a de manche exploitable.
 */
export function buildSquadIntensityProfileOption(opts: IntensityProfileOpts): EChartsCoreOption {
  const resolved: ResolvedPanel[] = opts.panels
    .map((p) => ({ ...p, profile: phaseProfile(p.rows) }))
    .filter((p) => p.profile.nMatches > 0)

  if (resolved.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const grids = computeGrids(resolved.length)
  const yMax = sharedYMax(resolved)
  const refColor = tc.axisLabel

  const xAxes = grids.map((_, gi) => ({
    ...axis,
    gridIndex: gi,
    type: 'category' as const,
    data: SQUAD_INTENSITY_PHASE_LABELS,
    axisLabel: { ...axis.axisLabel, rotate: -25, interval: 0, fontSize: 8 },
  }))
  const yAxes = grids.map((_, gi) => ({
    ...axis,
    gridIndex: gi,
    type: 'value' as const,
    min: 0,
    max: yMax,
    axisLabel: { ...axis.axisLabel, formatter: (v: number) => asPct(v) },
  }))
  const titles = grids.map((g, gi) => ({
    text: resolved[gi].label,
    left: `${g.centerX}%`,
    top: `${g.titleTop}%`,
    textAlign: 'center' as const,
    textStyle: { color: resolved[gi].color, fontSize: 12, fontWeight: 600 as const },
  }))
  const series = resolved.flatMap((p, gi) =>
    buildPanelSeries(p, gi, refColor, opts.refLabel),
  )

  return {
    backgroundColor: CHART_BG,
    title: titles,
    grid: grids.map((g) => ({
      left: g.left,
      width: g.width,
      top: g.top,
      height: g.height,
      containLabel: true,
    })),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: buildTooltipFormatter(opts.medianLabel, opts.envelopeLabel),
    },
    xAxis: xAxes,
    yAxis: yAxes,
    series,
  }
}
