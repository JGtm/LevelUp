/**
 * Heatmap2DChart — wrapper ECharts pour `ChartSeries<ChartPointHeatmap>`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ x, y, value, detail? }`.
 *   - Axes X et Y sont des chaînes (déjà résolues côté service en labels
 *     lisibles).
 *
 * Cas d'usage Squad V2 :
 *   - S3 Heatmap player × map (perf score)
 *   - S6 Intensity match × bucket
 *   - S5 Impact heatmap match × player (potentiel future)
 *
 * Le `value` est mappé via `visualMap` ECharts en gradient cold→hot ou
 * divergent low→high selon `paletteMode`.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore, type ColorPalette } from '@/stores/settingsDraftStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { heatmapRampTokens } from './heatmapColors'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
  type EChartsThemeColors,
} from './_utils'

/**
 * Libellé de la légende affichée quand la série contient au moins une case
 * `value: null` (décision D3, plan vague C formes 2026-09-08). Dictionnaire local
 * plutôt qu'un manifeste TOML : une seule chaîne, propre à ce wrapper — parité
 * FR/EN garantie par le typage `Record<Locale, T>` (même patron que
 * `lib/review/i18n.ts`).
 */
const HEATMAP_EMPTY_CELL_TEXT: Record<Locale, string> = {
  fr: 'Aucune mesure sur cet axe',
  en: 'No measurement on this axis',
}

/** Caractère affiché SUR une case vide (décision D3 : « un tiret »), au lieu de rien. */
const EMPTY_CELL_LABEL = '—'

/** Épaisseur du liseré entre cases (décision D2 — « c'est aéré et joli »). */
const CELL_BORDER_WIDTH = 4
/** Arrondi des coins de case (décision D2). */
const CELL_BORDER_RADIUS = 2

export interface ChartPointHeatmap {
  x: string
  y: string
  /**
   * Valeur de la cellule, ou `null` pour une case VIDE — hors échelle (le
   * visualMap ne la classe pas) mais désormais VISIBLE : hachurée, avec un tiret,
   * et nommée par la légende du graphe (décision D3, plan vague C formes
   * 2026-09-08). Avant cette décision (ajout 2026-09-06, correction W1) elle était
   * non peinte et sans étiquette — la doctrine du dépôt l'a corrigé : « l'absence a
   * sa propre forme, jamais un vide ».
   *
   * POURQUOI UNE CASE VIDE PLUTÔT QU'UNE CASE ABSENTE. Les axes de ce wrapper sont
   * DÉDUITS de l'ordre d'apparition des points : omettre une case décale les
   * catégories, et une matrice carrée à diagonale omise sortait avec un axe X
   * décalé d'un cran par rapport à l'axe Y (roster [A,B,C,D] → colonnes B,C,D,A).
   * Une case impossible doit donc être ÉMISE, et dite vide.
   */
  value: number | null
  detail?: Record<string, unknown>
}

/**
 * Modes de rampe acceptes par ce wrapper — sous-ensemble de HeatmapRampMode.
 *
 * 'frequency' (ajout 2026-09-06, matrice d'echange de l'escouade) : rampe NEUTRE
 * mono-teinte, monotone en luminance dans toutes les palettes. C'est le mode d'une
 * intensite qui ne porte AUCUN jugement — un nombre de vengeances n'est ni chaud ni
 * froid, et la rampe cold->hot lui collerait un « bon / mauvais » que la donnee ne dit
 * pas. Elle interdit aussi de confondre l'echelle avec les couleurs par JOUEUR.
 */
export type HeatmapPaletteMode = 'sequential' | 'divergent' | 'frequency'

export interface Heatmap2DChartProps {
  title?: string
  series: ChartSeries<ChartPointHeatmap>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** sequential (cold→hot) ou divergent (low→neutral→high). Default 'sequential'. */
  paletteMode?: HeatmapPaletteMode
  /** Min/max forcés du visualMap (default = auto-fit). */
  valueRange?: [number, number]
  /**
   * Plafond de saturation OPTIONNEL du visualMap (`max`) — la maquette sature à 30
   * (points d'intensité) : au-delà, une case garde la couleur la plus chaude au lieu
   * d'étirer l'échelle. Ignoré si `valueRange` est fourni (celui-ci fixe déjà min ET
   * max). Absent = comportement HISTORIQUE inchangé (max = valeur réelle la plus
   * haute) — n'affecte donc aucun consommateur existant tant qu'il ne le passe pas.
   */
  saturationCap?: number
  /**
   * Contenu HTML du tooltip d'une cellule. Absent = le libellé historique de la
   * heatmap joueur × carte (taux de victoire + nombre de matchs), qui ne convient
   * qu'à ce cas d'usage — toute autre donnée DOIT passer sa propre fonction, sinon
   * la cellule s'annonce sous un nom qui n'est pas le sien.
   *
   * L'appelant est responsable de l'échappement de ce qu'il injecte.
   */
  formatTooltip?: (point: ChartPointHeatmap) => string
}

export function Heatmap2DChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  paletteMode = 'sequential',
  valueRange,
  saturationCap,
  formatTooltip,
}: Heatmap2DChartProps) {
  // Palette d'accessibilité active : pilote la rampe CVD-safe (rebuild via
  // useColorPaletteVersion dans ChartCard + ce sélecteur au changement de palette).
  const colorPalette = useSettingsDraftStore((s) => s.localUiPrefs.colorPalette)
  const locale = useAppShellStore((s) => s.locale)
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHeatmap>[]) =>
      buildHeatmap2DOption(s, { paletteMode, valueRange, saturationCap, colorPalette, formatTooltip }),
    [paletteMode, valueRange, saturationCap, colorPalette, formatTooltip],
  )

  // Décision D3 : une case sans mesure se nomme, sans que l'appelant ait à le
  // demander — c'est ce qui permet aux quatre consommateurs existants d'hériter
  // sans modification. Pas de légende quand aucune case n'est vide (comportement
  // historique inchangé pour ces séries-là).
  const hasEmptyCell = series.some((s) => s.datapoints.some((d) => d.value == null))

  return (
    <ChartCard
      title={title}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={buildOption}
      legend={
        hasEmptyCell ? (
          <p className="text-xs text-muted-foreground" data-testid="heatmap-empty-cell-legend">
            {HEATMAP_EMPTY_CELL_TEXT[locale]}
          </p>
        ) : undefined
      }
    />
  )
}

interface BuildOpts {
  paletteMode?: HeatmapPaletteMode
  valueRange?: [number, number]
  /** Plafond de saturation optionnel du visualMap (`max`) — cf. doc de la prop du composant. */
  saturationCap?: number
  /** Palette d'accessibilité active — pilote la rampe CVD-safe (cf. heatmapColors). */
  colorPalette?: ColorPalette
  formatTooltip?: (point: ChartPointHeatmap) => string
}

/**
 * Tuple brut d'une case : `[xIndex, yIndex, value, detail?]`. `value` vaut `'-'`
 * pour une case vide (convention ECharts « pas de donnée », cf. commentaire de
 * `buildCellData`).
 */
type HeatCellTuple = [number, number, number | string, Record<string, unknown> | undefined]

/**
 * Une case peut être un tuple brut (cas historique, cases mesurées) ou un objet
 * `{ value, itemStyle }` quand elle porte un style PROPRE à elle seule — c'est le
 * cas d'une case vide, qui reçoit une hachure que les cases voisines n'ont pas.
 * ECharts accepte les deux formes dans le même tableau `data` (il lit `data.value`
 * quand l'élément n'est pas un tableau).
 */
type HeatCellDatum = HeatCellTuple | { value: HeatCellTuple; itemStyle: Record<string, unknown> }

/** Lit le tuple `[x, y, value, detail]` d'une case, quelle que soit sa forme. */
function heatCellTuple(d: HeatCellDatum): HeatCellTuple {
  return Array.isArray(d) ? d : d.value
}

/**
 * Style d'une case VIDE (décision D3) : fond neutre (au lieu d'invisible/transparent)
 * + hachure diagonale dans l'encre des axes. Même patron de décal que
 * `features/match-view/MatchSummaryCharts.tsx` (`DECAL_HATCH`), mais en tokens de
 * thème plutôt qu'en rgba en dur, car ce composant n'a pas la même exemption —
 * `getEChartsThemeColors()` résout déjà les deux teintes nécessaires.
 */
function emptyCellItemStyle(tc: EChartsThemeColors) {
  return {
    color: tc.splitLine,
    decal: {
      symbol: 'rect',
      symbolSize: 0.8,
      dashArrayX: [1, 0],
      dashArrayY: [4, 4],
      rotation: -Math.PI / 4,
      color: tc.axisLabel,
    },
  }
}

/**
 * Construit le tableau `data` ECharts à partir des datapoints. Une case VIDE
 * (`value: null`) part en tuple `'-'` (convention ECharts « pas de donnée » — la
 * valeur ne classe pas dans le visualMap) MAIS, contrairement à une case mesurée,
 * porte un `itemStyle` d'objet propre : c'est ce qui la rend visible (décision D3)
 * sans changer le rendu des cases mesurées, qui restent de simples tuples.
 */
function buildCellData(dps: ChartPointHeatmap[], xs: string[], ys: string[], tc: EChartsThemeColors): HeatCellDatum[] {
  return dps.map((d) => {
    const tuple: HeatCellTuple = [xs.indexOf(d.x), ys.indexOf(d.y), d.value ?? '-', d.detail]
    return d.value == null ? { value: tuple, itemStyle: emptyCellItemStyle(tc) } : tuple
  })
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildHeatmap2DOption(
  series: ChartSeries<ChartPointHeatmap>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { paletteMode = 'sequential', valueRange, saturationCap, colorPalette = 'default' } = opts
  const { formatTooltip } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints

  // Axes X / Y déduits des datapoints (preserve l'ordre d'apparition).
  const xs: string[] = []
  const ys: string[] = []
  const xSet = new Set<string>()
  const ySet = new Set<string>()
  for (const d of dps) {
    if (!xSet.has(d.x)) {
      xSet.add(d.x)
      xs.push(d.x)
    }
    if (!ySet.has(d.y)) {
      ySet.add(d.y)
      ys.push(d.y)
    }
  }

  const tc = getEChartsThemeColors()
  const data = buildCellData(dps, xs, ys, tc)

  const remplies = dps.filter((d): d is ChartPointHeatmap & { value: number } => d.value != null)
  const valeurs = remplies.map((d) => d.value)
  const minV = valueRange?.[0] ?? (valeurs.length > 0 ? Math.min(...valeurs) : 0)
  // Plafond de saturation : ignoré si valueRange fixe déjà le max. Absent des deux
  // → comportement historique (max = valeur réelle la plus haute).
  const maxV = valueRange?.[1] ?? saturationCap ?? (valeurs.length > 0 ? Math.max(...valeurs) : 0)

  // Rampe centralisée : en palette d'accessibilité, une heatmap séquentielle
  // bascule sur la rampe de fréquence (luminance monotone, CVD-safe).
  const colors = heatmapRampTokens(paletteMode, colorPalette).map(resolveToken)

  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 80, left: 96, right: 24 },
    // Requis pour qu'ECharts applique les `itemStyle.decal` posés à la main
    // ci-dessus (même flag que `MatchSummaryCharts.ARIA_DECAL`).
    aria: { decal: { show: true } },
    tooltip: {
      ...getTooltipBase(tc),
      position: 'top',
      formatter: (params: { data: HeatCellDatum }) => {
        const [xi, yi, brut, detail] = heatCellTuple(params.data)
        // Case vide : rien à dire, pas même « 0 ».
        if (typeof brut !== 'number') return ''
        const v = brut
        if (formatTooltip) {
          return formatTooltip({ x: xs[xi], y: ys[yi], value: v, detail })
        }
        const count = detail?.count ?? 0
        return `${escapeHtml(ys[yi])} × ${escapeHtml(xs[xi])}<br/>Win Rate: <b>${(v * 100).toFixed(1)}%</b><br/>Matchs: <b>${count}</b>`
      },
    },
    xAxis: { ...axis, type: 'category', data: xs, splitArea: { show: true } },
    yAxis: { ...axis, type: 'category', data: ys, splitArea: { show: true } },
    visualMap: {
      min: minV,
      max: maxV,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: 8,
      inRange: { color: colors },
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        name: main.key,
        type: 'heatmap',
        data,
        // Décision D2 : liseré = padding visuel entre cases. `borderColor` prend
        // l'encre du FOND DE CARTE (tc.card, cf. doc de `EChartsThemeColors.card`) :
        // ça détache chaque case de sa voisine sans introduire de couleur nouvelle.
        itemStyle: {
          borderWidth: CELL_BORDER_WIDTH,
          borderColor: tc.card,
          borderRadius: CELL_BORDER_RADIUS,
        },
        label: {
          show: true,
          formatter: (params: { data: HeatCellDatum }) => {
            const [, , brut, detail] = heatCellTuple(params.data)
            // Case vide : un tiret (décision D3) — un « 0 » se lirait comme une mesure.
            if (typeof brut !== 'number') return EMPTY_CELL_LABEL
            return String(detail?.count ?? 0)
          },
        },
        emphasis: {
          itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.5)' }, // color-allow: 2026-09-06 (revue R1, C5) — voile NEUTRE d ombre/fond d infobulle ECharts, pas une couleur de charte ; dette PREEXISTANTE au lot v2 D, a porter sur un token le jour ou un token de voile existera
        },
      },
    ],
  }
}
