/**
 * usageEquipmentPartiesModel.test.ts — les DEUX donuts (décisions P10/P11,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.9/E6.3) : un anneau, des parts EXCLUSIVES
 * qui font 100 % du lobby, contiguës moi → mes amis → reste de mon équipe → eux, avec
 * les deux sous-totaux emboîtés.
 */
import { describe, expect, it } from 'vitest'

import type { EquipmentUsageParties, SessionUsageSquadPlayer } from '@/lib/api/types'

import { buildPartiesDonutModel } from './usageEquipmentPartiesModel'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('buildPartiesDonutModel — P10/P11', () => {
  it('absent (parties non fournies) : pas de donut, jamais un anneau a zero', () => {
    expect(buildPartiesDonutModel(undefined, [], 'objets pris dans le lobby', t, 'fr')).toBeNull()
  })

  it('lobby_total <= 0 : pas de donut', () => {
    const parties: EquipmentUsageParties = {
      lobby_total: 0,
      player: 0,
      friends: 0,
      rest_of_team: 0,
      opponents: 0,
    }
    expect(buildPartiesDonutModel(parties, [], 'objets pris dans le lobby', t, 'fr')).toBeNull()
  })

  it('solo (aucun ami suivi) : trois parts, un seul sous-total « mon equipe »', () => {
    const parties: EquipmentUsageParties = {
      lobby_total: 100,
      player: 20,
      friends: 0,
      rest_of_team: 32,
      opponents: 48,
    }
    const model = buildPartiesDonutModel(parties, [], 'objets pris dans le lobby', t, 'fr')!
    expect(model).not.toBeNull()
    expect(model.series[0].datapoints.map((p) => p.name)).toEqual(['Moi', 'Reste de mon équipe', 'Eux (anonyme)'])
    expect(model.subtotals).toHaveLength(1)
    expect(model.subtotals[0].label).toBe('Mon équipe')
    // (20 + 32) / 100 = 52 %
    expect(model.subtotals[0].value).toBe('52,0 %')
    expect(model.centerValue).toBe('100')
    expect(model.centerLabel).toBe('objets pris dans le lobby')
  })

  it('escouade (amis suivis) : parts par ami, couleurs squad-player-2.., deux sous-totaux', () => {
    const parties: EquipmentUsageParties = {
      lobby_total: 892,
      player: 143,
      friends: 223,
      rest_of_team: 116,
      opponents: 410,
      by_friend: [
        { xuid: 'f1', value: 150 },
        { xuid: 'f2', value: 73 },
      ],
    }
    const trackedPlayers: SessionUsageSquadPlayer[] = [
      { xuid: 'f1', gamertag: 'Madina' },
      { xuid: 'f2', gamertag: 'Choco' },
    ]
    const model = buildPartiesDonutModel(parties, trackedPlayers, 'objets pris dans le lobby', t, 'fr')!
    expect(model.series[0].datapoints.map((p) => p.name)).toEqual([
      'Moi',
      'Madina',
      'Choco',
      'Reste de mon équipe',
      'Eux (anonyme)',
    ])
    expect(model.sliceColors.Madina).toBe('squad-player-2')
    expect(model.sliceColors.Choco).toBe('squad-player-3')
    expect(model.sliceColors.Moi).toBe('squad-player-1')
    expect(model.sliceColors['Reste de mon équipe']).toBe('team-ally')
    expect(model.sliceColors['Eux (anonyme)']).toBe('team-enemy')
    expect(model.subtotals).toHaveLength(2)
    expect(model.subtotals[0].label).toBe('Mon escouade')
    // (143 + 223) / 892 = 41.03 %
    expect(model.subtotals[0].value).toBe('41,0 %')
    expect(model.subtotals[1].label).toBe('Mon équipe')
    // (143 + 223 + 116) / 892 = 54.03 %
    expect(model.subtotals[1].value).toBe('54,0 %')
  })

  it('un ami suivi absent de by_friend (0) : pas de part pour lui, pas de crash', () => {
    const parties: EquipmentUsageParties = {
      lobby_total: 50,
      player: 20,
      friends: 0,
      rest_of_team: 10,
      opponents: 20,
      by_friend: [],
    }
    const trackedPlayers: SessionUsageSquadPlayer[] = [{ xuid: 'f1', gamertag: 'Madina' }]
    const model = buildPartiesDonutModel(parties, trackedPlayers, 'objets pris dans le lobby', t, 'fr')!
    expect(model.series[0].datapoints.map((p) => p.name)).toEqual(['Moi', 'Reste de mon équipe', 'Eux (anonyme)'])
    // Aucun ami avec value>0 : un seul sous-total, comme le cas solo.
    expect(model.subtotals).toHaveLength(1)
  })

  it('chaque point porte un valueLabel formate (compte brut, jamais un pourcentage)', () => {
    const parties: EquipmentUsageParties = {
      lobby_total: 100,
      player: 20,
      friends: 0,
      rest_of_team: 32,
      opponents: 48,
    }
    const model = buildPartiesDonutModel(parties, [], 'objets pris dans le lobby', t, 'fr')!
    expect(model.series[0].datapoints[0].valueLabel).toBe('20')
  })
})
