/**
 * heatmap2DOption — construction de l'option ECharts de la grille canonique.
 *
 * Extrait de `Heatmap2DChart.tsx` le 2026-09-20 (lot 3 des ajustements pré-v7.5) : le
 * fichier du composant depassait le seuil de 500 lignes du depot, et la migration des
 * trois dernieres grilles locales (Relations, Explorer, Escouade) y ajoutait ses options.
 * Le composant reexporte `buildHeatmap2DOption` : aucun appelant ne change d'import.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import type { ColorPalette } from '@/stores/settingsDraftStore'

import type { ChartSeries } from './ChartCard'
import { heatmapRampTokens } from './heatmapColors'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
  type EChartsThemeColors,
} from './_utils'


/** Caractère affiché SUR une case vide (décision D3 : « un tiret »), au lieu de rien. */
const EMPTY_CELL_LABEL = '—'

/**
 * Motif de hachure INVISIBLE, posé au niveau de la SÉRIE.
 *
 * `aria.decal.show` est requis pour qu'ECharts applique le `decal` de la case hachurée —
 * mais il hachure AUSSI, automatiquement, toute série qui n'en déclare pas, les cases
 * MESURÉES comprises. Un motif transparent ne peint rien et occupe la place que la hachure
 * automatique aurait prise ; la case hachurée, elle, pose son propre motif, qui prend le
 * dessus.
 */
const DECAL_NEUTRE = { color: 'transparent' }

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

/**
 * Un palier d'une échelle DISCRÈTE (`visualMap.pieces` d'ECharts). Ajouté le 2026-09-20
 * pour la grille « Performance par joueur × carte », dont l'échelle n'est pas une rampe
 * continue mais cinq paliers nommés : sans ce mode, la migrer vers le wrapper canonique
 * aurait changé ce que ses couleurs DISENT.
 */
export interface HeatmapPiece {
  lt?: number
  gte?: number
  color: string
  label: string
}

/**
 * Réglages d'axe propres à un consommateur. Tous absents = le rendu historique du
 * wrapper. Ils existent parce que deux grilles migrées ont des étiquettes que le réglage
 * par défaut ne sait pas loger : des noms de carte (obliques, une sur K) et des gamertags
 * de largeur imprévisible (nom d'axe posé en tête, pas au milieu).
 */
export interface HeatmapAxisTuning {
  /** Rotation des étiquettes de l'axe X, en degrés. */
  xLabelRotate?: number
  /** Étiquettes X SAUTÉES entre deux écrites (`axisLabel.interval`, 0 = toutes). */
  xLabelInterval?: number
  /** Écart entre les étiquettes X et le trait d'axe. */
  xLabelMargin?: number
  /**
   * Position du titre de l'axe Y. `'start'` le pose EN TÊTE d'axe : sur un axe inversé
   * dont les étiquettes sont des gamertags, `'middle'` tombe hors du canvas.
   */
  yNameLocation?: 'start' | 'middle'
}

/**
 * Marges de trace imposées par l'appelant, fusionnées PAR-DESSUS celles que le wrapper
 * calcule. `containLabel` laisse ECharts réserver la place des étiquettes lui-même —
 * indispensable quand leur largeur n'est pas connue à l'avance.
 */
export interface HeatmapGridOverride {
  top?: number
  bottom?: number
  left?: number
  right?: number
  containLabel?: boolean
}

export interface BuildOpts {
  paletteMode?: HeatmapPaletteMode
  valueRange?: [number, number]
  /** Plafond de saturation optionnel du visualMap (`max`) — cf. doc de la prop du composant. */
  saturationCap?: number
  /** Palette d'accessibilité active — pilote la rampe CVD-safe (cf. heatmapColors). */
  colorPalette?: ColorPalette
  formatTooltip?: (point: ChartPointHeatmap) => string
  /** Cf. les props homonymes du composant — toutes absentes = rendu historique. */
  cellLabelColor?: string
  showVisualMap?: boolean
  yAxisInverse?: boolean
  axisNames?: { x?: string; y?: string }
  visualMapOrient?: 'horizontal' | 'vertical'
  visualMapFormatter?: (value: number) => string
  visualMapText?: [string, string]
  emptyCells?: 'hatched' | 'hidden' | 'blank'
  /** Échelle DISCRÈTE à paliers au lieu de la rampe continue (cf. `HeatmapPiece`). */
  visualMapPieces?: HeatmapPiece[]
  /**
   * Écrit le nombre DANS la case. Default `true` — le rendu de tous les appelants
   * antérieurs. `false` pour une grille dont la case est trop étroite pour un nombre : la
   * couleur y porte seule la mesure, et l'infobulle la nomme.
   */
  showCellLabel?: boolean
  /** Réglages d'étiquettes et de titre d'axe (cf. `HeatmapAxisTuning`). */
  axisTuning?: HeatmapAxisTuning
  /** Marges de trace imposées, fusionnées par-dessus celles du wrapper. */
  gridOverride?: HeatmapGridOverride
}

/**
 * Titre d'axe pose AU MILIEU de son axe, a distance des graduations — la seule position qui
 * ne peut chevaucher ni la premiere ni la derniere etiquette. `undefined` quand l'appelant ne
 * passe pas de titre : l'axe reste exactement celui d'avant l'ajout de l'option.
 */
function axisNameOpts(
  name: string | undefined,
  gap: number,
  location: 'start' | 'middle' = 'middle',
) {
  if (!name) return {}
  // En TÊTE d'axe, le titre ne longe plus les graduations : son écart devient un simple
  // décollement (12 px), pas la hauteur d'une bande d'étiquettes.
  const nameGap = location === 'start' ? 12 : gap
  return { name, nameLocation: location, nameGap }
}

/**
 * Position de l'echelle de couleur ET marge du trace, qui vont ensemble : une echelle
 * verticale a droite a besoin de la place que l'horizontale en pied prenait en bas — et une
 * echelle MASQUEE rend sa bande au trace (une bande vide de 56 px sous une matrice de trois
 * lignes se voit autant qu'une reglette).
 */
function visualMapLayout(orient: 'horizontal' | 'vertical', shown: boolean) {
  if (orient === 'vertical') {
    // `left`/`bottom` plus genereux que l'horizontale : ils logent les TITRES d'axes, poses
    // au milieu de leur axe (cf. `axisNameOpts`) — sans cette marge, le titre se superpose
    // aux graduations.
    return {
      grid: { top: 32, bottom: 56, left: 116, right: shown ? 130 : 24 },
      placement: { orient, right: 30, top: 'center', itemWidth: 12, itemHeight: 140 },
    }
  }
  return {
    grid: { top: 24, bottom: shown ? 80 : 24, left: 96, right: 24 },
    placement: { orient, left: 'center', bottom: 8 },
  }
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
function buildCellData(
  dps: ChartPointHeatmap[],
  xs: string[],
  ys: string[],
  tc: EChartsThemeColors,
  emptyCells: 'hatched' | 'hidden' | 'blank',
): HeatCellDatum[] {
  return dps.map((d) => {
    const tuple: HeatCellTuple = [xs.indexOf(d.x), ys.indexOf(d.y), d.value ?? '-', d.detail]
    // Seul `hatched` donne une FORME a la case vide ; `hidden` et `blank` la laissent au
    // tuple nu, donc sans peinture propre.
    if (d.value != null || emptyCells !== 'hatched') return tuple
    return { value: tuple, itemStyle: emptyCellItemStyle(tc) }
  })
}

/** Ce que le visualMap doit savoir pour se construire, continu comme discret. */
interface VisualMapInput {
  showVisualMap: boolean
  pieces?: HeatmapPiece[]
  minV: number
  maxV: number
  placement: Record<string, unknown>
  colors: string[]
  axisLabelColor: string
  formatter?: (value: number) => string
  text?: [string, string]
}

/**
 * Le composant `visualMap` de l'option — extrait du builder pour le tenir sous le seuil de
 * 80 lignes du dépôt après l'ajout de l'échelle discrète.
 *
 * `dimension: 2` EST OBLIGATOIRE dans les deux modes, et son absence etait un bug muet :
 * une case est le tuple `[xIdx, yIdx, value, detail]`, et ECharts classe par defaut sur la
 * DERNIERE dimension — donc sur `detail`, un objet, que le visualMap ne sait pas ranger.
 * Aucune case ne recevait sa couleur : le trace n'etait plus qu'une grille de nombres, et
 * seuls les bandeaux d'axes (`splitArea`) donnaient l'illusion d'un remplissage.
 */
function heatVisualMap(v: VisualMapInput): Record<string, unknown> {
  const commun = {
    show: v.showVisualMap,
    dimension: 2,
    ...v.placement,
    textStyle: { color: v.axisLabelColor, fontSize: 10 },
  }
  if (v.pieces && v.pieces.length > 0) {
    return { ...commun, type: 'piecewise', pieces: v.pieces }
  }
  return {
    ...commun,
    min: v.minV,
    max: v.maxV,
    calculable: true,
    inRange: { color: v.colors },
    formatter: v.formatter,
    text: v.text,
  }
}

export function buildHeatmap2DOption(
  series: ChartSeries<ChartPointHeatmap>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { paletteMode = 'sequential', valueRange, saturationCap, colorPalette = 'default' } = opts
  const { formatTooltip, cellLabelColor, yAxisInverse, axisNames } = opts
  const { visualMapFormatter, visualMapText, showVisualMap = true } = opts
  const { visualMapPieces, showCellLabel = true, axisTuning, gridOverride } = opts
  const emptyCells = opts.emptyCells ?? 'hatched'
  const layout = visualMapLayout(opts.visualMapOrient ?? 'horizontal', showVisualMap)
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
  const data = buildCellData(dps, xs, ys, tc, emptyCells)

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
    // Marges de l'appelant PAR-DESSUS celles du wrapper : une grille dont les étiquettes
    // ont une largeur imprévisible (gamertags, noms de carte obliques) ne peut pas se
    // contenter des marges fixes calculées ici.
    grid: { ...layout.grid, ...gridOverride },
    // Requis pour qu'ECharts applique les `itemStyle.decal` poses a la main ci-dessus (meme
    // flag que `MatchSummaryCharts.ARIA_DECAL`). Inutile hors du mode `hatched` : plus
    // aucune hachure a rendre.
    aria: { decal: { show: emptyCells === 'hatched' } },
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
    xAxis: {
      ...axis,
      type: 'category',
      data: xs,
      splitArea: { show: emptyCells === 'hatched' },
      axisLabel: {
        ...axis.axisLabel,
        ...(axisTuning?.xLabelRotate != null ? { rotate: axisTuning.xLabelRotate } : {}),
        ...(axisTuning?.xLabelInterval != null ? { interval: axisTuning.xLabelInterval } : {}),
        ...(axisTuning?.xLabelMargin != null ? { margin: axisTuning.xLabelMargin } : {}),
      },
      ...axisNameOpts(axisNames?.x, 28),
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: ys,
      splitArea: { show: emptyCells === 'hatched' },
      ...axisNameOpts(axisNames?.y, 76, axisTuning?.yNameLocation),
      inverse: yAxisInverse,
    },
    visualMap: heatVisualMap({
      showVisualMap,
      pieces: visualMapPieces,
      minV,
      maxV,
      placement: layout.placement,
      colors,
      axisLabelColor: tc.axisLabel,
      formatter: visualMapFormatter,
      text: visualMapText,
    }),
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
          // MOTIF NEUTRE EXPLICITE : `aria.decal.show` (requis par la case hachurée)
          // demande sinon à ECharts de hachurer AUTOMATIQUEMENT toute série qui ne déclare
          // pas son motif — les cases MESURÉES comprises. Un motif transparent ne peint
          // rien et occupe la place que la hachure automatique aurait prise ; la case
          // hachurée, elle, pose son propre motif, qui prend le dessus.
          decal: DECAL_NEUTRE,
        },
        label: {
          // Une case trop étroite pour un nombre ne l'écrit pas : la couleur porte seule
          // la mesure, et l'infobulle la nomme.
          show: showCellLabel,
          formatter: (params: { data: HeatCellDatum }) => {
            const [, , brut, detail] = heatCellTuple(params.data)
            // Case vide : un tiret (décision D3) — un « 0 » se lirait comme une mesure. En
            // mode `hidden`, rien : la case n'a pas de forme, elle n'a pas d'étiquette.
            if (typeof brut !== 'number') return emptyCells === 'hatched' ? EMPTY_CELL_LABEL : ''
            return String(detail?.count ?? 0)
          },
          // Encre de l'étiquette, fournie par l'appelant : lui seul sait quelle rampe il a
          // demandée, donc quelle encre s'y lit. Une CHAÎNE, jamais une fonction — un
          // `label.color` fonctionnel fait disparaître TOUS les nombres de la heatmap
          // (mesuré sur capture le 2026-09-13, deux fois de suite).
          ...(cellLabelColor ? { color: cellLabelColor } : {}),
        },
        emphasis: {
          itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.5)' }, // color-allow: 2026-09-06 (revue R1, C5) — voile NEUTRE d ombre/fond d infobulle ECharts, pas une couleur de charte ; dette PREEXISTANTE au lot v2 D, a porter sur un token le jour ou un token de voile existera
        },
      },
    ],
  }
}
