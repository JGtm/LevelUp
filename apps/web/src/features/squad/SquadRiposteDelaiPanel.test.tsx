/**
 * Le repli « Combien de temps on met » — les deux barres hors fenêtre sont montrées,
 * jamais comptées, et le panneau le dit en toutes lettres.
 *
 * CE N'EST PLUS UNE CARTE (D19, 2026-09-21) : plus de SectionCard, plus d'infobulle ⓘ
 * propre — la méthode vit désormais dans l'unique infobulle de la carte « Riposte ».
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadRiposteDelaiPanel } from './SquadRiposteDelaiPanel'
import { echangeDe } from './squadRiposte.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadRiposteDelaiPanel', () => {
  it('ne pose AUCUN chrome de carte : il vit dans la carte « Riposte »', () => {
    const { container } = renderWithProviders(<SquadRiposteDelaiPanel echange={echangeDe()} />)
    expect(container.querySelector('section')).toBeNull()
    expect(screen.getByTestId('squad-riposte-delai')).toBeTruthy()
  })

  it('nomme la fenêtre et les barres hors fenêtre sous le graphe', () => {
    renderWithProviders(<SquadRiposteDelaiPanel echange={echangeDe()} />)
    const texte = screen.getByTestId('squad-riposte-delai').textContent ?? ''
    expect(texte).toMatch(/Fenêtre de riposte/i)
    expect(texte).toMatch(/hachurée/i)
  })

  it('ÉTAT VIDE quand aucune riposte n’a été mesurée', () => {
    const vide = echangeDe({
      delais: (echangeDe().delais ?? []).map((b) => ({ ...b, nombre: 0 })),
    })
    renderWithProviders(<SquadRiposteDelaiPanel echange={vide} />)
    expect(screen.getAllByText(/Aucune riposte mesurée/i)[0]).toBeTruthy()
  })
})
