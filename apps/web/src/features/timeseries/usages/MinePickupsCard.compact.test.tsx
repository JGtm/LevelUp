/**
 * MinePickupsCard.compact.test.tsx — « Mes prises dans mon camp » en vue compacte (tiroir de
 * comparaison de Sessions, maquette `renderMineCompact`) : une barre par RESSOURCE, ma part et celle
 * du reste de mon camp en pourcentage (comptes au survol), bonus perdus des deux camps en pourcentage.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { soloEmprise } from './usages.fixtures'
import { buildMinePickups, mineByResource } from './usages.logic'
import { MinePickupsCard } from './MinePickupsCard'
import { EMPRISE_TEXT_SOLO, USAGES_TEXT } from './usagesText'

const nameOf = (o: { key: string; label?: string }) => o.label ?? o.key
const mine = buildMinePickups(soloEmprise(), nameOf)!

describe('mineByResource — mes prises, ressource par ressource', () => {
  it('ma part et celle de mon camp, sommées sur les objets de la ressource, dans l’ordre du bilan', () => {
    expect(mineByResource(mine)).toEqual([
      { resource: 'powerup', me: 31, camp: 111 },
      { resource: 'power_weapon', me: 30, camp: 229 },
      { resource: 'rack', me: 2, camp: 5 },
    ])
  })
})

describe('MinePickupsCard — compact', () => {
  function renderCompact() {
    render(
      <MinePickupsCard mine={mine} itemName={nameOf} player="JGtm" t={EMPRISE_TEXT_SOLO.fr} ut={USAGES_TEXT.fr.cards} compact={{ resourceSub: 'prises de mon camp' }} />,
    )
  }

  it('une ligne par ressource, aucune ligne d’objet ni bouton de râteliers', () => {
    renderCompact()
    expect(screen.getAllByText('prises de mon camp')).toHaveLength(3)
    expect(screen.queryByTestId('usages-mine-row-spnkr')).toBeNull()
    expect(screen.queryByTestId('usages-mine-racks-toggle')).toBeNull()
  })

  it('ma part et celle du reste en pourcentage dans les segments', () => {
    renderCompact()
    expect(screen.getByTestId('usages-mine-me-power_weapon').textContent).toBe('13 %')
    expect(screen.getByTestId('usages-mine-rest-power_weapon').textContent).toBe('87 %')
    expect(screen.getByTestId('usages-mine-me-powerup').textContent).toBe('28 %')
    expect(screen.getByTestId('usages-mine-rest-powerup').textContent).toBe('72 %')
  })

  it('bonus perdus des deux camps en pourcentage', () => {
    renderCompact()
    expect(screen.getByTestId('usages-mine-losses').textContent).toBe('Bonus perdus11 %8 %')
  })
})
