/**
 * _scoreEvents.test.ts — la projection du calque de score en MARQUES datées.
 *
 * Le témoin de référence est le CTF `530820e5` (3-0) : trois captures pour un camp, et un
 * camp qui n'émet RIEN du tout — le film dit son zéro en se taisant.
 */
import { describe, expect, it } from 'vitest'

import { normalizeScoreTimeline } from '@/lib/replay/scoreTimeline'

import { buildScoreEvents, type ScoreEventsInput } from './_scoreEvents'

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
function input(over: Partial<ScoreEventsInput> = {}): ScoreEventsInput {
  return {
    timeline: timelineOf([
      equipe(0, [
        [100, 1],
        [300, 2],
        [700, 3],
      ]),
      equipe(1, []),
    ]),
    frameIntervalMs: 100,
    frameCount: 1001,
    teamIds: [0, 1],
    allyOf: (id) => id === 0,
    labelOf: (id) => `Équipe ${id}`,
    ...over,
  }
}

describe('buildScoreEvents — quand le graphe existe, et quand il n’existe pas', () => {
  it('rend null sans calque : pas d’artefact, RIEN à l’écran (jamais un cadre vide)', () => {
    expect(buildScoreEvents(input({ timeline: undefined }))).toBeNull()
  })

  it('rend null sans échelle temporelle — dater les marques serait une invention', () => {
    expect(buildScoreEvents(input({ frameIntervalMs: undefined }))).toBeNull()
  })

  it('rend null à moins de deux images : il n’y a pas d’axe des temps', () => {
    expect(buildScoreEvents(input({ frameCount: 1 }))).toBeNull()
  })

  it('rend null à moins de deux camps : une seule voie ne compare rien', () => {
    expect(buildScoreEvents(input({ teamIds: [0] }))).toBeNull()
  })

  it('rend null quand AUCUN camp n’est publié — le calque existe mais ne dit rien', () => {
    expect(buildScoreEvents(input({ timeline: timelineOf([]) }))).toBeNull()
  })

  it('rend null quand personne n’a marqué : un graphe de barres SANS barre est un cadre vide', () => {
    expect(buildScoreEvents(input({ timeline: timelineOf([equipe(0, []), equipe(1, [])]) }))).toBeNull()
  })

  it('rend une voie par camp dès qu’une marque existe, et borne l’axe à la fin du rejeu', () => {
    const events = buildScoreEvents(input())
    expect(events?.teams.map((s) => s.teamId)).toEqual([0, 1])
    expect(events?.durationMs).toBe(100_000)
  })
})

describe('buildScoreEvents — les marques et leur valeur', () => {
  it('date chaque marque en millisecondes de rejeu et compte son DELTA', () => {
    const events = buildScoreEvents(input())
    expect(events?.teams[0].events).toEqual([
      { ms: 10_000, points: 1, total: 1 },
      { ms: 30_000, points: 1, total: 2 },
      { ms: 70_000, points: 1, total: 3 },
    ])
  })

  it('lit le PREMIER palier depuis zéro : le match commence à 0-0, c’est une mesure', () => {
    const events = buildScoreEvents(
      input({ timeline: timelineOf([equipe(0, [[100, 4]]), equipe(1, [])]) }),
    )
    expect(events?.teams[0].events).toEqual([{ ms: 10_000, points: 4, total: 4 }])
  })

  it('reporte un delta de PLUSIEURS points tel quel (un mode peut en donner plus d’un)', () => {
    const events = buildScoreEvents(
      input({
        timeline: timelineOf([
          equipe(0, [
            [100, 1],
            [200, 6],
          ]),
          equipe(1, []),
        ]),
      }),
    )
    expect(events?.teams[0].events[1]).toEqual({ ms: 20_000, points: 5, total: 6 })
  })

  it('écarte les paliers qui ne font PAS monter le compteur — une barre à zéro mentirait', () => {
    const events = buildScoreEvents(
      input({
        timeline: timelineOf([
          equipe(0, [
            [100, 1],
            [200, 1],
            [300, 0],
            [400, 2],
          ]),
          equipe(1, []),
        ]),
      }),
    )
    // Le CUMUL reporté reste celui du film : après le retour à 0, le palier 2 vaut +2.
    expect(events?.teams[0].events).toEqual([
      { ms: 10_000, points: 1, total: 1 },
      { ms: 40_000, points: 2, total: 2 },
    ])
  })

  it('garde le camp SANS série : sa voie vide dit « il n’a pas marqué » (témoin CTF 3-0)', () => {
    const events = buildScoreEvents(
      input({ timeline: timelineOf([equipe(0, [[100, 1]])]) }),
    )
    const muet = events?.teams.find((s) => s.teamId === 1)
    expect(muet?.published).toBe(false)
    expect(muet?.events).toEqual([])
  })

  it('porte le camp et le nom de chaque équipe — l’identité ne tient pas à la seule couleur', () => {
    const events = buildScoreEvents(input())
    expect(events?.teams.map((s) => [s.ally, s.label])).toEqual([
      [true, 'Équipe 0'],
      [false, 'Équipe 1'],
    ])
  })
})
