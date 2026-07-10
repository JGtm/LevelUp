/**
 * Test contrat V8b — CareerPage top matches.
 *
 * Régression prouvée (audit A2 §7) : le front consommait `data.top_matches_preview`
 * (absent du CareerPageResponse Go) et `CareerTopMatchesResponse.items` (la réponse
 * réelle est `{ best_matches, worst_matches: TopMatchDTO[] }`). Les deux chemins
 * rendaient `undefined` → section toujours vide à l'écran.
 *
 * Ce test monte CareerPage avec une réponse /top-matches au shape RÉEL du backend
 * et vérifie que les lignes s'affichent (rendu non-vide) — la garantie fonctionnelle
 * demandée par le lot.
 */
import { describe, it, expect, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import type { CareerTopMatchesResponse } from '@/lib/api/types'
import { CareerPage } from './CareerPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

// Shape RÉELLE du backend : best_matches / worst_matches (TopMatchDTO), jamais `items`.
const TOP_MATCHES: CareerTopMatchesResponse = {
  best_matches: [
    {
      match_id: 'm-best-1',
      start_time: '2026-06-01T20:00:00Z',
      performance_score: 92.5,
      map_ui: 'Aquarius',
      mode_ui: 'Slayer',
      outcome_code: 2,
      outcome_label: 'Victoire',
      kills: 25,
      deaths: 8,
      kda: 3.1,
    },
  ],
  worst_matches: [
    {
      match_id: 'm-worst-1',
      start_time: '2026-06-02T21:00:00Z',
      performance_score: 12.3,
      map_ui: 'Streets',
      mode_ui: 'Oddball',
      outcome_code: 3,
      outcome_label: 'Défaite',
      kills: 4,
      deaths: 18,
      kda: 0.4,
    },
  ],
}

describe('CareerPage — top matches (contrat V8b)', () => {
  it('affiche les top matchs au shape réel { best_matches, worst_matches }', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/career/top-matches', () =>
        HttpResponse.json(TOP_MATCHES),
      ),
    )

    renderWithProviders(<CareerPage />)

    // Le meilleur match (best_matches[0]) doit rendre sa map + son outcome.
    await waitFor(() => {
      expect(screen.getByText('Aquarius')).toBeInTheDocument()
    })
    // La section n'est plus le placeholder vide.
    expect(screen.queryByText(/Top matchs indisponibles/i)).not.toBeInTheDocument()
    // Bouton "voir tout" présent car worst_matches non vide.
    expect(screen.getByText(/Voir tous les top matchs/i)).toBeInTheDocument()
  })
})
