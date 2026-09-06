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
