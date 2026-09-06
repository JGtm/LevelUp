/**
 * Tests du gating DATA-LEVEL (règle des deux portes, 2026-09-05 — registre L1/L3/L4/L5).
 *
 * Ce que ces tests verrouillent, et que rien ne verrouillait avant : le front CONSOMME
 * enfin `GET /titles/{slug}/capabilities`, et un titre qui ne déclare pas une clé `film.*`
 * ne voit pas la surface correspondante — au lieu d'un état vide qui promettait une donnée
 * qui n'arriverait jamais.
 *
 * Les portes elles-mêmes se testent où elles vivent (`MatchKillDistanceSection.test.tsx`,
 * `queries.gate.test.tsx`) : ici, le socle seul.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'

import { api } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

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
  const connu = (caps: Record<string, 'supported' | 'degraded' | 'not_exposed'>) =>
    ({ kind: 'known', caps }) as const

  it('supported et degraded valent OUI ; not_exposed et l\'absence valent NON', () => {
    expect(hasDataCapabilityIn(connu({ 'film.kill_positions': 'supported' }), 'film.kill_positions')).toBe(true)
    expect(hasDataCapabilityIn(connu({ 'film.kill_positions': 'degraded' }), 'film.kill_positions')).toBe(true)
    expect(hasDataCapabilityIn(connu({ 'film.kill_positions': 'not_exposed' }), 'film.kill_positions')).toBe(false)
    expect(hasDataCapabilityIn(connu({}), 'film.kill_positions')).toBe(false)
  })

  // LES TROIS ÉTATS, ET ILS NE SE TRAITENT PAS PAREIL (revue C-R1, constat C4).
  it('fail-CLOSED pendant le chargement : rien ne parait puis ne disparait', () => {
    expect(hasDataCapabilityIn({ kind: 'loading' }, 'film.kill_positions')).toBe(false)
  })

  it('fail-open sur erreur : une panne de cet endpoint ne mutile pas la page', () => {
    expect(hasDataCapabilityIn({ kind: 'error' }, 'film.kill_positions')).toBe(true)
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
    await waitFor(() => expect(result.current).toBe(true))
  })

  // LE FLASH FERME (revue C-R1, constat C4) : avant la reponse, la porte est CLOSE. Sans
  // cette assertion, une surface peint son contenu puis le retire sur un titre qui n aura
  // jamais la donnee — et c est la que la promesse est la plus trompeuse.
  it('fail-CLOSED tant que la reponse n est pas la : false AU PREMIER rendu', () => {
    servirCapabilities({ 'film.kill_positions': 'supported' })
    const { result } = renderHook(() => useDataCapability('film.kill_positions'), { wrapper })
    expect(result.current).toBe(false)
  })
})
