/**
 * Le repli « Qui riposte pour qui » — ce qu'il montre et ce qu'il tait.
 *
 * Ce que ces tests cadenassent : un état vide n'est jamais un graphe à zéro ; la rangée
 * « reçu N » est là ; et le panneau ne pose plus aucun chrome de carte (D19, 2026-09-21 —
 * pas de SectionCard dans une SectionCard).
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadRiposteMatricePanel } from './SquadRiposteMatricePanel'
import { echangeDe } from './squadRiposte.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadRiposteMatricePanel', () => {
  it('ne pose AUCUN chrome de carte, ni bandeau de couverture, ni légende de rampe', () => {
    const { container } = renderWithProviders(<SquadRiposteMatricePanel echange={echangeDe()} />)
    expect(container.querySelector('section')).toBeNull()
    expect(screen.queryByTestId('squad-echange-coverage')).toBeNull()
    expect(screen.queryByTestId('squad-echange-ramp')).toBeNull()
  })

  it('pose la rangée « reçu N » sous la grille', () => {
    renderWithProviders(<SquadRiposteMatricePanel echange={echangeDe()} />)
    expect(screen.getByTestId('squad-riposte-recus').textContent ?? '').toMatch(/reçu/i)
  })

  it('ÉTAT VIDE, jamais un graphe à zéro, quand aucune riposte interne', () => {
    renderWithProviders(<SquadRiposteMatricePanel echange={echangeDe({ cellules: [] })} />)
    expect(screen.getAllByText(/Aucune riposte/i)[0]).toBeTruthy()
  })

  it('PARITÉ FR/EN : les deux langues rendent un texte, et deux textes différents', () => {
    const { unmount } = renderWithProviders(<SquadRiposteMatricePanel echange={echangeDe()} />)
    const fr = screen.getByTestId('squad-riposte-matrice').textContent ?? ''
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadRiposteMatricePanel echange={echangeDe()} />)
    const en = screen.getByTestId('squad-riposte-matrice').textContent ?? ''
    expect(fr.length).toBeGreaterThan(0)
    expect(en).not.toBe(fr)
    // Aucune clé de manifest non résolue ne doit fuir à l'écran.
    expect(en.includes('squad.riposte.')).toBe(false)
    expect(fr.includes('squad.riposte.')).toBe(false)
  })
})
