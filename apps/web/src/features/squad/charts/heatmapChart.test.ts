/**
 * heatmapChart.test.ts — Libellés en argument, plus aucune string FR
 * libre, et `mapLabelOf` résout les IDs bruts de cartes vers leur libellé
 * localisé (assets.map du titre courant).
 */
import { describe, it, expect, vi } from 'vitest'
import { buildHeatmapChart } from './heatmapChart'
import type { MapBreakdownRow } from '@/lib/api/types'

const ROWS: MapBreakdownRow[] = [
  { map_ui: 'Aquarius', match_count: 4, win_rate: 75 },
  { map_ui: 'Live Fire', match_count: 6, win_rate: 50 },
]

const identityMapLabel = (id: string) => id

describe('buildHeatmapChart', () => {
  const labels = {
    title: 'H_TITLE',
    winAxis: 'H_AXIS',
    matchesLabel: 'H_MATCHES',
    mapLabelOf: identityMapLabel,
  }

  it('retourne null pour rows vide', () => {
    expect(buildHeatmapChart({ rows: [], ...labels })).toBeNull()
  })

  it('reporte les libellés fournis dans le layout et hovertemplate', () => {
    const fig = buildHeatmapChart({ rows: ROWS, ...labels })
    expect(fig!.layout.title).toEqual({ text: 'H_TITLE', font: { size: 13 } })
    const trace = fig!.data[0] as {
      y: string[]
      hovertemplate: string
      colorbar?: { title?: string }
    }
    expect(trace.y).toEqual(['H_AXIS'])
    expect(trace.hovertemplate).toContain('H_AXIS')
    expect(trace.hovertemplate).toContain('H_MATCHES')
    expect(trace.colorbar?.title).toBe('H_AXIS')
  })

  it('trie les cartes par win_rate décroissant', () => {
    const fig = buildHeatmapChart({ rows: ROWS, ...labels })
    const trace = fig!.data[0] as { x: string[]; z: number[][] }
    expect(trace.x).toEqual(['Aquarius', 'Live Fire'])
    expect(trace.z[0]).toEqual([75, 50])
  })

  it('aucun libellé hardcodé en français résiduel', () => {
    const fig = buildHeatmapChart({ rows: ROWS, ...labels })
    const json = JSON.stringify(fig)
    expect(json).not.toMatch(/Win rate par carte/)
    expect(json).not.toMatch(/Win %/)
    expect(json).not.toMatch(/Win rate \(%\)/)
    expect(json).not.toMatch(/Matchs:/)
  })

  it('applique mapLabelOf pour chaque carte (axe x localisé)', () => {
    const mapLabelOf = vi.fn((id: string) => `LOC_${id}`)
    const fig = buildHeatmapChart({ rows: ROWS, ...labels, mapLabelOf })
    const trace = fig!.data[0] as { x: string[] }
    expect(trace.x).toEqual(['LOC_Aquarius', 'LOC_Live Fire'])
    expect(mapLabelOf).toHaveBeenCalledWith('Aquarius')
    expect(mapLabelOf).toHaveBeenCalledWith('Live Fire')
  })

  it('mapLabelOf qui retourne l\'ID brut équivaut à pas de localisation (fallback)', () => {
    const fig = buildHeatmapChart({
      rows: ROWS,
      ...labels,
      mapLabelOf: identityMapLabel,
    })
    const trace = fig!.data[0] as { x: string[] }
    expect(trace.x).toEqual(['Aquarius', 'Live Fire'])
  })
})
