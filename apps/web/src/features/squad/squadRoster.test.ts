import { describe, expect, it } from 'vitest'
import type { SquadWithMembers } from '@/lib/prestige'
import { findSquadByRoster } from './squadRoster'

function sq(id: string, members: Array<{ xuid: string; user_id?: string }>): SquadWithMembers {
  return {
    squad: { id, name: id, created_by: 'me', created_at: '' },
    members: members.map((m) => ({ squad_id: id, xuid: m.xuid, user_id: m.user_id, joined_at: '' })),
  }
}

describe('findSquadByRoster', () => {
  const squads = [
    sq('s1', [{ xuid: 'me-x', user_id: 'me' }, { xuid: 'a' }, { xuid: 'b' }]),
    sq('s2', [{ xuid: 'me-x', user_id: 'me' }, { xuid: 'c' }]),
  ]

  it('matche le roster exact hors joueur principal', () => {
    expect(findSquadByRoster(squads, ['a', 'b'], 'me')?.squad.id).toBe('s1')
    expect(findSquadByRoster(squads, ['b', 'a'], 'me')?.squad.id).toBe('s1') // ordre indifférent
    expect(findSquadByRoster(squads, ['c'], 'me')?.squad.id).toBe('s2')
  })

  it('ne matche pas un roster partiel, sur-ensemble ou inconnu', () => {
    expect(findSquadByRoster(squads, ['a'], 'me')).toBeNull()
    expect(findSquadByRoster(squads, ['a', 'b', 'c'], 'me')).toBeNull()
    expect(findSquadByRoster(squads, ['z'], 'me')).toBeNull()
  })

  it('sélection vide ou squads absents → null', () => {
    expect(findSquadByRoster(squads, [], 'me')).toBeNull()
    expect(findSquadByRoster(undefined, ['a'], 'me')).toBeNull()
  })
})
