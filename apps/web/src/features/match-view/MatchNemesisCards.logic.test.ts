/**
 * Tests — MatchNemesisCards.logic : némésis et souffre-douleur de la page du match, lus par la
 * même fonction que le rejeu (`lib/antagonists`) sur les éliminations de `highlight_events`.
 *
 * Ce qui est protégé : le départage par le TENANT (pas par l'ordre d'une liste), l'immunité aux
 * doublons du fil, l'exclusion des bots et des trahisons, l'état vide.
 */
import { describe, expect, it } from 'vitest'

import type { MatchHighlightEvent, MatchScoreboardRow } from '@/lib/api/types'
import { readMatchAntagonists } from './MatchNemesisCards.logic'

const ME = 'me'

function row(xuid: string, teamSide: string | undefined, gamertag = xuid.toUpperCase()): MatchScoreboardRow {
  return { xuid, gamertag, is_me: xuid === ME, outcome: 'win', team_side: teamSide }
}

/** Moi et un coéquipier côté `eagle` ; trois adversaires côté `cobra`. */
const SCOREBOARD: MatchScoreboardRow[] = [
  row(ME, 'eagle'),
  row('ally', 'eagle'),
  row('e1', 'cobra'),
  row('e2', 'cobra'),
  row('e3', 'cobra'),
]

function kill(
  actor: string | undefined,
  victim: string | undefined,
  t: number,
  extra: Partial<MatchHighlightEvent> = {},
): MatchHighlightEvent {
  return { event_type: 'kill', actor_xuid: actor, victim_xuid: victim, event_time_ms: t, ...extra }
}

describe('readMatchAntagonists : départage par le tenant', () => {
  it('némésis : sur une égalité finale, le premier à atteindre le compte garde le titre, pas le premier de la liste', () => {
    // e1 prend le titre (1), e2 le dépasse (2 contre 1), puis e1 revient à 2 : e2 reste tenant.
    // Le scoreboard et la liste nomment e1 avant e2 : un départage par la liste rendrait e1, pas le tenant.
    const events = [kill('e1', ME, 1000), kill('e2', ME, 2000), kill('e2', ME, 3000), kill('e1', ME, 4000)]
    const r = readMatchAntagonists(events, SCOREBOARD, ME)
    expect(r.nemesis).toEqual({ xuid: 'e2', gamertag: 'E2', count: 2 })
  })

  it('souffre-douleur : même règle, côté éliminations du joueur', () => {
    const events = [kill(ME, 'e1', 1000), kill(ME, 'e2', 2000), kill(ME, 'e2', 3000), kill(ME, 'e1', 4000)]
    const r = readMatchAntagonists(events, SCOREBOARD, ME)
    expect(r.bully).toEqual({ xuid: 'e2', gamertag: 'E2', count: 2 })
  })

  it("ne dépend pas de l'ordre du fil : les éliminations sont lues dans l'ordre du temps", () => {
    const events = [kill('e1', ME, 4000), kill('e2', ME, 3000), kill('e2', ME, 2000), kill('e1', ME, 1000)]
    expect(readMatchAntagonists(events, SCOREBOARD, ME).nemesis?.xuid).toBe('e2')
  })

  it('lit les deux titres sur le même fil', () => {
    const events = [kill('e1', ME, 1000), kill('e1', ME, 2000), kill(ME, 'e3', 3000)]
    const r = readMatchAntagonists(events, SCOREBOARD, ME)
    expect(r.nemesis).toEqual({ xuid: 'e1', gamertag: 'E1', count: 2 })
    expect(r.bully).toEqual({ xuid: 'e3', gamertag: 'E3', count: 1 })
  })
})

describe('readMatchAntagonists : doublons du fil', () => {
  it('une élimination portée deux fois (même tueur, même victime, même instant) compte une fois', () => {
    const simple = [kill('e1', ME, 1000), kill(ME, 'e2', 2000)]
    const doubled = [...simple, ...simple]
    const r = readMatchAntagonists(doubled, SCOREBOARD, ME)
    expect(r).toEqual(readMatchAntagonists(simple, SCOREBOARD, ME))
    expect(r.nemesis?.count).toBe(1)
    expect(r.bully?.count).toBe(1)
  })

  it("le doublon reste un doublon quelle que soit la décoration de l'arme", () => {
    const events = [kill('e1', ME, 1000), kill('e1', ME, 2000), kill('e1', ME, 2000, { weapon_key: 'autre' })]
    expect(readMatchAntagonists(events, SCOREBOARD, ME).nemesis?.count).toBe(2)
  })
})

describe('readMatchAntagonists : bots, trahisons, adversaires', () => {
  it("un bot victime (sans xuid) n'est jamais un souffre-douleur", () => {
    const bot = { victim_gamertag: 'Bot Un [bot]' }
    const events = [
      kill(ME, undefined, 1000, bot),
      kill(ME, '', 2000, bot),
      kill(ME, undefined, 3000, bot),
      kill(ME, 'e2', 4000),
    ]
    const r = readMatchAntagonists(events, SCOREBOARD, ME)
    expect(r.bully).toEqual({ xuid: 'e2', gamertag: 'E2', count: 1 })
  })

  it("un bot tueur (sans xuid) n'est jamais une némésis", () => {
    const events = [kill(undefined, ME, 1000), kill('', ME, 2000)]
    expect(readMatchAntagonists(events, SCOREBOARD, ME).nemesis).toBeNull()
  })

  it('une trahison est ignorée, dans les deux sens', () => {
    const events = [kill('ally', ME, 1000), kill('ally', ME, 2000), kill(ME, 'ally', 3000), kill(ME, 'e3', 4000)]
    const r = readMatchAntagonists(events, SCOREBOARD, ME)
    expect(r.nemesis).toBeNull()
    expect(r.bully).toEqual({ xuid: 'e3', gamertag: 'E3', count: 1 })
  })

  it("sans équipe connue pour le joueur, tout autre joueur est un adversaire", () => {
    const sb = [row(ME, undefined), row('a', undefined)]
    const r = readMatchAntagonists([kill('a', ME, 1000)], sb, ME)
    expect(r.nemesis).toEqual({ xuid: 'a', gamertag: 'A', count: 1 })
  })

  it("un joueur absent du tableau de score n'est pas du camp du joueur : adversaire, nommé par le fil", () => {
    const events = [kill('parti', ME, 1000, { actor_gamertag: 'Parti' })]
    expect(readMatchAntagonists(events, SCOREBOARD, ME).nemesis).toEqual({
      xuid: 'parti',
      gamertag: 'Parti',
      count: 1,
    })
  })

  it("une auto-élimination n'est comptée nulle part", () => {
    expect(readMatchAntagonists([kill(ME, ME, 1000)], SCOREBOARD, ME)).toEqual({ nemesis: null, bully: null })
  })
})

describe('readMatchAntagonists : état vide', () => {
  it('aucune élimination : deux titres vides', () => {
    expect(readMatchAntagonists([], SCOREBOARD, ME)).toEqual({ nemesis: null, bully: null })
  })

  it('les morts, médailles et éliminations sans instant ne comptent pas', () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'death', actor_xuid: ME, victim_xuid: ME, event_time_ms: 1000 },
      { event_type: 'medal', actor_xuid: 'e1', event_time_ms: 2000 },
      kill('e1', ME, 3000, { event_time_ms: undefined }),
    ]
    expect(readMatchAntagonists(events, SCOREBOARD, ME)).toEqual({ nemesis: null, bully: null })
  })

  it('sans joueur de la page, aucun titre', () => {
    expect(readMatchAntagonists([kill('e1', ME, 1000)], SCOREBOARD, null)).toEqual({ nemesis: null, bully: null })
  })
})
