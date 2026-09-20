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
import { useCallback, type ReactNode } from 'react'

import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import {
  buildHeatmap2DOption,
  type ChartPointHeatmap,
  type HeatmapAxisTuning,
  type HeatmapGridOverride,
  type HeatmapPaletteMode,
  type HeatmapPiece,
} from './heatmap2DOption'

// Le builder et ses types vivent dans `heatmap2DOption.ts` (seuil de 500 lignes) et sont
// RÉEXPORTÉS ici : aucun appelant ne change d'import. La règle de Fast Refresh ne sait pas
// distinguer une réexportation d'une définition — c'est la même dérogation qu'avant la
// coupe du fichier.
/* eslint-disable react-refresh/only-export-components */
export { buildHeatmap2DOption } from './heatmap2DOption'
export type {
  ChartPointHeatmap,
  HeatmapPaletteMode,
  HeatmapAxisTuning,
  HeatmapGridOverride,
  HeatmapPiece,
} from './heatmap2DOption'
/* eslint-enable react-refresh/only-export-components */

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
  /**
   * Échelle DISCRÈTE à paliers nommés au lieu de la rampe continue. Absente = rampe
   * continue, le comportement de tous les appelants antérieurs.
   */
  visualMapPieces?: HeatmapPiece[]
  /** Écrit le nombre DANS la case. Default `true` (rendu historique). */
  showCellLabel?: boolean
  /** Réglages d'étiquettes et de titre d'axe (cf. `HeatmapAxisTuning`). */
  axisTuning?: HeatmapAxisTuning
  /** Marges de trace imposées, fusionnées par-dessus celles du wrapper. */
  gridOverride?: HeatmapGridOverride
  /**
   * Légende POSÉE EN DOM sous le graphe, à la place de celle que le wrapper écrit pour
   * les cases vides. Un appelant dont l'échelle est discrète nomme ses paliers lui-même :
   * deux légendes sous le même graphe, c'en est une de trop.
   */
  legend?: ReactNode
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
  visualMapPieces,
  showCellLabel,
  axisTuning,
  gridOverride,
  legend,
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
        visualMapPieces,
        showCellLabel,
        axisTuning,
        gridOverride,
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
      visualMapPieces,
      showCellLabel,
      axisTuning,
      gridOverride,
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
        legend ?? (hasEmptyCell ? (
          <p className="text-xs text-muted-foreground" data-testid="heatmap-empty-cell-legend">
            {HEATMAP_EMPTY_CELL_TEXT[locale]}
          </p>
        ) : undefined)
      }
    />
  )
}
