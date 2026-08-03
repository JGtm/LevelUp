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
  hoverRevealSymbol,
} from '@/components/charts/_utils'
import {
  MIN_MATCHES_FOR_ENVELOPE,
  PHASE_COUNT,
  phaseProfile,
  type PhaseProfileResult,
} from '@/lib/charts/phaseProfile'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'

/** Libellés des 10 tranches de phase (0-10 %, …, 90-100 %) — data de l'axe X + tooltip. */
export const INTENSITY_PHASE_LABELS = [
  '0-10%', '10-20%', '20-30%', '30-40%', '40-50%',
  '50-60%', '60-70%', '70-80%', '80-90%', '90-100%',
]

/** Index de la tranche « milieu » (0-based) affichée sur l'axe X. */
const PHASE_MID_INDEX = Math.floor(PHASE_COUNT / 2)

/**
 * Étiquettes de l'axe X du profil d'intensité. L'axe ne montre que 3 repères
 * (début / milieu / fin du match) — les 10 tranches restent en data (tooltip
 * précis), mais afficher 10 pourcentages était illisible et se confondait avec
 * l'axe Y (lui aussi en %). `rangeSuffix` complète la tranche dans le tooltip
 * (ex. « 40-50% du match »).
 */
export interface IntensityAxisLabels {
  start: string
  mid: string
  end: string
  rangeSuffix: string
}

/**
 * Étiquettes d'axe X partagées par les 3 surfaces (Escouade / Timeseries /
 * Sessions) — source unique via le manifest i18n `common.charts.intensity_axis_*`
 * (parité FR/EN garantie par le manifest, `formatMessage` étant pur et utilisable
 * hors composant React).
 */
export function intensityAxisLabels(locale: Locale): IntensityAxisLabels {
  return {
    start: formatMessage(commonManifest, 'common.charts.intensity_axis_start', locale),
    mid: formatMessage(commonManifest, 'common.charts.intensity_axis_mid', locale),
    end: formatMessage(commonManifest, 'common.charts.intensity_axis_end', locale),
    rangeSuffix: formatMessage(commonManifest, 'common.charts.intensity_axis_range_suffix', locale),
  }
}

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

/**
 * Courbe de référence d'ÉQUIPE superposée à chaque panneau (escouade de 3+
 * joueurs). Les manches attendues sont celles de la ligne agrégée `all` du
 * payload — un match y porte les frags de TOUS les joueurs sélectionnés — et non
 * une moyenne des médianes par joueur : l'agrégation reste faite par le helper
 * canonique `phaseProfile`, comme pour les panneaux joueur.
 */
export interface IntensityTeamOverlay {
  /** Libellé de la courbe (titre de la ligne dans le tooltip). */
  label: string
  /** Manches d'équipe (`intensity_profile.rows.all`). */
  rows: Array<{ phases: number[] | null }>
}

export interface IntensityProfileOpts {
  panels: IntensityPanelInput[]
  /** Libellés tooltip (i18n caller). */
  medianLabel: string
  envelopeLabel: string
  /** Libellé du repère 10 %. */
  refLabel: string
  /** Étiquettes de l'axe X (début / milieu / fin) + suffixe tooltip de tranche. */
  axisLabels: IntensityAxisLabels
  /**
   * Repère d'équipe superposé à CHAQUE panneau. Absent (défaut) = rendu
   * strictement identique à l'existant : Sessions et Timeseries, qui montent le
   * même builder en solo, n'héritent de rien.
   */
  teamOverlay?: IntensityTeamOverlay
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

/** Repère d'équipe résolu : médiane par phase + habillage (label / couleur). */
interface ResolvedTeam {
  label: string
  color: string
  median: number[]
}

/**
 * Médiane par phase de l'ÉQUIPE, agrégée par le MÊME helper canonique que les
 * panneaux joueur (`phaseProfile`) sur les manches d'escouade fournies par
 * l'appelant. `undefined` si aucune manche n'est exploitable (0 frag sur toute la
 * sélection) → aucune courbe plate n'est tracée.
 *
 * color-allow: `color` reçoit `tc.text`, neutre de thème réservé aux séries
 * partagées (précédent : courbe « MMR équipe » de squadPerformanceLineCharts) — il
 * ne peut entrer en collision avec aucune couleur joueur.
 */
function resolveTeam(overlay: IntensityTeamOverlay | undefined, color: string): ResolvedTeam | undefined {
  if (!overlay) return undefined
  const profile = phaseProfile(overlay.rows)
  if (profile.nMatches === 0) return undefined
  return { label: overlay.label, color, median: profile.median }
}

/** Contexte commun aux panneaux (repère 10 %, courbe d'équipe optionnelle). */
interface PanelContext {
  refColor: string
  refLabel: string
  team?: ResolvedTeam
}

/** Borne Y partagée : max des P75 (ou médianes si enveloppe omise) + le repère. */
function sharedYMax(panels: ResolvedPanel[], team?: ResolvedTeam): number {
  let max = UNIFORM_SHARE
  for (const p of panels) {
    const useEnvelope = p.profile.nMatches >= MIN_MATCHES_FOR_ENVELOPE
    const top = useEnvelope ? p.profile.p75 : p.profile.median
    for (const v of top) if (v > max) max = v
  }
  // La courbe d'équipe est superposée aux panneaux : elle doit tenir dans l'échelle.
  if (team) for (const v of team.median) if (v > max) max = v
  // Marge de tête ~12 %, plafonnée à 1 (100 % des frags).
  return Math.min(1, max * 1.12)
}

/** Séries d'un panneau (base P25 + bande + médiane + repère + équipe), liées à la grille i. */
function buildPanelSeries(panel: ResolvedPanel, gi: number, ctx: PanelContext) {
  const { refColor, refLabel, team } = ctx
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
    // Point révélé au survol de la tranche (helper partagé) ; les séries
    // d'enveloppe restent sans symbole (elles sont `silent`).
    ...hoverRevealSymbol(color),
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

  // Courbe d'ÉQUIPE superposée (escouade de 3+ joueurs) : même médiane par phase,
  // calculée sur les manches agrégées de l'escouade. Trait fin pointillé sous la
  // courbe du joueur (z inférieur) pour rester un repère, pas un concurrent.
  if (team) {
    series.push({
      id: `team-${gi}`,
      name: team.label,
      type: 'line',
      data: team.median,
      lineStyle: { color: team.color, width: 1.5, type: 'dashed', opacity: 0.85 },
      ...hoverRevealSymbol(team.color, 6),
      ...axisBinding,
      z: 3,
    })
  }
  return series
}

const asPct = (v: number): string => `${Math.round(v * 100)}%`

/** Tooltip axis : tranche précise + médiane + fourchette P25–P75 du panneau survolé. */
function buildTooltipFormatter(medianLabel: string, envelopeLabel: string, rangeSuffix: string) {
  return (params: unknown): string => {
    const arr = (Array.isArray(params) ? params : [params]) as Array<{
      seriesId?: string
      seriesName?: string
      dataIndex?: number
      value?: number
    }>
    if (arr.length === 0) return ''
    // L'axe X n'affiche que 3 repères (début/milieu/fin) : la tranche précise du
    // tooltip vient de dataIndex (jamais de axisValueLabel, masqué hors des 3 repères).
    const idx = typeof arr[0].dataIndex === 'number' ? arr[0].dataIndex : 0
    const range = INTENSITY_PHASE_LABELS[idx] ?? ''
    const phase = rangeSuffix ? `${range} ${rangeSuffix}` : range
    const median = arr.find((p) => String(p.seriesId ?? '').startsWith('median-'))
    const base = arr.find((p) => String(p.seriesId ?? '').startsWith('base-'))
    const band = arr.find((p) => String(p.seriesId ?? '').startsWith('band-'))
    const team = arr.find((p) => String(p.seriesId ?? '').startsWith('team-'))
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
    // Repère d'équipe (présent uniquement quand la courbe agrégée est montée).
    if (team?.seriesName && Number.isFinite(team.value)) {
      lines.push(`${escapeHtml(team.seriesName)} : ${asPct(team.value as number)}`)
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
  const team = resolveTeam(opts.teamOverlay, tc.text)
  const yMax = sharedYMax(resolved, team)
  const refColor = tc.axisLabel

  const { start, mid, end } = opts.axisLabels
  const xAxes = grids.map((_, gi) => ({
    ...axis,
    gridIndex: gi,
    type: 'category' as const,
    data: INTENSITY_PHASE_LABELS,
    // Seuls 3 repères affichés (début / milieu / fin) — les 10 tranches restent en
    // data pour le tooltip. Évite l'illisibilité de 10 pourcentages face à l'axe Y.
    axisLabel: {
      ...axis.axisLabel,
      interval: (index: number) =>
        index === 0 || index === PHASE_MID_INDEX || index === PHASE_COUNT - 1,
      formatter: (_value: string, index: number) =>
        index === 0 ? start : index === PHASE_MID_INDEX ? mid : index === PHASE_COUNT - 1 ? end : '',
      fontSize: 9,
    },
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
    buildPanelSeries(p, gi, { refColor, refLabel: opts.refLabel, team }),
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
      formatter: buildTooltipFormatter(opts.medianLabel, opts.envelopeLabel, opts.axisLabels.rangeSuffix),
    },
    xAxis: xAxes,
    yAxis: yAxes,
    series,
  }
}
