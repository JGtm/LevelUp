/**
 * _scoreEvents.test.ts — la projection du calque de score en MARQUES datées.
 *
 * Le témoin de référence est le CTF `530820e5` (3-0) : trois captures pour un camp, et un
 * camp qui n'émet RIEN du tout — le film dit son zéro en se taisant.
 *
 * L'HORLOGE DU TÉMOIN N'EST PAS NEUTRE, ET C'EST VOULU (registre 2026-09-05, P0-7) : l'image
 * zéro du film tombe 5 s après le début du match, le coup d'envoi 20 s après — le film
 * commence donc 15 s AVANT le coup d'envoi, et toute abscisse attendue ci-dessous a été
 * calculée à la main comme `image × 100 − 15 000`.
 */
import { describe, expect, it } from 'vitest'

import { matchClock, type MatchClock } from '@/lib/replay/matchClock'
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

/**
 * L'horloge du témoin : 100 ms par image, origine du film à 5 s du début du match, coup
 * d'envoi à 20 s. Le coup d'envoi tombe donc sur l'image 150.
 */
function horloge(frameCount = 1001): MatchClock {
  return matchClock({ originMs: 5_000, frameIntervalMs: 100, frameCount }, { t0_ms: 20_000 })!
}

/** L'entrée nominale : le témoin ci-dessus, 1 001 images (100 s de film), deux camps. */
function input(over: Partial<ScoreEventsInput> = {}): ScoreEventsInput {
  return {
    timeline: timelineOf([
      equipe(0, [
        [200, 1],
        [400, 2],
        [800, 3],
      ]),
      equipe(1, []),
    ]),
    clock: horloge(),
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

  it('rend null SANS HORLOGE — sans origine du film, dater une marque serait une invention', () => {
    expect(buildScoreEvents(input({ clock: null }))).toBeNull()
    expect(buildScoreEvents(input({ clock: undefined }))).toBeNull()
  })

  it('rend null quand le film s’arrête AVANT le coup d’envoi : aucun gameplay à tracer', () => {
    expect(buildScoreEvents(input({ clock: horloge(141) }))).toBeNull()
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

  it('rend une voie par camp dès qu’une marque existe, et borne l’axe à la fin du film', () => {
    const events = buildScoreEvents(input())
    expect(events?.teams.map((s) => s.teamId)).toEqual([0, 1])
    // 1 000 images de film = 100 s, dont 15 s avant le coup d'envoi : 85 s de gameplay.
    expect(events?.endMs).toBe(85_000)
  })
})

describe('buildScoreEvents — les marques et leur valeur', () => {
  it('date chaque marque DEPUIS LE COUP D’ENVOI et compte son DELTA', () => {
    const events = buildScoreEvents(input())
    // Images 200 / 400 / 800 = 20 / 40 / 80 s de film, moins les 15 s d'avant-match.
    expect(events?.teams[0].events).toEqual([
      { ms: 5_000, points: 1, total: 1 },
      { ms: 25_000, points: 1, total: 2 },
      { ms: 65_000, points: 1, total: 3 },
    ])
  })

  it('RANGE AU COUP D’ENVOI une marque que le film date avant lui', () => {
    // L'image 100 tombe 5 s avant le coup d'envoi : sur l'axe, elle se pose à 0 plutôt que
    // de sortir du cadre sans le dire.
    const events = buildScoreEvents(
      input({ timeline: timelineOf([equipe(0, [[100, 1]]), equipe(1, [])]) }),
    )
    expect(events?.teams[0].events).toEqual([{ ms: 0, points: 1, total: 1 }])
  })

  it('lit le PREMIER palier depuis zéro : le match commence à 0-0, c’est une mesure', () => {
    const events = buildScoreEvents(
      input({ timeline: timelineOf([equipe(0, [[200, 4]]), equipe(1, [])]) }),
    )
    expect(events?.teams[0].events).toEqual([{ ms: 5_000, points: 4, total: 4 }])
  })

  it('reporte un delta de PLUSIEURS points tel quel (un mode peut en donner plus d’un)', () => {
    const events = buildScoreEvents(
      input({
        timeline: timelineOf([
          equipe(0, [
            [200, 1],
            [300, 6],
          ]),
          equipe(1, []),
        ]),
      }),
    )
    expect(events?.teams[0].events[1]).toEqual({ ms: 15_000, points: 5, total: 6 })
  })

  it('écarte les paliers qui ne font PAS monter le compteur — une barre à zéro mentirait', () => {
    const events = buildScoreEvents(
      input({
        timeline: timelineOf([
          equipe(0, [
            [200, 1],
            [300, 1],
            [400, 0],
            [500, 2],
          ]),
          equipe(1, []),
        ]),
      }),
    )
    // Le CUMUL reporté reste celui du film : après le retour à 0, le palier 2 vaut +2.
    expect(events?.teams[0].events).toEqual([
      { ms: 5_000, points: 1, total: 1 },
      { ms: 35_000, points: 2, total: 2 },
    ])
  })

  it('garde le camp SANS série : sa voie vide dit « il n’a pas marqué » (témoin CTF 3-0)', () => {
    const events = buildScoreEvents(
      input({ timeline: timelineOf([equipe(0, [[200, 1]])]) }),
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
