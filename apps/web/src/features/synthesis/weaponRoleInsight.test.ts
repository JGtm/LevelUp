import { describe, expect, it } from 'vitest'
import { NON_COMBAT_WEAPON_ROLES, weaponRoleInsight } from './weaponRoleInsight'

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

  // ── Exclusion des rôles non-combat (hors-arsenal H5) ──────────────────────
  it('exclut les rôles non-combat du total (pas de faux angle mort armes lourdes)', () => {
    // Sans exclusion : power 30 / total 5090 ≈ 0,6 % → faux blind_spot_power.
    // Avec exclusion : power 30 / combat 90 = 33 % → aucun insight.
    const insight = weaponRoleInsight([
      { role: 'precision', kills: 60 },
      { role: 'power', kills: 30 },
      { role: 'unattributed', kills: 5000 }, // bucket « Spartan »
    ])
    expect(insight).toBeNull()
  })

  it('ne signale jamais une sur-dépendance sur un rôle non-combat', () => {
    // « Spartan » (unattributed) domine en volume brut mais ne doit pas être signalé.
    const insight = weaponRoleInsight([
      { role: 'unattributed', kills: 900 },
      { role: 'precision', kills: 40 },
      { role: 'automatic', kills: 30 },
      { role: 'power', kills: 20 },
    ])
    expect(insight).toBeNull()
  })

  it('détecte une vraie sur-dépendance en ignorant le volume non-combat', () => {
    // automatic = 80 % des frags de COMBAT (100), même si 'vehicle' pèse plus en brut.
    const insight = weaponRoleInsight([
      { role: 'vehicle', kills: 500 },
      { role: 'automatic', kills: 80 },
      { role: 'precision', kills: 15 },
      { role: 'power', kills: 5 },
    ])
    expect(insight).toEqual({ kind: 'over_reliance', role: 'automatic', pct: 80 })
  })

  it('retourne null si seuls des rôles non-combat sont présents', () => {
    expect(
      weaponRoleInsight([
        { role: 'unattributed', kills: 8000 },
        { role: 'vehicle', kills: 1200 },
      ]),
    ).toBeNull()
  })

  it('expose l\'ensemble des rôles non-combat exclus', () => {
    expect([...NON_COMBAT_WEAPON_ROLES].sort()).toEqual([
      'environmental',
      'other',
      'turret',
      'unattributed',
      'vehicle',
    ])
  })
})
