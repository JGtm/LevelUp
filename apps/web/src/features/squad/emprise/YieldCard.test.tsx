/**
 * YieldCard.test.tsx — les lignes « non mesurable » de la Vue match (`pending`) se rangent avec les
 * rendements calculés, dans l'ordre des ressources, la raison à la place de la barre.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { EMPRISE_TEXT } from './empriseStrings'
import { YieldCard } from './YieldCard'

const T = EMPRISE_TEXT.fr

describe('YieldCard — `pending`', () => {
  it('rangées dans l’ordre des ressources avec les rendements calculés', () => {
    const { container } = render(
      <YieldCard
        rows={[{ resource: 'power_weapon', gap: -0.65, us: 1.21, them: 3.45 }]}
        pending={[
          { resource: 'vehicle', text: 'Non mesuré' },
          { resource: 'powerup', text: 'Non mesuré : frags pendant l’effet non publiés' },
        ]}
        t={T}
      />,
    )
    const pendings = [...container.querySelectorAll<HTMLElement>('[data-testid^="emprise-yield-pending-"]')]
    expect(pendings.map((p) => p.dataset.testid)).toEqual(['emprise-yield-pending-powerup', 'emprise-yield-pending-vehicle'])
    expect(screen.getByTestId('emprise-yield-pending-powerup').textContent).toContain('Non mesuré : frags pendant l’effet non publiés')
    // La ligne calculée se pose entre les deux (armes spéciales, après les bonus, avant les véhicules).
    const order = [...container.querySelectorAll<HTMLElement>('[data-testid^="emprise-yield-"]')].map((n) => n.dataset.testid)
    const iPow = order.indexOf('emprise-yield-pending-powerup')
    const iVeh = order.indexOf('emprise-yield-pending-vehicle')
    expect(order.slice(iPow + 1, iVeh).some((id) => id?.includes('power_weapon'))).toBe(true)
  })

  it('sans `pending` : le rendu des pages sœurs, aucune ligne en attente', () => {
    const { container } = render(<YieldCard rows={[{ resource: 'power_weapon', gap: 0.2, us: 1.2, them: 1 }]} t={T} />)
    expect(container.querySelectorAll('[data-testid^="emprise-yield-pending-"]')).toHaveLength(0)
  })
})