/**
 * _scoreCurve.test.ts — la projection du calque de score en courbes.
 *
 * Les cas reprennent les trois témoins re-cuits de la phase 1 : Slayer `000d5950` (43/50,
 * aucun retournement), CTF `530820e5` (3-0, UNE seule série publiée pour deux camps),
 * Oddball `24dbb67d` (200/121, trois retournements).
 *
 * L'HORLOGE DU TÉMOIN N'EST PAS NEUTRE, ET C'EST VOULU (registre 2026-09-05, P0-7) : l'image
 * zéro du film tombe 5 s après le début du match, le coup d'envoi 20 s après — le film
 * commence donc 15 s AVANT le coup d'envoi, et toute abscisse attendue ci-dessous a été
 * calculée à la main comme `image × 100 − 15 000`. Une projection qui oublierait la
 * soustraction ferait rougir chacun de ces oracles.
 */
import { describe, expect, it } from 'vitest'

import { matchClock, type MatchClock } from '@/lib/replay/matchClock'
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

/**
 * L'horloge du témoin : 100 ms par image, origine du film à 5 s du début du match, coup
 * d'envoi à 20 s. Le coup d'envoi tombe donc sur l'image 150, et l'axe du gameplay vaut
 * `image × 100 − 15 000`.
 */
function horloge(frameCount = 1001): MatchClock {
  return matchClock({ originMs: 5_000, frameIntervalMs: 100, frameCount }, { t0_ms: 20_000 })!
}

/** L'entrée nominale : le témoin ci-dessus, 1 001 images (100 s de film), deux camps. */
function input(over: Partial<ScoreCurveInput> = {}): ScoreCurveInput {
  return {
    timeline: timelineOf([equipe(0, [[200, 1]]), equipe(1, [[300, 1], [400, 2]])]),
    clock: horloge(),
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

  it('rend null SANS HORLOGE — sans origine du film, l’instant d’un palier est un inconnu', () => {
    expect(buildScoreCurve(input({ clock: null }))).toBeNull()
    expect(buildScoreCurve(input({ clock: undefined }))).toBeNull()
  })

  it('rend null à moins de deux camps : une seule courbe ne compare rien', () => {
    expect(buildScoreCurve(input({ teamIds: [0] }))).toBeNull()
  })

  it('rend null quand AUCUN camp n’est publié — le calque existe mais ne dit rien', () => {
    expect(buildScoreCurve(input({ timeline: timelineOf([]) }))).toBeNull()
  })

  it('rend null quand le film s’arrête AVANT le coup d’envoi : aucun gameplay à tracer', () => {
    // 140 images de film (14 s) alors que le coup d'envoi tombe à l'image 150.
    expect(buildScoreCurve(input({ clock: horloge(141) }))).toBeNull()
  })

  it('rend une courbe par camp quand au moins un est publié', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series.map((s) => s.teamId)).toEqual([0, 1])
    // 1 000 images de film = 100 s, dont 15 s avant le coup d'envoi : 85 s de gameplay.
    expect(curve?.endMs).toBe(85_000)
  })
})

describe('buildScoreCurve — les paliers et leurs deux bornes', () => {
  it('commence à ZÉRO au coup d’envoi : sans ce point, la courbe naîtrait au premier but', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series[0].points[0]).toEqual([0, 0])
  })

  it('date chaque palier en millisecondes DEPUIS LE COUP D’ENVOI', () => {
    const curve = buildScoreCurve(input())
    // Images 300 et 400 = 30 s et 40 s de film = 15 s et 25 s après un coup d'envoi
    // tombé, lui, à 15 s de film.
    expect(curve?.series[1].points).toEqual([
      [0, 0],
      [15_000, 1],
      [25_000, 2],
      [85_000, 2],
    ])
  })

  it('TIENT jusqu’à la fin du match : le dernier palier ne s’arrête pas au dernier but', () => {
    const curve = buildScoreCurve(input())
    const derniers = curve?.series[0].points.slice(-1)[0]
    expect(derniers).toEqual([85_000, 1])
  })

  it('REPLIE sur le coup d’envoi ce que le film date avant lui, sans doubler le point', () => {
    // L'image 100 tombe 5 s AVANT le coup d'envoi : ce n'est pas un but d'avant-match,
    // c'est l'état du compteur au coup d'envoi.
    const curve = buildScoreCurve(
      input({ timeline: timelineOf([equipe(0, [[100, 1]]), equipe(1, [[300, 1]])]) }),
    )
    expect(curve?.series[0].points[0]).toEqual([0, 1])
    expect(curve?.series[0].points).toHaveLength(2)
  })

  it('trace le camp SANS série à plat, à zéro — témoin CTF 3-0', () => {
    const ctf = timelineOf([equipe(0, [[1347, 1], [1808, 2], [4678, 3]])])
    const curve = buildScoreCurve(input({ timeline: ctf, clock: horloge(4751) }))
    const perdant = curve?.series.find((s) => s.teamId === 1)
    expect(perdant?.published).toBe(false)
    expect(perdant?.final).toBe(0)
    // 4 750 images = 475 s de film, moins les 15 s d'avant coup d'envoi.
    expect(perdant?.points).toEqual([
      [0, 0],
      [460_000, 0],
    ])
    // Et le camp qui marque garde son escalier complet, jusqu'à 3.
    const vainqueur = curve?.series.find((s) => s.teamId === 0)
    expect(vainqueur?.published).toBe(true)
    expect(vainqueur?.final).toBe(3)
    expect(vainqueur?.points).toEqual([
      [0, 0],
      [119_700, 1],
      [165_800, 2],
      [452_800, 3],
      [460_000, 3],
    ])
  })

  it('reporte le camp du joueur de la page et le libellé, sans les recalculer', () => {
    const curve = buildScoreCurve(input())
    expect(curve?.series.map((s) => s.ally)).toEqual([true, false])
    expect(curve?.series.map((s) => s.label)).toEqual(['Équipe 0', 'Équipe 1'])
  })
})

describe('buildScoreCurve — les retournements, datés depuis le coup d’envoi', () => {
  it('n’en rend aucun sur un match mené de bout en bout (témoin Slayer)', () => {
    const slayer = timelineOf([
      equipe(0, [[399, 1], [430, 2], [656, 3]]),
      equipe(1, [[317, 1], [408, 2], [486, 3], [700, 5]]),
    ])
    expect(buildScoreCurve(input({ timeline: slayer, clock: horloge(4985) }))?.leads).toEqual([])
  })

  it('date chacun sur l’horloge du gameplay (témoin Oddball : trois)', () => {
    const oddball = timelineOf([
      equipe(0, [[595, 1], [653, 4], [1807, 53], [2345, 79], [5000, 200]]),
      equipe(1, [[493, 1], [1807, 54], [2345, 78], [5017, 121]]),
    ])
    // Images 653 / 1 807 / 2 345 = 65,3 / 180,7 / 234,5 s de film, moins 15 s.
    expect(buildScoreCurve(input({ timeline: oddball, clock: horloge(5137) }))?.leads).toEqual([
      { ms: 50_300, teamId: 0 },
      { ms: 165_700, teamId: 1 },
      { ms: 219_500, teamId: 0 },
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
