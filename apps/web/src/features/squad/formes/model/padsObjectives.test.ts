/**
 * padsObjectives.test.ts — LES SOCLES ET L'OBJECTIF.
 *
 * Deux invariants que l'artefact 2ec1b8eb pose explicitement :
 *   - le dénominateur des parts de socle est le nombre de prises NOMMÉES, et les
 *     occupations sans ramasseur restent comptées à part (la réserve affichée) ;
 *   - une durée ne se compare qu'à sa propre parité, et les colonnes d'un mode
 *     sont celles que le serveur publie — jamais une liste écrite ici.
 */
import { describe, expect, it } from 'vitest'

import { FORMES_MAIN_XUID, formesFixture } from '../formes.fixtures'
import {
  aggregateColumns,
  aggregateRole,
  columnsOfFamily,
  matchesOfFamily,
  objectiveFamilies,
  objectiveMatches,
  objectiveCell,
  roleLobbyParts,
} from './objectives'
import {
  aggregateWeaponClass,
  matchesWithWeapon,
  namedPickups,
  unnamedOccupations,
  weaponIndex,
  weaponOccupations,
  weaponPickupsByPlayer,
  weaponsByVolume,
} from './pads'

const block = formesFixture()
const weapons = weaponIndex(block)

describe('socles', () => {
  it('compte les occupations et les prises nommées séparément', () => {
    expect(namedPickups(block)).toBe(9)
    expect(unnamedOccupations(block)).toBe(5)
    expect(weaponOccupations(block, '71ab0a2c')).toBe(4)
  })

  it('trie les armes par volume de prises nommées, la clé départageant', () => {
    // SPNKr et carabine sont à 3 prises nommées : la clé tranche (contrat stable,
    // jamais l'ordre d'arrivée d'une map). L'arme non cataloguée, à 1, ferme la
    // marche avec le S7 à 2.
    const sorted = weaponsByVolume(block).map((w) => w.key)
    expect(sorted).toEqual(['230447b1', '71ab0a2c', '0a1992bc', 'a1b2c3d4'])
  })

  it('compte les matchs où une arme avait un socle', () => {
    expect(matchesWithWeapon(block, '230447b1')).toBe(1)
    expect(matchesWithWeapon(block, '71ab0a2c')).toBe(1)
  })

  it('attribue les prises au bon joueur', () => {
    expect(weaponPickupsByPlayer(block, '71ab0a2c', FORMES_MAIN_XUID)).toBe(2)
    expect(weaponPickupsByPlayer(block, '0a1992bc', FORMES_MAIN_XUID)).toBe(0)
  })

  it('range les prises par famille d’arme du registre', () => {
    // Lourdes : SPNKr + S7. Moi : 2 prises (SPNKr). Mon camp : 2 + 1 = 3.
    // Lobby : 3 + 2 (l'adversaire prend un SPNKr et un S7) = 5.
    const heavy = aggregateWeaponClass(block, weapons, 'heavy')
    expect(heavy.me).toBe(2)
    expect(heavy.team).toBe(3)
    expect(heavy.lobby).toBe(5)
    // Précision : la carabine du match 2 — moi 1, mon camp 2, lobby 3.
    const precision = aggregateWeaponClass(block, weapons, 'precision')
    expect(precision.me).toBe(1)
    expect(precision.team).toBe(2)
    expect(precision.lobby).toBe(3)
    // Autres socles : l'arme non cataloguée, prise par un allié hors escouade.
    const other = aggregateWeaponClass(block, weapons, 'other')
    expect(other.me).toBe(0)
    expect(other.team).toBe(1)
  })
})

describe('objectifs', () => {
  it('ne retient que les matchs qui en portent un', () => {
    expect(objectiveMatches(block)).toHaveLength(1)
    expect(objectiveFamilies(block)).toEqual(['ctf'])
    expect(matchesOfFamily(block, 'ctf')).toHaveLength(1)
  })

  it('sert les colonnes publiées par le serveur, avec leur unité', () => {
    const cols = columnsOfFamily(block, 'ctf')
    expect(cols.map((c) => c.key)).toEqual([
      'flag_captures',
      'flag_returns',
      'time_as_flag_carrier_seconds',
    ])
    expect(cols[2].duration).toBe(true)
  })

  it('agrège une colonne sur les deux camps', () => {
    const agg = aggregateColumns(matchesOfFamily(block, 'ctf'), FORMES_MAIN_XUID, ['flag_returns'])
    // Retours : moi 2, mon camp 2, lobby 3.
    expect(agg.me).toBe(2)
    expect(agg.team).toBe(2)
    expect(agg.lobby).toBe(3)
    expect(agg.myShareOfTeamPct).toBeCloseTo(100, 6)
    expect(agg.teamShareOfLobbyPct).toBeCloseTo((2 / 3) * 100, 6)
  })

  it('réconcilie les modes par rôle', () => {
    const take = aggregateRole(block, 'take')
    expect(take.me).toBe(0)
    expect(take.team).toBe(3)
    expect(take.lobby).toBe(4)
    const hold = aggregateRole(block, 'hold')
    expect(hold.lobby).toBeCloseTo(77.4, 6)
  })

  it('sépare l’escouade, le reste du camp et l’adversaire par rôle', () => {
    const parts = roleLobbyParts(block, 'defend', [FORMES_MAIN_XUID])
    expect(parts.bySquad[FORMES_MAIN_XUID]).toBe(2)
    expect(parts.teamRest).toBe(0)
    expect(parts.opponents).toBe(1)
  })

  it('rend 0 pour un joueur absent de la feuille d’objectif', () => {
    const match = matchesOfFamily(block, 'ctf')[0]
    expect(objectiveCell(match, 'x-inconnu', { key: 'flag_returns', role: 'defend' })).toBe(0)
  })

  it('rend « non mesuré » (null), pas 0, pour une grandeur optionnelle absente', () => {
    // Une grandeur lue du film manque sur un match sans artefact : l’afficher à
    // zéro dirait « il n’a rien pris » là où la vérité est « on n’a pas regardé ».
    const match = matchesOfFamily(block, 'ctf')[0]
    const col = { key: 'flag_grabs_net', role: 'take', optional: true }
    expect(objectiveCell(match, FORMES_MAIN_XUID, col)).toBeNull()
    expect(objectiveCell(match, FORMES_MAIN_XUID, { ...col, optional: false })).toBe(0)
  })
})
