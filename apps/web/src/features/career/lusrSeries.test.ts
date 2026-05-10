import { describe, expect, it } from 'vitest'
import { buildLusrSeries, normalizeGroup } from './lusrSeries'
import type { CareerLusrCheckpoint } from '@/lib/api/types'

function cp(overrides: Partial<CareerLusrCheckpoint>): CareerLusrCheckpoint {
  return {
    recorded_at: '2026-04-01T12:00:00Z',
    rating_type: 'LUSR',
    rating_value: 1500,
    tier_label: null,
    playlist_group: 'arena',
    playlist_name: 'Partie rapide',
    ...overrides,
  }
}

describe('normalizeGroup', () => {
  it('mappe social (legacy) sur arena', () => {
    expect(normalizeGroup('social')).toBe('arena')
  })
  it('mappe valeur inconnue ou vide sur arena', () => {
    expect(normalizeGroup(null)).toBe('arena')
    expect(normalizeGroup('')).toBe('arena')
    expect(normalizeGroup('weird')).toBe('arena')
  })
  it('préserve les 4 groupes canoniques', () => {
    expect(normalizeGroup('arena')).toBe('arena')
    expect(normalizeGroup('btb')).toBe('btb')
    expect(normalizeGroup('fun')).toBe('fun')
    expect(normalizeGroup('ranked')).toBe('ranked')
  })
})

describe('buildLusrSeries', () => {
  it('fusionne arena + social (legacy) sous une seule courbe "Arène"', () => {
    const series = buildLusrSeries(
      [
        cp({ playlist_group: 'arena',  playlist_name: 'Partie rapide', recorded_at: '2026-04-01' }),
        cp({ playlist_group: 'social', playlist_name: 'Partie rapide', recorded_at: '2026-04-02' }),
      ],
      'fr',
    )
    expect(series).toHaveLength(1)
    const meta = series[0].meta as { label: string; groupKey: string }
    expect(meta.label).toBe('Arène (LUSR)')
    expect(meta.groupKey).toBe('arena')
  })

  it('produit 1 courbe par playlist_group (arena, btb, fun, ranked)', () => {
    const series = buildLusrSeries(
      [
        cp({ playlist_group: 'arena',  playlist_name: 'Partie rapide' }),
        cp({ playlist_group: 'btb',    playlist_name: 'Big Team Battle' }),
        cp({ playlist_group: 'fun',    playlist_name: 'Super Fiesta' }),
        cp({ playlist_group: 'ranked', playlist_name: 'Ranked Slayer' }),
      ],
      'fr',
    )
    const labels = series.map((s) => (s.meta as { label: string }).label).sort()
    expect(labels).toEqual([
      'Arène (LUSR)',
      'Classé (LUSR)',
      'Grand combat (LUSR)',
      'Social (LUSR)',
    ])
  })

  it('filtre les checkpoints dont playlist_name est un UUID brut', () => {
    const uuid1 = 'a446725e-b281-414c-a21e-31b8700e95a1'
    const uuid2 = 'bdceefb3-1c52-4848-a6b7-d49acd13109d'
    const series = buildLusrSeries(
      [
        cp({ playlist_group: 'arena', playlist_name: uuid1 }),
        cp({ playlist_group: 'arena', playlist_name: uuid2 }),
        cp({ playlist_group: 'arena', playlist_name: 'Partie rapide' }),
      ],
      'fr',
    )
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toHaveLength(1)
  })

  it('produit des libellés EN quand locale=en', () => {
    const series = buildLusrSeries(
      [
        cp({ playlist_group: 'arena', playlist_name: 'Quickplay' }),
        cp({ playlist_group: 'fun',   playlist_name: 'Super Fiesta' }),
        cp({ playlist_group: 'btb',   playlist_name: 'Big Team Battle' }),
        cp({ playlist_group: 'ranked', playlist_name: 'Ranked Arena' }),
      ],
      'en',
    )
    const labels = series.map((s) => (s.meta as { label: string }).label).sort()
    expect(labels).toEqual([
      'Arena (LUSR)',
      'Big Team Battle (LUSR)',
      'Ranked (LUSR)',
      'Social (LUSR)',
    ])
  })

  it('ignore les checkpoints sans recorded_at', () => {
    const series = buildLusrSeries([cp({ recorded_at: null })], 'fr')
    expect(series).toHaveLength(0)
  })

  it('sépare LUSR et CSR sur le même groupe', () => {
    const series = buildLusrSeries(
      [
        cp({ rating_type: 'LUSR', playlist_group: 'ranked', playlist_name: 'Ranked Arena' }),
        cp({ rating_type: 'CSR',  playlist_group: 'ranked', playlist_name: 'Ranked Arena' }),
      ],
      'fr',
    )
    expect(series).toHaveLength(2)
    const labels = series.map((s) => (s.meta as { label: string }).label).sort()
    expect(labels).toEqual(['Classé (CSR)', 'Classé (LUSR)'])
  })
})
