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
  /**
   * Encre du NOMBRE ecrit dans la case — une couleur deja resolue.
   *
   * POURQUOI L'APPELANT ET PAS LE WRAPPER. Seul l'appelant sait quelle rampe il a demandee,
   * donc quelle encre s'y lit sur toute sa longueur : la rampe de frequence part d'un bleu
   * tres sombre, ou l'encre par defaut d'ECharts (grise) disparait. Absent = couleur par
   * defaut, le comportement de tous les appelants anterieurs.
   */
  cellLabelColor?: string
  /**
   * Rend le `visualMap` ECharts (la reglette de rampe). Default `true` — le comportement de
   * tous les appelants anterieurs. Le MAPPING des couleurs reste actif dans les deux cas :
   * seule la reglette disparait.
   *
   * `false` quand l'appelant pose SA PROPRE legende de rampe en DOM (mots compris :
   * « 0 … N echanges »), que la reglette redirait sans les mots. Deux legendes pour la meme
   * echelle, c'est une de trop.
   */
  showVisualMap?: boolean
  /**
   * Axe Y INVERSE : la premiere categorie rencontree occupe le HAUT et non le bas
   * (`yAxis.inverse` d'ECharts). Absent = comportement historique.
   *
   * Sans cette option, un calendrier jour x heure devait emettre ses points a l'envers
   * (Dimanche -> Lundi) pour que Lundi finisse en haut : l'ordre des DONNEES portait une
   * decision d'AFFICHAGE, et la lecture du builder ne disait plus le sens de la semaine.
   */
  yAxisInverse?: boolean
  /**
   * Titres des deux axes (`axis.name`). Absents = axes sans titre (historique). Un
   * calendrier en a besoin : « Heure » et « Jour » ne se devinent pas d'une rangee de
   * nombres a deux chiffres.
   */
  axisNames?: { x?: string; y?: string }
  /**
   * Echelle de couleur VERTICALE, posee a droite du trace, au lieu de l'horizontale en
   * pied (defaut). Pour un trace large et peu haut (24 colonnes x 7 lignes), la barre
   * verticale longe la hauteur du graphe au lieu de lui voler une rangee.
   */
  visualMapOrient?: 'horizontal' | 'vertical'
  /** Formatage d'une graduation de l'echelle (ex. un taux 0..1 rendu en %). */
  visualMapFormatter?: (value: number) => string
  /** Libelles des deux extremites de l'echelle (`visualMap.text`, haut puis bas). */
  visualMapText?: [string, string]
  /**
   * Traitement d'une case SANS MESURE (`value: null`).
   *
   * `'hatched'` (defaut) applique la decision D3 : la case est peinte en neutre, hachuree,
   * porte un tiret et la legende la nomme — « l'absence a sa propre forme, jamais un vide ».
   *
   * `'hidden'` ne peint RIEN — ni la case, ni le BANDEAU D'AXE derriere elle
   * (`splitArea`, qui compose en damier et se voit la ou aucune case n'est peinte), ni le
   * tiret, ni la legende. Les quatre vont ensemble : dire « une case sans mesure ne se
   * montre pas » puis peindre un damier a sa place, c'est faire de l'absence la chose la
   * plus voyante du graphe.
   *
   * `'blank'` ne peint ni hachure, ni damier, ni legende d'absence — comme `'hidden'`,
   * mais SANS le sous-entendu « ces cases sont trop nombreuses pour se montrer ». C'est le
   * mode d'une case IMPOSSIBLE et rare : la diagonale de la matrice « Qui couvre qui »
   * (personne ne se venge soi-meme) n'est pas une mesure qui manque, et la nommer « aucune
   * mesure sur cet axe » annoncait un trou de donnees qui n'existe pas.
   *
   * CE MODE NE PEUT PAS PORTER DE TIRET, et c'est une limite d'ECharts, mesuree sur pieces
   * le 2026-09-13 : avec `visualMap.dimension: 2` actif, aucune etiquette n'est ecrite sur
   * une case dont la valeur n'est pas classable (`'-'`), peinte ou non. Seul `'hatched'`,
   * qui donne a la case un style propre ET la sort du mapping, garde son tiret.
   *
   * `'hidden'` est un opt-in reserve aux grilles ou les cases vides sont MAJORITAIRES et
   * REGULIERES : sur le calendrier jour x heure, les heures de nuit occupent la moitie du
   * trace toutes les semaines. Dans les trois modes la case reste EMISE (les axes de ce
   * wrapper sont deduits de l'ordre d'apparition des points : en omettre une decale les
   * categories) — seul son rendu change.
   */
  emptyCells?: 'hatched' | 'hidden' | 'blank'
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
  cellLabelColor,
  showVisualMap,
  yAxisInverse,
  axisNames,
  visualMapOrient,
  visualMapFormatter,
  visualMapText,
  emptyCells,
}: Heatmap2DChartProps) {
  // Palette d'accessibilité active : pilote la rampe CVD-safe (rebuild via
  // useColorPaletteVersion dans ChartCard + ce sélecteur au changement de palette).
  const colorPalette = useSettingsDraftStore((s) => s.localUiPrefs.colorPalette)
  const locale = useAppShellStore((s) => s.locale)
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHeatmap>[]) =>
      buildHeatmap2DOption(s, {
        paletteMode,
        valueRange,
        saturationCap,
        colorPalette,
        formatTooltip,
        cellLabelColor,
        showVisualMap,
        yAxisInverse,
        axisNames,
        visualMapOrient,
        visualMapFormatter,
        visualMapText,
        emptyCells,
      }),
    [
      paletteMode,
      valueRange,
      saturationCap,
      colorPalette,
      formatTooltip,
      cellLabelColor,
      showVisualMap,
      yAxisInverse,
      axisNames,
      visualMapOrient,
      visualMapFormatter,
      visualMapText,
      emptyCells,
    ],
  )

  // Décision D3 : une case sans mesure se nomme, sans que l'appelant ait à le
  // demander — c'est ce qui permet aux quatre consommateurs existants d'hériter
  // sans modification. Pas de légende quand aucune case n'est vide (comportement
  // historique inchangé pour ces séries-là).
  // La légende NOMME la forme des cases vides : sans forme à nommer (`emptyCells: 'hidden'`),
  // elle annoncerait une absence que rien ne montre.
  // `emptyCells` est ici la PROP BRUTE : son defaut (`'hatched'`) n'est applique que dans le
  // builder. Comparer sans le rappeler aurait prive de legende tous les appelants qui ne
  // passent pas l'option — c'est-a-dire les quatre consommateurs historiques.
  const hasEmptyCell =
    (emptyCells ?? 'hatched') === 'hatched' &&
    series.some((s) => s.datapoints.some((d) => d.value == null))

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
  /** Cf. les props homonymes du composant — toutes absentes = rendu historique. */
  cellLabelColor?: string
  showVisualMap?: boolean
  yAxisInverse?: boolean
  axisNames?: { x?: string; y?: string }
  visualMapOrient?: 'horizontal' | 'vertical'
  visualMapFormatter?: (value: number) => string
  visualMapText?: [string, string]
  emptyCells?: 'hatched' | 'hidden' | 'blank'
}

/**
 * Titre d'axe pose AU MILIEU de son axe, a distance des graduations — la seule position qui
 * ne peut chevaucher ni la premiere ni la derniere etiquette. `undefined` quand l'appelant ne
 * passe pas de titre : l'axe reste exactement celui d'avant l'ajout de l'option.
 */
function axisNameOpts(name: string | undefined, gap: number) {
  if (!name) return {}
  return { name, nameLocation: 'middle' as const, nameGap: gap }
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

// eslint-disable-next-line react-refresh/only-export-components
export function buildHeatmap2DOption(
  series: ChartSeries<ChartPointHeatmap>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { paletteMode = 'sequential', valueRange, saturationCap, colorPalette = 'default' } = opts
  const { formatTooltip, cellLabelColor, yAxisInverse, axisNames } = opts
  const { visualMapFormatter, visualMapText, showVisualMap = true } = opts
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
    grid: layout.grid,
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
      ...axisNameOpts(axisNames?.x, 28),
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: ys,
      splitArea: { show: emptyCells === 'hatched' },
      ...axisNameOpts(axisNames?.y, 76),
      inverse: yAxisInverse,
    },
    visualMap: {
      show: showVisualMap,
      // `dimension: 2` EST OBLIGATOIRE, et son absence etait un bug muet : une case est le
      // tuple `[xIdx, yIdx, value, detail]`, et ECharts classe par defaut sur la DERNIERE
      // dimension — donc sur `detail`, un objet, que le visualMap ne sait pas ranger. Aucune
      // case ne recevait sa couleur : le trace n'etait plus qu'une grille de nombres, et
      // seuls les bandeaux d'axes (`splitArea`) donnaient l'illusion d'un remplissage.
      // La valeur mesuree est en dimension 2, et c'est elle que la rampe encode.
      dimension: 2,
      min: minV,
      max: maxV,
      calculable: true,
      ...layout.placement,
      inRange: { color: colors },
      textStyle: { color: tc.axisLabel, fontSize: 10 },
      formatter: visualMapFormatter,
      text: visualMapText,
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
          // MOTIF NEUTRE EXPLICITE : `aria.decal.show` (requis par la case hachurée)
          // demande sinon à ECharts de hachurer AUTOMATIQUEMENT toute série qui ne déclare
          // pas son motif — les cases MESURÉES comprises. Un motif transparent ne peint
          // rien et occupe la place que la hachure automatique aurait prise ; la case
          // hachurée, elle, pose son propre motif, qui prend le dessus.
          decal: DECAL_NEUTRE,
        },
        label: {
          show: true,
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
