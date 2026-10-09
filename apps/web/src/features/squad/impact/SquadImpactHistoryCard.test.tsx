import { describe, expect, it } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import { impactHistory3 } from './impactHistory.fixtures'
import { SquadImpactHistoryCard } from './SquadImpactHistoryCard'

const inkOf = (gt: string) => `var(--ink-${gt})`

describe('SquadImpactHistoryCard', () => {
  it('titre, puis légende sur trois lignes : nets des joueurs, gains, pertes', () => {
    renderWithProviders(
      <SquadImpactHistoryCard history={impactHistory3} colorByPlayer={{}} inkOf={inkOf} locale="fr" />,
    )
    const card = screen.getByTestId('squad-impact-history')
    expect(within(card).getByText('Points d’impact par soirée et par rôle')).toBeInTheDocument()
    const rows = within(card).getAllByRole('list')
    expect(rows).toHaveLength(3)
    expect(rows[0].textContent).toBe('Net de la soirée :JGtmChocoboflorMadina97294')
    expect(within(rows[1]).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'Finisseur +2', 'Premier sang +2', 'Héros silencieux +1,5', 'Bourreau +1',
    ])
    expect(within(rows[2]).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'Boulet −2', 'Faux-frère −1,5', 'Autres rôles à −1 (Première victime, Touriste, Kamikaze, Voleur)',
    ])
  })

  it('en anglais', () => {
    renderWithProviders(
      <SquadImpactHistoryCard history={impactHistory3} colorByPlayer={{}} inkOf={inkOf} locale="en" />,
    )
    expect(screen.getByText('Impact points per evening and role')).toBeInTheDocument()
    expect(screen.getByText('Other −1 roles (First down, Late starter, Kamikaze, Thief)')).toBeInTheDocument()
  })

  it('rien sans soirée', () => {
    renderWithProviders(
      <SquadImpactHistoryCard history={{ ...impactHistory3, evenings: [] }} colorByPlayer={{}} inkOf={inkOf} locale="fr" />,
    )
    expect(screen.queryByTestId('squad-impact-history')).toBeNull()
  })
})
