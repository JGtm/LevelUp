/**
 * SquadMapHeatmapChart — « Performance par joueur x carte » (wrapper teammates.03).
 *
 * DEUX CHOSES MANQUAIENT ET ONT ETE AJOUTEES LE 2026-09-13 (retour utilisateur) :
 *
 *   1. LES AXES N'AVAIENT PAS DE NOM. Une grille de gamertags par cartes laisse deviner ce
 *      qui est en ligne et ce qui est en colonne ; ils sont desormais nommes (« Carte » /
 *      « Joueur »), par l'appelant, donc en FR comme en EN.
 *   2. LA LEGENDE SORTAIT DU CANVAS. La hauteur valait `joueurs x 60 + 160`, soit 280 px a
 *      deux joueurs — moins que la somme `grid.bottom` (132) + la reglette du visualMap +
 *      les etiquettes rotees. La reglette etait donc coupee. La hauteur part maintenant du
 *      SOCLE reel (ce qui n'est pas la grille) et n'ajoute que les lignes.
 *
 * ET LA LEGENDE EXISTE AUSSI EN DOM. Les cinq paliers `perf-tier-*` sont rendus en pied de
 * `ChartCard` : ils survivent a un canvas trop court, se lisent a la loupe du navigateur,
 * et sont du texte selectionnable. La reglette du canvas reste — c'est elle qui permet de
 * filtrer les paliers d'un clic.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { SquadMapHeatmap } from '@/lib/api/types'
import {
  buildSquadMapHeatmapOption,
  type SquadMapHeatmapOpts,
} from './charts/squadMapHeatmapChart'

/**
 * Ce que la carte doit loger SOUS la premiere ligne de la grille : `grid.bottom` (132 px,
 * qui couvre les etiquettes rotees et le nom d'axe X) plus la reglette du visualMap posee
 * a `bottom: 4`, plus le `grid.top`. Mesure sur pieces, pas devinee.
 */
const HAUTEUR_SOCLE = 200
/** Hauteur d'une ligne de la matrice (un joueur). */
const HAUTEUR_LIGNE = 60
/** Plafond : au-dela, la carte pousse tout le reste de la page hors de l'ecran. */
const HAUTEUR_MAX = 680

/**
 * Les cinq paliers de la legende, DU PLUS FAIBLE AU MEILLEUR — exactement l'ordre des
 * `pieces` du visualMap, qui vit juste au-dessus. Deux legendes de la meme echelle rangees
 * a l'envers l'une de l'autre se lisent comme deux echelles differentes.
 */
const PALIERS: { token: SemanticToken; cle: keyof SquadMapHeatmapOpts['pieceLabels'] }[] = [
  { token: 'perf-tier-5', cle: 'tier5' },
  { token: 'perf-tier-4', cle: 'tier4' },
  { token: 'perf-tier-3', cle: 'tier3' },
  { token: 'perf-tier-2', cle: 'tier2' },
  { token: 'perf-tier-1', cle: 'tier1' },
]

interface SquadMapHeatmapChartProps extends SquadMapHeatmapOpts {
  title?: string
  emptyMessage?: string
  data: SquadMapHeatmap | null | undefined
}

export function SquadMapHeatmapChart({ data, title, emptyMessage, ...opts }: SquadMapHeatmapChartProps) {
  const series = useMemo<ChartSeries<SquadMapHeatmap>[]>(
    () => (data ? [{ key: 'squad-map-heatmap', datapoints: [data] }] : []),
    [data],
  )
  const buildOption = useCallback(
    (s: ChartSeries<SquadMapHeatmap>[]) => buildSquadMapHeatmapOption(s, opts),
    [opts],
  )
  const playerCount = data?.players?.length ?? 0
  const height = Math.min(HAUTEUR_MAX, HAUTEUR_SOCLE + playerCount * HAUTEUR_LIGNE)

  const legendItems = useMemo(
    () =>
      PALIERS.map((p) => ({
        label: opts.pieceLabels[p.cle],
        color: resolveToken(p.token),
      })),
    [opts.pieceLabels],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={height}
      emptyMessage={emptyMessage}
      legend={<ChartLegend items={legendItems} />}
    />
  )
}
