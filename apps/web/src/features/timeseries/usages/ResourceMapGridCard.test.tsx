/**
 * ResourceMapGridCard.test.tsx — « Contrôle des ressources, carte par carte » : une colonne par carte
 * (nom, matchs, bilan V / D / A), la colonne « Autres cartes » et son nombre de cartes, les cases de
 * la table de l'Emprise et l'infobulle « Chez moi » ouverte par l'en-tête de la carte.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { soloEmprise } from './usages.fixtures'
import { buildMapGrid } from './usages.logic'
import { ResourceMapGridCard } from './ResourceMapGridCard'
import { EMPRISE_TEXT_SOLO, USAGES_TEXT } from './usagesText'

function renderCard() {
  render(
    <ResourceMapGridCard
      grid={buildMapGrid(soloEmprise())}
      itemName={(row) => row.object?.label ?? row.object?.key ?? ''}
      playerName={(x) => (x === 'xj' ? 'JGtm' : '')}
      t={EMPRISE_TEXT_SOLO.fr}
      ut={USAGES_TEXT.fr.cards}
    />,
  )
}

const text = (id: string) => screen.getByTestId(id).textContent ?? ''

describe('ResourceMapGridCard', () => {
  it('titre de la maquette', () => {
    renderCard()
    expect(screen.getByText('Contrôle des ressources, carte par carte')).toBeTruthy()
  })

  it('en-têtes : nom, matchs, V / D ; « A » seulement quand il y a d’autres issues', () => {
    renderCard()
    expect(text('usages-map-head-aq')).toBe('Carte Alpha2 matchs1 V · 1 D')
    expect(text('usages-map-head-rc')).toBe('Carte Bravo1 match1 V · 0 D')
  })

  it('« Autres cartes » : son nombre de cartes devant ses matchs', () => {
    renderCard()
    expect(text('usages-map-head-others')).toBe('Autres cartes3 cartes · 1 match0 V · 0 D · 1 A')
  })

  it('cases : valeur, non classé (sans niveaux), sans film', () => {
    renderCard()
    const cells = Array.from(screen.getByTestId('emprise-grid-table').querySelectorAll('[data-cell]')).map((c) => c.getAttribute('data-cell'))
    // Bonus (3 colonnes), camouflage (3), armes spéciales (3), frags aux armes spéciales (3), râteliers repliés.
    expect(cells.slice(0, 3)).toEqual(['value', 'value', 'nofilm'])
    expect(cells).toContain('untiered')
  })

  it('infobulle : l’en-tête de la carte, puis « Chez moi »', () => {
    renderCard()
    const camo = Array.from(screen.getByTestId('emprise-grid-table').querySelectorAll('[data-cell="value"]'))[2]
    fireEvent.mouseEnter(camo.parentElement!)
    const tip = screen.getByRole('tooltip').textContent ?? ''
    expect(tip).toContain('Carte Alpha (2 matchs, 2 filmés)')
    // Moi d'abord, puis le reste du camp (maquette), quel que soit le volume.
    expect(tip).toContain('Chez moi : JGtm 15, reste du camp 35')
  })
})
