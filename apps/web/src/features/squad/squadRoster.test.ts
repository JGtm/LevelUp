import { describe, expect, it } from 'vitest'
import type { SquadMember, SquadWithMembers } from '@/lib/prestige'
import { findSquadByRoster, squadTeammates } from './squadRoster'

function member(xuid: string, gamertag?: string, user_id?: string): SquadMember {
  return { squad_id: 's', xuid, gamertag, user_id, joined_at: '' }
}

function sq(id: string, members: SquadMember[]): SquadWithMembers {
  return { squad: { id, name: id, created_by: 'creator', created_at: '' }, members }
}

describe('squadTeammates', () => {
  it('exclut le viewer par son XUID absolu (pas par le slug)', () => {
    const members = [member('me-x', 'JGtm', 'jgtm'), member('a-x', 'AllyA'), member('b-x', 'AllyB')]
    const others = squadTeammates(members, 'me-x')
    expect(others.map((m) => m.xuid)).toEqual(['a-x', 'b-x'])
  })

  it('ignore les membres sans xuid', () => {
    const members = [member('', 'NoXuid'), member('a-x', 'AllyA')]
    expect(squadTeammates(members, 'me-x').map((m) => m.xuid)).toEqual(['a-x'])
  })
})

describe('findSquadByRoster (clé XUID, player-agnostic)', () => {
  // Escouade « Big Bsses » créée par JGtm (me-x), roster {JGtm, Madina, Choco}.
  const bigBsses = sq('big', [
    member('me-x', 'JGtm', 'jgtm'),
    member('madina-x', 'Madina97294'),
    member('choco-x', 'Chocoboflor'),
  ])

  it('viewer == créateur : roster hors viewer reconnu', () => {
    // JGtm (me-x) sélectionne Madina + Choco → matche Big Bsses.
    expect(findSquadByRoster([bigBsses], ['madina-x', 'choco-x'], 'me-x')?.squad.id).toBe('big')
  })

  it('escouade créée par A, vue par B : reconnue, sans doublon du viewer', () => {
    // Chocoboflor (choco-x) voit Big Bsses : roster hors lui = {JGtm, Madina}.
    const others = squadTeammates(bigBsses.members, 'choco-x').map((m) => m.xuid)
    expect(others).toEqual(['me-x', 'madina-x'])
    expect(others).not.toContain('choco-x') // pas de doublon de lui-même
    expect(findSquadByRoster([bigBsses], ['me-x', 'madina-x'], 'choco-x')?.squad.id).toBe('big')
  })

  it('viewer dont le gamertag diffère du slug : toujours matché (clé xuid)', () => {
    // user_id/slug non pertinents : seule la clé xuid compte.
    const s = sq('s', [member('me-x', 'GamerTagXYZ', 'url-slug-different'), member('a-x', 'AllyA')])
    expect(findSquadByRoster([s], ['a-x'], 'me-x')?.squad.id).toBe('s')
  })

  it('ne matche pas un roster partiel, sur-ensemble ou inconnu', () => {
    expect(findSquadByRoster([bigBsses], ['madina-x'], 'me-x')).toBeNull()
    expect(findSquadByRoster([bigBsses], ['madina-x', 'choco-x', 'x-x'], 'me-x')).toBeNull()
    expect(findSquadByRoster([bigBsses], ['inconnu-x'], 'me-x')).toBeNull()
  })

  it('sélection vide ou squads absents → null', () => {
    expect(findSquadByRoster([bigBsses], [], 'me-x')).toBeNull()
    expect(findSquadByRoster(undefined, ['a-x'], 'me-x')).toBeNull()
  })
})
