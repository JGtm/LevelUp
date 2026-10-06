/**
 * MinePickupsCard.test.tsx — « Mes prises dans mon camp » : groupes par ressource, objets triés par
 * volume de mon camp, segments moi / reste avec leurs comptes, « moi n · camp m » au bout, barre à
 * l'échelle du plus gros objet, râteliers repliés, bonus perdus des deux camps.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { soloEmprise } from './usages.fixtures'
import { buildMinePickups } from './usages.logic'
import { MinePickupsCard } from './MinePickupsCard'
import { EMPRISE_TEXT_SOLO, USAGES_TEXT } from './usagesText'

const nameOf = (o: { key: string; label?: string }) => o.label ?? o.key

function renderCard() {
  render(<MinePickupsCard mine={buildMinePickups(soloEmprise(), nameOf)!} itemName={nameOf} t={EMPRISE_TEXT_SOLO.fr} ut={USAGES_TEXT.fr.cards} />)
}

const width = (id: string) => parseFloat((screen.getByTestId(id) as HTMLElement).style.width)

describe('MinePickupsCard', () => {
  it('titre et légende de la maquette', () => {
    renderCard()
    expect(screen.getByText('Mes prises dans mon camp')).toBeTruthy()
    expect(screen.getByText('Moi')).toBeTruthy()
    expect(screen.getByText('Reste de mon camp')).toBeTruthy()
  })

  it('une ligne par objet pris par mon camp, « moi n · camp m » au bout', () => {
    renderCard()
    expect(screen.getByTestId('usages-mine-value-spnkr').textContent).toBe('moi 30 · camp 129')
    expect(screen.getByTestId('usages-mine-value-sniper').textContent).toBe('moi 0 · camp 100')
  })

  it('barre à l’échelle du plus gros objet ; segments moi / reste dans la barre', () => {
    renderCard()
    expect(width('usages-mine-track-spnkr')).toBeCloseTo(100)
    expect(width('usages-mine-track-sniper')).toBeCloseTo((100 / 129) * 100)
    expect(width('usages-mine-me-spnkr')).toBeCloseTo((30 / 129) * 100)
    expect(width('usages-mine-rest-spnkr')).toBeCloseTo((99 / 129) * 100)
    expect(screen.getByTestId('usages-mine-me-spnkr').textContent).toBe('30')
  })

  it('aucune prise à moi : pas de segment « moi »', () => {
    renderCard()
    expect(screen.queryByTestId('usages-mine-me-sniper')).toBeNull()
    expect(width('usages-mine-rest-sniper')).toBeCloseTo(100)
  })

  it('armes de râtelier repliées derrière leur intertitre', () => {
    renderCard()
    expect(screen.queryByTestId('usages-mine-row-br')).toBeNull()
    const toggle = screen.getByTestId('usages-mine-racks-toggle')
    expect(toggle.textContent).toContain('(1, repliées)')
    fireEvent.click(toggle)
    expect(screen.getByTestId('usages-mine-row-br')).toBeTruthy()
  })

  it('bonus perdus des deux camps', () => {
    renderCard()
    expect(screen.getByTestId('usages-mine-losses').textContent).toBe('Bonus perdus12 sur 111 (11 %)7 sur 87 (8 %)')
  })
})
