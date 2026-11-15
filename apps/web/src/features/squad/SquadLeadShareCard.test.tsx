import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: ({ title, series }: { title?: ReactNode; series?: unknown[] }) => (
    <div data-testid="chart-card" data-empty={series && series.length === 0 ? 'oui' : undefined}>
      {title}
    </div>
  ),
}))

import { renderWithProviders } from '@/test/render-utils'
import type { SquadLeadSharePoint } from '@/lib/api/types'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadLeadShareCard } from './SquadLeadShareCard'

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const SERIE: SquadLeadSharePoint[] = [
  { match_id: 'm1', start_time: '2026-11-03T20:00:00Z', outcome: 2, map_name: 'Bazaar', lead_share: 0.7, lead_changes: 2 },
  { match_id: 'm2', start_time: '2026-11-03T20:20:00Z', outcome: 3 },
]

describe('SquadLeadShareCard', () => {
  it('titre neutre « Mainmise, match par match » et une courbe quand un match au moins est mesuré', () => {
    renderWithProviders(<SquadLeadShareCard series={SERIE} />)
    expect(screen.getByText(LEAD_SHARE_TEXT.fr.perMatchTitle)).toBeInTheDocument()
    expect(screen.getByTestId('chart-card').getAttribute('data-empty')).toBeNull()
  })

  it('aucun match mesuré, ou série absente : état vide, jamais zéro', () => {
    renderWithProviders(<SquadLeadShareCard series={[SERIE[1]]} />)
    expect(screen.getByTestId('chart-card').getAttribute('data-empty')).toBe('oui')
  })

  it('EN : « Lead share, match by match »', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadLeadShareCard series={undefined} />)
    expect(screen.getByText('Lead share, match by match')).toBeInTheDocument()
  })
})
