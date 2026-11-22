/**
 * Tests — QUELLES LIGNES DU FIL CONCERNENT LE JOUEUR SUIVI (`feedFollowed`) : auteur ou cible de chaque forme de
 * ligne, les deux clés d'un joueur (roster et tableau de score), l'assistance de capture qui n'en est pas une, les
 * lignes sans joueur, et l'absence de joueur suivi.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { FeedEventLine } from './feedEventLines'
import { concernsFollowed, playerKeys } from './feedFollowed'
import type { ReplayDeath, ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import type { MedalEvent } from './medalEvent'
import type { PresenceEvent } from './presenceFeed'

const ME = 'xuid(1)'
const OTHER = 'xuid(2)'
const THIRD = 'xuid(3)'

const base: ReplayFeedEntry = { key: 'k', replayMs: 1000, kill: null, medal: null, death: null }

const kill = (xuid: string, victimXuid: string): ReplayFeedEntry => ({
  ...base,
  kill: { xuid, victimXuid } as ReplayKill,
})
const death = (xuid: string): ReplayFeedEntry => ({ ...base, death: { xuid } as ReplayDeath })
const medal = (xuid: string): ReplayFeedEntry => ({ ...base, medal: { xuid } as MedalEvent })
const presence = (xuid: string): ReplayFeedEntry => ({ ...base, presence: { xuid } as PresenceEvent })
const event = (line: unknown): ReplayFeedEntry => ({ ...base, event: line as FeedEventLine })

const followed = new Set([ME])

describe('playerKeys — les clés sous lesquelles le fil désigne un joueur', () => {
  it('la clé de roster, et le xuid de base de la ligne du tableau de score quand il diffère', () => {
    const player = { xuid: 'film-key', board: { xuid: ME } } as unknown as ReplayPlayer
    expect([...playerKeys(player)].sort()).toEqual(['film-key', ME].sort())
  })

  it('sans ligne de tableau de score : la clé de roster seule', () => {
    expect([...playerKeys({ xuid: ME } as ReplayPlayer)]).toEqual([ME])
  })

  it('personne n\'est suivi : l\'ensemble vide, le même d\'un appel à l\'autre', () => {
    expect(playerKeys(null).size).toBe(0)
    expect(playerKeys(undefined)).toBe(playerKeys(null))
  })

  it('un même joueur rend le MÊME ensemble (pas de nouvelle identité à chaque rendu)', () => {
    const player = { xuid: ME } as ReplayPlayer
    expect(playerKeys(player)).toBe(playerKeys(player))
  })
})

describe('concernsFollowed — le joueur suivi est l\'auteur ou la cible de la ligne', () => {
  it('frag : le tueur, ou la victime', () => {
    expect(concernsFollowed(kill(ME, OTHER), followed)).toBe(true)
    expect(concernsFollowed(kill(OTHER, ME), followed)).toBe(true)
    expect(concernsFollowed(kill(OTHER, THIRD), followed)).toBe(false)
  })

  it('mort sans tueur : le défunt', () => {
    expect(concernsFollowed(death(ME), followed)).toBe(true)
    expect(concernsFollowed(death(OTHER), followed)).toBe(false)
  })

  it('médaille seule : le décoré', () => {
    expect(concernsFollowed(medal(ME), followed)).toBe(true)
    expect(concernsFollowed(medal(OTHER), followed)).toBe(false)
  })

  it('entrée ou sortie de partie : l\'arrivant ou le partant', () => {
    expect(concernsFollowed(presence(ME), followed)).toBe(true)
    expect(concernsFollowed(presence(OTHER), followed)).toBe(false)
  })

  it('ligne d\'événement : l\'acteur (objectif, prise, montée, crâne)', () => {
    expect(concernsFollowed(event({ family: 'objectives', xuid: ME, stat: 'flag_captures' }), followed)).toBe(true)
    expect(concernsFollowed(event({ family: 'pickups', xuid: ME, name: 'x', nature: 'weaponRack' }), followed)).toBe(true)
    expect(concernsFollowed(event({ family: 'vehicles', xuid: ME, name: 'Warthog' }), followed)).toBe(true)
    expect(concernsFollowed(event({ family: 'objectives', kind: 'skull', xuid: ME, change: 'taken' }), followed)).toBe(true)
    expect(concernsFollowed(event({ family: 'vehicles', xuid: OTHER, name: 'Warthog' }), followed)).toBe(false)
  })

  it('destruction de véhicule : le destructeur, ou un joueur tué avec lui ; sans auteur, personne d\'autre', () => {
    const destroyed = (xuid: string | null, killedXuids: string[]) =>
      event({ family: 'vehicles', destroyed: true, xuid, botName: null, killedXuids, name: 'Ghost' })
    expect(concernsFollowed(destroyed(ME, []), followed)).toBe(true)
    expect(concernsFollowed(destroyed(OTHER, [ME]), followed)).toBe(true)
    expect(concernsFollowed(destroyed(null, [OTHER]), followed)).toBe(false)
    expect(concernsFollowed(destroyed(null, []), followed)).toBe(false)
  })

  it('une assistance de capture n\'est ni l\'auteur ni la cible', () => {
    const capture = event({ family: 'objectives', xuid: OTHER, stat: 'flag_captures', assistXuids: [ME] })
    expect(concernsFollowed(capture, followed)).toBe(false)
  })

  it('fin de manche et colline : aucun joueur, donc personne', () => {
    expect(concernsFollowed(event({ family: 'objectives', kind: 'roundEnd', index: 1, scores: [] }), followed)).toBe(false)
    expect(concernsFollowed(event({ family: 'objectives', kind: 'hill', teamId: 0 }), followed)).toBe(false)
  })

  it('une ligne sans aucun contenu ne concerne personne', () => {
    expect(concernsFollowed(base, followed)).toBe(false)
  })

  it('aucun joueur suivi : aucune ligne, quelle que soit sa forme', () => {
    const none = playerKeys(null)
    for (const entry of [kill(ME, OTHER), death(ME), medal(ME), presence(ME), event({ family: 'vehicles', xuid: ME, name: 'x' })]) {
      expect(concernsFollowed(entry, none)).toBe(false)
    }
  })

  it('les deux clés d\'un joueur désignent la même personne', () => {
    const keys = playerKeys({ xuid: 'film-key', board: { xuid: ME } } as unknown as ReplayPlayer)
    expect(concernsFollowed(kill(ME, OTHER), keys)).toBe(true)
    expect(concernsFollowed(kill(OTHER, 'film-key'), keys)).toBe(true)
  })
})
