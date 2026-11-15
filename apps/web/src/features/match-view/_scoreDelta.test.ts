/**
 * _scoreDelta.test.ts — l'écart de score de l'équipe du joueur, bâti sur les paliers de la
 * courbe de score.
 *
 * Même horloge de témoin que `_scoreCurve.test.ts` : 100 ms par image, origine du film à 5 s
 * du début du match, coup d'envoi à 20 s — l'axe du gameplay vaut `image x 100 - 15 000`,
 * calculé à la main. Les cas reprennent les témoins re-cuits (Slayer, CTF 3-0 à une seule série,
 * Oddball à retournements) : l'écart doit en dire la même chose que le calque.
 */
import { describe, expect, it } from 'vitest'

import { matchClock, type MatchClock } from '@/lib/replay/matchClock'
import {
  campCountOf,
  leaderStates,
  normalizeScoreTimeline,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'

import type { ScoreCurveInput } from './_scoreCurve'
import { buildScoreDelta, scoreDeltaAt } from './_scoreDelta'

function timelineOf(teams: unknown): ReplayScoreTimelineReady | undefined {
  return normalizeScoreTimeline({ teams, players: null } as never)
}

function equipe(teamId: number | undefined, points: Array<[number, number]>) {
  const pts = points.map(([t, v]) => ({ t, v }))
  return { teamId, rounds: [{ round: 0, points: pts }], total: pts }
}

function horloge(frameCount = 1001): MatchClock {
  return matchClock({ originMs: 5_000, frameIntervalMs: 100, frameCount }, { t0_ms: 20_000 })!
}

/** L'abscisse de gameplay d'une image du film (cf. l'en-tête). */
const ms = (frame: number) => frame * 100 - 15_000

function input(over: Partial<ScoreCurveInput> = {}): ScoreCurveInput {
  return {
    timeline: timelineOf([equipe(0, [[200, 1], [500, 2]]), equipe(1, [[300, 1], [400, 2], [600, 3]])]),
    clock: horloge(),
    teamIds: [0, 1],
    allyOf: (id) => id === 0,
    labelOf: (id) => `Équipe ${id}`,
    ...over,
  }
}

describe('buildScoreDelta — la série', () => {
  it('rend l’écart signé de l’équipe du joueur : positif quand elle mène', () => {
    const d = buildScoreDelta(input())!
    // Paliers (images -> ms) : 200 -> A 1 ; 300 -> B 1 ; 400 -> B 2 ; 500 -> A 2 ; 600 -> B 3.
    expect(d.points.map((p) => [p.ms, p.ally, p.opponent, p.delta])).toEqual([
      [0, 0, 0, 0],
      [ms(200), 1, 0, 1],
      [ms(300), 1, 1, 0],
      [ms(400), 1, 2, -1],
      [ms(500), 2, 2, 0],
      [ms(600), 2, 3, -1],
      [d.endMs, 2, 3, -1],
    ])
  })

  it('le signe est relatif au JOUEUR : le même match vu de l’autre camp s’inverse', () => {
    const vuDeB = buildScoreDelta(input({ allyOf: (id) => id === 1 }))!
    const vuDeA = buildScoreDelta(input())!
    expect(vuDeB.points.map((p) => p.delta)).toEqual(vuDeA.points.map((p) => 0 - p.delta))
  })

  it('la série part du coup d’envoi et court jusqu’à la fin du film, sur l’horloge du gameplay', () => {
    const d = buildScoreDelta(input())!
    expect(d.points[0].ms).toBe(0)
    expect(d.points[d.points.length - 1].ms).toBe(d.endMs)
    expect(d.endMs).toBe(100_000 - 15_000)
  })

  it('un camp qui n’a jamais marqué est un zéro MESURÉ : le 3-0 à une seule série se lit', () => {
    const d = buildScoreDelta(
      input({ timeline: timelineOf([equipe(0, [[200, 1], [400, 3]])]), teamIds: [0, 1] }),
    )!
    expect(d.points.map((p) => p.delta)).toEqual([0, 1, 3, 3])
  })

  it('un palier qui ne change aucun des deux scores comparés n’ajoute pas de point', () => {
    // Trois camps : B marque alors que C mène ; l'écart contre la meilleure équipe adverse (C)
    // ne bouge pas, le point n'a rien à dire.
    const d = buildScoreDelta(
      input({
        timeline: timelineOf([
          equipe(0, [[200, 1]]),
          equipe(1, [[250, 1]]),
          equipe(2, [[220, 5]]),
        ]),
        teamIds: [0, 1, 2],
        allyOf: (id) => id === 0,
      }),
    )!
    expect(d.points.map((p) => p.ms)).toEqual([0, ms(200), ms(220), d.endMs])
  })

  it('rend les noms des camps pour l’infobulle', () => {
    const d = buildScoreDelta(input())!
    expect(d.allyLabel).toBe('Équipe 0')
    expect(d.opponentLabel).toBe('Équipe 1')
  })
})

describe('buildScoreDelta — plus de deux camps', () => {
  const multi = () =>
    buildScoreDelta(
      input({
        timeline: timelineOf([
          equipe(0, [[200, 4]]),
          equipe(1, [[300, 2]]),
          equipe(2, [[400, 6]]),
        ]),
        teamIds: [0, 1, 2],
        allyOf: (id) => id === 0,
      }),
    )!

  it('statue : l’écart se prend contre la MEILLEURE équipe adverse, et ne nomme aucun adversaire', () => {
    const d = multi()
    expect(d.multiTeam).toBe(true)
    expect(d.opponentLabel).toBeNull()
    // Après l'image 400 : 4 contre max(2, 6) = 6.
    expect(d.points.find((p) => p.ms === ms(400))).toMatchObject({ ally: 4, opponent: 6, delta: -2 })
  })

  it('deux camps : multiTeam vaut false et l’adversaire est nommé', () => {
    const d = buildScoreDelta(input())!
    expect(d.multiTeam).toBe(false)
  })
})

describe('buildScoreDelta — quand il n’y a rien à tracer', () => {
  it('rend null sans calque, sans horloge, ou avec moins de deux camps (comme la courbe)', () => {
    expect(buildScoreDelta(input({ timeline: undefined }))).toBeNull()
    expect(buildScoreDelta(input({ clock: null }))).toBeNull()
    expect(buildScoreDelta(input({ teamIds: [0] }))).toBeNull()
  })

  it('rend null quand le CAMP DU JOUEUR n’est pas résolu : aucune équipe alliée connue', () => {
    expect(buildScoreDelta(input({ allyOf: () => null }))).toBeNull()
    expect(buildScoreDelta(input({ allyOf: () => false }))).toBeNull()
  })

  it('rend null quand un camp d’en face est d’identité inconnue : rien n’est deviné', () => {
    expect(buildScoreDelta(input({ allyOf: (id) => (id === 0 ? true : null) }))).toBeNull()
  })

  it('rend null quand deux camps se disent alliés : la référence est ambiguë', () => {
    expect(buildScoreDelta(input({ allyOf: () => true }))).toBeNull()
  })

  it('rend null quand une série du calque n’a pas de camp (`teamIdentity` non résolue)', () => {
    expect(
      buildScoreDelta(
        input({ timeline: timelineOf([equipe(0, [[200, 1]]), equipe(undefined, [[300, 1]])]) }),
      ),
    ).toBeNull()
  })
})

describe('scoreDeltaAt — la valeur tenue à un instant', () => {
  const d = buildScoreDelta(input())!

  it('rend le dernier point au plus tard à l’instant lu : le score ATTEND entre deux paliers', () => {
    expect(scoreDeltaAt(d, ms(250))).toMatchObject({ ally: 1, opponent: 0, delta: 1 })
    expect(scoreDeltaAt(d, ms(300))).toMatchObject({ delta: 0 })
    expect(scoreDeltaAt(d, d.endMs)).toMatchObject({ delta: -1 })
  })

  it('borne aux extrémités : avant 0, le premier point ; après la fin, le dernier', () => {
    expect(scoreDeltaAt(d, -5_000).ms).toBe(0)
    expect(scoreDeltaAt(d, d.endMs + 9_999).ms).toBe(d.endMs)
  })
})

/**
 * LA PARITÉ AVEC `leaderStates` (D-11 : une seule définition du meneur dans le dépôt). Le signe
 * de l'écart doit dire la même chose que le meneur du calque : positif = le camp du joueur est
 * seul en tête, nul = égalité, négatif = un autre camp mène.
 */
describe('buildScoreDelta — parité avec leaderStates', () => {
  const CAS: Array<{ nom: string; teams: ReturnType<typeof equipe>[] }> = [
    {
      nom: 'Slayer (aucun retournement, deux égalités)',
      teams: [equipe(0, [[200, 1], [400, 2], [800, 40]]), equipe(1, [[300, 1], [450, 1], [700, 10]])],
    },
    {
      nom: 'CTF 3-0 (une seule série)',
      teams: [equipe(0, [[300, 1], [600, 2], [900, 3]])],
    },
    {
      nom: 'Oddball (retournements)',
      teams: [
        equipe(0, [[200, 10], [500, 12], [800, 20]]),
        equipe(1, [[300, 5], [450, 15], [700, 16]]),
      ],
    },
  ]

  for (const cas of CAS) {
    it(`${cas.nom} : le signe de l'écart est le meneur de leaderStates`, () => {
      const timeline = timelineOf(cas.teams)
      const delta = buildScoreDelta(input({ timeline }))!
      const etats = leaderStates(timeline, campCountOf([{ team: 0 }, { team: 1 }]))
      expect(etats.length).toBeGreaterThan(0)
      for (const etat of etats) {
        const attendu = etat.teamId === null ? 0 : etat.teamId === 0 ? 1 : -1
        expect(Math.sign(scoreDeltaAt(delta, ms(etat.frame)).delta), `image ${etat.frame}`).toBe(attendu)
      }
    })
  }
})
