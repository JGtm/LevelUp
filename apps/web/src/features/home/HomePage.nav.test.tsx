/**
 * Tests — navigation des cards "Sessions récentes" (HomePage).
 *
 * Card solo → Timeseries (session épinglée dans soloFilterStore).
 * Card escouade → /squad?session=…&teammates=… (composition pré-sélectionnée).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { HomePage } from './HomePage'

const navigateSpy = vi.fn()

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigateSpy,
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

const SOLO_LABEL = '09/06/2026 18:00–18:40 (4)'
const SQUAD_LABEL = '10/06/2026 19:52–20:14 (3)'

function sessionPayload(label: string, teammates?: string[]) {
  return {
    session_label: label,
    match_count: 3,
    win_rate: 0.66,
    global_ratio: 1.2,
    started_at: '2026-06-10T19:52:00Z',
    ended_at: '2026-06-10T20:14:00Z',
    wins: 2,
    losses: 1,
    draws: 0,
    dnfs: 0,
    avg_player_performance: 80,
    avg_team_performance: 75,
    avg_kda: 1.5,
    dominant_playlist: 'Ranked',
    dominant_mode: 'Slayer',
    ...(teammates ? { teammates } : {}),
  }
}

beforeEach(() => {
  navigateSpy.mockReset()
  useSoloFilterStore.getState().resetFilters()
  server.use(
    http.get('/api/v1/players/:playerSlug/pages/home', () =>
      HttpResponse.json({
        hero: { player_name: 'TestPlayer', kpis: { win_rate: 0.5, total_matches: 10, wins: 5, losses: 5 }, trend: null },
        highlights: [],
        recent_matches: [],
        favorite_matches: [],
        recent_media: [],
        solo_session: null,
        squad_session: null,
        solo_sessions: [sessionPayload(SOLO_LABEL)],
        squad_sessions: [sessionPayload(SQUAD_LABEL, ['Alice', 'Bob'])],
      }),
    ),
  )
})

describe('HomePage — navigation cards Sessions récentes', () => {
  it('card solo → Timeseries + épingle la session dans soloFilterStore', async () => {
    renderWithProviders(<HomePage />)
    const card = await screen.findByRole('button', {
      name: `Voir le détail de la session ${SOLO_LABEL}`,
    })
    fireEvent.click(card)

    expect(navigateSpy).toHaveBeenCalledWith(
      expect.objectContaining({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries' }),
    )
    expect(useSoloFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual([SOLO_LABEL])
  })

  it('card escouade → /squad avec session + coéquipiers (joints par virgule)', async () => {
    renderWithProviders(<HomePage />)
    const card = await screen.findByRole('button', {
      name: `Voir le détail de la session ${SQUAD_LABEL}`,
    })
    fireEvent.click(card)

    expect(navigateSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        to: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad',
        search: { session: SQUAD_LABEL, teammates: 'Alice,Bob' },
      }),
    )
  })

  it('card escouade sans coéquipiers résolus → teammates undefined', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () =>
        HttpResponse.json({
          hero: { player_name: 'TestPlayer', kpis: { win_rate: 0.5, total_matches: 10, wins: 5, losses: 5 }, trend: null },
          highlights: [],
          recent_matches: [],
          favorite_matches: [],
          recent_media: [],
          solo_session: null,
          squad_session: null,
          squad_sessions: [sessionPayload(SQUAD_LABEL)],
        }),
      ),
    )
    renderWithProviders(<HomePage />)
    const card = await screen.findByRole('button', {
      name: `Voir le détail de la session ${SQUAD_LABEL}`,
    })
    fireEvent.click(card)

    expect(navigateSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        to: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad',
        search: { session: SQUAD_LABEL, teammates: undefined },
      }),
    )
  })
})
