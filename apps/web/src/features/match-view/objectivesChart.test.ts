/**
 * Tests — objectivesChart (la projection des deux vues « Objectifs »).
 *
 * CE QU'ILS PROTÈGENT, et le premier point est LE point critique de la section :
 *   1. LES DEUX VUES SUIVENT LE MODE. Aucune grandeur n'est écrite en dur : les six modes à
 *      objectif du titre passent, de trois à cinq colonnes, avec le même code.
 *   2. L'AGRÉGAT D'ÉQUIPE RESPECTE `objectiveTeamTotal` : cumul, ou MAXIMUM pour un
 *      « meilleur temps » — additionner des meilleurs temps n'a aucun sens.
 *   3. CHAQUE LIGNE DU FACE-À-FACE A SON ÉCHELLE, zéro au centre : la longueur d'un côté dit
 *      l'avantage, pas la valeur absolue.
 *   4. `null` N'EST PAS ZÉRO, et une durée mesurée à zéro s'écrit « 0:00 » — le tiret est
 *      réservé au NON MESURÉ.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { objectiveColsFor, type ObjectiveMode } from './MatchScoreboard.logic'
import { buildObjectiveDuel, buildObjectiveGrid, objectiveValueText } from './objectivesChart'

/** Une ligne de scoreboard réduite à ce que la section lit. */
function ligne(
  gamertag: string,
  side: string,
  objective: Record<string, number | null> | null,
  isMe = false,
): MatchScoreboardRow {
  return { xuid: gamertag, gamertag, team_side: side, is_me: isMe, objective } as unknown as MatchScoreboardRow
}

/** Bastion (zones) : trois grandeurs, dont une durée. */
const ZONES = [
  ligne('Alpha', 't0', { zone_captures: 10, zone_secures: 6, time_in_zones_seconds: 128 }, true),
  ligne('Bravo', 't0', { zone_captures: 9, zone_secures: 1, time_in_zones_seconds: 112 }),
  ligne('Charlie', 't1', { zone_captures: 6, zone_secures: 0, time_in_zones_seconds: 83 }),
]

const VISUAL = {
  colLabel: (col: { key: string | number | symbol }) => `col:${String(col.key)}`,
  teamLabel: (side: string) => `Équipe ${side}`,
  teamColor: (side: string) => `var(--ac-team-${side === 't0' ? 'ally' : 'enemy'})`,
}

describe('objectiveValueText — une durée mesurée à zéro n’est pas une absence', () => {
  it('écrit « 0:00 » pour une durée nulle MESURÉE, et le nombre nu sinon', () => {
    expect(objectiveValueText(0, { key: 'time_in_zones_seconds', agg: 'sum', duration: true })).toBe(
      '0:00',
    )
    expect(objectiveValueText(128, { key: 'time_in_zones_seconds', agg: 'sum', duration: true })).toBe(
      '2:08',
    )
    expect(objectiveValueText(0, { key: 'zone_captures', agg: 'sum' })).toBe('0')
  })
})

describe('buildObjectiveGrid — la vue par joueur', () => {
  const grille = (rows = ZONES, mode: ObjectiveMode = 'zones') =>
    buildObjectiveGrid({
      rows,
      cols: objectiveColsFor(mode),
      ...VISUAL,
      tipFmt: (player, metric, value) => `${player} — ${metric} : ${value}`,
    })

  it('rend une colonne par grandeur du mode, chacune avec SON échelle', () => {
    const m = grille()
    expect(m.columns.map((c) => c.key)).toEqual([
      'zone_captures',
      'zone_secures',
      'time_in_zones_seconds',
    ])
    expect(m.columns[0].bound).toBe(10)
    expect(m.columns[1].bound).toBe(6)
    expect(m.columns[2].bound).toBe(150)
  })

  it('n’écrit PAS de total sur une colonne dont la somme n’a pas de sens (agrégat max)', () => {
    const oddball = grille(
      [
        ligne('Alpha', 't0', {
          skull_grabs: 2,
          time_as_skull_carrier_seconds: 40,
          longest_time_as_skull_carrier_seconds: 25,
        }),
      ],
      'oddball',
    )
    expect(oddball.columns.map((c) => c.totalText)).toEqual(['2', '0:40', null])
  })

  it('donne à chaque barre l’encre du CAMP du joueur, et sépare les camps', () => {
    const m = grille()
    expect(m.cells[0][0].color).toBe('var(--ac-team-ally)')
    expect(m.cells[2][0].color).toBe('var(--ac-team-enemy)')
    expect(m.separators).toEqual([2])
  })

  it('met en avant la ligne du joueur de la page', () => {
    expect(grille().rows.map((r) => r.emphasis === true)).toEqual([true, false, false])
  })

  it('écrit « — », jamais un zéro, pour une grandeur que la ligne ne porte pas', () => {
    const m = grille([ligne('Alpha', 't0', { zone_captures: 3 })])
    expect(m.cells[0][0]).toMatchObject({ value: 3, text: '3' })
    expect(m.cells[0][1]).toMatchObject({ value: null, text: '—', fraction: 0 })
  })

  it('encaisse LES SIX MODES à objectif sans une ligne de code par mode', () => {
    const modes: ObjectiveMode[] = ['ctf', 'zones', 'oddball', 'stockpile', 'extraction', 'vip']
    for (const mode of modes) {
      const cols = objectiveColsFor(mode)
      const m = buildObjectiveGrid({
        rows: [ligne('Alpha', 't0', Object.fromEntries(cols.map((c, i) => [String(c.key), i + 1])))],
        cols,
        ...VISUAL,
        tipFmt: (player, metric, value) => `${player} — ${metric} : ${value}`,
      })
      expect(m.columns).toHaveLength(cols.length)
      expect(m.cells[0]).toHaveLength(cols.length)
    }
    // Et la plus large des six (VIP) porte bien cinq grandeurs : la grille doit l'encaisser.
    expect(objectiveColsFor('vip')).toHaveLength(5)
  })
})

describe('buildObjectiveDuel — le face-à-face', () => {
  const duel = (rows = ZONES, mode: ObjectiveMode = 'zones') =>
    buildObjectiveDuel({
      rows,
      teams: ['t0', 't1'],
      cols: objectiveColsFor(mode),
      ...VISUAL,
      tipFmt: (team, metric, value) => `${team} — ${metric} : ${value}`,
    })

  it('cumule le camp sur une colonne « sum »', () => {
    const d = duel()
    expect(d[0].left.value).toBe(19)
    expect(d[0].right.value).toBe(6)
  })

  it('prend le MAXIMUM du camp sur un « meilleur temps », jamais la somme', () => {
    const d = duel(
      [
        ligne('Alpha', 't0', { skull_grabs: 1, longest_time_as_skull_carrier_seconds: 25 }),
        ligne('Bravo', 't0', { skull_grabs: 1, longest_time_as_skull_carrier_seconds: 40 }),
        ligne('Charlie', 't1', { skull_grabs: 3, longest_time_as_skull_carrier_seconds: 10 }),
      ],
      'oddball',
    )
    const meilleur = d.find((r) => r.key === 'longest_time_as_skull_carrier_seconds')!
    expect(meilleur.left.value).toBe(40)
    expect(meilleur.left.text).toBe('0:40')
  })

  it('donne à CHAQUE LIGNE son échelle : le plus grand des deux côtés remplit le sien', () => {
    const d = duel()
    expect(d[0].left.fraction).toBe(1)
    expect(d[0].right.fraction).toBeCloseTo(6 / 19)
    // Zones sécurisées : 7 contre 0 — l'avantage se lit malgré des nombres bien plus petits
    // que les captures, parce que la ligne ne partage pas leur échelle.
    expect(d[1].left.fraction).toBe(1)
    expect(d[1].right.fraction).toBe(0)
  })

  it('ne divise jamais par zéro quand les deux camps sont à zéro', () => {
    const d = duel([ligne('Alpha', 't0', { zone_captures: 0 }), ligne('Charlie', 't1', { zone_captures: 0 })])
    expect(d[0].left.fraction).toBe(0)
    expect(d[0].right.fraction).toBe(0)
  })

  it('écrit « — » pour un camp dont aucune ligne ne porte la grandeur', () => {
    const d = duel([
      ligne('Alpha', 't0', { zone_captures: 4 }),
      ligne('Charlie', 't1', { zone_captures: 2 }),
    ])
    expect(d[1].left).toMatchObject({ value: null, text: '—', fraction: 0 })
  })

  it('porte l’équipe, la grandeur et la valeur écrite dans son infobulle', () => {
    expect(duel()[0].left.tooltip).toBe('Équipe t0 — col:zone_captures : 19')
  })
})
