/**
 * UsageEquipmentDonutCard.test.tsx — le rendu du donut + sa légende + ses sous-totaux
 * emboîtés (P10/P11, E5.9/E6.3).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { EquipmentUsageParties } from '@/lib/api/types'

import { UsageEquipmentDonutCard } from './UsageEquipmentDonutCard'
import { buildPartiesDonutModel } from './usageEquipmentPartiesModel'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('UsageEquipmentDonutCard', () => {
  it('ne rend rien quand le modele est null (parties absentes)', () => {
    const { container } = render(<UsageEquipmentDonutCard model={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('rend la legende (nom, pas de valeur) et les sous-totaux (valeur en %)', () => {
    const parties: EquipmentUsageParties = {
      lobby_total: 100,
      player: 20,
      friends: 0,
      rest_of_team: 32,
      opponents: 48,
    }
    const model = buildPartiesDonutModel(parties, [], t.donutEquipmentCenterLabel, t, 'fr')
    render(<UsageEquipmentDonutCard model={model} />)
    expect(screen.getByText('Moi')).toBeInTheDocument()
    expect(screen.getByText('Mon équipe')).toBeInTheDocument()
    expect(screen.getByText('52,0 %')).toBeInTheDocument()
    // La légende ne porte AUCUNE valeur (P11) — seul le sous-total en porte une.
    expect(screen.queryByText('20')).not.toBeInTheDocument()
  })
})
