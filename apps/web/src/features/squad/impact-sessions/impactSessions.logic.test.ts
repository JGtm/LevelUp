import { describe, expect, it } from 'vitest'

import type { SquadImpactSessions } from '@/lib/api/types'

import { impactSessionsFixture } from './impactSessions.fixtures'
import {
  ALL_SESSIONS_KEY,
  buildImpactSessionsView,
  formatSignedNet,
  initialOpenState,
  netBarRatio,
  netSign,
  roleCellStates,
  scoreCellStates,
  scoreMark,
} from './impactSessions.logic'

const view = buildImpactSessionsView(impactSessionsFixture)!

describe('buildImpactSessionsView', () => {
  it('rend null sans joueur ou sans session', () => {
    expect(buildImpactSessionsView(null)).toBeNull()
    expect(buildImpactSessionsView(undefined)).toBeNull()
    expect(buildImpactSessionsView({ ...impactSessionsFixture, sessions: [] })).toBeNull()
    expect(buildImpactSessionsView({ ...impactSessionsFixture, players: [] })).toBeNull()
  })

  it('met « Toutes les sessions » en tête, puis les sessions de la plus récente à la plus ancienne', () => {
    expect(view.rows.map((r) => r.key)).toEqual([
      ALL_SESSIONS_KEY,
      's:14/11/2026 20:05–23:10',
      's:10/11/2026 21:00–22:30',
      's:07/11/2026 18:30–19:40',
      's:31/10/2026 22:00–22:45',
    ])
    expect(view.rows[1].label).toBe('14/11/2026 20:05–23:10')
  })

  it('compte les matchs et le bilan victoires / défaites de chaque session (une égalité n’est ni l’un ni l’autre)', () => {
    const [, s14, s10] = view.rows
    expect([s14.matchCount, s14.wins, s14.losses]).toEqual([5, 3, 1])
    expect([s10.matchCount, s10.wins, s10.losses]).toEqual([3, 1, 2])
    expect(s14.matches.map((m) => m.number)).toEqual([1, 2, 3, 4, 5])
    expect(s14.matches[4].outcome).toBe('other')
  })

  it('somme le bilan de « Toutes les sessions » sur les sessions affichées', () => {
    const [all, ...sessions] = view.rows
    expect(all.matchCount).toBe(sessions.reduce((n, r) => n + r.matchCount, 0))
    expect(all.wins).toBe(sessions.reduce((n, r) => n + r.wins, 0))
    expect(all.losses).toBe(sessions.reduce((n, r) => n + r.losses, 0))
    expect(all.matchCount).toBe(12)
  })

  it('calcule le net par match : points de la session / matchs joués', () => {
    const jgtm14 = view.rows[1].players[0]
    // premier sang +2, bourreau +1, finisseur +2, boulet −2, kamikaze −0,5 : 2,5 points sur 5 matchs.
    expect(jgtm14.points).toBe(2.5)
    expect(jgtm14.matchesPlayed).toBe(5)
    expect(jgtm14.netPerMatch).toBeCloseTo(0.5)
  })

  it('divise par les matchs joués, pas par les matchs de la session (joueur absent d’un match)', () => {
    const joueur165_14 = view.rows[1].players[2]
    expect(joueur165_14.matchesPlayed).toBe(4)
    expect(joueur165_14.netPerMatch).toBeCloseTo(joueur165_14.points / 4)
  })

  it('rend un net nul absent pour un joueur qui n’a joué aucun match de la ligne', () => {
    const absent = view.rows[4].players[0]
    expect(absent.matchesPlayed).toBe(0)
    expect(absent.netPerMatch).toBeNull()
  })

  it('garde tous les rôles dans les comptes (zéros compris) et les rôles par match', () => {
    const p = view.rows[1].players[0]
    expect(Object.keys(p.counts)).toEqual(view.roleOrder)
    expect(p.counts.first_blood).toBe(1)
    expect(p.counts.glass_jaw).toBe(0)
    expect(p.rolesByMatch['m-14-1']).toEqual(['first_blood', 'top_killer'])
    expect(p.rolesByMatch['m-14-5']).toBeUndefined()
  })

  it('lit les totaux servis pour « Toutes les sessions » ; ils égalent la somme des sessions', () => {
    const [all, ...sessions] = view.rows
    all.players.forEach((p, i) => {
      expect(p.points).toBeCloseTo(sessions.reduce((n, r) => n + r.players[i].points, 0))
      expect(p.matchesPlayed).toBe(sessions.reduce((n, r) => n + r.players[i].matchesPlayed, 0))
    })
  })

  it('tire les rôles qui coûtent des points du barème servi', () => {
    expect([...view.costlyRoles].sort()).toEqual(
      ['false_brother', 'first_group_death', 'glass_jaw', 'kamikaze', 'last_casualty', 'last_group_kill', 'thief'].sort(),
    )
    expect(view.costlyRoles.has('top_killer')).toBe(false)
  })

  it('tolère les listes nulles du contrat', () => {
    const sparse: SquadImpactSessions = {
      players: ['A'],
      role_order: null,
      scale: null,
      totals: null,
      sessions: [{ session_label: 'S (1)', start_time: '2026-01-01T00:00:00Z', matches: null, players: null }],
    }
    const v = buildImpactSessionsView(sparse)!
    expect(v.rows[1].matchCount).toBe(0)
    expect(v.rows[1].players[0].netPerMatch).toBeNull()
    expect(v.costlyRoles.size).toBe(0)
  })
})

describe('initialOpenState', () => {
  it('sans sélection : la session la plus récente s’ouvre, sans teinte', () => {
    expect(initialOpenState(view.rows, [])).toEqual({ open: ['s:14/11/2026 20:05–23:10'], selected: [] })
  })

  it('ouvre et teinte les sessions sélectionnées (plusieurs possibles)', () => {
    const s = initialOpenState(view.rows, ['10/11/2026 21:00–22:30 (3)', '31/10/2026 22:00–22:45 (2)'])
    expect(s.open).toEqual(['s:10/11/2026 21:00–22:30', 's:31/10/2026 22:00–22:45'])
    expect(s.selected).toEqual(s.open)
  })

  it('ignore le suffixe de compte volatil : un libellé sélectionné avec un ancien compte reste reconnu', () => {
    const s = initialOpenState(view.rows, ['10/11/2026 21:00–22:30 (1)'])
    expect(s.selected).toEqual(['s:10/11/2026 21:00–22:30'])
  })

  it('une sélection hors des sessions affichées retombe sur la plus récente, sans teinte', () => {
    expect(initialOpenState(view.rows, ['01/01/2020 10:00–11:00 (4)'])).toEqual({
      open: ['s:14/11/2026 20:05–23:10'],
      selected: [],
    })
  })

  it('« Toutes les sessions » n’est jamais ouverte d’office', () => {
    expect(initialOpenState(view.rows, []).open).not.toContain(ALL_SESSIONS_KEY)
  })
})

describe('roleCellStates', () => {
  it('surligne le plus grand compte en meilleur et le plus petit en pire pour un rôle qui rapporte', () => {
    expect(roleCellStates([3, 1, 2], false)).toEqual(['best', 'worst', 'neutral'])
  })

  it('inverse pour un rôle qui coûte des points', () => {
    expect(roleCellStates([3, 1, 2], true)).toEqual(['worst', 'best', 'neutral'])
  })

  it('ignore les comptes nuls : seuls les non nuls se comparent', () => {
    expect(roleCellStates([2, 0, 1], false)).toEqual(['best', 'neutral', 'worst'])
  })

  it('ne surligne rien sans deux valeurs distinctes', () => {
    expect(roleCellStates([2, 2, 0], false)).toEqual(['neutral', 'neutral', 'neutral'])
    expect(roleCellStates([0, 0, 0], true)).toEqual(['neutral', 'neutral', 'neutral'])
    expect(roleCellStates([1, 0, 0], false)).toEqual(['neutral', 'neutral', 'neutral'])
  })
})

describe('scoreCellStates et scoreMark', () => {
  it('surligne le meilleur et le pire score ; un joueur absent n’entre pas dans la comparaison', () => {
    const players = view.rows[3].players // session du 7 : le troisième joueur est absent
    const states = scoreCellStates(players)
    expect(players[2].matchesPlayed).toBe(0)
    expect(states[2]).toBe('neutral')
    expect(states.filter((s) => s === 'best')).toHaveLength(1)
    expect(states.filter((s) => s === 'worst')).toHaveLength(1)
  })

  it('ne surligne rien quand tous les scores sont égaux', () => {
    const players = view.rows[4].players // un seul joueur a joué
    expect(scoreCellStates(players)).toEqual(['neutral', 'neutral', 'neutral'])
  })

  it('couronne au premier, banane au dernier s’il est négatif, passager clandestin sinon', () => {
    expect(scoreMark('best', 4)).toBe('champion')
    expect(scoreMark('worst', -1)).toBe('maillon_faible')
    expect(scoreMark('worst', 0)).toBe('passager_clandestin')
    expect(scoreMark('worst', 2)).toBe('passager_clandestin')
    expect(scoreMark('neutral', 2)).toBeNull()
  })
})

describe('formats du net', () => {
  it('signe, une décimale, séparateur de la langue', () => {
    expect(formatSignedNet(1.5, 'fr')).toBe('+1,5')
    expect(formatSignedNet(1.5, 'en')).toBe('+1.5')
    expect(formatSignedNet(-2, 'fr')).toBe('−2,0')
    expect(formatSignedNet(0, 'fr')).toBe('±0,0')
  })

  it('arrondit à une décimale : un net qui arrondit à zéro est nul, sans signe moins', () => {
    expect(formatSignedNet(-0.04, 'fr')).toBe('±0,0')
    expect(formatSignedNet(0.25, 'en')).toBe('+0.3')
  })

  it('le sens du net suit la valeur affichée', () => {
    expect(netSign(0.5)).toBe('gain')
    expect(netSign(-0.5)).toBe('loss')
    expect(netSign(0.04)).toBe('zero')
  })

  it('borne la mini-barre à ±2 points par match', () => {
    expect(netBarRatio(1)).toBe(0.5)
    expect(netBarRatio(-5)).toBe(-1)
    expect(netBarRatio(9)).toBe(1)
    expect(netBarRatio(0)).toBe(0)
  })
})
