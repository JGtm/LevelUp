/**
 * mapPerfVsHistoryChart.test.ts — grouped-bar Performance par carte (teammates.13).
 *
 * tokenCssVar stubbed pour retourner le nom du token comme valeur de couleur.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildMapPerfVsHistoryOption } from './mapPerfVsHistoryChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

function row(
  mapUI: string,
  perfSession: number | undefined,
  perfHistory: number | undefined,
  matchCount = 5,
): MapBreakdownRow {
  return {
    map_ui: mapUI,
    match_count: matchCount,
    win_rate: 0.5,
    performance_avg: perfSession,
    historical_performance_avg: perfHistory,
  }
}

const OPTS = {
  mapLabelOf: (m: string) => m.toUpperCase(),
  sessionLabel: 'Session',
  historyLabel: 'Historique',
}

function makeSeries(rows: MapBreakdownRow[]): ChartSeries<MapBreakdownRow>[] {
  return [{ key: 'map-perf-vs-history', datapoints: rows }]
}

type Datum = { value: number; itemStyle?: { color: string; opacity?: number } }
type SeriesShape = {
  name: string
  data: Datum[]
  markLine?: { data: Array<{ xAxis: number }> }
}

function findByName(opt: ReturnType<typeof buildMapPerfVsHistoryOption>, name: string): SeriesShape {
  const series = opt.series as SeriesShape[]
  const found = series.find((s) => s.name === name)
  if (!found) throw new Error(`série "${name}" introuvable`)
  return found
}

describe('buildMapPerfVsHistoryOption', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('série vide → option minimale', () => {
    const opt = buildMapPerfVsHistoryOption([], OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('aucune carte commune (perf manquante) → option minimale', () => {
    const rows = [row('a', 70, undefined), row('b', undefined, 50)]
    const opt = buildMapPerfVsHistoryOption(makeSeries(rows), OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('jointure inner : ne garde que les cartes avec session ET historique', () => {
    const rows = [
      row('a', 70, 60),
      row('b', 50, undefined), // exclue
      row('c', 80, 75),
    ]
    const opt = buildMapPerfVsHistoryOption(makeSeries(rows), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toEqual(['A', 'C'])
  })

  it('respecte l\'ordre reçu du backend, aucun re-tri par match_count — I12', () => {
    // Le backend (computeMapBreakdown) trie déjà par première apparition
    // chronologique ; matchCount distinct ne doit PAS réordonner l'affichage
    // (ancien comportement : tri par match_count desc).
    const rows = [row('few', 60, 60, 2), row('many', 60, 60, 20), row('mid', 60, 60, 8)]
    const opt = buildMapPerfVsHistoryOption(makeSeries(rows), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toEqual(['FEW', 'MANY', 'MID'])
  })

  it('cap à 20 cartes max : garde les 20 PLUS JOUÉES, affichées dans l\'ordre reçu — I12', () => {
    // 25 cartes reçues dans l'ordre chronologique (m0 = plus ancienne), avec un
    // match_count croissant avec l'index. Les 5 moins jouées (m0..m4) doivent
    // être exclues de la sélection top-20 ; les 20 retenues (m5..m24) restent
    // dans leur ordre chronologique d'arrivée (pas re-triées par match_count).
    const rows = Array.from({ length: 25 }, (_, i) => row(`m${i}`, i * 3, 50, i + 1))
    const opt = buildMapPerfVsHistoryOption(makeSeries(rows), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toHaveLength(20)
    expect(yAxis.data).toEqual(Array.from({ length: 20 }, (_, i) => `M${i + 5}`))
  })

  it('couleur session par palier perf (≥75 → perf-tier-1)', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 80, 60)]), OPTS)
    const datum = findByName(opt, 'Session').data[0]
    expect(datum.itemStyle?.color).toBe('color:perf-tier-1')
  })

  it('palier perf (≥60 → perf-tier-2)', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 65, 50)]), OPTS)
    expect(findByName(opt, 'Session').data[0].itemStyle?.color).toBe('color:perf-tier-2')
  })

  it('palier perf (≥45 → perf-tier-3)', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 50, 40)]), OPTS)
    expect(findByName(opt, 'Session').data[0].itemStyle?.color).toBe('color:perf-tier-3')
  })

  it('palier perf (≥30 → perf-tier-4)', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 35, 50)]), OPTS)
    expect(findByName(opt, 'Session').data[0].itemStyle?.color).toBe('color:perf-tier-4')
  })

  it('palier perf (<30 → perf-tier-5)', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 20, 40)]), OPTS)
    expect(findByName(opt, 'Session').data[0].itemStyle?.color).toBe('color:perf-tier-5')
  })

  it('série historique colorée en gris neutre rgba structurel', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 50, 60)]), OPTS)
    const datum = findByName(opt, 'Historique').data[0]
    expect(datum.itemStyle?.color).toBe('rgba(120, 120, 120, 0.45)')
  })

  it('valeurs arrondies à 1 décimale', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 62.345, 71.987)]), OPTS)
    expect(findByName(opt, 'Session').data[0].value).toBe(62.3)
    expect(findByName(opt, 'Historique').data[0].value).toBe(72.0)
  })

  it('yAxis inverse activé (1ère carte du tri en haut)', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 50, 60)]), OPTS)
    const yAxis = opt.yAxis as { inverse: boolean }
    expect(yAxis.inverse).toBe(true)
  })

  it('markLine de référence à xAxis: 0 sur la série historique', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('x', 50, 60)]), OPTS)
    const markLine = findByName(opt, 'Historique').markLine
    expect(markLine?.data?.[0]?.xAxis).toBe(0)
  })

  it('mapLabelOf appliqué sur les noms de cartes', () => {
    const opt = buildMapPerfVsHistoryOption(makeSeries([row('aquarius', 50, 60)]), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toContain('AQUARIUS')
  })
})
