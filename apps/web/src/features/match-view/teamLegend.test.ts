import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { MATCH_VIEW_TEXT } from './i18n'
import { teamLegend } from './teamLegend'

const t = MATCH_VIEW_TEXT.fr

function feuille(rows: Array<Partial<MatchScoreboardRow>>): MatchScoreboardRow[] {
  return rows as unknown as MatchScoreboardRow[]
}

describe('teamLegend — les noms de la légende des graphes divergents', () => {
  it('nomme l’équipe du joueur et l’unique équipe d’en face, comme « Score dans le temps »', () => {
    const l = teamLegend(
      feuille([
        { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
        { xuid: 'e', gamertag: 'Autre', team_side: 't1', is_me: false },
      ]),
      'me',
      t,
    )
    expect(l.ally).toContain('Eagle')
    expect(l.opponent).toContain('Cobra')
    expect(l.multiTeam).toBe(false)
  })

  it('le nom du backend (team_name) prime sur le nom officiel', () => {
    const l = teamLegend(
      feuille([
        { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true, team_name: 'Rouge' },
        { xuid: 'e', gamertag: 'Autre', team_side: 't1', is_me: false, team_name: 'Bleu' },
      ]),
      'me',
      t,
    )
    expect(l.ally).toContain('Rouge')
    expect(l.opponent).toContain('Bleu')
  })

  it('plusieurs équipes en face : pas de nom d’adversaire, et le dit', () => {
    const l = teamLegend(
      feuille([
        { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
        { xuid: 'b', gamertag: 'B', team_side: 't1', is_me: false },
        { xuid: 'c', gamertag: 'C', team_side: 't2', is_me: false },
      ]),
      'me',
      t,
    )
    expect(l.ally).toContain('Eagle')
    expect(l.opponent).toBeNull()
    expect(l.multiTeam).toBe(true)
  })

  it('camp du joueur non situé, ou feuille absente : aucun nom deviné', () => {
    const sansMoi = feuille([
      { xuid: 'a', gamertag: 'A', team_side: 't0', is_me: false },
      { xuid: 'b', gamertag: 'B', team_side: 't1', is_me: false },
    ])
    expect(teamLegend(sansMoi, null, t)).toEqual({ ally: null, opponent: null, multiTeam: false })
    expect(teamLegend(null, 'me', t)).toEqual({ ally: null, opponent: null, multiTeam: false })
  })
})
