/**
 * SquadMapHeatmapChart — « Performance par joueur x carte » (wrapper teammates.03).
 *
 * RENDU CANONIQUE (2026-09-20, lot 3 des ajustements pré-v7.5) : la grille passe par
 * `components/charts/Heatmap2DChart`, comme « Activité par jour et heure » qui est le rendu
 * de référence — cases aérées, infobulle du wrapper, plus aucune option ECharts écrite ici.
 * L'échelle reste DISCRÈTE (cinq paliers `perf-tier-*`) : c'est le mode `visualMapPieces`
 * du wrapper, ajouté pour cette migration, parce qu'une rampe continue aurait changé ce que
 * les couleurs disent.
 *
 * DEUX RÉGLAGES SURVIVENT À LA MIGRATION, et tous deux ont été mesurés le 2026-09-13 :
 *
 *   1. LE NOM DE L'AXE Y SE POSE EN TÊTE D'AXE, pas au milieu. Les étiquettes de cet axe
 *      sont des gamertags (jusqu'à une centaine de pixels) : `containLabel` réserve leur
 *      place sans réserver celle du NOM, qui au milieu tombait hors du canvas.
 *   2. L'AXE X N'A PLUS DE NOM (retour utilisateur du 2026-09-19) : « Carte » redisait ce
 *      que chaque étiquette montre déjà. Les étiquettes, elles, restent obliques et une
 *      sur K au-delà de 18 cartes, sans quoi elles se superposent en un pâté illisible.
 *
 * ET IL N'Y A QU'UNE LEGENDE : celle du DOM. Les cinq paliers `perf-tier-*` sont rendus en
 * pied de carte — ils survivent a un canvas trop court, se lisent a la loupe du navigateur,
 * et sont du texte selectionnable.
 */
import { useMemo } from 'react'
import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { escapeHtml } from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { SquadMapHeatmap } from '@/lib/api/types'
import {
  buildSquadMapHeatmapPoints,
  xLabelInterval,
  type SquadMapHeatmapOpts,
} from './charts/squadMapHeatmapChart'

/**
 * Ce que la carte doit loger hors des lignes : `grid.bottom` (104 px — etiquettes rotees)
 * plus `grid.top`. Mesure sur pieces, pas devinee.
 */
const HAUTEUR_SOCLE = 170
/** Hauteur d'une ligne de la matrice (un joueur). */
const HAUTEUR_LIGNE = 60
/** Plafond : au-dela, la carte pousse tout le reste de la page hors de l'ecran. */
const HAUTEUR_MAX = 680

/** Marges du trace : les etiquettes de carte (obliques, deux lignes) vivent en bas. */
const MARGES = { top: 30, bottom: 104, left: 8, right: 8, containLabel: true }
/** Inclinaison des etiquettes de carte, et leur decollement du trait d'axe. */
const ETIQUETTE_X = { rotate: -35, margin: 14 }

/**
 * Les cinq paliers de la legende, DU PLUS FAIBLE AU MEILLEUR — exactement l'ordre des
 * `pieces` du visualMap. Deux legendes de la meme echelle rangees a l'envers l'une de
 * l'autre se lisent comme deux echelles differentes.
 */
const PALIERS: { token: SemanticToken; cle: keyof SquadMapHeatmapOpts['pieceLabels'] }[] = [
  { token: 'perf-tier-5', cle: 'tier5' },
  { token: 'perf-tier-4', cle: 'tier4' },
  { token: 'perf-tier-3', cle: 'tier3' },
  { token: 'perf-tier-2', cle: 'tier2' },
  { token: 'perf-tier-1', cle: 'tier1' },
]

/** Seuils de l'echelle discrete (spec teammates.03) : 30 / 45 / 60 / 75. */
const SEUILS = [30, 45, 60, 75]

interface SquadMapHeatmapChartProps extends SquadMapHeatmapOpts {
  title?: string
  emptyMessage?: string
  data: SquadMapHeatmap | null | undefined
}

export function SquadMapHeatmapChart({
  data,
  title,
  emptyMessage,
  mapLabelOf,
  pieceLabels,
  noScoreLabel,
  yAxisName,
}: SquadMapHeatmapChartProps) {
  const points = useMemo(() => buildSquadMapHeatmapPoints(data, mapLabelOf), [data, mapLabelOf])
  const series = useMemo<ChartSeries<ChartPointHeatmap>[]>(
    () => (points.length > 0 ? [{ key: 'squad-map-heatmap', datapoints: points }] : []),
    [points],
  )

  const playerCount = data?.players?.length ?? 0
  const height = Math.min(HAUTEUR_MAX, HAUTEUR_SOCLE + playerCount * HAUTEUR_LIGNE)

  const pieces = useMemo(
    () => [
      { lt: SEUILS[0], color: resolveToken('perf-tier-5'), label: pieceLabels.tier5 },
      { gte: SEUILS[0], lt: SEUILS[1], color: resolveToken('perf-tier-4'), label: pieceLabels.tier4 },
      { gte: SEUILS[1], lt: SEUILS[2], color: resolveToken('perf-tier-3'), label: pieceLabels.tier3 },
      { gte: SEUILS[2], lt: SEUILS[3], color: resolveToken('perf-tier-2'), label: pieceLabels.tier2 },
      { gte: SEUILS[3], color: resolveToken('perf-tier-1'), label: pieceLabels.tier1 },
    ],
    [pieceLabels],
  )

  const legendItems = useMemo(
    () => PALIERS.map((p) => ({ label: pieceLabels[p.cle], color: resolveToken(p.token) })),
    [pieceLabels],
  )

  const formatTooltip = useMemo(
    () => (point: ChartPointHeatmap) => {
      const mapName = (point.detail?.mapName as string | undefined) ?? ''
      const n = (point.detail?.matchCount as number | undefined) ?? 0
      const perf = point.value == null ? noScoreLabel : point.value.toFixed(1)
      return `${escapeHtml(point.y)} — ${escapeHtml(mapName)}<br/>Perf: ${perf}<br/>N: ${n}`
    },
    [noScoreLabel],
  )

  const axisTuning = useMemo(
    () => ({
      xLabelRotate: ETIQUETTE_X.rotate,
      xLabelInterval: xLabelInterval(data?.maps_topn?.length ?? 0),
      xLabelMargin: ETIQUETTE_X.margin,
      yNameLocation: 'start' as const,
    }),
    [data?.maps_topn?.length],
  )

  return (
    <Heatmap2DChart
      title={title}
      series={series}
      emptyMessage={emptyMessage}
      height={height}
      visualMapPieces={pieces}
      showVisualMap={false}
      showCellLabel={false}
      formatTooltip={formatTooltip}
      yAxisInverse
      axisNames={{ y: yAxisName }}
      axisTuning={axisTuning}
      gridOverride={MARGES}
      emptyCells="hidden"
      legend={<ChartLegend items={legendItems} />}
    />
  )
}
