/**
 * squadMapHeatmapChart — teammates.03 : « Performance par joueur × carte », rangement des
 * cellules du bloc en points de la grille canonique.
 *
 * Spec : .ai/charts_specs/teammates/03_squad_heatmap.yaml
 *
 * RENDU CANONIQUE (2026-09-20, lot 3 des ajustements pré-v7.5) : ce fichier n'écrit plus
 * d'option ECharts. `components/charts/Heatmap2DChart` rend la grille — mêmes cases aérées
 * et même infobulle que « Activité par jour et heure », le rendu de référence — et ce
 * module ne fait plus que ranger les cellules et calculer l'espacement des étiquettes de
 * carte.
 *
 * Accessibilité (CVD) : l'échelle reste ORDINALE discrète à 5 paliers (seuils 75/60/45/30),
 * encodée par les tokens sémantiques perf-tier-* (palette-aware et CVD-safe par
 * construction : axe bleu→jaune→vermillon en palette daltonienne, cf. palettes/okabe-ito)
 * ET chaque palier porte un label texte qui désambiguïse même à teintes proches. C'est le
 * mode `visualMapPieces` du wrapper : la migration ne change pas ce que les couleurs disent.
 */
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { SquadMapHeatmap, SquadMapHeatmapCell } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

/**
 * Combien d'étiquettes de carte la bande sous le graphe peut porter SANS que deux voisines
 * se chevauchent.
 *
 * MESURÉ, PAS DEVINÉ (capture du 2026-09-13, Escouade sur 55 sessions) : à 56 cartes les 56
 * étiquettes obliques se superposaient en un pâté illisible. Une étiquette tronquée
 * (`truncateMap`, 9 caractères + points de suspension) occupe ~45 px d'emprise horizontale
 * une fois tournée ; sur la largeur utile d'une carte pleine (~1 200 px) cela fait 18
 * étiquettes qui respirent. Au-delà de ce compte, on n'en écrit plus qu'UNE SUR K — la
 * numérotation « #N » de chaque étiquette dit combien de colonnes le saut a passées, et
 * l'infobulle de chaque case nomme toujours sa carte en entier.
 */
const MAX_ETIQUETTES_X = 18

/**
 * `axisLabel.interval` d'ECharts : nombre d'étiquettes SAUTÉES entre deux écrites (0 = toutes).
 * Exporté pour être vérifié hors rendu.
 */
export function xLabelInterval(mapCount: number): number {
  if (mapCount <= MAX_ETIQUETTES_X) return 0
  return Math.ceil(mapCount / MAX_ETIQUETTES_X) - 1
}

export interface SquadMapHeatmapOpts {
  mapLabelOf: (mapUI: string) => string
  pieceLabels: { tier1: string; tier2: string; tier3: string; tier4: string; tier5: string }
  noScoreLabel: string
  /** Nom de l'axe Y (« Joueur ») — localise par l'appelant. */
  yAxisName: string
}

/**
 * Les cellules du bloc rangées en points de la grille canonique.
 *
 * Étiquette de colonne « #N\nCarte » (format compact 2 lignes, comme les autres charts) ;
 * le nom COMPLET de la carte voyage dans `detail.mapName`, pour l'infobulle. Toutes les
 * cases du produit (joueur × carte) sont émises, une sans score mesuré valant `null` : les
 * axes du wrapper sont déduits de l'ordre d'apparition des points, en omettre une
 * décalerait les catégories.
 */
export function buildSquadMapHeatmapPoints(
  data: SquadMapHeatmap | null | undefined,
  mapLabelOf: (mapUI: string) => string,
): ChartPointHeatmap[] {
  const players = data?.players ?? []
  const mapsTopn = data?.maps_topn ?? []
  if (players.length === 0 || mapsTopn.length === 0) return []

  const mapNames = mapsTopn.map(mapLabelOf)
  const xLabels = mapNames.map((name, i) => `#${i + 1}\n${truncateMap(name)}`)

  const cellByKey = new Map<string, SquadMapHeatmapCell>()
  for (const c of data?.cells ?? []) {
    cellByKey.set(`${c.player}|${c.map_ui}`, c)
  }

  const points: ChartPointHeatmap[] = []
  for (let yi = 0; yi < players.length; yi += 1) {
    for (let xi = 0; xi < mapsTopn.length; xi += 1) {
      const c = cellByKey.get(`${players[yi]}|${mapsTopn[xi]}`)
      points.push({
        x: xLabels[xi],
        y: players[yi],
        value: c?.perf_avg !== undefined ? Number(c.perf_avg.toFixed(1)) : null,
        detail: { mapName: mapNames[xi], matchCount: c?.match_count ?? 0 },
      })
    }
  }
  return points
}
