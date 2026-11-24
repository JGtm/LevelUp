/**
 * Tests — les éliminations liées à l'objectif sur la fiche de match.
 *
 * Le serveur sert déjà, par joueur, les éliminations de porteurs, de rapatrieurs, de défenseurs et
 * d'attaquants de zone, de porteurs de crâne et les aides sur le VIP. La section « Objectifs » les
 * affiche avec la grandeur de leur mode, comme l'onglet Objectif de l'Escouade les exploite déjà.
 *
 * CE QU'ILS PROTÈGENT :
 *   1. chaque mode porte ses grandeurs d'élimination, à leur place dans l'ordre du mode ;
 *   2. une grandeur d'élimination est un COMPTE (ni durée, ni maximum) : le total d'équipe est
 *      la somme ;
 *   3. chaque colonne de chaque mode a son libellé et son infobulle en français ET en anglais ;
 *   4. une valeur absente reste « — », jamais zéro.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { MATCH_VIEW_TEXT } from './i18n'
import {
  objectiveColsFor,
  objectiveTeamTotal,
  type ObjectiveMode,
} from './MatchScoreboard.logic'
import { buildObjectiveGrid } from './objectivesChart'

const MODES: ObjectiveMode[] = ['ctf', 'zones', 'oddball', 'stockpile', 'extraction', 'vip', 'bomb']

const cles = (mode: ObjectiveMode) => objectiveColsFor(mode).map((c) => String(c.key))

function ligne(gamertag: string, side: string, objective: Record<string, number | null>): MatchScoreboardRow {
  return { xuid: gamertag, gamertag, team_side: side, is_me: false, objective } as unknown as MatchScoreboardRow
}

describe('éliminations liées à l’objectif — colonnes par mode', () => {
  it('drapeau : porteurs et rapatrieurs éliminés, avant le temps de portage', () => {
    expect(cles('ctf')).toEqual([
      'flag_captures',
      'flag_returns',
      'flag_steals',
      'flag_carriers_killed',
      'flag_returners_killed',
      'time_as_flag_carrier_seconds',
    ])
  })

  it('zones : éliminations en attaque puis en défense, avant le temps en zone', () => {
    expect(cles('zones')).toEqual([
      'zone_captures',
      'zone_secures',
      'zone_offensive_kills',
      'zone_defensive_kills',
      'time_in_zones_seconds',
    ])
  })

  it('crâne : porteurs éliminés', () => {
    expect(cles('oddball')).toContain('skull_carriers_killed')
  })

  it('VIP : aides sur le VIP adverse, juste après les VIP abattus', () => {
    const vip = cles('vip')
    expect(vip.indexOf('vip_assists')).toBe(vip.indexOf('vip_kills') + 1)
  })

  it('compte les éliminations (somme d’équipe, ni durée ni maximum)', () => {
    for (const [mode, key] of [
      ['ctf', 'flag_carriers_killed'],
      ['ctf', 'flag_returners_killed'],
      ['zones', 'zone_offensive_kills'],
      ['zones', 'zone_defensive_kills'],
      ['oddball', 'skull_carriers_killed'],
      ['vip', 'vip_assists'],
    ] as const) {
      const col = objectiveColsFor(mode).find((c) => String(c.key) === key)
      expect(col, key).toMatchObject({ agg: 'sum' })
      expect(col?.duration, key).toBeFalsy()
    }
    const col = objectiveColsFor('zones').find((c) => c.key === 'zone_offensive_kills')!
    const rows = [
      ligne('Alpha', 't0', { zone_offensive_kills: 3 }),
      ligne('Bravo', 't0', { zone_offensive_kills: 2 }),
    ]
    expect(objectiveTeamTotal(rows, col)).toBe(5)
  })

  it('écrit « — », jamais un zéro, pour une élimination non mesurée', () => {
    const cols = objectiveColsFor('ctf')
    const m = buildObjectiveGrid({
      rows: [ligne('Alpha', 't0', { flag_captures: 1, flag_carriers_killed: 0 })],
      cols,
      colLabel: (c) => String(c.key),
      teamLabel: (s) => s,
      teamColor: () => 'currentColor',
      tipFmt: (p, metric, v) => `${p} ${metric} ${v}`,
    })
    const i = (key: string) => cols.findIndex((c) => String(c.key) === key)
    expect(m.cells[0][i('flag_carriers_killed')]).toMatchObject({ value: 0, text: '0' })
    expect(m.cells[0][i('flag_returners_killed')]).toMatchObject({ value: null, text: '—' })
  })
})

describe('éliminations liées à l’objectif — libellés en toutes langues', () => {
  it.each(Object.keys(MATCH_VIEW_TEXT))('%s : chaque colonne de chaque mode est nommée', (locale) => {
    const text = MATCH_VIEW_TEXT[locale as keyof typeof MATCH_VIEW_TEXT]
    for (const mode of MODES) {
      for (const key of cles(mode)) {
        const entry = text.objectives.cols[key]
        expect(entry?.label, `${locale} ${key} label`).toBeTruthy()
        expect(entry?.tooltip, `${locale} ${key} tooltip`).toBeTruthy()
      }
    }
  })
})
