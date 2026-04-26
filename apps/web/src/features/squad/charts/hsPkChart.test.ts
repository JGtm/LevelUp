/**
 * hsPkChart.test.ts — Vérifie que le builder ne crée plus de strings
 * libres : les libellés sont passés en argument et reportés tels quels
 * dans le payload Plotly.
 */
import { describe, it, expect } from 'vitest'
import { buildHsPkChart } from './hsPkChart'
import { SQUAD_HSPK_METRICS } from '../metrics'
import type { TeammateRow } from '@/lib/api/types'

const makeRow = (gamertag: string, hs: number, pk: number): TeammateRow => ({
  gamertag,
  xuid: 'x',
  encounter_count: 5,
  last_seen_at: null,
  with_kpis: {
    match_count: 5,
    wins: 3,
    kd_ratio: 1.2,
    win_rate: 0.6,
    accuracy: 0.4,
    kills_per_game: 10,
    assists_per_game: 4,
    headshot_kills_per_game: hs,
    perfect_kills_per_game: pk,
  },
  without_kpis: null,
})

describe('buildHsPkChart', () => {
  const args = {
    hsMetric: SQUAD_HSPK_METRICS.hs,
    pkMetric: SQUAD_HSPK_METRICS.pk,
    hsLabel: 'HS Label',
    pkLabel: 'PK Label',
    title: 'TITRE_TEST',
  }

  it('retourne null pour rows vide', () => {
    expect(buildHsPkChart({ rows: [], ...args })).toBeNull()
  })

  it('reporte le titre et les noms de traces fournis', () => {
    const fig = buildHsPkChart({
      rows: [makeRow('A', 3, 1), makeRow('B', 4, 0)],
      ...args,
    })
    expect(fig).not.toBeNull()
    expect(fig!.layout.title).toEqual({ text: 'TITRE_TEST', font: { size: 13 } })
    const names = fig!.data.map((t) => (t as { name?: string }).name)
    expect(names).toContain('HS Label')
    expect(names).toContain('PK Label')
  })

  it('aucun libellé hardcodé en français résiduel', () => {
    const fig = buildHsPkChart({
      rows: [makeRow('A', 3, 1)],
      ...args,
    })
    const json = JSON.stringify(fig)
    expect(json).not.toMatch(/Headshot kills\/partie/)
    expect(json).not.toMatch(/Perfect kills\/partie/)
  })

  it('utilise les extracteurs SquadMetric pour les valeurs', () => {
    const fig = buildHsPkChart({
      rows: [makeRow('A', 4, 2), makeRow('B', 0, 0)],
      ...args,
    })
    const hsTrace = fig!.data.find((t) => (t as { name?: string }).name === 'HS Label')
    const pkTrace = fig!.data.find((t) => (t as { name?: string }).name === 'PK Label')
    expect((hsTrace as { y: number[] }).y).toEqual([4, 0])
    expect((pkTrace as { y: number[] }).y).toEqual([2, 0])
  })
})
