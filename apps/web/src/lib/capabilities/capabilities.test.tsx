/**
 * Tests du socle de gating par capability (Phase 5 — pilier fonctionnel
 * multi-titre). Vérifie useCapability (présence/absence/fail-open) + FeatureGate
 * (rend / masque / fallback). Le titre courant + ses capabilities sont pilotés
 * via le store appShell (source = bootstrap.availableTitles[].capabilities).
 */
import { afterEach, describe, expect, it } from 'vitest'
import { render, renderHook, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'

import { FeatureGate } from './FeatureGate'
import { RouteCapabilityGate } from './RouteCapabilityGate'
import { hasCapabilityIn, useCapability, useCapabilityStrict, useTitleCapabilities } from './capabilities'

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true, effective_hp_to_kill: 225 },
    ],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('useCapability', () => {
  it('true quand le titre courant déclare la capability', () => {
    setTitleCaps(['firefight', 'ranked'])
    const { result } = renderHook(() => useCapability('firefight'))
    expect(result.current).toBe(true)
  })

  it('false quand la capability est absente du titre courant', () => {
    setTitleCaps(['ranked'])
    const { result } = renderHook(() => useCapability('firefight'))
    expect(result.current).toBe(false)
  })

  it('fail-open : true quand le titre courant est introuvable (bootstrap non chargé)', () => {
    useAppShellStore.setState({ currentTitleSlug: 'unknown', availableTitles: [] })
    const { result } = renderHook(() => useCapability('firefight'))
    expect(result.current).toBe(true)
  })

  it('expected_stats : gate les 3 charts « Écart au FDA attendu » (présent → affiché, absent → masqué)', () => {
    setTitleCaps(['expected_stats', 'ranked'])
    expect(renderHook(() => useCapability('expected_stats')).result.current).toBe(true)
    setTitleCaps(['ranked'])
    expect(renderHook(() => useCapability('expected_stats')).result.current).toBe(false)
  })
})

describe('useCapabilityStrict (fail-closed)', () => {
  it('false quand aucun titre n\'est chargé (availableTitles vide) — jamais fail-open', () => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_5', availableTitles: [] })
    const { result } = renderHook(() => useCapabilityStrict('spartan_customizer'))
    expect(result.current).toBe(false)
  })

  it('false quand le titre courant est introuvable (fenêtre transitoire de re-bootstrap)', () => {
    useAppShellStore.setState({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [
        { slug: 'halo_5', name: 'Halo 5', status: 'active', capabilities: ['spartan_customizer'], is_default: false, effective_hp_to_kill: 225 },
      ],
    })
    const { result } = renderHook(() => useCapabilityStrict('spartan_customizer'))
    expect(result.current).toBe(false)
  })

  it('false quand le titre résolu ne déclare pas la capability', () => {
    setTitleCaps(['ranked'])
    const { result } = renderHook(() => useCapabilityStrict('spartan_customizer'))
    expect(result.current).toBe(false)
  })

  it('true uniquement quand le titre résolu déclare la capability', () => {
    setTitleCaps(['spartan_customizer'])
    const { result } = renderHook(() => useCapabilityStrict('spartan_customizer'))
    expect(result.current).toBe(true)
  })
})

describe('FeatureGate', () => {
  it('rend les enfants si la capability est présente', () => {
    setTitleCaps(['media'])
    render(
      <FeatureGate capability="media">
        <span>contenu</span>
      </FeatureGate>,
    )
    expect(screen.getByText('contenu')).toBeInTheDocument()
  })

  it('masque (rend le fallback) si la capability est absente', () => {
    setTitleCaps(['ranked'])
    render(
      <FeatureGate capability="media" fallback={<span>indispo</span>}>
        <span>contenu</span>
      </FeatureGate>,
    )
    expect(screen.queryByText('contenu')).not.toBeInTheDocument()
    expect(screen.getByText('indispo')).toBeInTheDocument()
  })
})

describe('useTitleCapabilities / hasCapabilityIn', () => {
  it('retourne la liste des capabilities du titre courant', () => {
    setTitleCaps(['firefight', 'media'])
    const { result } = renderHook(() => useTitleCapabilities())
    expect(result.current).toEqual(['firefight', 'media'])
  })

  it('retourne null (fail-open) si le titre courant est introuvable', () => {
    useAppShellStore.setState({ currentTitleSlug: 'unknown', availableTitles: [] })
    const { result } = renderHook(() => useTitleCapabilities())
    expect(result.current).toBeNull()
  })

  it('hasCapabilityIn : présence/absence + fail-open sur null', () => {
    expect(hasCapabilityIn(['media', 'ranked'], 'media')).toBe(true)
    expect(hasCapabilityIn(['ranked'], 'media')).toBe(false)
    expect(hasCapabilityIn(null, 'media')).toBe(true)
  })
})

describe('RouteCapabilityGate', () => {
  it('rend la page si la capability est présente', () => {
    setTitleCaps(['career'])
    render(
      <RouteCapabilityGate capability="career">
        <span>page-carriere</span>
      </RouteCapabilityGate>,
    )
    expect(screen.getByText('page-carriere')).toBeInTheDocument()
  })

  it('rend le placeholder indisponible si la capability est absente', () => {
    setTitleCaps(['ranked'])
    render(
      <RouteCapabilityGate capability="career">
        <span>page-carriere</span>
      </RouteCapabilityGate>,
    )
    expect(screen.queryByText('page-carriere')).not.toBeInTheDocument()
    expect(screen.getByText('Indisponible pour ce titre')).toBeInTheDocument()
  })
})
