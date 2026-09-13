/**
 * squadMapHeatmapChart.test.ts — teammates.03.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadMapHeatmapOption, xLabelInterval } from './squadMapHeatmapChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadMapHeatmap } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

const OPTS = {
  mapLabelOf: (m: string) => m.toUpperCase(),
  pieceLabels: { tier1: 'T1', tier2: 'T2', tier3: 'T3', tier4: 'T4', tier5: 'T5' },
  noScoreLabel: '-',
  xAxisName: 'Carte',
  yAxisName: 'Joueur',
}

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

function makeSeries(d: SquadMapHeatmap | null): ChartSeries<SquadMapHeatmap>[] {
  return d ? [{ key: 'k', datapoints: [d] }] : []
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadMapHeatmapOption', () => {
  it('série vide → option minimale', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(null), OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('aucun joueur ou aucune carte → option minimale', () => {
    const empty: SquadMapHeatmap = { players: [], maps_topn: [], cells: [] }
    expect(buildSquadMapHeatmapOption(makeSeries(empty), OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('xAxis = "#N\\nCarte" (mapLabelOf appliqué + numérotation)', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const xAxis = opt.xAxis as { data: string[] }
    expect(xAxis.data).toEqual(['#1\nAQUARIUS', '#2\nBAZAAR'])
  })

  it('yAxis = liste des joueurs avec inverse', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const yAxis = opt.yAxis as { data: string[]; inverse: boolean }
    expect(yAxis.data).toEqual(['Me', 'Friend1'])
    expect(yAxis.inverse).toBe(true)
  })

  // Les deux axes n'avaient AUCUN nom jusqu'au 2026-09-13 : une grille de gamertags par
  // cartes laissait deviner ce qui etait en ligne et ce qui etait en colonne.
  it('nomme les DEUX axes, avec le libelle fourni par l appelant (donc localise)', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const xAxis = opt.xAxis as { name: string; nameLocation: string }
    const yAxis = opt.yAxis as { name: string; nameLocation: string }
    expect(xAxis.name).toBe('Carte')
    expect(yAxis.name).toBe('Joueur')
    expect(xAxis.nameLocation).toBe('middle')
    // L'axe Y est INVERSE : son nom se pose en `start` (donc EN HAUT). En `middle` il
    // tombait hors du canvas — `containLabel` reserve la place des etiquettes, pas celle
    // du nom — et en `end` il retombait sur les etiquettes de cartes.
    expect(yAxis.nameLocation).toBe('start')
  })

  it('reserve sous la grille de quoi loger etiquettes rotees ET nom d axe X', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const grid = opt.grid as { bottom: number }
    expect(grid.bottom).toBeGreaterThanOrEqual(100)
  })

  it('MASQUE la reglette du visualMap sans couper le mapping des couleurs', () => {
    // Les cinq paliers sont deja nommes par la legende DOM du pied de carte : deux
    // rangees identiques sous le meme graphe, c'est une de trop. `show: false` ne coupe
    // que l'affichage du composant — les `pieces` restent, donc la teinte des cases aussi.
    const vm = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS).visualMap as {
      show: boolean
      pieces: unknown[]
    }
    expect(vm.show).toBe(false)
    expect(vm.pieces).toHaveLength(5)
  })

  it('data heatmap = matrice (xi, yi, value) avec value=null pour cellule sans score', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const series = opt.series as Array<{ data: Array<[number, number, number | null]> }>
    expect(series[0].data).toHaveLength(4)
    // Friend1 × Aquarius (xi=0, yi=1) doit être null.
    const cellNull = series[0].data.find((d) => d[0] === 0 && d[1] === 1)
    expect(cellNull?.[2]).toBeNull()
    // Me × Aquarius (xi=0, yi=0) doit valoir 80.
    const cellMe = series[0].data.find((d) => d[0] === 0 && d[1] === 0)
    expect(cellMe?.[2]).toBe(80)
  })

  it('visualMap discret 5 paliers avec tokens perf-tier', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const vm = opt.visualMap as { type: string; pieces: Array<{ color: string; label: string }> }
    expect(vm.type).toBe('piecewise')
    expect(vm.pieces).toHaveLength(5)
    expect(vm.pieces[0].color).toBe('color:perf-tier-5') // <30
    expect(vm.pieces[4].color).toBe('color:perf-tier-1') // ≥75
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

  it('l option porte l intervalle calcule (et pas 0 en dur)', () => {
    const beaucoup: SquadMapHeatmap = {
      players: ['Me'],
      maps_topn: Array.from({ length: 56 }, (_, i) => `Carte${i + 1}`),
      cells: [],
    }
    const opt = buildSquadMapHeatmapOption(makeSeries(beaucoup), OPTS)
    const xAxis = opt.xAxis as { axisLabel: { interval: number; rotate: number } }
    expect(xAxis.axisLabel.interval).toBe(3)
    // La rotation ne change pas : a 12 cartes l oblique reste le plus lisible.
    expect(xAxis.axisLabel.rotate).toBe(-35)
  })
})
