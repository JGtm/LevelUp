/**
 * _sections.test.ts — LE MODÈLE DES SECTIONS de la colonne de session (D11 du plan
 * PLAN_SESSIONS_EMPRISE_2026-10-06) : une clé par CARTE, des groupes titrés, des sous-groupes
 * titrés, des paires en pleine page, une rangée partagée par clé en comparaison.
 */
import { describe, expect, it } from 'vitest'

import {
  groupSessionSections,
  mergeSessionSectionKeys,
  pairSessionKeys,
  sessionRowOpenings,
  sessionSectionKeys,
  type SessionSectionPresence,
} from './_sections'

/** Toutes les cartes présentes. */
const TOUT: SessionSectionPresence = {
  coordination: true,
  range: true,
  frag_donut: true,
  tools: true,
  weapon_accuracy: true,
  control: true,
  fil: true,
  grid: true,
  mine: true,
  production: true,
  yield: true,
  lives: true,
  objective_balance: true,
  objective_sheet: true,
  equipment: true,
}

const CARTES = [
  'frag_donut',
  'tools',
  'weapon_accuracy',
  'control',
  'fil',
  'grid',
  'mine',
  'production',
  'yield',
  'lives',
  'objective_balance',
  'objective_sheet',
  'equipment',
]

describe('sessionSectionKeys — une clé par carte', () => {
  it('ordre : les cartes A à L après la carrière, avant le tableau des matchs', () => {
    const keys = sessionSectionKeys(TOUT)
    const i = keys.indexOf('career_xp')
    expect(keys.slice(i + 1, i + 1 + CARTES.length)).toEqual(CARTES)
    expect(keys[keys.length - 1]).toBe('matches')
    expect(keys).not.toContain('frags')
    expect(keys).not.toContain('usage')
  })

  it('une carte absente n’a pas de clé', () => {
    const keys = sessionSectionKeys({ ...TOUT, mine: false, objective_sheet: false })
    expect(keys).not.toContain('mine')
    expect(keys).not.toContain('objective_sheet')
    expect(keys).toContain('objective_balance')
  })
})

describe('groupSessionSections — groupes et sous-groupes', () => {
  it('les cartes A-L sous « Frags et usages », sous-groupes dans l’ordre de la maquette', () => {
    const runs = groupSessionSections(sessionSectionKeys(TOUT))
    const ku = runs.find((r) => r.group === 'kills_usage')
    expect(ku?.keys).toEqual(CARTES)
    expect(ku?.subruns.map((s) => [s.subgroup, s.keys])).toEqual([
      [null, ['frag_donut', 'tools', 'weapon_accuracy']],
      ['resources', ['control', 'fil', 'grid', 'mine']],
      ['prendre', ['production', 'yield']],
      ['lives', ['lives']],
      ['objectif', ['objective_balance', 'objective_sheet']],
      ['equipment', ['equipment']],
    ])
  })

  it('un sous-groupe sans carte n’existe pas (pas d’intertitre au-dessus de rien)', () => {
    const runs = groupSessionSections(
      sessionSectionKeys({ ...TOUT, objective_balance: false, objective_sheet: false }),
    )
    const ku = runs.find((r) => r.group === 'kills_usage')
    expect(ku?.subruns.map((s) => s.subgroup)).not.toContain('objectif')
  })
})

describe('sessionRowOpenings — l’intertitre se pose sur la première clé PRÉSENTE', () => {
  it('groupe et sous-groupe dans la rangée de leur première clé', () => {
    const open = sessionRowOpenings(sessionSectionKeys({ ...TOUT, frag_donut: false, tools: false, weapon_accuracy: false, control: false }))
    // « Frags et usages » s'ouvre sur `fil` : A, B, B' et C manquent ; le sous-groupe aussi.
    expect(open.get('fil')).toEqual({ group: 'kills_usage', subgroup: 'resources' })
    expect(open.get('grid')).toBeUndefined()
    expect(open.get('production')).toEqual({ subgroup: 'prendre' })
    expect(open.has('control')).toBe(false)
  })

  it('une clé sans groupe ni sous-groupe n’ouvre rien', () => {
    const open = sessionRowOpenings(sessionSectionKeys(TOUT))
    expect(open.has('summary')).toBe(false)
    expect(open.has('matches')).toBe(false)
    expect(open.get('frag_donut')).toEqual({ group: 'kills_usage' })
    expect(open.get('tools')).toBeUndefined()
  })
})

describe('mergeSessionSectionKeys — union des deux colonnes', () => {
  it('l’ordre est l’ordre canonique, pas celui de la colonne de gauche', () => {
    const gauche = sessionSectionKeys({ ...TOUT, objective_balance: false, objective_sheet: false })
    const droite = sessionSectionKeys({ ...TOUT, mine: false })
    const union = mergeSessionSectionKeys(gauche, droite)
    expect(union).toEqual(sessionSectionKeys(TOUT))
    // Une clé présente à droite seulement prend sa place canonique.
    expect(union.indexOf('objective_balance')).toBeLessThan(union.indexOf('equipment'))
  })
})

describe('pairSessionKeys — les paires de la pleine page', () => {
  it('A|B, C|D, G|H partagent une rangée ; le reste est seul', () => {
    const rows = pairSessionKeys(CARTES as never)
    expect(rows).toEqual([
      ['frag_donut', 'tools'],
      ['weapon_accuracy'],
      ['control', 'fil'],
      ['grid'],
      ['mine'],
      ['production', 'yield'],
      ['lives'],
      ['objective_balance'],
      ['objective_sheet'],
      ['equipment'],
    ])
  })

  it('une carte de la paire seule reste seule', () => {
    expect(pairSessionKeys(['tools', 'control', 'grid'] as never)).toEqual([['tools'], ['control'], ['grid']])
  })
})
