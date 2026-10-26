/**
 * Tests — données du graphe « Évolution LUSR / CSR » : placements écartés, une
 * échelle à la fois.
 */
import { describe, it, expect } from 'vitest'
import type { CareerLusrCheckpoint } from '@/lib/api/types'
import {
  availableScales,
  chartableCheckpoints,
  checkpointsForScale,
  resolveScale,
} from './lusrEvolution.logic'

function cp(over: Partial<CareerLusrCheckpoint>): CareerLusrCheckpoint {
  return {
    recorded_at: '2026-03-01T20:00:00Z',
    rating_type: 'LUSR',
    rating_value: 1500,
    tier_label: 'Or III',
    playlist_group: 'social_slayer',
    playlist_name: 'Quick Play',
    ...over,
  }
}

describe('chartableCheckpoints', () => {
  it('écarte les matchs de placement (valeur stockée 0) et les points non datés', () => {
    const kept = chartableCheckpoints([
      cp({ rating_type: 'CSR', rating_value: 1475, tier_label: 'Diamant VI' }),
      cp({ rating_type: 'CSR', rating_value: 0, tier_label: 'Placement (4 restants)' }),
      cp({ rating_type: 'CSR', rating_value: 0, tier_label: 'Placement' }),
      cp({ recorded_at: null }),
    ])
    expect(kept.map(k => k.rating_value)).toEqual([1475])
  })
})

describe('échelle affichée', () => {
  const mixed = [cp({ rating_type: 'CSR', rating_value: 1600 }), cp({ rating_type: 'LUSR' })]

  it('ordre stable LUSR puis CSR, une seule échelle si une seule présente', () => {
    expect(availableScales(mixed)).toEqual(['lusr', 'csr'])
    expect(availableScales([cp({ rating_type: 'csr' })])).toEqual(['csr'])
    expect(availableScales([])).toEqual([])
  })

  it('la sélection prime ; absente des données → première échelle présente', () => {
    expect(resolveScale('csr', ['lusr', 'csr'])).toBe('csr')
    expect(resolveScale(null, ['lusr', 'csr'])).toBe('lusr')
    expect(resolveScale('csr', ['lusr'])).toBe('lusr')
    expect(resolveScale(null, [])).toBeNull()
  })

  it('ne mélange jamais les deux échelles', () => {
    expect(checkpointsForScale(mixed, 'csr').map(c => c.rating_type)).toEqual(['CSR'])
    expect(checkpointsForScale(mixed, 'lusr').map(c => c.rating_type)).toEqual(['LUSR'])
    expect(checkpointsForScale(mixed, null)).toEqual([])
  })
})
