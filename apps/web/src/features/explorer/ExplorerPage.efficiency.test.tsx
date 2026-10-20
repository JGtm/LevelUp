/**
 * ExplorerPage — efficacité (E2 revue 2026-07).
 *
 * Trois garde-rails :
 *   1. Mode Joueur → la query matches-query (incluant le briefing serveur) est
 *      DÉSACTIVÉE (enabled=false) et include_briefing=false : aucun recompute
 *      serveur inutile quand son résultat n'est pas consommé.
 *   2. Frappe rapide dans l'input match-ID → une seule valeur atteint la query
 *      après le debounce (250 ms) : 1 POST par rafale, pas un par caractère.
 *   3. Mode Joueur avec une cible → les deux tableaux (allié / ennemi) partagent
 *      UNE requête matches-query portant tous les matchs communs.
 *
 * On espionne les ARGUMENTS passés à useExplorerMatches (le proxy fidèle du
 * déclenchement réseau : React Query émet 1 POST par queryKey distincte, et
 * match_id_search est dans la queryKey). usePageScope est mocké en état local
 * pour que l'input contrôlé réponde à la frappe (le vrai passe par l'URL/router).
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { ExplorerPage } from './ExplorerPage'
import type { ExplorerScope } from './explorerScope'

const hoisted = vi.hoisted(() => ({
  search: {} as Record<string, unknown>,
  matchesCalls: [] as Array<{ request: Record<string, unknown>; enabled: boolean }>,
  player: undefined as Record<string, unknown> | undefined,
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => hoisted.search,
  }
})

vi.mock('./queries', () => ({
  useExplorerMatches: (
    _playerSlug: string,
    request: Record<string, unknown>,
    _hash: string,
    enabled = true,
  ) => {
    hoisted.matchesCalls.push({ request, enabled })
    return { data: undefined, isLoading: false, isError: false, isFetching: false, error: null }
  },
  useExplorerPlayer: () => ({ data: hoisted.player, isLoading: false, isError: false, error: null }),
}))

vi.mock('@/lib/page-scope/usePageScope', async () => {
  const React = await import('react')
  const { decodeExplorerScope } = await import('./explorerScope')
  return {
    usePageScope: () => {
      const [scope, setScope] = React.useState(() => decodeExplorerScope({}))
      return {
        scope,
        setScope: (patch: Partial<ExplorerScope>) => setScope((s) => ({ ...s, ...patch })),
        reset: () => setScope(decodeExplorerScope({})),
      }
    },
  }
})

// Filtre les appels de la query PRINCIPALE (seule à porter include_export_hint ;
// les tableaux allié/ennemi passent par match_ids sans ce flag).
function mainCalls() {
  return hoisted.matchesCalls.filter((c) => c.request.include_export_hint === true)
}

describe('ExplorerPage — efficacité (E2)', () => {
  beforeEach(() => {
    hoisted.search = {}
    hoisted.matchesCalls.length = 0
    hoisted.player = undefined
  })

  it('mode Joueur : query matches-query désactivée + include_briefing=false', () => {
    hoisted.search = { mode: 'player' }
    renderWithProviders(<ExplorerPage />)
    const calls = mainCalls()
    expect(calls.length).toBeGreaterThan(0)
    for (const c of calls) {
      expect(c.enabled).toBe(false)
      expect(c.request.include_briefing).toBe(false)
    }
  })

  it('mode Matchs : query active avec briefing', () => {
    hoisted.search = { mode: 'matches' }
    renderWithProviders(<ExplorerPage />)
    const last = mainCalls().at(-1)
    expect(last).toBeDefined()
    expect(last!.enabled).toBe(true)
    expect(last!.request.include_briefing).toBe(true)
  })

  it('frappe rapide dans match-ID : une seule valeur atteint la query après 250 ms', async () => {
    hoisted.search = { mode: 'matches' }
    renderWithProviders(<ExplorerPage />)
    const input = screen.getByPlaceholderText('ID match…')

    fireEvent.change(input, { target: { value: 'a' } })
    fireEvent.change(input, { target: { value: 'ab' } })
    fireEvent.change(input, { target: { value: 'abc' } })

    // Pendant la rafale (avant 250 ms) : aucune valeur n'a encore atteint la query.
    expect(mainCalls().every((c) => !c.request.match_id_search)).toBe(true)

    // Après le debounce : 'abc' est propagé.
    await waitFor(() => {
      expect(mainCalls().some((c) => c.request.match_id_search === 'abc')).toBe(true)
    })

    // Seule 'abc' a jamais atteint la query (jamais 'a' ni 'ab') → 1 POST par rafale.
    const distinct = [...new Set(mainCalls().map((c) => c.request.match_id_search).filter(Boolean))]
    expect(distinct).toEqual(['abc'])
  })

  it('mode Joueur : une seule requête matches-query pour les deux tableaux', () => {
    hoisted.search = { mode: 'player', target: 'Cible' }
    hoisted.player = {
      target_gamertag: 'Cible',
      common_matches: [
        { match_id: 'm1', were_teammates: true },
        { match_id: 'm2', were_teammates: false },
        { match_id: 'm3', were_teammates: true },
      ],
    }
    renderWithProviders(<ExplorerPage />)
    const tableCalls = hoisted.matchesCalls.filter(
      (c) => c.enabled && c.request.include_export_hint !== true,
    )
    const distinctBodies = new Set(tableCalls.map((c) => JSON.stringify(c.request.match_ids)))
    expect([...distinctBodies]).toEqual([JSON.stringify(['m1', 'm2', 'm3'])])
  })
})
