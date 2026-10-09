import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { buildPlayerMarks } from './playerMarks'

function row(over: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return { xuid: 'x', gamertag: 'GT', team_side: 't0', is_me: false, ...over } as MatchScoreboardRow
}

describe('buildPlayerMarks', () => {
  const board = [
    row({ xuid: 'ME', gamertag: 'Guillaume', is_me: true }),
    row({ xuid: 'A', gamertag: 'Ma Pote', team_side: 't0' }),
    row({ xuid: 'B', gamertag: 'Adversaire Ami', team_side: 't1' }),
    row({ xuid: 'C', gamertag: 'Inconnu', team_side: 't1' }),
  ]

  it('is_me -> me ; amis appariés sans casse ni espaces de bord ; les autres sans marque', () => {
    const marks = buildPlayerMarks(board, ['  MA POTE ', 'adversaire ami'])
    expect(marks.get('ME')).toBe('me')
    expect(marks.get('A')).toBe('friend')
    expect(marks.get('C')).toBeUndefined()
  })

  it('un ami ADVERSE est marqué aussi — la marque dit l’identité, pas le camp', () => {
    expect(buildPlayerMarks(board, ['Adversaire Ami']).get('B')).toBe('friend')
  })

  it('le joueur de la page n’est jamais marqué ami de lui-même', () => {
    expect(buildPlayerMarks(board, ['Guillaume']).get('ME')).toBe('me')
  })

  it('liste d’amis vide ou entrées vides : seul « moi » est marqué', () => {
    expect([...buildPlayerMarks(board, ['', '   ']).keys()]).toEqual(['ME'])
  })
})

/**
 * AJOUT DU 2026-09-06 (lot L2b) — la marque `me` suit le POINT DE VUE (décision 13 du plan
 * « frise, point de vue »). Ajouts seulement : les cas ci-dessus fixent le comportement sans
 * point de vue, celui des appelants qui n'en connaissent pas.
 */
describe('buildPlayerMarks — avec un point de vue', () => {
  const board = [
    row({ xuid: 'ME', gamertag: 'Guillaume', is_me: true }),
    row({ xuid: 'A', gamertag: 'Ma Pote', team_side: 't0' }),
    row({ xuid: 'B', gamertag: 'Adversaire Ami', team_side: 't1' }),
    row({ xuid: 'C', gamertag: 'Inconnu', team_side: 't1' }),
  ]

  it('le disque cerclé va au joueur REGARDÉ, pas à la ligne « moi »', () => {
    const marks = buildPlayerMarks(board, [], 'C')
    expect(marks.get('C')).toBe('me')
    expect(marks.get('ME')).toBeUndefined()
  })

  it('celui qu’on regarde n’est jamais marqué ami de lui-même', () => {
    expect(buildPlayerMarks(board, ['Adversaire Ami'], 'B').get('B')).toBe('me')
  })

  it('la ligne « moi » redevient un joueur comme un autre — amie si elle est amie', () => {
    expect(buildPlayerMarks(board, ['Guillaume'], 'C').get('ME')).toBe('friend')
  })

  it('point de vue = la ligne « moi » : identique à l’appel sans point de vue', () => {
    const avec = buildPlayerMarks(board, ['Ma Pote'], 'ME')
    const sans = buildPlayerMarks(board, ['Ma Pote'])
    expect(Object.fromEntries(avec)).toEqual(Object.fromEntries(sans))
  })

  it('point de vue absent du tableau de score : PERSONNE ne porte la marque « moi »', () => {
    // Pas de repli sur `is_me` : le disque cerclé désignerait alors quelqu'un d'autre que ce
    // qu'on regarde, ce qui est pire que pas de disque du tout.
    const marks = buildPlayerMarks(board, [], 'xuid-jamais-vu')
    expect([...marks.values()]).not.toContain('me')
  })

  it('point de vue à null ou undefined : le comportement d’origine', () => {
    expect(buildPlayerMarks(board, [], null).get('ME')).toBe('me')
    expect(buildPlayerMarks(board, [], undefined).get('ME')).toBe('me')
  })
})
