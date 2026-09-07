/**
 * La MATRICE « qui échange pour qui » — ce qu'elle montre et ce qu'elle tait.
 *
 * Ce que ces tests cadenassent : le bandeau de couverture vit AU-DESSUS du graphe
 * (les films expirent, le manque est définitif) et reste affiché même sans paire —
 * sans lui, « aucune » se confondrait avec « rien mesuré » ; un état vide n'est
 * jamais un graphe à zéro ; sous le plancher d'échantillon la réserve s'affiche et
 * aucun badge ne classe personne ; et les deux langues rendent deux textes.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadEchangeMatrixCard } from './SquadEchangeMatrixCard'
import { couverture, echangeDe } from './squadEchange.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadEchangeMatrixCard', () => {
  it('affiche le bandeau de couverture AU-DESSUS du graphe (« mesuré sur N des M »)', () => {
    renderWithProviders(<SquadEchangeMatrixCard echange={echangeDe()} />)
    const bandeau = screen.getByTestId('squad-echange-coverage')
    expect(bandeau.textContent).toContain('9')
    expect(bandeau.textContent).toContain('12')
  })

  it('pose une ligne narrative chiffrée au-dessus du graphe', () => {
    renderWithProviders(<SquadEchangeMatrixCard echange={echangeDe()} />)
    const narratif = screen.getByTestId('squad-echange-narrative').textContent ?? ''
    expect(narratif).toContain('18')
    expect(narratif).toContain('45')
  })

  it('ÉTAT VIDE, jamais un graphe à zéro, quand aucune vengeance interne', () => {
    renderWithProviders(<SquadEchangeMatrixCard echange={echangeDe({ cellules: [] })} />)
    expect(screen.getByText(/Aucune vengeance/i)).toBeTruthy()
    // La couverture RESTE affichée : sans elle, « aucune » se lirait « rien mesuré ».
    expect(screen.getByTestId('squad-echange-coverage')).toBeTruthy()
  })

  it('pose la réserve « échantillon faible » sous le plancher, et la tait au-dessus', () => {
    renderWithProviders(
      <SquadEchangeMatrixCard echange={echangeDe({ couverture: couverture(8, 8, 3) })} />,
    )
    expect(screen.getByTestId('squad-echange-low-sample')).toBeTruthy()
    expect(screen.queryByTestId('squad-echange-badges')).toBeNull()
  })

  it('PARITÉ FR/EN : les deux langues rendent un texte, et deux textes différents', () => {
    const { unmount } = renderWithProviders(<SquadEchangeMatrixCard echange={echangeDe()} />)
    const fr = screen.getByTestId('squad-echange-coverage').textContent ?? ''
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadEchangeMatrixCard echange={echangeDe()} />)
    const en = screen.getByTestId('squad-echange-coverage').textContent ?? ''
    expect(fr.length).toBeGreaterThan(0)
    expect(en.length).toBeGreaterThan(0)
    expect(en).not.toBe(fr)
    // Aucune clé de manifest non résolue ne doit fuir à l'écran.
    expect(en.startsWith('squad.')).toBe(false)
    expect(fr.startsWith('squad.')).toBe(false)
  })
})
