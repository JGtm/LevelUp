import { describe, expect, it } from 'vitest'
import type { FragDistribution } from '@/lib/api/types'
import {
  NON_COMBAT_WEAPON_ROLES,
  insightFromRoles,
  rolesFromDistribution,
  weaponRoleInsight,
} from './weaponRoleInsight'

// ── Cœur de logique (insightFromRoles) — inchangé depuis kills_by_role ─────────
describe('insightFromRoles', () => {
  it('retourne null sous le seuil minimum de frags', () => {
    expect(insightFromRoles([{ role: 'precision', kills: 10 }])).toBeNull()
  })

  it('retourne null pour une entrée vide', () => {
    expect(insightFromRoles([])).toBeNull()
  })

  it('détecte l\'angle mort armes lourdes (power < 3%)', () => {
    const insight = insightFromRoles([
      { role: 'precision', kills: 60 },
      { role: 'automatic', kills: 40 },
    ])
    expect(insight).toEqual({ kind: 'blind_spot_power' })
  })

  it('ne signale pas l\'angle mort si power suffisant', () => {
    const insight = insightFromRoles([
      { role: 'precision', kills: 50 },
      { role: 'power', kills: 30 }, // 37.5% → pas un angle mort
    ])
    expect(insight).toBeNull()
  })

  it('détecte la sur-dépendance (un rôle > 70%)', () => {
    const insight = insightFromRoles([
      { role: 'automatic', kills: 80 },
      { role: 'precision', kills: 10 },
      { role: 'power', kills: 10 }, // power 10% → pas d'angle mort, mais automatic 80%
    ])
    expect(insight).toEqual({ kind: 'over_reliance', role: 'automatic', pct: 80 })
  })

  it('priorise l\'angle mort sur la sur-dépendance', () => {
    // automatic 95%, power 0% → les deux règles matchent, blind_spot gagne.
    const insight = insightFromRoles([
      { role: 'automatic', kills: 95 },
      { role: 'precision', kills: 5 },
    ])
    expect(insight).toEqual({ kind: 'blind_spot_power' })
  })

  // ── Exclusion des rôles non-combat (hors-arsenal H5) ──────────────────────
  it('exclut les rôles non-combat du total (pas de faux angle mort armes lourdes)', () => {
    // Sans exclusion : power 30 / total 5090 ≈ 0,6 % → faux blind_spot_power.
    // Avec exclusion : power 30 / combat 90 = 33 % → aucun insight.
    const insight = insightFromRoles([
      { role: 'precision', kills: 60 },
      { role: 'power', kills: 30 },
      { role: 'unattributed', kills: 5000 }, // bucket « Spartan »
    ])
    expect(insight).toBeNull()
  })

  it('ne signale jamais une sur-dépendance sur un rôle non-combat', () => {
    // « Spartan » (unattributed) domine en volume brut mais ne doit pas être signalé.
    const insight = insightFromRoles([
      { role: 'unattributed', kills: 900 },
      { role: 'precision', kills: 40 },
      { role: 'automatic', kills: 30 },
      { role: 'power', kills: 20 },
    ])
    expect(insight).toBeNull()
  })

  it('détecte une vraie sur-dépendance en ignorant le volume non-combat', () => {
    // automatic = 80 % des frags de COMBAT (100), même si 'vehicle' pèse plus en brut.
    const insight = insightFromRoles([
      { role: 'vehicle', kills: 500 },
      { role: 'automatic', kills: 80 },
      { role: 'precision', kills: 15 },
      { role: 'power', kills: 5 },
    ])
    expect(insight).toEqual({ kind: 'over_reliance', role: 'automatic', pct: 80 })
  })

  it('retourne null si seuls des rôles non-combat sont présents', () => {
    expect(
      insightFromRoles([
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

// ── Dérivation depuis la FragDistribution (gun classes) ────────────────────────
describe('rolesFromDistribution', () => {
  it('retourne [] pour une distribution absente/vide', () => {
    expect(rolesFromDistribution(undefined)).toEqual([])
    expect(rolesFromDistribution(null)).toEqual([])
    expect(rolesFromDistribution({ total_kills: 0, classes: [] })).toEqual([])
  })

  it('n\'agrège QUE les gun classes (melee/grenade/spartan/unattributed exclus)', () => {
    const fd: FragDistribution = {
      total_kills: 100,
      classes: [
        { class: 'shoulder', kills: 30, authoritative: false, roles: [{ role: 'precision', kills: 30 }] },
        { class: 'melee', kills: 20, authoritative: true },
        { class: 'grenade', kills: 10, authoritative: true },
        { class: 'spartan_ability', kills: 5, authoritative: true, roles: [{ role: 'ground_pound', kills: 5 }] },
        { class: 'unattributed', kills: 35, authoritative: false },
      ],
    }
    expect(rolesFromDistribution(fd)).toEqual([{ role: 'precision', kills: 30 }])
  })

  it('reconstruit une gun class FEUILLE comme un rôle portant son nom de classe', () => {
    // Poing/sidearm est une feuille (roles nil) → rôle 'sidearm' (byte-équiv kills_by_role).
    const fd: FragDistribution = {
      total_kills: 20,
      classes: [{ class: 'sidearm', kills: 20, authoritative: false }],
    }
    expect(rolesFromDistribution(fd)).toEqual([{ role: 'sidearm', kills: 20 }])
  })

  it('somme un rôle trans-classes présent sous plusieurs gun classes', () => {
    // shotgun sous Épaule (Bulldog) + Lourde (Heatwave) → un seul rôle sommé.
    const fd: FragDistribution = {
      total_kills: 40,
      classes: [
        { class: 'shoulder', kills: 15, authoritative: false, roles: [{ role: 'shotgun', kills: 15 }] },
        { class: 'heavy', kills: 25, authoritative: false, roles: [{ role: 'shotgun', kills: 25 }] },
      ],
    }
    expect(rolesFromDistribution(fd)).toEqual([{ role: 'shotgun', kills: 40 }])
  })
})

describe('weaponRoleInsight (depuis FragDistribution)', () => {
  it('retourne null si aucune distribution', () => {
    expect(weaponRoleInsight(undefined)).toBeNull()
    expect(weaponRoleInsight(null)).toBeNull()
  })

  it('détecte l\'angle mort armes lourdes depuis les gun classes', () => {
    // Épaule 100 (precision+automatic), aucune arme lourde (power) → blind spot.
    const fd: FragDistribution = {
      total_kills: 100,
      classes: [
        {
          class: 'shoulder',
          kills: 100,
          authoritative: false,
          roles: [
            { role: 'precision', kills: 60 },
            { role: 'automatic', kills: 40 },
          ],
        },
      ],
    }
    expect(weaponRoleInsight(fd)).toEqual({ kind: 'blind_spot_power' })
  })

  it('détecte la sur-dépendance (rôle > 70%) sans compter le résidu non-combat', () => {
    // automatic 80 / combat 100 = 80 % ; le gros unattributed (classe API) est ignoré.
    const fd: FragDistribution = {
      total_kills: 600,
      classes: [
        {
          class: 'shoulder',
          kills: 90,
          authoritative: false,
          roles: [
            { role: 'automatic', kills: 80 },
            { role: 'precision', kills: 10 },
          ],
        },
        { class: 'heavy', kills: 10, authoritative: false, roles: [{ role: 'power', kills: 10 }] },
        { class: 'unattributed', kills: 500, authoritative: false },
      ],
    }
    expect(weaponRoleInsight(fd)).toEqual({ kind: 'over_reliance', role: 'automatic', pct: 80 })
  })
})
