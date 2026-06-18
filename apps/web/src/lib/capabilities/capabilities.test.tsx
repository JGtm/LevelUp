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
import { useCapability } from './capabilities'

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true },
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
