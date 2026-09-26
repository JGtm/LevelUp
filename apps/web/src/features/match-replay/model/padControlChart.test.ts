/**
 * Tests — padControlChart (la projection du contrôle des armes spéciales).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. UNE ARME, UNE BARRE : le rail vaut 100 % des occupations NOMMÉES de ce socle, et le
 *      total de la ligne est publié à côté du nom de l'arme.
 *   2. LA TEINTE D'UN JOUEUR EST STABLE D'UNE LIGNE À L'AUTRE : elle suit son rang DANS LE CAMP,
 *      pas son rang parmi les preneurs de CE socle.
 *   3. CE QUI N'A PAS DE RAMASSEUR NOMMÉ N'EST VERSÉ À AUCUN CAMP — il ressort en annotation.
 *   4. L'ORDRE DES ARMES est celui de `padControlLogic` ; l'ordre des camps est celui que
 *      l'appelant demande (le camp du joueur de la page en haut).
 */
import { describe, expect, it } from 'vitest'

import { buildPadControlBars, padTint } from './padControlChart'
import type { PadControl } from './padControlLogic'

const SNIPER = 'sniper'
const ROCKET = 'rocket'

/** Deux camps de deux joueurs, deux socles, et une occupation sans nom sur le lance-roquettes. */
const CONTROL = {
  weapons: [SNIPER, ROCKET],
  byTeam: [
    {
      side: 't1',
      players: [
        { xuid: 'b1', name: 'Charlie', side: 't1', total: 3, byWeapon: { [SNIPER]: 3 } },
        { xuid: 'b2', name: 'Delta', side: 't1', total: 1, byWeapon: { [ROCKET]: 1 } },
      ],
      total: { total: 4, byWeapon: { [SNIPER]: 3, [ROCKET]: 1 } },
    },
    {
      side: 't0',
      players: [
        { xuid: 'a1', name: 'Alpha', side: 't0', total: 2, byWeapon: { [SNIPER]: 1, [ROCKET]: 1 } },
        { xuid: 'a2', name: 'Bravo', side: 't0', total: 0, byWeapon: {} },
      ],
      total: { total: 2, byWeapon: { [SNIPER]: 1, [ROCKET]: 1 } },
    },
  ],
  coverage: null,
  attributed: 6,
  unjoined: 0,
  unnamedByWeapon: { [ROCKET]: 2 },
  hasData: true,
} as unknown as PadControl

function barres() {
  return buildPadControlBars({
    control: CONTROL,
    weaponLabel: (w) => (w === SNIPER ? 'S7 Sniper' : 'M41 SPNKr'),
    teamLabel: (side) => `Équipe ${side ?? 'inconnue'}`,
    teamColor: (side) => `var(--ac-team-${side === 't0' ? 'ally' : 'enemy'})`,
    // Le camp du joueur de la page (t0) en haut, l'adverse ensuite.
    teamRank: (side) => (side === 't0' ? 0 : 1),
  })
}

describe('padTint — l’éclaircissement d’un joueur dans son camp', () => {
  it('donne l’encre pure au premier et la plus claire au dernier', () => {
    expect(padTint(0, 4)).toBe(100)
    expect(padTint(3, 4)).toBe(40)
  })

  it('adapte le pas à l’effectif : huit joueurs restent distinguables', () => {
    expect(padTint(1, 8)).toBeCloseTo(100 - 60 / 7)
    expect(padTint(7, 8)).toBe(40)
  })

  it('donne l’encre pure au joueur unique de son camp (aucune division par zéro)', () => {
    expect(padTint(0, 1)).toBe(100)
  })
})

describe('buildPadControlBars — une arme, une barre', () => {
  it('publie le total NOMMÉ de chaque socle : c’est le dénominateur du rail', () => {
    const m = barres()
    expect(m.rows.map((r) => [r.label, r.total])).toEqual([
      ['S7 Sniper', 4],
      ['M41 SPNKr', 2],
    ])
  })

  it('rapporte chaque segment au rail de SA ligne, pas à une borne commune', () => {
    const m = barres()
    const sniper = m.rows[0].segments
    // Alpha ouvre la barre (camp du joueur de la page), Charlie suit : 1/4 puis 3/4.
    expect(sniper.map((s) => s.name)).toEqual(['Alpha', 'Charlie'])
    expect(sniper[0].fraction).toBeCloseTo(1 / 4)
    expect(sniper[1].fraction).toBeCloseTo(3 / 4)
  })

  it('pose le filet sur le PREMIER segment du second camp, jamais au bord du rail', () => {
    const m = barres()
    expect(m.rows[0].segments.map((s) => s.startsSide)).toEqual([false, true])
  })

  it('garde à un joueur la MÊME teinte d’une arme à l’autre (rang dans le camp)', () => {
    const m = barres()
    // Delta est 2e de son camp : sur le lance-roquettes, où il est le SEUL preneur de son camp,
    // il garde sa teinte de 2e — jamais l'encre pure du premier.
    const delta = m.rows[1].segments.find((s) => s.name === 'Delta')!
    expect(delta.tint).toBe(40)
    expect(delta.color).toContain('color-mix(in oklab,')
    // Alpha est 1er du sien : encre pure, sans mélange.
    expect(m.rows[0].segments[0].color).toBe('var(--ac-team-ally)')
  })

  it('écarte les joueurs sans aucune prise sur ce socle, sans décaler les teintes', () => {
    // Bravo (2e du camp t0, zéro prise) n'a de segment sur aucune ligne.
    expect(barres().rows.flatMap((r) => r.segments.map((s) => s.name))).not.toContain('Bravo')
  })

  it('respecte l’ordre des armes du modèle amont et l’ordre de camps demandé', () => {
    const m = barres()
    expect(m.rows.map((r) => r.label)).toEqual(['S7 Sniper', 'M41 SPNKr'])
    expect(m.rows[0].segments.map((s) => s.side)).toEqual(['t0', 't1'])
    expect(m.teams.map((t) => t.side)).toEqual(['t0', 't1'])
  })

  it('sort les occupations sans ramasseur nommé de la barre, en annotation de ligne', () => {
    const m = barres()
    expect(m.rows[0].unnamed).toBe(0)
    expect(m.rows[1].unnamed).toBe(2)
    // Et elles n'entrent dans le compte d'aucun camp.
    expect(m.rows[1].segments.reduce((a, s) => a + s.count, 0)).toBe(2)
  })
})
