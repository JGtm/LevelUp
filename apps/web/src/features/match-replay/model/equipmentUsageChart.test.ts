/**
 * Tests — equipmentUsageChart (la projection des deux vues du bilan d'équipement).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. LA COULEUR SUIT LA FAMILLE, JAMAIS SON RANG. Une famille absente d'un match ne doit pas
 *      repeindre les autres — c'est ce qui rend deux matchs comparables à l'œil.
 *   2. LA VUE DES PARTS SUIT LA GRILLE : une ligne par COLONNE, même liste et même ordre, et
 *      toutes les lignes sur UNE échelle commune bornée par le plus gros total (D20, 5.A).
 *   3. UNE FAMILLE QU'AUCUN CAMP N'A EMPLOYÉE N'A PAS DE LIGNE : une barre vide n'est pas une
 *      part.
 *   4. L'ORDRE DES LIGNES DE LA GRILLE est celui des camps puis du roster — un joueur se lit en
 *      ligne d'une colonne à l'autre.
 */
import { describe, expect, it } from 'vitest'

import {
  USAGE_GROUP_TOKENS,
  buildUsageGrid,
  buildUsageFamilyBars,
  usageGroupColor,
  usageLeaves,
} from './equipmentUsageChart'
import type { UsageColumnGroup } from './equipmentUsageColumns'
import type { EquipmentUsageTally, EquipmentUsageTeam } from './equipmentUsageLogic'

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
      deployed: { wall: 10 },
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
    key: 'equipment',
    label: 'Équipement',
    hint: 'réserve équipement',
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
  teamLabel: (side: string | null) => `Équipe ${side ?? 'inconnue'}`,
  teamAccent: (side: string | null) => `var(--ac-team-${side === 't0' ? 'ally' : 'enemy'})`,
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

describe('buildUsageFamilyBars — la part de chaque équipe (5.A, 2026-09-21)', () => {
  const barres = (allySide: string | null = 't0') =>
    buildUsageFamilyBars({ teams: TEAMS, groups: GROUPS, allySide, ...VISUAL })
  /** La clé d'une ligne = celle de la colonne de la grille : groupe puis colonne. */
  const cle = (group: string, column: string) => `${group}.${column}`
  const ligne = (group: string, column: string) =>
    barres().rows.find((r) => r.key === cle(group, column))!

  it('rend UNE ligne par colonne de la grille, dans le MÊME ordre — pas une par groupe', () => {
    expect(barres().rows.map((r) => r.key)).toEqual([
      cle('grapple', 'pulls'),
      cle('equipment', 'camo.count'),
      cle('equipment', 'wall'),
    ])
    expect(barres().rows.map((r) => r.label)).toEqual(['Grappin', 'Camouflage', 'Mur de protection'])
  })

  it('met toutes les lignes sur UNE échelle commune, bornée par le plus gros total', () => {
    const grappin = ligne('grapple', 'pulls')
    expect(grappin.total).toBe(4)
    // 3 tractions sur une borne de 10 : la piste est remplie à 30 %, pas à 75 %.
    expect(grappin.segments.map((s) => s.widthPct)).toEqual([30, 10])
    expect(ligne('equipment', 'wall').segments[0].widthPct).toBe(100)
  })

  it('garde le pourcentage DE LA FAMILLE, pour l’infobulle et elle seule', () => {
    const grappin = ligne('grapple', 'pulls')
    expect(grappin.segments.map((s) => [s.label, s.count, s.percent])).toEqual([
      ['Équipe t0', 3, 75],
      ['Équipe t1', 1, 25],
    ])
  })

  it('ouvre chaque barre par MON camp, quel que soit l’ordre du film', () => {
    expect(barres('t1').rows[0].segments.map((s) => s.side)).toEqual(['t1', 't0'])
    expect(barres('t1').legend.map((l) => l.side)).toEqual(['t1', 't0'])
    // Sans camp connu, l'ordre du film reste — aucune des deux encres n'est « la mienne ».
    expect(barres(null).rows[0].segments.map((s) => s.side)).toEqual(['t0', 't1'])
  })

  it('n’écrit aucun segment pour un camp qui n’a rien fait de cette famille', () => {
    expect(ligne('equipment', 'wall').segments.map((s) => s.count)).toEqual([10])
  })

  it('ne rend aucune ligne pour une famille qu’aucun camp n’a employée', () => {
    // `camo.kills` n'est pas mesurée (value -> null) : elle n'a pas de ligne, et une piste
    // vide n'apparaît jamais.
    expect(barres().rows.some((r) => r.key === cle('equipment', 'camo.kills'))).toBe(false)
    const vide = buildUsageFamilyBars({
      teams: TEAMS.map((team) => ({ ...team, total: tally() })),
      groups: GROUPS,
      allySide: 't0',
      ...VISUAL,
    })
    expect(vide.rows).toEqual([])
  })

  it('porte la réserve de mesure du GROUPE sur chacune de ses lignes', () => {
    expect(ligne('equipment', 'wall').hint).toBe('réserve équipement')
    expect(barres().rows[0].hint).toBe('réserve grappin')
  })
})
