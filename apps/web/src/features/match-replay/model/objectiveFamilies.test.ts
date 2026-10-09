/**
 * Tests — objectiveFamilies : la famille d'objectif d'une statistique nommée.
 *
 * CE QU'ILS PROTÈGENT : les deux statistiques SANS famille que `doc.objectives` transporte
 * (`kills`, `assists`) restent hors des familles, et le préfixe exige son séparateur — un
 * nom qui commence par les mêmes lettres n'est pas de la famille.
 */
import { describe, expect, it } from 'vitest'

import {
  isObjectiveFamilyStat,
  OBJECTIVE_FAMILIES,
  objectiveFamilyOf,
} from './objectiveFamilies'

describe('objectiveFamilyOf', () => {
  it('nomme la famille de chaque statistique d’objectif servie par le décodeur', () => {
    // Les noms canoniques de `objectiveevents/named.go` (constantes Stat*), famille par famille.
    const attendu: Record<string, string> = {
      flag_captures: 'flag',
      flag_capture_assists: 'flag',
      flag_grabs: 'flag',
      flag_secures: 'flag',
      flag_returns: 'flag',
      flag_steals: 'flag',
      flag_carriers_killed: 'flag',
      zone_captures: 'zone',
      zone_secures: 'zone',
      vip_selected: 'vip',
      bomb_detonations: 'bomb',
      skull_grabs: 'skull',
    }
    for (const [stat, famille] of Object.entries(attendu)) {
      expect(objectiveFamilyOf(stat), stat).toBe(famille)
    }
  })

  it('les deux statistiques HORS objectif du statborg n’ont aucune famille', () => {
    // `kills` est l'ancre d'identité du balayage, `assists` son voisin de contrôle croisé :
    // publiées dans `doc.objectives`, elles ne sont pas des objectifs.
    expect(objectiveFamilyOf('kills')).toBeNull()
    expect(objectiveFamilyOf('assists')).toBeNull()
    expect(isObjectiveFamilyStat('kills')).toBe(false)
    expect(isObjectiveFamilyStat('assists')).toBe(false)
  })

  it('le séparateur est exigé : un nom qui commence par les mêmes lettres ne suffit pas', () => {
    expect(objectiveFamilyOf('flagrant')).toBeNull()
    expect(objectiveFamilyOf('zones')).toBeNull()
    expect(objectiveFamilyOf('')).toBeNull()
  })

  it('les six familles sont celles des ObjectiveType* du serveur, et elles se résolvent', () => {
    expect([...OBJECTIVE_FAMILIES]).toEqual(['flag', 'zone', 'hill', 'skull', 'vip', 'bomb'])
    for (const famille of OBJECTIVE_FAMILIES) {
      expect(objectiveFamilyOf(`${famille}_quoi_que_ce_soit`)).toBe(famille)
    }
  })
})
