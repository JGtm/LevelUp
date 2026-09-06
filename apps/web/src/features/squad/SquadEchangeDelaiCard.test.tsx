/**
 * La DISTRIBUTION DU DÉLAI — les deux barres hors fenêtre sont montrées, jamais
 * comptées, et la carte le dit en toutes lettres.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadEchangeDelaiCard } from './SquadEchangeDelaiCard'
import { echangeDe } from './squadEchange.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadEchangeDelaiCard', () => {
  it('dit combien de ripostes tombent dans la fenêtre et combien sont hors comptage', () => {
    renderWithProviders(<SquadEchangeDelaiCard echange={echangeDe()} />)
    const narratif = screen.getByTestId('squad-echange-delai-narrative').textContent ?? ''
    expect(narratif).toContain('6') // 2 + 4 dans la fenêtre
    expect(narratif).toContain('4') // 3 + 1 hors fenêtre
  })

  it('ÉTAT VIDE quand aucune riposte n’a été mesurée', () => {
    const vide = echangeDe({
      delais: (echangeDe().delais ?? []).map((b) => ({ ...b, nombre: 0 })),
    })
    renderWithProviders(<SquadEchangeDelaiCard echange={vide} />)
    expect(screen.getByText(/Aucune riposte mesurée/i)).toBeTruthy()
  })

  it('porte la définition de l’échange ET la couverture dans son pied de carte', () => {
    renderWithProviders(<SquadEchangeDelaiCard echange={echangeDe()} />)
    expect(screen.getByTestId('squad-echange-delai-coverage').textContent).toContain('9')
    expect(screen.getByText(/dans les 5 s qui suivent votre mort/i)).toBeTruthy()
  })
})
