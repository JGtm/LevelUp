/**
 * playerInstantEstimates.arrondi.test.ts — DEUX PROMESSES DES TIRS ESTIMÉS : les lignes par arme somment au
 * total, et la précision n'est jamais écrite au-dessus de 100 %.
 *
 * Un match minimal à un joueur (slot 512, pas de 100 ms) : les tirs écrits par arme, les touches et le rapport
 * du bilan (`shots / shotEvents`) se règlent à la main.
 */
import { describe, expect, it } from 'vitest'

import { buildPlayers } from '@/lib/replay/rosterLogic'

import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc, type TestReplayDocOverrides } from '../test/testDoc'

import { playerInstantEstimatesAt, readInstantEstimatesSeries } from './playerInstantEstimates'

const shot = (t: number, w?: string) => ({ t, slot: 512, x: 0, y: 0, ...(w ? { w } : {}) })
const hit = (t: number) => ({ t, shooter: 'A', victim: 'B', effect: 'e', w: 'W1' })
const bilan = (shots: number, shotEvents: number) => ({
  xuid: 'A', sufficient: true, coverage: 0.9, damageDealt: 0, damageTaken: 0, duplicates: 1, hits: 0, messages: 0, shots, shotEvents,
})

/** Les tirs estimés de A à l'image `frame`, sur un document à un joueur surchargé par `over`. */
function shotsOf(over: TestReplayDocOverrides, frame: number) {
  const doc = testReplayDoc({
    frameIntervalMs: 100,
    frameCount: 200,
    roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
    tracks: [{ slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }] }],
    ...over,
  })
  const players = buildPlayers(doc, [scoreboardRow('A', 'Alpha', 'Eagle')])
  const series = readInstantEstimatesSeries({ doc, players })
  return playerInstantEstimatesAt(series, players[0], frame).shots!
}

describe('tirs estimés — les lignes par arme somment au total', () => {
  it('rapport 1,5 et trois armes à un tir écrit : 2 + 2 + 1 = 5, le plus grand reste d’abord, à égalité l’ordre du film', () => {
    const s = shotsOf({ shots: [shot(5, 'W1'), shot(15, 'W2'), shot(25, 'W3')], shotStats: [bilan(6, 4)] }, 100)
    expect(s.fired).toBe(3)
    expect(s.firedEstimated).toBe(5)
    expect(s.byWeapon.map((w) => [w.weapon, w.firedEstimated])).toEqual([['W1', 2], ['W2', 2], ['W3', 1]])
    expect(s.byWeapon.reduce((n, w) => n + w.firedEstimated, 0)).toBe(s.firedEstimated)
  })

  it('le reste va aux plus grandes parties fractionnaires, pas à la première arme', () => {
    // Rapport 1,2 : W1 (1 tir) -> 1,2 ; W2 (3 tirs) -> 3,6 ; total round(4,8) = 5 ; le reste d’une unité va à W2 (0,6).
    const s = shotsOf(
      { shots: [shot(5, 'W1'), shot(15, 'W2'), shot(25, 'W2'), shot(35, 'W2')], shotStats: [bilan(6, 5)] },
      100,
    )
    expect(s.firedEstimated).toBe(5)
    expect(Object.fromEntries(s.byWeapon.map((w) => [w.weapon, w.firedEstimated]))).toEqual({ W1: 1, W2: 4 })
  })

  it('une seule arme : sa ligne est le total', () => {
    const s = shotsOf({ shots: [shot(5, 'W1')], shotStats: [bilan(6, 4)] }, 100)
    expect(s.firedEstimated).toBe(2)
    expect(s.byWeapon.map((w) => w.firedEstimated)).toEqual([2])
  })

  it('aucun tir : total nul et aucune ligne de tirs estimés', () => {
    const s = shotsOf({ shots: [], shotStats: [bilan(6, 4)] }, 100)
    expect(s.firedEstimated).toBe(0)
    expect(s.byWeapon.reduce((n, w) => n + w.firedEstimated, 0)).toBe(0)
  })

  it('rapport 7/4, à chaque image : la somme des lignes égale le total', () => {
    for (let f = 0; f <= 60; f += 5) {
      const s = shotsOf({ shots: [shot(5, 'W1'), shot(15, 'W2'), shot(25, 'W3'), shot(35)], shotStats: [bilan(7, 4)] }, f)
      expect(s.byWeapon.reduce((n, w) => n + w.firedEstimated, 0), `image ${f}`).toBe(s.firedEstimated)
    }
  })
})

describe('tirs estimés — la précision n’est jamais au-dessus de 100 %', () => {
  const doc = { shots: [shot(5, 'W1')], hits: [hit(6), hit(16)], shotStats: [bilan(0, 0)] }

  it('autant de touches que de tirs : précision de 100 %, établie', () => {
    const s = shotsOf(doc, 10)
    expect([s.fired, s.hit, s.accuracy]).toEqual([1, 1, 1])
  })

  it('plus de touches que de tirs estimés (arme à zone, tir pas encore écrit) : précision absente, jamais 200 %', () => {
    const s = shotsOf(doc, 20)
    expect([s.firedEstimated, s.hit]).toEqual([1, 2])
    expect(s.accuracy).toBeNull()
  })
})
