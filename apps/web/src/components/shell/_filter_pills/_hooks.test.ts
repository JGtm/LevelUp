/**
 * Tests des helpers de _hooks.ts — focus sur computePendingHash.
 *
 * Régression : computePendingHash sert à comparer l'état pending au hash
 * commité dans le store (`isDirty` dans FilterOmnibar/SquadLayout). Une
 * implémentation tronquée à 32 chars du base64 du JSON ne couvrait pas la
 * cascade (qui apparaît en fin de structure), donc toggler une option dans
 * FiltresPill ne marquait pas le formulaire comme dirty et React Query
 * ne refetchait pas le preview — toute la feature smart-filter-counts
 * paraissait inerte. Les tests ci-dessous interdisent ce retour en arrière.
 */
import { describe, expect, it } from 'vitest'
import { computePendingHash, DEFAULT_CASCADE, DEFAULT_PERIOD, DEFAULT_SESSIONS } from './_hooks'
import type { FilterContextInput } from '@/lib/api/types'

const baseCtx: FilterContextInput = {
  filter_mode: 'period',
  period: DEFAULT_PERIOD,
  sessions: DEFAULT_SESSIONS,
  cascade: DEFAULT_CASCADE,
}

describe('computePendingHash', () => {
  it('retourne la même valeur pour un même contexte', () => {
    expect(computePendingHash(baseCtx)).toBe(computePendingHash(baseCtx))
  })

  it('change quand experience_types change (cascade en fin de JSON)', () => {
    const before = computePendingHash(baseCtx)
    const after = computePendingHash({
      ...baseCtx,
      cascade: { ...DEFAULT_CASCADE, experience_types: ['PVE'] },
    })
    expect(after).not.toBe(before)
  })

  it('change quand playlists change', () => {
    const before = computePendingHash(baseCtx)
    const after = computePendingHash({
      ...baseCtx,
      cascade: { ...DEFAULT_CASCADE, playlists: ['Arène classée'] },
    })
    expect(after).not.toBe(before)
  })

  it('change quand modes change', () => {
    const before = computePendingHash(baseCtx)
    const after = computePendingHash({
      ...baseCtx,
      cascade: { ...DEFAULT_CASCADE, modes: ['Slayer'] },
    })
    expect(after).not.toBe(before)
  })

  it('change quand maps change', () => {
    const before = computePendingHash(baseCtx)
    const after = computePendingHash({
      ...baseCtx,
      cascade: { ...DEFAULT_CASCADE, maps: ['Recharge'] },
    })
    expect(after).not.toBe(before)
  })

  it('distingue deux cascades différentes (toggle d\'une option)', () => {
    const a = computePendingHash({
      ...baseCtx,
      cascade: { ...DEFAULT_CASCADE, experience_types: ['PVE'] },
    })
    const b = computePendingHash({
      ...baseCtx,
      cascade: { ...DEFAULT_CASCADE, experience_types: ['PVP non classé'] },
    })
    expect(a).not.toBe(b)
  })

  it('change quand period change', () => {
    const before = computePendingHash(baseCtx)
    const after = computePendingHash({
      ...baseCtx,
      period: { start_date: '2025-01-01', end_date: '2025-01-31' },
    })
    expect(after).not.toBe(before)
  })

  it('change quand picked_sessions change', () => {
    const before = computePendingHash(baseCtx)
    const after = computePendingHash({
      ...baseCtx,
      sessions: { ...DEFAULT_SESSIONS, picked_sessions: ['sess-1'] },
    })
    expect(after).not.toBe(before)
  })
})
