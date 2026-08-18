/**
 * _scoreCurve.test.ts — la projection du calque de score en courbes.
 *
 * Les cas reprennent les trois témoins re-cuits de la phase 1 : Slayer `000d5950` (43/50,
 * aucun retournement), CTF `530820e5` (3-0, UNE seule série publiée pour deux camps),
 * Oddball `24dbb67d` (200/121, trois retournements).
 */
import { describe, expect, it } from 'vitest'

import { normalizeScoreTimeline } from '@/lib/replay/scoreTimeline'

import { buildScoreCurve, formatClock, teamIdsOf, type ScoreCurveInput } from './_scoreCurve'

import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

/** Un calque normalisé, bâti par la frontière du calque (aucun document complet requis). */
function timelineOf(teams: unknown): ReplayScoreTimelineReady | undefined {
  return normalizeScoreTimeline({ teams, players: null } as never)
}

/** Une équipe à manche unique, décrite par ses paliers `[frame, valeur]`. */
function equipe(teamId: number, points: Array<[number, number]>) {
  const pts = points.map(([t, v]) => ({ t, v }))
  return { teamId, rounds: [{ round: 0, points: pts }], total: pts }
}

/** L'entrée nominale : 100 ms par image, 1 001 images (100 s), deux camps identifiés. */
function input(over: Partial<ScoreCurveInput> = {}): ScoreCurveInput {
  return {
    timeline: timelineOf([equipe(0, [[100, 1]]), equipe(1, [[200, 1], [300, 2]])]),
    frameIntervalMs: 100,
    frameCount: 1001,
    teamIds: [0, 1],
    allyOf: (id) => id === 0,
    labelOf: (id) => `Équipe ${id}`,
    ...over,
  }
}

describe('buildScoreCurve — quand la courbe existe, et quand elle n’existe pas', () => {
  it('rend null sans calque : pas d’artefact, RIEN à l’écran (jamais un cadre vide)', () => {
    expect(buildScoreCurve(input({ timeline: undefined }))).toBeNull()
  })

  it('rend null sans échelle temporelle — un axe en mm:ss sans durée d’image serait inventé', () => {
    expect(buildScoreCurve(input({ frameIntervalMs: undefined }))).toBeNull()
  })

  it('rend null à moins de deux camps : une seule courbe ne compare rien', () => {
    expect(buildScoreCurve(input({ teamIds: [0] }))).toBeNull()
  })

  it('rend null quand AUCUN camp n’est publié — le calque existe mais ne dit rien', () => {
    expect(buildScoreCurve(input({ timeline: timelineOf([]) }))).toBeNull()
  })

  it('rend une courbe par camp quand au moins un est publié', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series.map((s) => s.teamId)).toEqual([0, 1])
    expect(curve?.durationMs).toBe(100_000)
  })
})

describe('buildScoreCurve — les paliers et leurs deux bornes', () => {
  it('commence à ZÉRO au coup d’envoi : sans ce point, la courbe naîtrait au premier but', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series[0].points[0]).toEqual([0, 0])
  })

  it('convertit chaque palier en millisecondes de rejeu', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series[1].points).toEqual([
      [0, 0],
      [20_000, 1],
      [30_000, 2],
      [100_000, 2],
    ])
  })

  it('TIENT jusqu’à la fin du match : le dernier palier ne s’arrête pas au dernier but', () => {
    const curve = buildScoreCurve(input())
    const derniers = curve?.series[0].points.slice(-1)[0]
    expect(derniers).toEqual([100_000, 1])
  })

  it('ne DOUBLE pas le point de départ quand un palier tombe à l’image 0', () => {
    const curve = buildScoreCurve(input({ timeline: timelineOf([equipe(0, [[0, 1]]), equipe(1, [[50, 1]])]) }))
    expect(curve?.series[0].points[0]).toEqual([0, 1])
    expect(curve?.series[0].points).toHaveLength(2)
  })

  it('trace le camp SANS série à plat, à zéro — témoin CTF 3-0', () => {
    const ctf = timelineOf([equipe(0, [[1347, 1], [1808, 2], [4678, 3]])])
    const curve = buildScoreCurve(input({ timeline: ctf, frameCount: 4751 }))
    const perdant = curve?.series.find((s) => s.teamId === 1)
    expect(perdant?.published).toBe(false)
    expect(perdant?.final).toBe(0)
    expect(perdant?.points).toEqual([
      [0, 0],
      [475_000, 0],
    ])
    // Et le camp qui marque garde son escalier complet, jusqu'à 3.
    const vainqueur = curve?.series.find((s) => s.teamId === 0)
    expect(vainqueur?.published).toBe(true)
    expect(vainqueur?.final).toBe(3)
  })

  it('reporte le camp du joueur de la page et le libellé, sans les recalculer', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series.map((s) => s.ally)).toEqual([true, false])
    expect(curve?.series.map((s) => s.label)).toEqual(['Équipe 0', 'Équipe 1'])
  })
})

describe('buildScoreCurve — les retournements, datés en millisecondes', () => {
  it('n’en rend aucun sur un match mené de bout en bout (témoin Slayer)', () => {
    const slayer = timelineOf([
      equipe(0, [[399, 1], [430, 2], [656, 3]]),
      equipe(1, [[317, 1], [408, 2], [486, 3], [700, 5]]),
    ])
    expect(buildScoreCurve(input({ timeline: slayer, frameCount: 4985 }))?.leads).toEqual([])
  })

  it('date chacun en ms de rejeu (témoin Oddball : trois)', () => {
    const oddball = timelineOf([
      equipe(0, [[595, 1], [653, 4], [1807, 53], [2345, 79], [5000, 200]]),
      equipe(1, [[493, 1], [1807, 54], [2345, 78], [5017, 121]]),
    ])
    expect(buildScoreCurve(input({ timeline: oddball, frameCount: 5137 }))?.leads).toEqual([
      { ms: 65_300, teamId: 0 },
      { ms: 180_700, teamId: 1 },
      { ms: 234_500, teamId: 0 },
    ])
  })
})

describe('teamIdsOf — les camps à tracer', () => {
  it('prend ceux du scoreboard, dans un ordre stable et sans doublon', () => {
    expect(teamIdsOf([0, 1, 0, 1, null], undefined)).toEqual([0, 1])
  })

  it('ajoute un camp que SEUL le film connaît — un but doit appartenir à quelqu’un', () => {
    expect(teamIdsOf([0], timelineOf([equipe(0, [[1, 1]]), equipe(3, [[2, 1]])]))).toEqual([0, 3])
  })

  it('ignore les camps sans identité résolue', () => {
    const flou = timelineOf([equipe(0, [[1, 1]]), { rounds: null, total: [{ t: 2, v: 1 }] }])
    expect(teamIdsOf([], flou)).toEqual([0])
  })
})

describe('formatClock', () => {
  it('écrit l’instant en mm:ss, secondes sur deux chiffres', () => {
    expect(formatClock(0)).toBe('0:00')
    expect(formatClock(65_300)).toBe('1:05')
    expect(formatClock(180_700)).toBe('3:01')
    expect(formatClock(600_000)).toBe('10:00')
  })

  it('ne rend jamais un temps négatif', () => {
    expect(formatClock(-5_000)).toBe('0:00')
  })
})
