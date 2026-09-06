/**
 * LA PREMIÈRE DES DEUX PORTES : la barre d'onglets d'Ascension.
 *
 * Ces tests cadenassent le 5e onglet « Tactique » (2026-09-06) :
 *   - il apparaît, en FR comme en EN, quand le titre déclare `replay` ;
 *   - il DISPARAÎT quand le titre ne le déclare pas — un onglet qui mène à une page
 *     « indisponible » est un onglet qui ment ;
 *   - les quatre onglets historiques ne bougent pas.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { AscensionLayout } from './AscensionLayout'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'JGtm' }),
    useMatchRoute: () => () => false,
    Link: ({ children, ...rest }: { children: React.ReactNode }) => (
      <a {...(rest as Record<string, unknown>)}>{children}</a>
    ),
    Outlet: () => null,
  }
})

function poserTitre(capabilities: string[], locale: 'fr' | 'en' = 'fr') {
  useAppShellStore.setState({
    locale,
    currentTitleSlug: 'un_titre',
    availableTitles: [{ slug: 'un_titre', name: 'Un titre', capabilities }] as never,
  })
}

beforeEach(() => poserTitre(['replay']))
afterEach(() => useAppShellStore.setState({ locale: 'fr', availableTitles: [] }))

describe('AscensionLayout — la rangée d’onglets', () => {
  it('titre AVEC `replay` : les cinq onglets, Tactique en dernier', () => {
    renderWithProviders(<AscensionLayout />)
    const onglets = screen.getAllByRole('tab').map((n) => n.textContent)
    expect(onglets).toEqual([
      'Profil',
      'Objectifs',
      'Entraînement',
      'Réalisations',
      'Tactique',
    ])
  })

  it('titre SANS `replay` : pas d’onglet Tactique — un onglet mort ment', () => {
    poserTitre(['matchmaking'])
    renderWithProviders(<AscensionLayout />)
    expect(screen.queryByText('Tactique')).toBeNull()
    expect(screen.getAllByRole('tab')).toHaveLength(4)
  })

  it('en anglais, l’onglet s’appelle « Tactics »', () => {
    poserTitre(['replay'], 'en')
    renderWithProviders(<AscensionLayout />)
    expect(screen.getByText('Tactics')).toBeInTheDocument()
  })
})
