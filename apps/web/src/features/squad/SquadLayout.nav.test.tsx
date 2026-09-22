/**
 * SquadLayout.nav.test.tsx — LA BARRE D'ONGLETS DE L'ESCOUADE : quatre onglets, pas
 * cinq (lot 3 « sections », 2026-09-22). Passée de trois à quatre avec l'arrivée
 * d'Usages, elle a un PLAFOND : quatre onglets maximum, un axe de lecture par onglet.
 * Ce test est le ratchet de ce plafond autant que du libellé des quatre.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { SquadLayout } from './SquadLayout'
import { getSquadText } from './i18n'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'p' }),
    useSearch: () => ({}),
    useMatchRoute: () => () => null,
    useNavigate: () => vi.fn(),
    Outlet: () => <div data-testid="contenu-onglet" />,
    Link: ({ children, to }: { children?: ReactNode; to?: string }) => <a href={to}>{children}</a>,
  }
})

/** Réponse teammates minimale mais NON vide : le contenu (donc la barre) est monté. */
const teammatesReponse = {
  options: [],
  teammates: [],
  total_matches: 3,
  session_labels: { solo: [], squad: [] },
  friends_count: 0,
  composition_sessions: [],
  latest_composition_session: '',
  match_history: [],
}

const resolveReponse = {
  effective: {
    filter_mode: 'period',
    period: { start_date: null, end_date: null },
    sessions: { picked_sessions: [], gap_minutes: 120 },
    cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
  },
  available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
  period_presets: [
    { preset_id: '7d', count: 1 },
    { preset_id: '30d', count: 3 },
    { preset_id: '90d', count: 3 },
    { preset_id: 'all', count: 3 },
  ],
}

beforeEach(() => {
  localStorage.clear()
  useSquadFilterStore.getState().resetFilters()
  useAppShellStore.setState({ locale: 'fr' })
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/teammates', () =>
      HttpResponse.json(teammatesReponse),
    ),
    http.post('/api/v1/players/:playerSlug/filters/resolve', () =>
      HttpResponse.json(resolveReponse),
    ),
  )
})

async function monter() {
  renderWithProviders(<SquadLayout />)
  await screen.findByTestId('contenu-onglet')
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 150))
  })
}

describe('SquadLayout — barre d\'onglets', () => {
  it('rend QUATRE onglets, dans l\'ordre, vers les quatre sous-routes', async () => {
    await monter()
    const t = getSquadText('fr')
    const onglets = screen.getAllByRole('navigation')[0]
    const liens = Array.from(onglets.querySelectorAll('a'))
    expect(liens.map((a) => a.textContent)).toEqual([
      t.nav.synergies,
      t.nav.contributions,
      t.nav.dynamique,
      t.nav.usages,
    ])
    expect(liens.map((a) => a.getAttribute('href'))).toEqual([
      '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies',
      '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/contributions',
      '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique',
      '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/usages',
    ])
  })
})
