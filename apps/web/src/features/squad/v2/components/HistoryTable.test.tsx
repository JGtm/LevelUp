import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { HistoryTable } from './HistoryTable'
import type { HistoryTableRow } from '../types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(${token})`,
}))

const baseLabels = {
  date: 'Date',
  mode: 'Mode',
  map: 'Map',
  outcome: 'Outcome',
  duration: 'Duration',
  kdaSuffix: 'K/D/A',
}

describe('HistoryTable', () => {
  it('rend une ligne par match', () => {
    const rows: HistoryTableRow[] = [
      {
        match_id: 'm1',
        started_at_utc: '2026-04-01T10:00:00Z',
        duration_seconds: 600,
        map_label: 'Aquarius',
        mode_label: 'Slayer',
        main_outcome: 'win',
        player_stats: {
          main: { kills: 10, deaths: 5, assists: 3, outcome: 'win' },
          f1: { kills: 6, deaths: 7, assists: 1, outcome: 'win' },
        },
      },
      {
        match_id: 'm2',
        started_at_utc: '2026-04-01T11:00:00Z',
        main_outcome: 'loss',
        player_stats: {
          main: { kills: 8, deaths: 9, assists: 0 },
        },
      },
    ]
    render(
      <HistoryTable
        rows={rows}
        squadOrder={['main', 'f1']}
        locale="fr-FR"
        labels={baseLabels}
      />,
    )
    expect(screen.getByTestId('history-table')).toBeTruthy()
    expect(screen.getAllByRole('row')).toHaveLength(3) // 1 header + 2 data
    expect(screen.getByText('Aquarius')).toBeTruthy()
    expect(screen.getByText('Slayer')).toBeTruthy()
    // Cellule manquante (f1 absent du m2) -> "-"
    const cells = screen.getAllByText('-')
    expect(cells.length).toBeGreaterThan(0)
  })

  it('formate la duree mm:ss', () => {
    const rows: HistoryTableRow[] = [
      {
        match_id: 'm1',
        started_at_utc: '2026-04-01T10:00:00Z',
        duration_seconds: 605,
        main_outcome: 'win',
        player_stats: {},
      },
    ]
    render(
      <HistoryTable rows={rows} squadOrder={['main']} locale="fr-FR" labels={baseLabels} />,
    )
    expect(screen.getByText('10:05')).toBeTruthy()
  })

  it('rows vides → composant absent', () => {
    const { container } = render(
      <HistoryTable rows={[]} squadOrder={['main']} locale="fr-FR" labels={baseLabels} />,
    )
    expect(container.firstChild).toBeNull()
  })
})
