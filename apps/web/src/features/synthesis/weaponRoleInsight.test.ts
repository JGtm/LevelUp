import { describe, expect, it } from 'vitest'
import { weaponRoleInsight } from './weaponRoleInsight'

describe('weaponRoleInsight', () => {
  it('retourne null sous le seuil minimum de frags', () => {
    expect(weaponRoleInsight([{ role: 'precision', kills: 10 }])).toBeNull()
  })

  it('retourne null pour une entrée vide/undefined', () => {
    expect(weaponRoleInsight(undefined)).toBeNull()
    expect(weaponRoleInsight([])).toBeNull()
  })

  it('détecte l\'angle mort armes lourdes (power < 3%)', () => {
    const insight = weaponRoleInsight([
      { role: 'precision', kills: 60 },
      { role: 'automatic', kills: 40 },
    ])
    expect(insight).toEqual({ kind: 'blind_spot_power' })
  })

  it('ne signale pas l\'angle mort si power suffisant', () => {
    const insight = weaponRoleInsight([
      { role: 'precision', kills: 50 },
      { role: 'power', kills: 30 }, // 37.5% → pas un angle mort
    ])
    expect(insight).toBeNull()
  })

  it('détecte la sur-dépendance (un rôle > 70%)', () => {
    const insight = weaponRoleInsight([
      { role: 'automatic', kills: 80 },
      { role: 'precision', kills: 10 },
      { role: 'power', kills: 10 }, // power 10% → pas d'angle mort, mais automatic 80%
    ])
    expect(insight).toEqual({ kind: 'over_reliance', role: 'automatic', pct: 80 })
  })

  it('priorise l\'angle mort sur la sur-dépendance', () => {
    // automatic 95%, power 0% → les deux règles matchent, blind_spot gagne.
    const insight = weaponRoleInsight([
      { role: 'automatic', kills: 95 },
      { role: 'precision', kills: 5 },
    ])
    expect(insight).toEqual({ kind: 'blind_spot_power' })
  })
})
