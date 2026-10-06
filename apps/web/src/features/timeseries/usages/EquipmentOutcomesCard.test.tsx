/**
 * EquipmentOutcomesCard.test.tsx — « Équipement pris, et ce que j'en ai fait » : une ligne par famille
 * dans l'ordre du Go ; mesurées : servi / gardé / lâché pour moi (comptes dans les segments), barre
 * fine et ligne de parts pour le reste de mon camp ; non mesurées : « Non mesuré » et mes lâchers.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { soloEmprise } from './usages.fixtures'
import { buildEquipmentRows } from './usages.logic'
import { EquipmentOutcomesCard } from './EquipmentOutcomesCard'
import { USAGES_TEXT } from './usagesText'

const NAMES: Record<string, string> = { wall: 'Mur de protection', sensor: 'Capteur de menaces', shroud_screen: 'Écran occultant', grapple: 'Grappin', thruster: 'Propulseur' }

function renderCard() {
  render(<EquipmentOutcomesCard rows={buildEquipmentRows(soloEmprise())} familyLabel={(f) => NAMES[f] ?? f} ut={USAGES_TEXT.fr.cards} />)
}

const width = (id: string) => parseFloat((screen.getByTestId(id) as HTMLElement).style.width)

describe('EquipmentOutcomesCard', () => {
  it('une ligne par famille, dans l’ordre du Go', () => {
    renderCard()
    const rows = screen.getAllByTestId(/^usages-equip-row-/).map((n) => n.getAttribute('data-testid'))
    expect(rows).toEqual(['usages-equip-row-grapple', 'usages-equip-row-wall', 'usages-equip-row-sensor', 'usages-equip-row-shroud_screen', 'usages-equip-row-thruster'])
  })

  it('mur : 84 objets dont 23 pris sur la carte ; servi 52 et lâché 32 dans leurs segments, aucun gardé', () => {
    renderCard()
    expect(screen.getByTestId('usages-equip-sub-wall').textContent).toBe('84 objets, dont 23 pris sur la carte')
    expect(width('usages-equip-me-wall-used')).toBeCloseTo((52 / 84) * 100)
    expect(screen.getByTestId('usages-equip-me-wall-used').textContent).toBe('52')
    expect(screen.queryByTestId('usages-equip-me-wall-kept')).toBeNull()
    expect(width('usages-equip-me-wall-dropped')).toBeCloseTo((32 / 84) * 100)
  })

  it('reste de mon camp : barre fine et ligne de parts', () => {
    renderCard()
    expect(width('usages-equip-rest-wall-used')).toBeCloseTo((146 / 304) * 100)
    expect(screen.getByTestId('usages-equip-restline-wall').textContent).toBe('reste de mon camp : 146 servis · 7 gardés · 151 lâchés48 % servis')
  })

  it('non mesurées : le libellé, mes lâchers, « Non mesuré »', () => {
    renderCard()
    expect(screen.getByTestId('usages-equip-sub-grapple').textContent).toBe('84 lâchés')
    expect(screen.getByTestId('usages-equip-row-grapple').textContent).toContain('Non mesuré : ni prise ni usage publiés pour cette famille')
    expect(screen.getByTestId('usages-equip-sub-thruster').textContent).toBe('65 lâchés')
  })

  it('famille sans objet : « 0 objet », piste vide, reste à zéro dit', () => {
    renderCard()
    expect(screen.getByTestId('usages-equip-sub-shroud_screen').textContent).toBe('0 objet')
    expect(screen.queryByTestId('usages-equip-me-shroud_screen-used')).toBeNull()
    expect(screen.getByTestId('usages-equip-restline-shroud_screen').textContent).toBe('reste de mon camp : 0 objet')
  })

  it('légende de la maquette', () => {
    renderCard()
    for (const l of ['Servi', 'Gardé sans servir', 'Lâché', 'Barre fine : reste de mon camp']) expect(screen.getByText(l)).toBeTruthy()
  })
})
