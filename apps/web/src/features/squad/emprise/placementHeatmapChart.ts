/**
 * placementHeatmapChart.ts — la série « cases » de la carte de chaleur « Distance au coéquipier le
 * plus proche et frags, par vie » : un rectangle par case non vide, posé sur les axes VALEUR du
 * graphe (X en portées de radar, Y en frags), de sorte que le repère du radar, la frontière
 * « au moins un frag » et les gros points des joueurs se superposent à leur position exacte.
 *
 * Série `custom` et non `heatmap` : le type `heatmap` d'ECharts demande deux axes catégoriels, qui
 * ne portent ni le trait du seuil d'isolement (une valeur quelconque du bloc, pas une borne de
 * case) ni les gros points à position continue.
 *
 * Teinte : UNE teinte de la rampe séquentielle neutre (`heatmapRampTokens('frequency')`, bleu
 * monotone en luminance, sûre en daltonisme), dont seule l'opacité monte avec l'intensité (racine
 * carrée de la part sur la part maximale) ; aucune couleur écrite ici.
 */
import { escapeHtml, hexToRgba } from '@/components/charts/_utils'
import { segmentInk } from '@/lib/accessibility/segmentInk'

import type { HeatCell, HeatGrid } from './placementHeatmap.logic'
import { cellShowsLabel } from './placementHeatmap.logic'
import type { PlacementText } from './placementStrings'

/** Z de la série : sous les repères (3) et les gros points (6). */
export const HEAT_Z = 1
/** Espace entre deux cases (px) : la grille se lit case par case. */
const CELL_GAP = 2
const CELL_RADIUS = 2
const LABEL_FONT_SIZE = 11
const LABEL_OUTLINE_WIDTH = 2

export interface HeatFormats {
  /** Un chiffre après la virgule au plus (« 12,5 »), séparateur de la langue. */
  pct: Intl.NumberFormat
  /** Deux décimales (« 0,25 »). */
  ratio: Intl.NumberFormat
}

/** L'infobulle d'une case : sa part, son nombre de vies, ses bornes. */
export function cellTip(cell: HeatCell, t: PlacementText, f: HeatFormats): string {
  const from = f.ratio.format(cell.x0)
  const distance = t.life.cellDistance(from, cell.openRight ? null : f.ratio.format(cell.x1))
  return [
    escapeHtml(t.life.cellHead(f.pct.format(cell.share * 100), cell.lives)),
    `${escapeHtml(distance)} · ${escapeHtml(t.life.cellKills(cell.row, cell.openTop))}`,
  ].join('<br>')
}

interface RenderApi {
  coord: (v: number[]) => number[]
}

/** La série ECharts des cases ; `fill` est la couleur résolue de la rampe, `grid` ce que `binLives` a compté. */
export function buildHeatCellsSeries(grid: HeatGrid, fill: string, t: PlacementText, f: HeatFormats): unknown {
  const { ink, outline } = segmentInk()
  const cells = grid.cells
  return {
    type: 'custom',
    z: HEAT_Z,
    silent: false,
    // Une case se lit par x0, x1 (portées) et la ligne (frags) ; la part vit dans `cells`.
    encode: { x: [0, 1], y: 2 },
    data: cells.map((cell) => ({
      value: [cell.x0, cell.x1, cell.row],
      tip: cellTip(cell, t, f),
    })),
    renderItem: (params: unknown, api: unknown) => {
      const cell = cells[(params as { dataIndex: number }).dataIndex]
      const a = api as RenderApi
      const [left, top] = a.coord([cell.x0, cell.row + 0.5])
      const [right, bottom] = a.coord([cell.x1, cell.row - 0.5])
      const width = right - left - CELL_GAP
      const height = bottom - top - CELL_GAP
      const children: unknown[] = [
        {
          type: 'rect',
          shape: { x: left + CELL_GAP / 2, y: top + CELL_GAP / 2, width, height, r: CELL_RADIUS },
          style: { fill: hexToRgba(fill, cell.intensity) },
        },
      ]
      if (cellShowsLabel(cell)) {
        children.push({
          type: 'text',
          style: {
            text: t.life.cellValue(Math.round(cell.share * 100)),
            x: left + (right - left) / 2,
            y: top + (bottom - top) / 2,
            textAlign: 'center',
            textVerticalAlign: 'middle',
            fontSize: LABEL_FONT_SIZE,
            fontWeight: 600,
            fill: ink,
            stroke: outline,
            lineWidth: LABEL_OUTLINE_WIDTH,
          },
        })
      }
      return { type: 'group', children }
    },
  }
}
