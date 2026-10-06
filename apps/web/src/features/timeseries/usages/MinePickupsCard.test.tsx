/**
 * MinePickupsCard.test.tsx — « Contribution aux prises » : groupes par ressource, objets triés par
 * volume de l’équipe, segments joueur / reste avec leurs comptes, « JGtm n · équipe m » au bout, barre à
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
  render(<MinePickupsCard mine={buildMinePickups(soloEmprise(), nameOf)!} itemName={nameOf} player="JGtm" t={EMPRISE_TEXT_SOLO.fr} ut={USAGES_TEXT.fr.cards} />)
}

const width = (id: string) => parseFloat((screen.getByTestId(id) as HTMLElement).style.width)

describe('MinePickupsCard', () => {
  it('titre et légende : le joueur par son gamertag, le reste de l’équipe', () => {
    renderCard()
    expect(screen.getByText('Contribution aux prises')).toBeTruthy()
    expect(screen.getAllByText('JGtm').length).toBeGreaterThan(0)
    expect(screen.getByText('Reste de l’équipe')).toBeTruthy()
  })

  it('une ligne par objet pris par l’équipe, « JGtm n · équipe m » au bout', () => {
    renderCard()
    expect(screen.getByTestId('usages-mine-value-spnkr').textContent).toBe('JGtm 30 · équipe 129')
    expect(screen.getByTestId('usages-mine-value-sniper').textContent).toBe('JGtm 0 · équipe 100')
  })

  it('barre à l’échelle du plus gros objet ; segments moi / reste dans la barre', () => {
    renderCard()
    expect(width('usages-mine-track-spnkr')).toBeCloseTo(100)
    expect(width('usages-mine-track-sniper')).toBeCloseTo((100 / 129) * 100)
    expect(width('usages-mine-me-spnkr')).toBeCloseTo((30 / 129) * 100)
    expect(width('usages-mine-rest-spnkr')).toBeCloseTo((99 / 129) * 100)
    expect(screen.getByTestId('usages-mine-me-spnkr').textContent).toBe('30')
  })

  it('aucune prise du joueur : pas de segment du joueur', () => {
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
