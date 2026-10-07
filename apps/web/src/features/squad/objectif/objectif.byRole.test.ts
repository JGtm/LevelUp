/**
 * objectif.byRole.test.ts — le rapport de force PAR RÔLE (vue compacte du tiroir de comparaison de
 * Sessions, maquette `makeBalance` avec `cp`) : par famille, la somme des actions de chaque rôle
 * (Tenir en durée), camp contre camp, puis les colonnes facultatives (prises nettes de drapeau) en
 * ligne à part. Témoin : la soirée du 07/09 (relevés `.ai/V7.5/MESURES_SESSIONS_2026-10-06.md` §3).
 */
import { describe, expect, it } from 'vitest'

import { block0709 } from './objectif.fixtures'
import { buildBalanceByRole, buildObjectiveBalance } from './objectif.logic'

const pct = (share: number) => Math.round(share * 1000) / 10

describe('buildBalanceByRole — 07/09', () => {
  const fams = buildBalanceByRole(buildObjectiveBalance(block0709()))

  it('Bases : prendre 93-109 (46 %), défendre 30-60 (33,3 %), tenir en durée (53,6 %)', () => {
    const bases = fams.find((f) => f.family === 'zones_strongholds')!
    expect(bases.lines.map((l) => [l.key, l.us, l.them, pct(l.share)])).toEqual([
      ['take', 93, 109, 46],
      ['defend', 30, 60, 33.3],
      ['hold', bases.lines[2].us, bases.lines[2].them, 53.6],
    ])
    expect(bases.lines[2].duration).toBe(true)
  })

  it('Drapeau : les prises nettes à part (29-41, 41,4 %), hors de la somme de « prendre »', () => {
    const ctf = fams.find((f) => f.family === 'ctf')!
    const take = ctf.lines.find((l) => l.key === 'take')!
    expect([take.us, take.us + take.them]).toEqual([1 + 1 + 15 + 0, 10 + 8 + 31 + 6])
    const net = ctf.lines.find((l) => l.key === 'flag_grabs_net')!
    expect([net.us, net.them, pct(net.share)]).toEqual([29, 41, 41.4])
    expect(net.role).toBeNull()
  })
})
