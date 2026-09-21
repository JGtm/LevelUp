/**
 * D16 — rangees partagees en vue comparaison.
 *
 * Verifie l'INVARIANT de structure, pas le pixel : les deux colonnes emettent la MEME
 * liste ordonnee de cles de section (meme index = meme rangee de la grille racine),
 * et un cote qui n'a pas la section rend le placeholder « Sans equivalent dans cette
 * session ». En pleine page (drawer ferme) : pile simple, aucun placeholder.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'

import { SessionDetailPage } from './SessionDetailPage'

vi.mock('echarts-for-react', () => ({ default: () => null }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
  }
})

const CURRENT_SESSION = {
  session_label: '2026-04-21 19h30',
  start_time: '2026-04-21T19:30:00Z',
  end_time: '2026-04-21T20:05:00Z',
  total_matches: 2,
  wins: 2,
  losses: 0,
  kda: 2.4,
  performance_score: 68.5,
  win_rate: 100,
  kdr: 1.8,
  kills_per_match: 13,
  with_friends: false,
  dominant_category: 'Ranked',
}

const COMPARE_SESSION = { ...CURRENT_SESSION, session_label: '2026-04-21 18h', wins: 1, losses: 2 }

const MATCH = {
  match_id: 'match-1',
  start_time: '2026-04-21T19:45:00Z',
  outcome: 2,
  playlist_name: 'Ranked Arena',
  pair_name: 'Oddball',
  is_ranked: true,
  kills: 13,
  deaths: 5,
  assists: 6,
  kda: 2.6,
  accuracy: 64.8,
  personal_score: 2450,
  performance_score: 70,
  session_label: '2026-04-21 19h30',
  dominant_category: 'Ranked',
}

/** Bloc « usages » minimal : seule sa PRESENCE decide de l'existence de la section. */
const USAGE_BLOCK = {
  available: false,
  matches_measured: 0,
  matches_total: 2,
  unavailable_reason: 'no_film',
}

function baseResponse() {
  return {
    current_session: CURRENT_SESSION,
    available_sessions: ['2026-04-21 19h30', '2026-04-21 18h'],
    matches: [MATCH],
    suggested_compare: { session_label: '2026-04-21 18h', strategy: 's', reason: 'r' },
    compare_enabled: false,
    compare_session: null,
    compare_metrics: [],
  }
}

/** `usageSides` : quelles colonnes recoivent le bloc « usages ». */
function mockDetail(usageSides: { left: boolean; right: boolean }) {
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/sessions/detail', async ({ request }) => {
      const body = (await request.json()) as { enable_compare?: boolean }
      const payload: Record<string, unknown> = baseResponse()
      if (usageSides.left) payload.usage = USAGE_BLOCK
      if (body.enable_compare) {
        payload.compare_enabled = true
        payload.compare_session = COMPARE_SESSION
        payload.compare_matches = [{ ...MATCH, match_id: 'match-2', session_label: '2026-04-21 18h' }]
        if (usageSides.right) payload.compare_usage = USAGE_BLOCK
      }
      return HttpResponse.json(payload)
    }),
  )
}

async function openCompare() {
  await waitFor(() => {
    expect(screen.getByRole('button', { name: /Comparer/i })).toBeInTheDocument()
  })
  fireEvent.click(screen.getByRole('button', { name: /Comparer/i }))
  await waitFor(() => {
    expect(screen.getByRole('heading', { name: 'Comparaison' })).toBeInTheDocument()
  })
}

/** Cles de section dans l'ordre du DOM — colonne principale d'abord, puis drawer. */
function sectionKeys(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('[data-session-section]')).map(
    (node) => node.getAttribute('data-session-section') ?? '',
  )
}

describe('SessionDetailPage — rangees partagees (D16)', () => {
  it('emet la meme liste ordonnee de sections dans les deux colonnes', async () => {
    mockDetail({ left: true, right: true })
    const { container } = renderWithProviders(<SessionDetailPage />)
    await openCompare()

    await waitFor(() => {
      expect(sectionKeys(container).length).toBeGreaterThan(0)
    })
    const keys = sectionKeys(container)
    expect(keys.length % 2).toBe(0)
    const half = keys.length / 2
    const left = keys.slice(0, half)
    const right = keys.slice(half)
    // Meme index = meme rangee de la grille : l'egalite des deux listes EST l'alignement.
    expect(left).toEqual(right)
    // Chaque cle apparait une fois par colonne.
    expect(new Set(left).size).toBe(left.length)
    expect(left).toContain('usage')
    expect(left).toContain('summary')
    expect(left).toContain('matches')
    expect(screen.queryByTestId('session-section-placeholder')).not.toBeInTheDocument()
  })

  it('rend le placeholder du cote qui n’a pas la section', async () => {
    mockDetail({ left: true, right: false })
    const { container } = renderWithProviders(<SessionDetailPage />)
    await openCompare()

    await waitFor(() => {
      expect(screen.getByTestId('session-section-placeholder')).toBeInTheDocument()
    })
    const keys = sectionKeys(container)
    const half = keys.length / 2
    expect(keys.slice(0, half)).toEqual(keys.slice(half))
    // La rangee « usages » existe des DEUX cotes : a droite, c'est le placeholder.
    const cells = Array.from(container.querySelectorAll('[data-session-section="usage"]'))
    expect(cells).toHaveLength(2)
    expect(cells[0].querySelector('[data-testid="session-section-placeholder"]')).toBeNull()
    expect(cells[1].querySelector('[data-testid="session-section-placeholder"]')).not.toBeNull()
    expect(screen.getByText('Sans équivalent dans cette session')).toBeInTheDocument()
  })

  it('ne pose ni rangees ni placeholder en pleine page (drawer ferme)', async () => {
    mockDetail({ left: true, right: false })
    const { container } = renderWithProviders(<SessionDetailPage />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Comparer/i })).toBeInTheDocument()
    })
    expect(sectionKeys(container)).toHaveLength(0)
    expect(screen.queryByTestId('session-section-placeholder')).not.toBeInTheDocument()
  })
})
