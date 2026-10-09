/**
 * queries.gate.test.tsx — LA PORTE DE TITRE DES DEUX HOOKS DU FILM de la Match View.
 *
 * Condition n°2 du lot v2(C) : « sur halo_5, aucune requête de film n'est émise ». Elle ne
 * tenait pas — la revue adversariale C-R1 l'a relevée : `useMatchObjectiveEvents` et
 * `useMatchPositions` n'avaient aucune garde de capability dans leur `enabled`, si bien que
 * deux requêtes partaient à l'ouverture de l'onglet Chronologie et prenaient le 503 posé par
 * v2(C.2). Aucune erreur visible (les consommateurs lisent `data ?? []`, la carte de chaleur
 * rend `null` sur une liste vide), mais deux allers-retours réseau garantis perdants.
 *
 * Ces tests figent les DEUX sens : rien ne part sans la clé, tout part avec.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'

import { api } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

import { useMatchObjectiveEvents, useMatchPositions } from './queries'

vi.mock('@/lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn() },
  getApiTitleSlug: () => 'titre_test',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

const apiGet = vi.mocked(api.get)

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

/** Sert les capabilities data-level du titre courant, et RIEN d'autre. */
function servirCapabilites(statut: 'supported' | 'not_exposed') {
  useAppShellStore.setState({ currentTitleSlug: 'titre_test' })
  apiGet.mockImplementation(async (url: string) => {
    if (url.includes('/capabilities')) {
      return {
        title_slug: 'titre_test',
        schema_version: 1,
        capabilities: { 'film.replay_artifact': statut },
      }
    }
    return []
  })
}

/** Les URL demandées, hors requête de capabilities (qui, elle, doit partir). */
function urlsDuFilm(): string[] {
  return apiGet.mock.calls
    .map(([url]) => String(url))
    .filter((url) => !url.includes('/capabilities'))
}

beforeEach(() => {
  apiGet.mockReset()
})

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('useMatchObjectiveEvents / useMatchPositions — porte de titre', () => {
  it("titre sans film.replay_artifact : AUCUNE des deux requêtes ne part", async () => {
    servirCapabilites('not_exposed')
    renderHook(
      () => {
        useMatchObjectiveEvents('me', 'match-1')
        useMatchPositions('me', 'match-1')
      },
      { wrapper },
    )
    // La requête de capabilities, elle, part : c'est elle qui ferme la porte.
    await waitFor(() => expect(apiGet).toHaveBeenCalled())
    expect(urlsDuFilm()).toEqual([])
  })

  it('titre AVEC la clé : les deux requêtes partent', async () => {
    servirCapabilites('supported')
    renderHook(
      () => {
        useMatchObjectiveEvents('me', 'match-1')
        useMatchPositions('me', 'match-1')
      },
      { wrapper },
    )
    await waitFor(() => expect(urlsDuFilm()).toHaveLength(2))
    expect(urlsDuFilm().some((u) => u.endsWith('/objective-events'))).toBe(true)
    expect(urlsDuFilm().some((u) => u.endsWith('/positions'))).toBe(true)
  })

  // La porte de titre s'AJOUTE au paramètre `enabled` existant (onglet actif), elle ne le
  // remplace pas : un titre supporté dont l'onglet est fermé ne tire toujours rien.
  it('titre avec la clé mais onglet fermé : rien ne part non plus', async () => {
    servirCapabilites('supported')
    renderHook(
      () => {
        useMatchObjectiveEvents('me', 'match-1', false)
        useMatchPositions('me', 'match-1', false)
      },
      { wrapper },
    )
    await waitFor(() => expect(apiGet).toHaveBeenCalled())
    expect(urlsDuFilm()).toEqual([])
  })
})
