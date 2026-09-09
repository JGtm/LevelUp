/**
 * Tests — equipmentUsageLogic, LE GARDÉ ET LA RÉSERVE (E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md).
 *
 * EXTRAIT de `equipmentUsageLogic.test.ts` le 2026-09-09 (seuil de taille du dépôt, CLAUDE.md
 * n°5 : 500 lignes) — même découpe de principe que `equipmentUsageColumns.ts` extrait de
 * `equipmentUsageLogic.ts` le 2026-08-25 : d'un côté ce que le pont slot -> joueur -> équipe
 * mesure (fichier voisin), de l'autre ce que la TROISIÈME ISSUE (« gardé sans l'utiliser »)
 * ajoute par-dessus. Les fixtures partagées (`temoin`, `pose`, `SB`) vivent dans
 * `../test/equipmentUsageFixtures.ts` (CLAUDE.md n°6 — jamais une deuxième copie, et jamais
 * un import d'un fichier `.test.ts` voisin, qui rejouerait ses `describe` une seconde fois).
 *
 * CE QU'ILS PROTÈGENT :
 *   - `equipmentChangeFamilyOf` reconnaît les huit familles du bilan par la RACINE du libellé
 *     publié, dans les deux langues, et rend `null` pour un rang labellisé mais HORS BILAN
 *     (grappin, propulseur, répulseur — P4) comme pour un rang SANS AUCUN label ;
 *   - `kept` se DÉRIVE (`taken - utilisé - lâché`, jamais lu d'un canal direct) — décision
 *     utilisateur du 2026-09-09, qui amende la sortie de E0 (garder les trois issues) ;
 *   - le RÉPULSEUR n'ouvre jamais la colonne équipement fusionnée, même pris et déployé (P4) ;
 *   - un objet pris dont le rang n'a AUCUN label va dans la réserve du MATCH
 *     (`EquipmentUsage.unnamedTaken`), jamais sur une ligne de joueur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { equipmentChangeFamilyOf } from './equipmentKeptLogic'
import { buildEquipmentUsage } from './equipmentUsageLogic'
import { pose, SB, temoin } from '../test/equipmentUsageFixtures'

describe('equipmentChangeFamilyOf — la reconnaissance rang -> famille (E2)', () => {
  const LABELS_A = {
    '5': { fr: 'Mur de protection', en: 'Drop wall' },
    '6': { fr: 'Capteur de menaces', en: 'Threat sensor' },
    '7': { fr: 'Traqueur de menaces', en: 'Threat seeker' },
    '8': { fr: 'Champ de réparation', en: 'Repair field' },
    '9': { fr: 'Écran occultant', en: 'Shroud screen' },
    '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator' },
    '1': { fr: 'Camouflage actif', en: 'Active camouflage' },
    '2': { fr: 'Surbouclier', en: 'Overshield' },
    '3': { fr: 'Grappin', en: 'Grappleshot' },
    '4': { fr: 'Propulseur', en: 'Thruster' },
    '10': { fr: 'Répulseur', en: 'Repulsor' },
  }

  it('reconnaît chacune des huit familles du bilan, dans les deux langues', () => {
    expect(equipmentChangeFamilyOf(LABELS_A, 5)).toBe('wall')
    expect(equipmentChangeFamilyOf(LABELS_A, 6)).toBe('sensor')
    expect(equipmentChangeFamilyOf(LABELS_A, 7)).toBe('threat_seeker')
    expect(equipmentChangeFamilyOf(LABELS_A, 8)).toBe('repair_field')
    expect(equipmentChangeFamilyOf(LABELS_A, 9)).toBe('shroud_screen')
    expect(equipmentChangeFamilyOf(LABELS_A, 11)).toBe('translocator_beacon')
    expect(equipmentChangeFamilyOf(LABELS_A, 1)).toBe('camo')
    expect(equipmentChangeFamilyOf(LABELS_A, 2)).toBe('overshield')
  })

  it('rend null pour un rang LABELLISÉ mais hors bilan (grappin, propulseur, répulseur — P4)', () => {
    expect(equipmentChangeFamilyOf(LABELS_A, 3)).toBeNull()
    expect(equipmentChangeFamilyOf(LABELS_A, 4)).toBeNull()
    expect(equipmentChangeFamilyOf(LABELS_A, 10)).toBeNull()
  })

  it('rend null quand le rang n’a AUCUN label — c’est à l’appelant de le compter en réserve', () => {
    expect(equipmentChangeFamilyOf(LABELS_A, 42)).toBeNull()
    expect(equipmentChangeFamilyOf(undefined, 5)).toBeNull()
  })
})

describe('buildEquipmentUsage — le gardé se DÉRIVE de `equipmentChanges` (E2, décision utilisateur 2026-09-09)', () => {
  const LABELS = { '5': { fr: 'Mur de protection', en: 'Drop wall' } }

  it('un objet PRIS et jamais posé ni lâché est GARDÉ — taken(1) - utilisé(0) - lâché(0) = 1', () => {
    const doc = temoin({
      abilityLabels: LABELS,
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.kept.wall).toBe(1)
    expect(u.columns.equipment).toEqual(['wall'])
  })

  it('un objet PRIS PUIS POSÉ n’est plus gardé — taken(1) - utilisé(1) - lâché(0) = 0', () => {
    const doc = temoin({
      abilityLabels: LABELS,
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
      equipmentPlacements: [pose('wall', 'deployed', 1, '0x528fce46')],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.deployed.wall).toBe(1)
    expect(alpha?.kept.wall ?? 0).toBe(0)
  })

  it('un objet PRIS PUIS LÂCHÉ n’est pas gardé — taken(1) - utilisé(0) - lâché(1) = 0', () => {
    const doc = temoin({
      abilityLabels: LABELS,
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
      equipmentPlacements: [pose('wall', 'dropped', 1)],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.dropped.wall).toBe(1)
    expect(alpha?.kept.wall ?? 0).toBe(0)
  })

  it('un power-up ACTIVÉ (camouflage) entre dans la colonne équipement fusionnée (D9 amendée)', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 60 }],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.columns.equipment).toEqual(['camo'])
  })

  it('le RÉPULSEUR n’ouvre jamais la colonne équipement, même pris et déployé (P4)', () => {
    const doc = temoin({
      abilityLabels: { '10': { fr: 'Répulseur', en: 'Repulsor' } },
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 10, from: -1 }],
      equipmentPlacements: [pose('repulsor', 'deployed', 1)],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.columns.equipment).toEqual([])
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.kept.repulsor ?? 0).toBe(0)
  })

  it('un objet PRIS sans famille connue va dans la RÉSERVE du match, jamais sur une ligne de joueur', () => {
    const doc = temoin({
      // Aucune table `abilityLabels` : le rang 5 n'a AUCUN nom dans CE film.
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.unnamedTaken).toBe(1)
    expect(u.columns.equipment).toEqual([])
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha ? Object.keys(alpha.kept).length : 0).toBe(0)
  })
})
