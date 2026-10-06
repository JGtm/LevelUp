/**
 * EquipmentOutcomesCard.compact.test.tsx — « Équipement pris, et ce que j'en ai fait » en vue
 * compacte (tiroir de comparaison de Sessions, maquette `renderEquip` avec `cp`) : parts entières dans
 * la barre épaisse, barre fine gardée, sous-libellé « n objets », ligne du reste réduite à sa part de
 * servis, « Non mesuré » court pour grappin et propulseur.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { soloEmprise } from './usages.fixtures'
import { buildEquipmentRows } from './usages.logic'
import { EquipmentOutcomesCard } from './EquipmentOutcomesCard'
import { USAGES_TEXT } from './usagesText'

const COMPACT = {
  sub: (n: number) => `${n} objets`,
  unmeasured: 'Non mesuré',
  restUsed: (pct: string) => `reste de l’équipe : ${pct} servis`,
}

function renderCompact() {
  render(<EquipmentOutcomesCard rows={buildEquipmentRows(soloEmprise())} familyLabel={(f) => f} player="JGtm" ut={USAGES_TEXT.fr.cards} compact={COMPACT} />)
}

describe('EquipmentOutcomesCard — compact', () => {
  it('parts entières dans la barre épaisse (mur : 52 · 0 · 32 → 62 % · 38 %)', () => {
    renderCompact()
    expect(screen.getByTestId('usages-equip-me-wall-used').textContent).toBe('62 %')
    expect(screen.getByTestId('usages-equip-me-wall-dropped').textContent).toBe('38 %')
  })

  it('barre fine du reste gardée, sa ligne réduite à la part de servis', () => {
    renderCompact()
    expect(screen.getByTestId('usages-equip-rest-wall-used')).toBeTruthy()
    expect(screen.getByTestId('usages-equip-restline-wall').textContent).toBe('reste de l’équipe : 48 % servis')
  })

  it('sous-libellé « n objets » sans les prises ; non mesurée : « Non mesuré » court', () => {
    renderCompact()
    expect(screen.getByTestId('usages-equip-sub-wall').textContent).toBe('84 objets')
    expect(screen.getByTestId('usages-equip-row-grapple').textContent).toContain('Non mesuré')
    expect(screen.getByTestId('usages-equip-row-grapple').textContent).not.toContain('ni prise ni usage')
  })
})
