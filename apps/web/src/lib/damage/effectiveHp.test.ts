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

// Mirror de useEffectiveHpToKill : lecture title-aware depuis appShellStore, défaut
// Infinite (provides_damage_taken ?? true). Garantit que la Résistance défensive est
// neutralisée (N/A) pour les titres sans damage_taken (Halo 5) sans toucher Infinite.
describe('useProvidesDamageTaken', () => {
  const base = {
    name: '',
    status: 'active' as const,
    capabilities: [],
    is_default: false,
    effective_hp_to_kill: 225,
  }
  const title = (slug: string, provides?: boolean): TitleSummary => ({
    ...base,
    slug,
    ...(provides === undefined ? {} : { provides_damage_taken: provides }),
  })

  afterEach(() => {
    useAppShellStore.setState({ availableTitles: [], currentTitleSlug: 'halo_infinite' })
  })

  it('défaut true quand le titre courant omet provides_damage_taken (Infinite inchangé)', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_infinite')],
      currentTitleSlug: 'halo_infinite',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(true)
  })

  it('false quand le titre courant déclare provides_damage_taken=false (Halo 5)', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_infinite'), title('halo_5', false)],
      currentTitleSlug: 'halo_5',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(false)
  })

  it('true quand le titre courant déclare provides_damage_taken=true', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_infinite', true)],
      currentTitleSlug: 'halo_infinite',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(true)
  })

  it('défaut true quand le slug courant est introuvable', () => {
    useAppShellStore.setState({
      availableTitles: [title('halo_5', false)],
      currentTitleSlug: 'inconnu',
    })
    const { result } = renderHook(() => useProvidesDamageTaken())
    expect(result.current).toBe(true)
  })
})
