/**
 * Tests — equipmentUsageChart (la projection de la grille d'usage d'équipements).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. LA COULEUR SUIT LA FAMILLE, JAMAIS SON RANG. Une famille absente d'un match ne doit pas
 *      repeindre les autres — c'est ce qui rend deux matchs comparables à l'œil.
 *   2. L'ORDRE DES LIGNES DE LA GRILLE est celui des camps puis du roster — un joueur se lit en
 *      ligne d'une colonne à l'autre.
 */
import { describe, expect, it } from 'vitest'

import {
  USAGE_GROUP_TOKENS,
  buildUsageGrid,
  usageGroupColor,
  usageLeaves,
} from './equipmentUsageChart'
import type { UsageColumnGroup } from './equipmentUsageColumns'
import type { EquipmentUsageTally, EquipmentUsageTeam } from './equipmentUsageLogic'
import type { ReplayCamp } from '../../../lib/replay/replayCamps'

function tally(over: Partial<EquipmentUsageTally> = {}): EquipmentUsageTally {
  return {
    grapplePulls: 0,
    episodes: {},
    deployed: {},
    dropped: {},
    spent: {},
    grenades: {},
    kept: {},
    ...over,
  }
}

const ALPHA = tally({
  grapplePulls: 2,
  episodes: { camo: { count: 1, ms: 5000, kills: 0 } },
  grenades: { 0: 4 },
})
const BRAVO = tally({ grapplePulls: 1, grenades: { 0: 6 } })
const CHARLIE = tally({ grapplePulls: 1 })

/** Deux camps du film : le 0 (nommé `t0` par la feuille) et le 1 (`t1`). */
const TEAMS: EquipmentUsageTeam[] = [
  {
    team: 0,
    side: 't0',
    players: [
      { ...ALPHA, xuid: 'a1', name: 'Alpha', team: 0, side: 't0' },
      { ...BRAVO, xuid: 'a2', name: 'Bravo', team: 0, side: 't0' },
    ],
    total: tally({
      grapplePulls: 3,
      episodes: { camo: { count: 1, ms: 5000, kills: 0 } },
      deployed: { wall: 10 },
    }),
  },
  {
    team: 1,
    side: 't1',
    players: [{ ...CHARLIE, xuid: 'b1', name: 'Charlie', team: 1, side: 't1' }],
    total: tally({ grapplePulls: 1 }),
  },
]

const GROUPS: UsageColumnGroup[] = [
  {
    key: 'grapple',
    columns: [
      {
        key: 'pulls',
        label: 'Grappin',
        value: (x) => x.grapplePulls,
        format: (v) => String(v),
      },
    ],
  },
  {
    key: 'equipment',
    columns: [
      {
        key: 'camo.count',
        label: 'Camouflage',
        value: (x) => x.episodes.camo?.count ?? 0,
        format: (v) => String(v),
      },
      {
        key: 'camo.kills',
        label: 'Frags sous camo',
        // Jointure non tentée : NON MESURÉ, pas zéro.
        value: () => null,
        format: (v) => String(v),
      },
      {
        key: 'wall',
        label: 'Mur de protection',
        value: (x) => x.deployed.wall ?? 0,
        format: (v) => String(v),
      },
    ],
  },
]

const VISUAL = {
  teamLabel: (camp: ReplayCamp) => `Équipe ${camp.side ?? camp.team}`,
  teamAccent: (camp: ReplayCamp) => `var(--ac-team-${camp.side === 't0' ? 'ally' : 'enemy'})`,
}

describe('l’encre d’une famille de geste', () => {
  it('donne une teinte DIFFÉRENTE à chacune des deux familles (E2 : deployed+dropped -> equipment)', () => {
    const encres = Object.values(USAGE_GROUP_TOKENS)
    expect(new Set(encres).size).toBe(encres.length)
  })

  it('rend une variable CSS de jeton, jamais un hex', () => {
    expect(usageGroupColor('equipment')).toMatch(/^var\(--ac-[a-z-]+\)$/)
  })

  it('suit la FAMILLE et pas son rang : une famille absente ne repeint pas les autres', () => {
    const complet = usageLeaves(GROUPS)
    const sansGrappin = usageLeaves(GROUPS.slice(1))
    const encre = (leaves: ReturnType<typeof usageLeaves>, label: string) =>
      usageGroupColor(leaves.find((l) => l.column.label === label)!.group)
    expect(encre(sansGrappin, 'Mur de protection')).toBe(encre(complet, 'Mur de protection'))
  })
})

describe('buildUsageGrid — la grille par joueur', () => {
  const grille = () =>
    buildUsageGrid({
      teams: TEAMS,
      groups: GROUPS,
      meXUID: 'a1',
      ...VISUAL,
      tipFmt: (player, column, value) => `${player} — ${column} : ${value}`,
    })

  it('range les joueurs camp par camp, dans l’ordre du roster, et sépare les camps', () => {
    const m = grille()
    expect(m.rows.map((r) => r.label)).toEqual(['Alpha', 'Bravo', 'Charlie'])
    expect(m.separators).toEqual([2])
  })

  it('le filet suit le CAMP DU FILM : deux camps que la feuille ne nomme pas restent deux groupes', () => {
    const sansFeuille = TEAMS.map((team) => ({
      ...team,
      side: null,
      players: team.players.map((p) => ({ ...p, side: null })),
    }))
    const m = buildUsageGrid({
      teams: sansFeuille,
      groups: GROUPS,
      meXUID: null,
      ...VISUAL,
      tipFmt: (player, column, value) => `${player} — ${column} : ${value}`,
    })
    expect(m.separators).toEqual([2])
    expect(m.rows.map((r) => r.hint)).toEqual(['Alpha — Équipe 0', 'Bravo — Équipe 0', 'Charlie — Équipe 1'])
  })

  it('met en avant la ligne du joueur de la page, et elle seule', () => {
    expect(grille().rows.map((r) => r.emphasis === true)).toEqual([true, false, false])
  })

  it('donne à chaque colonne l’encre de SA famille, la même sur toute la colonne', () => {
    const m = grille()
    expect(m.cells.map((row) => row[0].color)).toEqual([
      usageGroupColor('grapple'),
      usageGroupColor('grapple'),
      usageGroupColor('grapple'),
    ])
    expect(m.cells[0][3].color).toBe(usageGroupColor('equipment'))
  })

  it('laisse vide une colonne NON MESURÉE, sans la confondre avec un zéro', () => {
    const m = grille()
    expect(m.cells[0][2]).toMatchObject({ value: null, text: '—', fraction: 0 })
    expect(m.cells[2][3]).toMatchObject({ value: 0, text: '0' })
  })
})
