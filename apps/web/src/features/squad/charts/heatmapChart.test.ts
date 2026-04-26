/**
 * heatmapChart.test.ts — Libellés en argument, plus aucune string FR
 * libre, et le commentaire TODO multi-title est conservé pour signaler
 * la migration future vers useAssetLabel('map', id).
 */
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { buildHeatmapChart } from './heatmapChart'
import type { MapBreakdownRow } from '@/lib/api/types'

const ROWS: MapBreakdownRow[] = [
  { map_ui: 'Aquarius', match_count: 4, win_rate: 75 },
  { map_ui: 'Live Fire', match_count: 6, win_rate: 50 },
]

describe('buildHeatmapChart', () => {
  const labels = {
    title: 'H_TITLE',
    winAxis: 'H_AXIS',
    matchesLabel: 'H_MATCHES',
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

  it('conserve le TODO multi-title pour la migration useAssetLabel', () => {
    // Le TODO dans le source est load-bearing : il documente la dette de
    // localisation des noms de cartes (raw string aujourd'hui, à migrer
    // vers useAssetLabel('map', id) en Phase 3 du PLAN_FINITION).
    const here = dirname(fileURLToPath(import.meta.url))
    const src = readFileSync(resolve(here, 'heatmapChart.ts'), 'utf-8')
    expect(src).toContain('TODO(multi-title)')
    expect(src).toContain('useAssetLabel')
  })
})
