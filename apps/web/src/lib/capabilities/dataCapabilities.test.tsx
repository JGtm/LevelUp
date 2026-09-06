/**
 * Tests du gating DATA-LEVEL (règle des deux portes, 2026-09-05 — registre L1/L3/L4/L5).
 *
 * Ce que ces tests verrouillent, et que rien ne verrouillait avant : le front CONSOMME
 * enfin `GET /titles/{slug}/capabilities`, et un titre qui ne déclare pas une clé `film.*`
 * ne voit pas la surface correspondante — au lieu d'un état vide qui promettait une donnée
 * qui n'arriverait jamais.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, renderHook, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'

import { api } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

import { FeatureGate } from './FeatureGate'
import { hasDataCapabilityIn, useDataCapability } from './dataCapabilities'

vi.mock('@/lib/api/client', () => ({
  api: { get: vi.fn() },
  getApiTitleSlug: () => 'halo_infinite',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

const apiGet = vi.mocked(api.get)

/** Le wrapper porte SON QueryClient : chaque test repart d'un cache vide. */
function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

/** Sert la reponse de l'endpoint pour le titre courant. */
function servirCapabilities(caps: Record<string, string>) {
  useAppShellStore.setState({ currentTitleSlug: 'titre_test' })
  apiGet.mockResolvedValue({
    title_slug: 'titre_test',
    schema_version: 1,
    capabilities: caps,
  })
}

beforeEach(() => {
  apiGet.mockReset()
})

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('hasDataCapabilityIn (prédicat pur)', () => {
  it('supported et degraded valent OUI ; not_exposed et l\'absence valent NON', () => {
    expect(hasDataCapabilityIn({ 'film.kill_positions': 'supported' }, 'film.kill_positions')).toBe(true)
    expect(hasDataCapabilityIn({ 'film.kill_positions': 'degraded' }, 'film.kill_positions')).toBe(true)
    expect(hasDataCapabilityIn({ 'film.kill_positions': 'not_exposed' }, 'film.kill_positions')).toBe(false)
    expect(hasDataCapabilityIn({}, 'film.kill_positions')).toBe(false)
  })

  it('fail-open tant que les capabilities ne sont pas connues (null)', () => {
    expect(hasDataCapabilityIn(null, 'film.kill_positions')).toBe(true)
  })
})

describe('useDataCapability', () => {
  it('true quand le titre déclare la clé (supported)', async () => {
    servirCapabilities({ 'film.kill_positions': 'supported' })
    const { result } = renderHook(() => useDataCapability('film.kill_positions'), { wrapper })
    await waitFor(() => expect(result.current).toBe(true))
    expect(apiGet).toHaveBeenCalledWith('/titles/titre_test/capabilities')
  })

  it('false quand le titre la déclare not_exposed', async () => {
    servirCapabilities({ 'film.kill_positions': 'not_exposed' })
    const { result } = renderHook(() => useDataCapability('film.kill_positions'), { wrapper })
    await waitFor(() => expect(result.current).toBe(false))
  })

  it('false quand la clé est absente de la réponse', async () => {
    servirCapabilities({ 'match.history': 'supported' })
    const { result } = renderHook(() => useDataCapability('film.kill_positions'), { wrapper })
    await waitFor(() => expect(result.current).toBe(false))
  })

  it('fail-open quand la requête échoue : une panne de cet endpoint n\'ampute pas l\'app', async () => {
    useAppShellStore.setState({ currentTitleSlug: 'titre_test' })
    apiGet.mockRejectedValue(new Error('500'))
    const { result } = renderHook(() => useDataCapability('film.kill_positions'), { wrapper })
    await waitFor(() => expect(apiGet).toHaveBeenCalled())
    expect(result.current).toBe(true)
  })
})

describe('FeatureGate — les deux portes dans un seul composant', () => {
  function poserTitre(caps: string[]) {
    useAppShellStore.setState({
      currentTitleSlug: 'titre_test',
      availableTitles: [
        {
          slug: 'titre_test', name: 'Test', status: 'active', capabilities: caps, is_default: true,
          effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true,
          provides_max_killing_spree: true, offensive_conversion_p80: 0.9,
          defensive_resistance_p80: 1.65,
        },
      ],
    })
  }

  it('porte data-level seule : masque quand la clé est not_exposed', async () => {
    servirCapabilities({ 'film.kill_positions': 'not_exposed' })
    render(
      <FeatureGate dataCapability="film.kill_positions" fallback={<span>indispo</span>}>
        <span>contenu</span>
      </FeatureGate>,
      { wrapper },
    )
    await waitFor(() => expect(screen.getByText('indispo')).toBeInTheDocument())
    expect(screen.queryByText('contenu')).not.toBeInTheDocument()
  })

  it('LES DEUX portes : il faut les deux — la title-level absente suffit à masquer', async () => {
    poserTitre(['ranked'])
    apiGet.mockResolvedValue({
      title_slug: 'titre_test', schema_version: 1,
      capabilities: { 'film.kill_positions': 'supported' },
    })
    render(
      <FeatureGate capability="replay" dataCapability="film.kill_positions">
        <span>contenu</span>
      </FeatureGate>,
      { wrapper },
    )
    await waitFor(() => expect(apiGet).toHaveBeenCalled())
    expect(screen.queryByText('contenu')).not.toBeInTheDocument()
  })

  it('LES DEUX portes ouvertes : le contenu est rendu', async () => {
    poserTitre(['replay'])
    apiGet.mockResolvedValue({
      title_slug: 'titre_test', schema_version: 1,
      capabilities: { 'film.kill_positions': 'supported' },
    })
    render(
      <FeatureGate capability="replay" dataCapability="film.kill_positions">
        <span>contenu</span>
      </FeatureGate>,
      { wrapper },
    )
    await waitFor(() => expect(screen.getByText('contenu')).toBeInTheDocument())
  })

  it('sans aucune porte : rend ses enfants (un gate qui ne garde rien ne masque rien)', () => {
    render(
      <FeatureGate>
        <span>contenu</span>
      </FeatureGate>,
      { wrapper },
    )
    expect(screen.getByText('contenu')).toBeInTheDocument()
  })
})
