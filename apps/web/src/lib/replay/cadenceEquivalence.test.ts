/**
 * cadenceEquivalence.test.ts — SUPÉRIORITÉ NUMÉRIQUE, MENEUR DU SCORE ET HORLOGE : LE MÊME RÉSULTAT
 * SUR UNE GRILLE DE 100 MS ET SUR UNE GRILLE DE 33 MS.
 *
 * Les vies et les paliers de score sont posés en MILLISECONDES, puis convertis en images avec le pas du
 * document : ce que le joueur lit à l'écran (effectif en vie à un instant, camp en tête à un instant)
 * ne doit pas changer avec la densité de la grille.
 */
import { describe, expect, it } from 'vitest'

import { aliveBalanceAt, buildAliveBalance } from './aliveBalanceLogic'
import { matchClock } from './matchClock'
import type { ReplayTrackReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'
import { leaderStates, type ReplayScoreTimelineReady } from './scoreTimeline'

const PAS = [100, 33] as const
const DUREE_MS = 9_900 // multiple de 100 et de 33

/** Une vie sur [debutMs, finMs], en images du pas donné (bornes comprises, comme `isAliveAt`). */
function vie(pas: number, debutMs: number, finMs: number): ReplayTrackReady {
  return { slot: 0, startFrame: Math.ceil(debutMs / pas), endFrame: Math.floor(finMs / pas), points: [] } as unknown as ReplayTrackReady
}

function joueur(xuid: string, team: number, lives: ReplayTrackReady[]): ReplayPlayer {
  return { xuid, team, lives }
}

function horloge(pas: number) {
  return matchClock({ originMs: 0, frameIntervalMs: pas, frameCount: DUREE_MS / pas + 1 }, { t0_ms: 0 })
}

describe('buildAliveBalance — l’effectif en vie à un instant ne dépend pas du pas', () => {
  function balance(pas: number) {
    return buildAliveBalance({
      players: [
        joueur('a1', 0, [vie(pas, 0, DUREE_MS)]),
        joueur('a2', 0, [vie(pas, 0, 4_000), vie(pas, 7_000, DUREE_MS)]),
        joueur('e1', 1, [vie(pas, 0, DUREE_MS)]),
        joueur('e2', 1, [vie(pas, 0, 2_000)]),
      ],
      allyTeam: 0,
      clock: horloge(pas),
    })!
  }

  it('mêmes effectifs aux instants relevés, éloignés d’au moins un pas d’une borne de vie', () => {
    const ref = balance(100)
    const autre = balance(33)
    for (const ms of [0, 1_000, 3_000, 5_000, 6_000, 8_000, 9_000]) {
      const a = aliveBalanceAt(ref, ms)
      const b = aliveBalanceAt(autre, ms)
      expect({ ms, ally: b.ally, opponent: b.opponent }, `${ms} ms`).toEqual({ ms, ally: a.ally, opponent: a.opponent })
    }
  })
})

describe('leaderStates — le camp en tête à un instant ne dépend pas du pas', () => {
  const PALIERS_MS = {
    0: [1_000, 4_000, 8_000],
    1: [2_000, 5_000, 6_000],
  } as const

  function timeline(pas: number): ReplayScoreTimelineReady {
    const teams = Object.entries(PALIERS_MS).map(([id, instants]) => ({
      teamId: Number(id),
      rounds: [],
      total: instants.map((ms, i) => ({ t: Math.round(ms / pas), v: i + 1 })),
    }))
    return { teams, players: [], holdTicks: [] } as unknown as ReplayScoreTimelineReady
  }

  it('la suite des meneurs (camp ou égalité) est la même, à 100 ms et à 33 ms', () => {
    const suite = (pas: number) => leaderStates(timeline(pas), 2).map((e) => e.teamId)
    expect(suite(33)).toEqual(suite(100))
    expect(suite(100).length).toBeGreaterThan(2)
  })

  it('chaque changement de meneur tombe au même instant, à un pas près', () => {
    for (const pas of PAS) {
      const instants = leaderStates(timeline(pas), 2).map((e) => e.frame * pas)
      const ref = leaderStates(timeline(100), 2).map((e) => e.frame * 100)
      expect(instants.length).toBe(ref.length)
      for (const [i, ms] of instants.entries()) expect(Math.abs(ms - ref[i])).toBeLessThanOrEqual(pas)
    }
  })
})

describe('matchClock — l’axe de lecture est en millisecondes', () => {
  it('la durée de gameplay du film ne bouge pas avec le pas, à un pas près', () => {
    const fin = (pas: number) => horloge(pas)!.gameplayMsOfFrame(horloge(pas)!.frameCount - 1)
    expect(Math.abs(fin(33) - fin(100))).toBeLessThanOrEqual(100)
    expect(fin(33)).toBeCloseTo(DUREE_MS, -2)
  })
})
