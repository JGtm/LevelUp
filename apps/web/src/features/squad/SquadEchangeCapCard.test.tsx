/**
 * Le CAP DU MOMENT — deux seuils, deux cadrages, et rien du tout en dessous.
 *
 * La règle elle-même est testée dans squadEchange.logic.test.ts ; ici on vérifie que
 * le composant ne rend RIEN quand elle dit non (aucun état vide, aucun bruit) et
 * qu'il ne culpabilise pas quand elle dit oui.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadEchangeCapCard } from './SquadEchangeCapCard'
import { couverture, echangeDe } from './squadEchange.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadEchangeCapCard', () => {
  it('ne rend RIEN quand la section est absente du contrat', () => {
    const { container } = renderWithProviders(<SquadEchangeCapCard echange={undefined} />)
    expect(container.textContent).toBe('')
  })

  it('ne rend RIEN sous les seuils (aucun état vide, aucun bruit)', () => {
    const sousLePlancher = echangeDe({
      couverture: couverture(8, 8, 3),
      habituel: couverture(40, 100),
    })
    const { container } = renderWithProviders(<SquadEchangeCapCard echange={sousLePlancher} />)
    expect(container.textContent).toBe('')
  })

  it('cadre l’écart négatif en ATTENTION, jamais en alerte', () => {
    renderWithProviders(
      <SquadEchangeCapCard
        echange={echangeDe({ couverture: couverture(9, 45), habituel: couverture(40, 100) })}
      />,
    )
    const phrase = screen.getByTestId('squad-echange-cap-phrase').textContent ?? ''
    expect(phrase).toMatch(/mérite votre attention/i)
  })

  it('cadre l’écart positif en CONSOLIDATION', () => {
    renderWithProviders(
      <SquadEchangeCapCard
        echange={echangeDe({ couverture: couverture(27, 45), habituel: couverture(40, 100) })}
      />,
    )
    expect(screen.getByTestId('squad-echange-cap-phrase').textContent).toMatch(/consolidez/i)
  })
})
