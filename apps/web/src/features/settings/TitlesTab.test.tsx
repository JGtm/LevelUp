import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

import { TitlesTab } from './TitlesTab'
import { getSettingsText } from './i18n'
import type { PlayerTitleStatus } from './queries'

// Mocks des hooks de données : on teste la logique d'affichage (invariant min-1),
// pas les appels réseau.
const mutate = vi.fn()
let titlesData: PlayerTitleStatus[] = []

vi.mock('./queries', () => ({
  usePlayerTitles: () => ({ data: titlesData, isLoading: false }),
  useSetTitleSync: () => ({ mutate, isPending: false }),
  usePurgeTitleData: () => ({ mutate, isPending: false }),
}))

const t = getSettingsText('fr')

beforeEach(() => {
  mutate.mockClear()
})

describe('TitlesTab', () => {
  it('grise le toggle + purge du DERNIER titre actif (invariant min 1)', () => {
    titlesData = [
      { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', enrolled: true, syncEnabled: true },
      { slug: 'halo_5', name: 'Halo 5', status: 'coming_soon', enrolled: true, syncEnabled: false },
    ]
    render(<TitlesTab t={t} />)

    // Le dernier titre actif (Halo Infinite, seul actif) : son bouton Purger est désactivé.
    const purgeButtons = screen.getAllByRole('button', { name: t.titlePurgeButton })
    expect(purgeButtons).toHaveLength(2)
    expect(purgeButtons[0]).toBeDisabled() // Halo Infinite (dernier actif)
    expect(purgeButtons[1]).not.toBeDisabled() // Halo 5 (en pause → purgeable)

    // Le hint min-1 est affiché.
    expect(screen.getByText(t.titleLastActiveHint)).toBeInTheDocument()
  })

  it('permet de mettre en pause quand 2 titres sont actifs', () => {
    titlesData = [
      { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', enrolled: true, syncEnabled: true },
      { slug: 'halo_5', name: 'Halo 5', status: 'active', enrolled: true, syncEnabled: true },
    ]
    render(<TitlesTab t={t} />)

    const purgeButtons = screen.getAllByRole('button', { name: t.titlePurgeButton })
    expect(purgeButtons[0]).not.toBeDisabled()
    expect(purgeButtons[1]).not.toBeDisabled()
  })

  it('grise tout en mode démo (frozen)', () => {
    titlesData = [
      { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', enrolled: true, syncEnabled: true },
      { slug: 'halo_5', name: 'Halo 5', status: 'active', enrolled: true, syncEnabled: true },
    ]
    render(<TitlesTab t={t} frozen />)
    for (const b of screen.getAllByRole('button', { name: t.titlePurgeButton })) {
      expect(b).toBeDisabled()
    }
  })
})
