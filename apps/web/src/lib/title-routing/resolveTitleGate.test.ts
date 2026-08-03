/**
 * Tests — resolveTitleGate (module title-routing, D-6/D-11).
 *
 * Fonction PURE : projette un slug de titre d'URL contre la liste bootstrap →
 * verdict wait | valid | unknown | coming_soon | archived.
 */
import { describe, it, expect } from 'vitest'
import { resolveTitleGate } from './resolveTitleGate'
import type { TitleSummary } from '@/lib/api/types'

function title(slug: string, status?: TitleSummary['status']): TitleSummary {
  return {
    slug,
    name: slug,
    status: status as TitleSummary['status'],
    capabilities: [],
    is_default: false,
    effective_hp_to_kill: 0,
    provides_damage_taken: true,
    provides_team_mmr: true,
    provides_max_killing_spree: true,
    offensive_conversion_p80: 0.9,
    defensive_resistance_p80: 1.65,
  } as TitleSummary
}

const TITLES: TitleSummary[] = [
  title('halo_infinite', 'active'),
  title('halo_5', 'coming_soon'),
  title('halo_mcc', 'archived'),
]

describe('resolveTitleGate', () => {
  it('wait tant que le store n’est pas bootstrappé (quel que soit le slug)', () => {
    expect(resolveTitleGate('halo_infinite', [], false)).toBe('wait')
    expect(resolveTitleGate('inconnu', TITLES, false)).toBe('wait')
  })

  it('valid pour un titre actif présent', () => {
    expect(resolveTitleGate('halo_infinite', TITLES, true)).toBe('valid')
  })

  it('unknown pour un slug absent de la liste', () => {
    expect(resolveTitleGate('halo_wars', TITLES, true)).toBe('unknown')
  })

  it('coming_soon remonté tel quel', () => {
    expect(resolveTitleGate('halo_5', TITLES, true)).toBe('coming_soon')
  })

  it('archived remonté tel quel', () => {
    expect(resolveTitleGate('halo_mcc', TITLES, true)).toBe('archived')
  })

  it('status absent traité comme active → valid (parité buildTitleSwitcherEntries)', () => {
    const noStatus = [
      { slug: 'halo_x', name: 'X', capabilities: [], is_default: false, effective_hp_to_kill: 0, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
    ] as unknown as TitleSummary[]
    expect(resolveTitleGate('halo_x', noStatus, true)).toBe('valid')
  })
})
