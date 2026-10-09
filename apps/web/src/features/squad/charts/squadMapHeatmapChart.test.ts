/**
 * squadMapHeatmapChart.test.ts — teammates.03.
 *
 * Depuis la migration vers la grille canonique (2026-09-20), ce module ne produit plus
 * d'option ECharts : il RANGE les cellules en points. Les tests suivent — ils vérifient le
 * rangement et l'espacement des étiquettes de carte, pas la forme d'une option que le
 * wrapper écrit désormais (et que `components/charts/Heatmap2DChart.test.ts` couvre).
 */
import { describe, it, expect } from 'vitest'
import { buildSquadMapHeatmapPoints, xLabelInterval } from './squadMapHeatmapChart'
import type { SquadMapHeatmap } from '@/lib/api/types'

const mapLabelOf = (m: string) => m.toUpperCase()

function makeData(): SquadMapHeatmap {
  return {
    players: ['Me', 'Friend1'],
    maps_topn: ['Aquarius', 'Bazaar'],
    cells: [
      { player: 'Me', map_ui: 'Aquarius', perf_avg: 80, match_count: 4 },
      { player: 'Me', map_ui: 'Bazaar', perf_avg: 45, match_count: 2 },
      { player: 'Friend1', map_ui: 'Aquarius', perf_avg: undefined, match_count: 0 },
      { player: 'Friend1', map_ui: 'Bazaar', perf_avg: 60, match_count: 1 },
    ],
  }
}

describe('buildSquadMapHeatmapPoints', () => {
  it('donnée absente → aucun point', () => {
    expect(buildSquadMapHeatmapPoints(null, mapLabelOf)).toEqual([])
    expect(buildSquadMapHeatmapPoints(undefined, mapLabelOf)).toEqual([])
  })

  it('aucun joueur ou aucune carte → aucun point', () => {
    const empty: SquadMapHeatmap = { players: [], maps_topn: [], cells: [] }
    expect(buildSquadMapHeatmapPoints(empty, mapLabelOf)).toEqual([])
  })

  it('colonne = "#N\\nCarte" (mapLabelOf appliqué + numérotation)', () => {
    const points = buildSquadMapHeatmapPoints(makeData(), mapLabelOf)
    expect([...new Set(points.map((p) => p.x))]).toEqual(['#1\nAQUARIUS', '#2\nBAZAAR'])
  })

  it('ligne = joueur, dans l’ordre du bloc (l’axe inversé du wrapper met le premier en haut)', () => {
    const points = buildSquadMapHeatmapPoints(makeData(), mapLabelOf)
    expect([...new Set(points.map((p) => p.y))]).toEqual(['Me', 'Friend1'])
  })

  it('toutes les cases sont émises, une sans score mesuré valant null', () => {
    const points = buildSquadMapHeatmapPoints(makeData(), mapLabelOf)
    expect(points).toHaveLength(4)
    const sansScore = points.find((p) => p.y === 'Friend1' && p.x.includes('AQUARIUS'))
    expect(sansScore?.value).toBeNull()
    const avecScore = points.find((p) => p.y === 'Me' && p.x.includes('AQUARIUS'))
    expect(avecScore?.value).toBe(80)
  })

  it('le nom COMPLET de la carte et le nombre de matchs voyagent pour l’infobulle', () => {
    const points = buildSquadMapHeatmapPoints(makeData(), mapLabelOf)
    const p = points.find((pt) => pt.y === 'Me' && pt.x.includes('AQUARIUS'))
    expect(p?.detail).toEqual({ mapName: 'AQUARIUS', matchCount: 4 })
  })
})

describe('etiquettes de cartes — lisibilite a 56 cartes (finitions 2026-09-13)', () => {
  it('peu de cartes : toutes les cartes sont etiquetees', () => {
    expect(xLabelInterval(2)).toBe(0)
    expect(xLabelInterval(12)).toBe(0)
    expect(xLabelInterval(18)).toBe(0)
  })

  it('beaucoup de cartes : une etiquette sur K, jamais plus de 18 ecrites', () => {
    expect(xLabelInterval(19)).toBe(1)
    expect(xLabelInterval(56)).toBe(3)
    for (const n of [19, 24, 36, 56, 120]) {
      const ecrites = Math.ceil(n / (xLabelInterval(n) + 1))
      expect(ecrites).toBeLessThanOrEqual(18)
    }
  })
})
