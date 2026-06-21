/**
 * Tests unitaires — résolution title-aware des groupes LUSR.
 */
import { describe, it, expect } from 'vitest'
import {
  knownLusrGroupsForTitle,
  resolveLusrGroupsForDisplay,
  lusrChainLabel,
} from './lusr-chains'

describe('knownLusrGroupsForTitle', () => {
  it('Halo Infinite → 4 groupes connus dans l\'ordre déclaré', () => {
    expect(knownLusrGroupsForTitle('halo_infinite')).toEqual([
      'arena_slayer',
      'arena_objectif',
      'btb',
      'chaos',
    ])
  })

  it('titre inconnu (halo_5) → aucun groupe connu', () => {
    expect(knownLusrGroupsForTitle('halo_5')).toEqual([])
  })
})

describe('resolveLusrGroupsForDisplay', () => {
  it('HINF : data ⊆ connus → exactement les 4 connus dans l\'ordre (no-op)', () => {
    expect(resolveLusrGroupsForDisplay('halo_infinite', ['arena_slayer'])).toEqual([
      'arena_slayer',
      'arena_objectif',
      'btb',
      'chaos',
    ])
  })

  it('HINF : groupe data hors connus → ajouté APRÈS les connus', () => {
    expect(
      resolveLusrGroupsForDisplay('halo_infinite', ['chaos', 'mystere']),
    ).toEqual(['arena_slayer', 'arena_objectif', 'btb', 'chaos', 'mystere'])
  })

  it('halo_5 : aucun connu → uniquement les groupes data', () => {
    expect(resolveLusrGroupsForDisplay('halo_5', ['h5_arena'])).toEqual(['h5_arena'])
  })

  it('groupes data supplémentaires triés alpha pour stabilité', () => {
    expect(resolveLusrGroupsForDisplay('halo_5', ['zeta', 'alpha'])).toEqual([
      'alpha',
      'zeta',
    ])
  })
})

describe('lusrChainLabel', () => {
  it('h5_arena → libellé i18n FR « Arène »', () => {
    expect(lusrChainLabel('h5_arena', 'fr')).toBe('Arène')
  })

  it('h5_arena → libellé i18n EN « Arena »', () => {
    expect(lusrChainLabel('h5_arena', 'en')).toBe('Arena')
  })
})
