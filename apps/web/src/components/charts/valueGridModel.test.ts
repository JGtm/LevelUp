/**
 * Tests — valueGridModel (la projection de la grille de valeurs).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. CHAQUE COLONNE A SON ÉCHELLE — une grandeur rare ne s'écrase pas contre le zéro d'une
 *      grandeur abondante, et la barre la plus longue d'une colonne remplit son rail.
 *   2. `null` N'EST PAS ZÉRO : une grandeur NON MESURÉE n'entre ni dans le maximum, ni dans le
 *      total, et sa barre reste vide. Un zéro MESURÉ, lui, compte.
 *   3. LE FILET SUIT LE GROUPE, pas le rang : il tombe entre deux lignes de groupes différents,
 *      où qu'elles soient, et nulle part ailleurs.
 */
import { describe, expect, it } from 'vitest'

import { buildValueGrid, valueGridBound, type ValueGridInput } from './valueGridModel'

const ROWS = [
  { key: 'a', label: 'Alpha', group: 't0' },
  { key: 'b', label: 'Bravo', group: 't0' },
  { key: 'c', label: 'Charlie', group: 't1' },
]
const COLS = [
  { key: 'grenades', label: 'Grenades', showTotal: true },
  { key: 'murs', label: 'Murs', showTotal: true },
]
/** grenades : 40 / 13 / 4 ; murs : 2 / 1 / 0. */
const VALEURS = [
  [40, 2],
  [13, 1],
  [4, 0],
]

function grille(over: Partial<ValueGridInput> = {}) {
  return buildValueGrid({
    rows: ROWS,
    columns: COLS,
    value: (r, c) => VALEURS[r][c],
    format: (v) => String(v),
    color: () => 'var(--ac-team-ally)',
    tooltip: (r, c, text) => `${ROWS[r].label} — ${COLS[c].label} : ${text}`,
    ...over,
  })
}

describe('valueGridBound — le palier rond au-dessus du maximum', () => {
  it('monte à l’unité sous 5, à la paire sous 12, au pas de 5 sous 60, à la demi-minute au-delà', () => {
    expect(valueGridBound(3)).toBe(3)
    expect(valueGridBound(4.2)).toBe(5)
    expect(valueGridBound(9)).toBe(10)
    expect(valueGridBound(13)).toBe(15)
    expect(valueGridBound(40)).toBe(40)
    expect(valueGridBound(128.1)).toBe(150)
  })

  it('ne rend jamais zéro : une colonne toute à zéro garde un rail gradué', () => {
    expect(valueGridBound(0)).toBe(1)
    expect(valueGridBound(-3)).toBe(1)
    expect(valueGridBound(Number.NaN)).toBe(1)
  })
})

describe('buildValueGrid — une échelle par colonne', () => {
  it('borne chaque colonne sur SON maximum, jamais sur celui de la grille', () => {
    const m = grille()
    expect(m.columns.map((c) => c.bound)).toEqual([40, 2])
  })

  it('la barre la plus longue d’une colonne remplit son rail, les autres s’y rapportent', () => {
    const m = grille()
    expect(m.cells[0][0].fraction).toBe(1)
    expect(m.cells[1][0].fraction).toBeCloseTo(13 / 40)
    // Un mur sur deux occupe la moitié du rail de SA colonne — pas 1/40e de celui des grenades.
    expect(m.cells[1][1].fraction).toBe(0.5)
  })

  it('gradue le pied de colonne en zéro · milieu · borne, la borne arrondie à l’entier', () => {
    const m = grille()
    expect(m.columns[0].axis).toEqual(['0', '20', '40'])
  })

  it('ne PAS arrondir le milieu d’une colonne de DURÉE : 2:30 est une graduation juste', () => {
    const m = buildValueGrid({
      rows: ROWS,
      columns: [{ key: 'temps', label: 'Temps en zone', duration: true }],
      value: (r) => [128.1, 112.7, 83][r],
      format: (v) => `${Math.floor(v / 60)}:${String(Math.round(v % 60)).padStart(2, '0')}`,
      color: () => 'var(--ac-team-ally)',
      tooltip: (_r, _c, text) => text,
    })
    expect(m.columns[0].bound).toBe(150)
    expect(m.columns[0].axis).toEqual(['0:00', '1:15', '2:30'])
  })

  it('écrit le total d’une colonne qui en demande un, et rien sur celle qui n’en veut pas', () => {
    const m = buildValueGrid({
      rows: ROWS,
      columns: [COLS[0], { key: 'murs', label: 'Murs' }],
      value: (r, c) => VALEURS[r][c],
      format: (v) => String(v),
      color: () => 'var(--ac-team-ally)',
      tooltip: (_r, _c, text) => text,
    })
    expect(m.columns[0].totalText).toBe('57')
    expect(m.columns[1].totalText).toBeNull()
  })
})

describe('buildValueGrid — une valeur non mesurée n’est pas un zéro', () => {
  it('l’exclut du maximum et du total, laisse sa barre vide et écrit le repli', () => {
    const m = grille({ value: (r, c) => (r === 0 && c === 0 ? null : VALEURS[r][c]) })
    expect(m.columns[0].bound).toBe(15) // 13 devient le maximum, 40 ne compte plus
    expect(m.columns[0].totalText).toBe('17')
    expect(m.cells[0][0]).toMatchObject({ value: null, text: '—', fraction: 0 })
  })

  it('un zéro MESURÉ reste une mesure : il s’écrit « 0 », jamais le repli', () => {
    expect(grille().cells[2][1]).toMatchObject({ value: 0, text: '0', fraction: 0 })
  })

  it('accepte un repli d’appelant autre que le tiret cadratin', () => {
    const m = grille({ value: () => null, notMeasured: 'n. m.' })
    expect(m.cells[0][0].text).toBe('n. m.')
  })
})

describe('buildValueGrid — le filet suit le groupe', () => {
  it('tombe entre deux lignes de groupes différents, et nulle part ailleurs', () => {
    expect(grille().separators).toEqual([2])
  })

  it('ne pose aucun filet quand toutes les lignes partagent un groupe', () => {
    const m = grille({ rows: ROWS.map((r) => ({ ...r, group: 't0' })) })
    expect(m.separators).toEqual([])
  })

  it('en pose autant qu’il y a de changements, même sur des groupes alternés', () => {
    const m = grille({
      rows: [
        { key: 'a', label: 'A', group: 't0' },
        { key: 'b', label: 'B', group: 't1' },
        { key: 'c', label: 'C', group: 't0' },
      ],
    })
    expect(m.separators).toEqual([1, 2])
  })
})

describe('buildValueGrid — l’infobulle porte l’identité, la grandeur et la valeur', () => {
  it('reçoit le texte DÉJÀ formaté, pas le nombre brut', () => {
    const m = grille()
    expect(m.cells[0][0].tooltip).toBe('Alpha — Grenades : 40')
  })
})
