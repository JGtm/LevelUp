import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { RelationInsight } from '@/lib/api/types'

import { getPalmaresText } from './i18n'
import { RelationsWhatsNewStrip } from './RelationsWhatsNewStrip'

const labels = getPalmaresText('fr').relations
const DAY = 86_400_000

function rel(overrides: Partial<RelationInsight>): RelationInsight {
  return {
    xuid: overrides.xuid ?? 'x',
    gamertag: overrides.gamertag ?? 'GT',
    total_matches: 5,
    teammate_matches: 3,
    teammate_wins: 2,
    teammate_win_rate: 0.5,
    enemy_matches: 2,
    enemy_wins: 1,
    enemy_win_rate: 0.5,
    avg_kda_with: 1,
    avg_kda_against: 1,
    kills_dealt: 4,
    deaths_suffered: 4,
    duel_ratio: 1,
    first_seen_at: null,
    last_seen_at: null,
    category: 'mixed',
    is_core: false,
    is_revived: false,
    badges: [],
    ...overrides,
  }
}

describe('RelationsWhatsNewStrip', () => {
  it('ne rend rien quand aucune donnée ne matche', () => {
    const rows = [rel({ xuid: 'a', gamertag: 'OldOnly', first_seen_at: new Date(Date.now() - 300 * DAY).toISOString() })]
    renderWithProviders(<RelationsWhatsNewStrip relations={rows} labels={labels} onPlayerClick={vi.fn()} />)
    expect(screen.queryByTestId('palmares-relations-whats-new')).not.toBeInTheDocument()
  })

  it('affiche les nouvelles têtes et les retrouvailles avec les bons gamertags', () => {
    const rows = [
      rel({ xuid: 'a', gamertag: 'FreshFace', first_seen_at: new Date(Date.now() - 3 * DAY).toISOString() }),
      rel({ xuid: 'b', gamertag: 'BackAgain', is_revived: true, last_seen_at: new Date(Date.now() - 2 * DAY).toISOString() }),
      rel({ xuid: 'c', gamertag: 'Neither', first_seen_at: new Date(Date.now() - 400 * DAY).toISOString() }),
    ]
    const onPlayerClick = vi.fn()
    renderWithProviders(<RelationsWhatsNewStrip relations={rows} labels={labels} onPlayerClick={onPlayerClick} />)

    expect(screen.getByTestId('palmares-relations-whats-new')).toBeInTheDocument()
    expect(screen.getByText('Nouvelles têtes')).toBeInTheDocument()
    expect(screen.getByText('Retrouvailles')).toBeInTheDocument()

    const fresh = screen.getByRole('button', { name: 'FreshFace' })
    expect(fresh).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'BackAgain' })).toBeInTheDocument()
    // La relation ni nouvelle ni ravivée n'apparaît pas.
    expect(screen.queryByText('Neither')).not.toBeInTheDocument()

    fresh.click()
    expect(onPlayerClick).toHaveBeenCalledWith('FreshFace')
  })
})
