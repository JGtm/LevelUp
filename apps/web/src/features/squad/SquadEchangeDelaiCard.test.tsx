/**
 * La DISTRIBUTION DU DÉLAI — les deux barres hors fenêtre sont montrées, jamais
 * comptées, et la carte le dit en toutes lettres.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadEchangeDelaiCard } from './SquadEchangeDelaiCard'
import { echangeDe } from './squadEchange.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadEchangeDelaiCard', () => {
  it('ne redit plus en toutes lettres ce que la distribution montre', () => {
    // La phrase narrative (« N ripostes sur M arrivent dans la fenêtre… ») et la note de
    // couverture ont été retirées le 2026-09-09 : le graphe porte déjà la répartition, et
    // les deux barres hors fenêtre sont nommées sur leur propre étiquette d'axe.
    renderWithProviders(<SquadEchangeDelaiCard echange={echangeDe()} />)
    expect(screen.queryByTestId('squad-echange-delai-narrative')).toBeNull()
    expect(screen.queryByTestId('squad-echange-delai-coverage')).toBeNull()
    expect(screen.queryByText(/Mesuré sur/i)).toBeNull()
  })

  it('ÉTAT VIDE quand aucune riposte n’a été mesurée', () => {
    const vide = echangeDe({
      delais: (echangeDe().delais ?? []).map((b) => ({ ...b, nombre: 0 })),
    })
    renderWithProviders(<SquadEchangeDelaiCard echange={vide} />)
    expect(screen.getByText(/Aucune riposte mesurée/i)).toBeTruthy()
  })

  it('porte la définition de l’échange et sa fenêtre dans l’aide ⓘ du titre', () => {
    renderWithProviders(<SquadEchangeDelaiCard echange={echangeDe()} />)
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toMatch(/dans les 5 s qui suivent votre mort/i)
    expect(aide).toMatch(/hors fenêtre/i)
  })
})
