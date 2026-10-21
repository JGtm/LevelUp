/**
 * Tests — « Résultats de la dernière session » (HomePage) : la séquence vient du serveur
 * (`last_session_outcomes`, calculée sur tout l'historique), plus d'un filtre sur la liste de tuiles.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { HomePage } from './HomePage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

// Le graphique ECharts n'est pas l'objet : le témoin expose ce qu'il reçoit, dans l'ordre reçu.
vi.mock('@/components/charts/OutcomeSequenceTape', () => ({
  OutcomeSequenceTape: ({ matches }: { matches: Array<{ matchId: string; outcome: string; map: string }> }) => (
    <ol data-testid="tape">
      {matches.map((m) => (
        <li key={m.matchId} data-outcome={m.outcome}>
          {m.matchId}|{m.map}
        </li>
      ))}
    </ol>
  ),
}))

function homePayload(extra: Record<string, unknown>) {
  return {
    hero: { player_name: 'TestPlayer', kpis: { win_rate: 0.5, total_matches: 400, wins: 200, losses: 200 }, trend: null },
    highlights: [],
    recent_matches: [],
    recent_matches_total: 400,
    recent_matches_has_more: false,
    solo_session: null,
    squad_session: null,
    ...extra,
  }
}

describe('HomePage — séquence de la dernière session', () => {
  it('rend les issues du serveur, la plus ancienne d\'abord, issues inconnues en égalité', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () =>
        HttpResponse.json(
          homePayload({
            last_session_outcomes: [
              { match_id: 'm3', outcome_tone: 'win', detail: 'Slayer · FDA 1.20 · 50 - 44' },
              { match_id: 'm2', outcome_tone: 'loss', detail: 'CTF · FDA 0.80 · 1 - 2' },
              { match_id: 'm1', outcome_tone: 'tie', detail: 'Oddball · FDA 1.00 · 90 - 90' },
            ],
          }),
        ),
      ),
    )
    renderWithProviders(<HomePage />)
    const tape = await screen.findByTestId('tape')
    const rows = Array.from(tape.querySelectorAll('li')).map((li) => li.textContent)
    // Le serveur sert la plus récente en tête ; la bande se lit de la plus ancienne à la plus récente.
    expect(rows).toEqual([
      'm1|Oddball · FDA 1.00 · 90 - 90',
      'm2|CTF · FDA 0.80 · 1 - 2',
      'm3|Slayer · FDA 1.20 · 50 - 44',
    ])
    expect(Array.from(tape.querySelectorAll('li')).map((li) => li.getAttribute('data-outcome'))).toEqual([
      'tie',
      'loss',
      'win',
    ])
  })

  it('aucune issue servie : pas de section', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () =>
        HttpResponse.json(homePayload({ last_session_outcomes: [] })),
      ),
    )
    renderWithProviders(<HomePage />)
    await waitFor(() => expect(screen.getByText('Taux de victoire')).toBeInTheDocument())
    expect(screen.queryByTestId('tape')).not.toBeInTheDocument()
  })
})
