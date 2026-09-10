/**
 * Tests — equipmentUsageLogic, LE GARDÉ ET LA RÉSERVE (E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md ;
 * lot 5.7 pour la bascule « utilisé » sur les consommations de charge).
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
 *   - `equipmentChangeFamilyOf` lit `abilityLabels[rank].family` (schéma 51), la bascule sur les
 *     deux vocabulaires (`powerup_camo` -> `camo`), et rend `null` pour un rang labellisé mais
 *     HORS BILAN (grappin, propulseur, répulseur — P4) comme pour un rang SANS AUCUNE famille ;
 *   - un label SANS `family` (artefact antérieur au schéma 51) est un REPLI EXPLICITE vers la
 *     réserve `unnamedTaken`, jamais une famille devinée (lot 5.7) ;
 *   - `kept` se DÉRIVE (`taken - utilisé - lâché`, jamais lu d'un canal direct) — décision
 *     utilisateur du 2026-09-09, qui amende la sortie de E0 (garder les trois issues) ;
 *   - « UTILISÉ » se lit sur les CONSOMMATIONS (`spent`) pour tout déployable qui n'engendre pas
 *     de pièce distincte, et sur les POSES (`deployed`) pour le seul MUR (lot 5.7, aligné sur le
 *     Go `us6`) — une pose seule sur un capteur/traqueur/écran/champ/balise ne compte plus comme
 *     un usage ;
 *   - `KEPT_FAMILIES_WITH_SPAWNED_PIECE` (via `isFamilyWithSpawnedPiece`) est figée à UNE seule
 *     famille : le mur ;
 *   - le RÉPULSEUR n'ouvre jamais la colonne équipement fusionnée, même pris et déployé (P4) ;
 *   - un objet pris dont le rang n'a AUCUNE famille va dans la réserve du MATCH
 *     (`EquipmentUsage.unnamedTaken`), jamais sur une ligne de joueur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { equipmentChangeFamilyOf, isFamilyWithSpawnedPiece } from './equipmentKeptLogic'
import { buildEquipmentUsage } from './equipmentUsageLogic'
import { pose, SB, temoin } from '../test/equipmentUsageFixtures'

describe('isFamilyWithSpawnedPiece — le périmètre est figé (lot 5.7, jumeau Go)', () => {
  it('n’est vrai que pour le mur — toute autre famille du bilan lit son « utilisé » sur `spent`', () => {
    const familles = [
      'wall',
      'sensor',
      'translocator_beacon',
      'shroud_screen',
      'threat_seeker',
      'repair_field',
      'camo',
      'overshield',
    ]
    expect(familles.filter(isFamilyWithSpawnedPiece)).toEqual(['wall'])
  })
})

describe('equipmentChangeFamilyOf — la reconnaissance rang -> famille sur `Label.family` (lot 5.7)', () => {
  const LABELS_A = {
    '5': { fr: 'Mur de protection', en: 'Drop wall', family: 'wall' },
    '6': { fr: 'Capteur de menaces', en: 'Threat sensor', family: 'sensor' },
    '7': { fr: 'Traqueur de menaces', en: 'Threat seeker', family: 'threat_seeker' },
    '8': { fr: 'Champ de réparation', en: 'Repair field', family: 'repair_field' },
    '9': { fr: 'Écran occultant', en: 'Shroud screen', family: 'shroud_screen' },
    '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator', family: 'translocator_beacon' },
    '1': { fr: 'Camouflage actif', en: 'Active camouflage', family: 'powerup_camo' },
    '2': { fr: 'Surbouclier', en: 'Overshield', family: 'powerup_overshield' },
    '3': { fr: 'Grappin', en: 'Grappleshot', family: 'grapple' },
    '4': { fr: 'Propulseur', en: 'Thruster', family: 'thruster' },
    '10': { fr: 'Répulseur', en: 'Repulsor', family: 'repulsor' },
  }

  it('reconnaît chacune des huit familles du bilan par `family`, camo/overshield via le pont D5', () => {
    expect(equipmentChangeFamilyOf(LABELS_A, 5)).toBe('wall')
    expect(equipmentChangeFamilyOf(LABELS_A, 6)).toBe('sensor')
    expect(equipmentChangeFamilyOf(LABELS_A, 7)).toBe('threat_seeker')
    expect(equipmentChangeFamilyOf(LABELS_A, 8)).toBe('repair_field')
    expect(equipmentChangeFamilyOf(LABELS_A, 9)).toBe('shroud_screen')
    expect(equipmentChangeFamilyOf(LABELS_A, 11)).toBe('translocator_beacon')
    // Le manifeste publie `powerup_camo`/`powerup_overshield` (vocabulaire de POSE) — le pont
    // D5 (`EPISODE_FAMILY_OF_POWERUP`) les ramène au vocabulaire `KEPT_FAMILIES` (`camo`/
    // `overshield`), jamais une seconde table.
    expect(equipmentChangeFamilyOf(LABELS_A, 1)).toBe('camo')
    expect(equipmentChangeFamilyOf(LABELS_A, 2)).toBe('overshield')
  })

  it('rend null pour un rang LABELLISÉ, avec une famille CONNUE, mais hors bilan (P4)', () => {
    expect(equipmentChangeFamilyOf(LABELS_A, 3)).toBeNull() // grapple
    expect(equipmentChangeFamilyOf(LABELS_A, 4)).toBeNull() // thruster
    expect(equipmentChangeFamilyOf(LABELS_A, 10)).toBeNull() // repulsor
  })

  it('rend null quand le rang n’a AUCUN label — c’est à l’appelant de le compter en réserve', () => {
    expect(equipmentChangeFamilyOf(LABELS_A, 42)).toBeNull()
    expect(equipmentChangeFamilyOf(undefined, 5)).toBeNull()
  })

  it('rend null quand le label existe mais SANS `family` — artefact antérieur au schéma 51', () => {
    // La table est là, mais aucun lot < 4.3 n'y a écrit de famille : repli explicite,
    // jamais une devinette sur la racine du libellé (lot 5.7).
    expect(equipmentChangeFamilyOf({ '5': { fr: 'Mur de protection', en: 'Drop wall' } }, 5)).toBeNull()
  })
})

describe('buildEquipmentUsage — le gardé se DÉRIVE de `equipmentChanges` (E2, décision utilisateur 2026-09-09)', () => {
  const LABELS = { '5': { fr: 'Mur de protection', en: 'Drop wall', family: 'wall' } }

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

  it('un MUR PRIS PUIS POSÉ n’est plus gardé — taken(1) - utilisé(1, sur SES poses) - lâché(0) = 0', () => {
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

  it('un CAPTEUR PRIS PUIS SEULEMENT POSÉ reste GARDÉ (lot 5.7) — une pose sur un objet porté n’est pas un usage', () => {
    // C'EST LE CŒUR DE LA BASCULE : avant le lot 5.7, `t.deployed.sensor` comptait comme
    // « utilisé » pour TOUTE famille, y compris le capteur — ce test aurait attendu kept=0.
    // Le Go (lot 5.5) a mesuré que ces poses sont des lâchers volontaires à mi-vie, pas des
    // déploiements : la pose ne doit donc plus effacer le gardé.
    const doc = temoin({
      abilityLabels: { '6': { fr: 'Capteur de menaces', en: 'Threat sensor', family: 'sensor' } },
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 6, from: -1 }],
      equipmentPlacements: [pose('sensor', 'deployed', 1)],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.deployed.sensor).toBe(1)
    expect(alpha?.kept.sensor).toBe(1)
  })

  it('un CAPTEUR PRIS PUIS CONSOMMÉ (`spent`) n’est plus gardé — taken(1) - utilisé(1, `spent`) - lâché(0) = 0', () => {
    const doc = temoin({
      abilityLabels: { '6': { fr: 'Capteur de menaces', en: 'Threat sensor', family: 'sensor' } },
      equipmentChanges: [
        { t: 5, slot: 1, kind: 'taken', r: 6, from: -1 },
        { t: 8, slot: 1, kind: 'spent', r: -1, from: 6 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.spent.sensor).toBe(1)
    expect(alpha?.kept.sensor ?? 0).toBe(0)
  })

  it('un `spent` sur une chaîne de compteur TROUÉE (`gap > 0`) ne ventile aucune famille', () => {
    // Même garde que le Go (`SpentUnreliableFrom`, Gap > 0) : `from` n'est alors pas une
    // identité fiable, la consommation ne doit ventiler ni compter comme un usage.
    const doc = temoin({
      abilityLabels: { '6': { fr: 'Capteur de menaces', en: 'Threat sensor', family: 'sensor' } },
      equipmentChanges: [
        { t: 5, slot: 1, kind: 'taken', r: 6, from: -1 },
        { t: 8, slot: 1, kind: 'spent', r: -1, from: 6, gap: 1 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.spent.sensor ?? 0).toBe(0)
    expect(alpha?.kept.sensor).toBe(1)
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
      abilityLabels: { '10': { fr: 'Répulseur', en: 'Repulsor', family: 'repulsor' } },
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

  it('un objet PRIS dont le label n’a AUCUNE `family` (artefact ancien) va aussi dans la RÉSERVE', () => {
    // La table existe (donc ce n'est PAS le cas ci-dessus) mais ne porte pas de `family` :
    // repli explicite vers la réserve (lot 5.7), jamais une famille devinée sur le libellé.
    const doc = temoin({
      abilityLabels: { '5': { fr: 'Mur de protection', en: 'Drop wall' } },
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
    } as unknown as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.unnamedTaken).toBe(1)
    expect(u.columns.equipment).toEqual([])
  })
})
