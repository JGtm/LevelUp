/**
 * Tests — equipmentUsageChart (la projection des deux vues du bilan d'équipement).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. LA COULEUR SUIT LA FAMILLE, JAMAIS SON RANG. Une famille absente d'un match ne doit pas
 *      repeindre les autres — c'est ce qui rend deux matchs comparables à l'œil.
 *   2. LA PART D'ÉQUIPE SE COMPTE EN GESTES : une famille peut porter plusieurs colonnes
 *      d'unités différentes (épisodes, durée, frags), et les additionner n'aurait aucun sens.
 *   3. UNE FAMILLE QU'AUCUN CAMP N'A EMPLOYÉE N'A PAS DE LIGNE : une barre vide n'est pas une
 *      part.
 *   4. L'ORDRE DES LIGNES DE LA GRILLE est celui des camps puis du roster — un joueur se lit en
 *      ligne d'une colonne à l'autre.
 */
import { describe, expect, it } from 'vitest'

import {
  USAGE_GROUP_TOKENS,
  buildUsageGrid,
  buildUsageShares,
  usageGestureCount,
  usageGroupColor,
  usageLeaves,
} from './equipmentUsageChart'
import type { UsageColumnGroup } from './equipmentUsageColumns'
import type { EquipmentUsageTally, EquipmentUsageTeam } from './equipmentUsageLogic'

function tally(over: Partial<EquipmentUsageTally> = {}): EquipmentUsageTally {
  return { grapplePulls: 0, episodes: {}, deployed: {}, dropped: {}, grenades: {}, ...over }
}

const ALPHA = tally({
  grapplePulls: 2,
  episodes: { camo: { count: 1, ms: 5000, kills: 0 } },
  grenades: { 0: 4 },
})
const BRAVO = tally({ grapplePulls: 1, grenades: { 0: 6 } })
const CHARLIE = tally({ grapplePulls: 1 })

const TEAMS: EquipmentUsageTeam[] = [
  {
    side: 't0',
    players: [
      { ...ALPHA, xuid: 'a1', name: 'Alpha', side: 't0' },
      { ...BRAVO, xuid: 'a2', name: 'Bravo', side: 't0' },
    ],
    total: tally({
      grapplePulls: 3,
      episodes: { camo: { count: 1, ms: 5000, kills: 0 } },
      grenades: { 0: 10 },
    }),
  },
  {
    side: 't1',
    players: [{ ...CHARLIE, xuid: 'b1', name: 'Charlie', side: 't1' }],
    total: tally({ grapplePulls: 1 }),
  },
]

const GROUPS: UsageColumnGroup[] = [
  {
    key: 'grapple',
    label: 'Grappin',
    hint: 'réserve grappin',
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
    key: 'episodes',
    label: 'États actifs',
    hint: 'réserve états actifs',
    columns: [
      {
        key: 'camo.count',
        label: 'Camouflage (épisodes)',
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
    ],
  },
  {
    key: 'grenades',
    label: 'Grenades lancées',
    hint: 'réserve grenades',
    columns: [
      {
        key: 'grenade.0',
        label: 'Fragmentation',
        value: (x) => x.grenades[0] ?? 0,
        format: (v) => String(v),
      },
    ],
  },
]

const VISUAL = {
  teamLabel: (side: string | null) => `Équipe ${side ?? 'inconnue'}`,
  teamAccent: (side: string | null) => `var(--ac-team-${side === 't0' ? 'ally' : 'enemy'})`,
}

describe('l’encre d’une famille de geste', () => {
  it('donne une teinte DIFFÉRENTE à chacune des cinq familles', () => {
    const encres = Object.values(USAGE_GROUP_TOKENS)
    expect(new Set(encres).size).toBe(encres.length)
  })

  it('rend une variable CSS de jeton, jamais un hex', () => {
    expect(usageGroupColor('grenades')).toMatch(/^var\(--ac-[a-z-]+\)$/)
  })

  it('suit la FAMILLE et pas son rang : une famille absente ne repeint pas les autres', () => {
    const complet = usageLeaves(GROUPS)
    const sansGrappin = usageLeaves(GROUPS.slice(1))
    const encre = (leaves: ReturnType<typeof usageLeaves>, label: string) =>
      usageGroupColor(leaves.find((l) => l.column.label === label)!.group)
    expect(encre(sansGrappin, 'Fragmentation')).toBe(encre(complet, 'Fragmentation'))
  })
})

describe('usageGestureCount — la part se compte en GESTES', () => {
  it('compte un épisode d’état actif pour UN geste, sans y ajouter sa durée ni ses frags', () => {
    expect(usageGestureCount(ALPHA, 'episodes')).toBe(1)
  })

  it('somme les familles d’une même catégorie de pose', () => {
    expect(usageGestureCount(tally({ deployed: { wall: 2, sensor: 1 } }), 'deployed')).toBe(3)
    expect(usageGestureCount(ALPHA, 'grenades')).toBe(4)
    expect(usageGestureCount(ALPHA, 'grapple')).toBe(2)
    expect(usageGestureCount(ALPHA, 'dropped')).toBe(0)
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
    expect(m.cells[0][3].color).toBe(usageGroupColor('grenades'))
  })

  it('laisse vide une colonne NON MESURÉE, sans la confondre avec un zéro', () => {
    const m = grille()
    expect(m.cells[0][2]).toMatchObject({ value: null, text: '—', fraction: 0 })
    expect(m.cells[2][3]).toMatchObject({ value: 0, text: '0' })
  })
})

describe('buildUsageShares — la part de chaque équipe', () => {
  it('somme les gestes du CAMP, et écrit compte brut et pourcentage', () => {
    const lignes = buildUsageShares({ teams: TEAMS, groups: GROUPS, ...VISUAL })
    const grappin = lignes.find((l) => l.key === 'grapple')!
    expect(grappin.total).toBe(4)
    expect(grappin.segments.map((s) => [s.label, s.count, s.percent])).toEqual([
      ['Équipe t0', 3, 75],
      ['Équipe t1', 1, 25],
    ])
  })

  it('n’écrit aucun segment pour un camp qui n’a rien fait de cette famille', () => {
    const lignes = buildUsageShares({ teams: TEAMS, groups: GROUPS, ...VISUAL })
    expect(lignes.find((l) => l.key === 'grenades')!.segments.map((s) => s.count)).toEqual([10])
  })

  it('ne rend aucune ligne pour une famille qu’aucun camp n’a employée', () => {
    const lignes = buildUsageShares({
      teams: TEAMS.map((team) => ({ ...team, total: tally() })),
      groups: GROUPS,
      ...VISUAL,
    })
    expect(lignes).toEqual([])
  })

  it('porte la réserve de mesure de la famille, pour que son nom puisse la dire', () => {
    const lignes = buildUsageShares({ teams: TEAMS, groups: GROUPS, ...VISUAL })
    expect(lignes.find((l) => l.key === 'episodes')!.hint).toBe('réserve états actifs')
  })
})
