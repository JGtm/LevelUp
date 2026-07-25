import { describe, it, expect, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'

import { substituteHpToken, useProvidesDamageTaken } from './effectiveHp'
import { useAppShellStore } from '@/stores/appShellStore'
import type { components } from '@/lib/api/generated'

type TitleSummary = components['schemas']['TitleSummary']

describe('substituteHpToken', () => {
  it('remplace toutes les occurrences du jeton {{HP}} par le barème', () => {
    expect(substituteHpToken('1 vie ({{HP}})', 115)).toBe('1 vie (115)')
    expect(substituteHpToken('≈ {{HP}} points · {{HP}} PV', 225)).toBe('≈ 225 points · 225 PV')
  })
  it('laisse le texte intact sans jeton', () => {
    expect(substituteHpToken('aucun jeton', 115)).toBe('aucun jeton')
  })
})

// useProvidesDamageTaken délègue désormais à useCapability('damage_taken') (source
// unique de masquage booléen). La Résistance défensive est neutralisée (N/A) pour
// les titres qui ne DÉCLARENT PAS la capability damage_taken (Halo 5), sans toucher
// Infinite (qui la déclare). Fail-open : true tant que le titre courant est inconnu
// / le bootstrap non chargé (cf. useCapability).
describe('useProvidesDamageTaken', () => {
  const base = {
    name: '',
    status: 'active' as const,
    is_default: false,
    effective_hp_to_kill: 225,
    provides_damage_taken: true,
    provides_team_mmr: true,
    provides_max_killing_spree: true,
    offensive_conversion_p80: 0.9,
  }
  const title = (slug: string, hasDamageTaken: boolean): TitleSummary => ({
    ...base,
    slug,
    capabilities: hasDamageTaken ? ['damage_taken'] : [],
  })

  afterEach(() => {
    useAppShellStore.setState({ availableTitles: [], currentTitleSlug: 'halo_infinite' })
  })

  it('true quand le titre courant déclare la capability damage_taken (Infinite)', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_infinite', true)],
      currentTitleSlug: 'halo_infinite',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(true)
  })

  it('false quand le titre courant ne déclare pas damage_taken (Halo 5)', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_infinite', true), title('halo_5', false)],
      currentTitleSlug: 'halo_5',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(false)
  })

  it('fail-open true quand le slug courant est introuvable', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_5', false)],
      currentTitleSlug: 'inconnu',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(true)
  })
})
