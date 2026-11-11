/**
 * SessionLeadShareChart — « Mainmise, match par match » de la page Sessions : la carte et sa
 * présence. Elle n'existe que sur un titre qui décode le film (capability `replay`), entre « Vies
 * nettes » et « Intensité », et dit pourquoi elle est vide quand aucun match n'est mesuré.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'
import { useAppShellStore } from '@/stores/appShellStore'

import { SessionColumnBody } from './SessionColumnBody'
import { SessionLeadShareChart } from './SessionLeadShareChart'
import { session2209 } from './sessionEmprise.fixtures'

vi.mock('echarts-for-react', () => ({ default: () => <div data-testid="echarts-stub" /> }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useParams: () => ({ playerSlug: 'moi' }), useSearch: () => ({}), useNavigate: () => vi.fn() }
})

afterEach(() => useAppShellStore.setState({ locale: 'fr', availableTitles: [], currentTitleSlug: 'halo_infinite' }))

function rows(over: Array<Partial<SessionDetailMatchRow>>): SessionDetailMatchRow[] {
  return over as unknown as SessionDetailMatchRow[]
}

describe('SessionLeadShareChart — la carte', () => {
  it('titre, bouton d’information et moyenne des matchs mesurés sous le graphique', async () => {
    const entry = { lead_share: 0.54, lead_share_matches: 5 } as unknown as SessionCompareEntry
    renderWithProviders(
      <SessionLeadShareChart
        matches={rows([{ match_id: 'm1', map_name: 'Bazaar', lead_share: 0.78 }, { match_id: 'm2', map_name: 'Recharge' }])}
        entry={entry}
      />,
    )
    expect(screen.getByText(LEAD_SHARE_TEXT.fr.perMatchTitle)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /informations/i })).toBeInTheDocument()
    expect(screen.getByText('Moyenne 54 % sur 5 matchs mesurés')).toBeInTheDocument()
    expect(await screen.findByTestId('echarts-stub')).toBeInTheDocument()
  })

  it('aucun match mesuré : l’état vide nomme la cause, ni graphique ni « N non mesurés »', () => {
    renderWithProviders(<SessionLeadShareChart matches={rows([{ match_id: 'm1' }])} entry={null} />)
    expect(screen.getByText(LEAD_SHARE_TEXT.fr.emptyNoMeasure)).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
    expect(screen.queryByText(/non mesuré/i)).toBeNull()
  })

  it('EN : « Lead share, match by match »', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SessionLeadShareChart matches={rows([{ match_id: 'm1', lead_share: 0.5 }])} entry={null} />)
    expect(screen.getByText('Lead share, match by match')).toBeInTheDocument()
  })
})

describe('SessionColumnBody — la carte Mainmise suit la capability du film', () => {
  function titre(capabilities: string[]) {
    useAppShellStore.setState({
      availableTitles: [{ slug: 'halo_infinite', capabilities }] as never,
      currentTitleSlug: 'halo_infinite',
    })
  }

  it('titre qui décode le film : la carte est là, entre « Vies nettes » et « Intensité »', () => {
    titre(['replay'])
    const { container } = renderWithProviders(<SessionColumnBody blocks={session2209()} playerSlug="moi" compact={false} />)
    expect(screen.getByTestId('lead-share-chart')).toBeInTheDocument()
    const cards = Array.from(container.querySelectorAll('[data-testid="lead-share-chart"], [data-testid="chart-card"]'))
    const i = cards.findIndex((n) => n.getAttribute('data-testid') === 'lead-share-chart')
    expect(i).toBeGreaterThan(0)
  })

  it('titre sans décodeur de film (Halo 5) : aucune carte', () => {
    titre(['analytics.timeseries'])
    renderWithProviders(<SessionColumnBody blocks={session2209()} playerSlug="moi" compact={false} />)
    expect(screen.queryByTestId('lead-share-chart')).toBeNull()
  })
})
