/**
 * scoreTimeline.test.ts — la lecture du score au frame courant, et ses trois pièges.
 *
 * Les cas ne sont pas inventés : ils reprennent les trois témoins re-cuits de la phase 1
 * (`LOTA_PHASE1.md`) — Slayer `000d5950` 43/50, CTF `530820e5` 3-0 avec UNE seule série
 * publiée, Oddball `24dbb67d` 200/121 en deux manches (100/78 puis 100/43).
 *
 * Les fixtures passent par `normalizeScoreTimeline`, la frontière du calque, qui vit dans
 * ce même module : le test n'a donc besoin d'AUCUN document de rejeu complet, ni d'aucune
 * feature. C'est ce qui rend cette logique déplaçable — et c'est tout l'objet du
 * déplacement dans `lib/` (ratchet cross-feature P8.5).
 */
import { describe, expect, it } from 'vitest'

import {
  filmClockTrusted,
  leadChanges,
  normalizeScoreTimeline,
  playerCountersAt,
  scoreAtFrame,
  scoreTimelineOf,
  teamIdOfSide,
  teamRoundScoreAtFrame,
  teamScoreAtFrame,
  teamSeriesFor,
} from './scoreTimeline'

import type { ReplayScoreDocument, ReplayScoreTimelineReady } from './scoreTimeline'

/** Fabrique un calque normalisé depuis une forme brute — par la frontière du calque. */
function timelineOf(raw: unknown): ReplayScoreTimelineReady {
  const timeline = normalizeScoreTimeline(raw as never)
  if (!timeline) throw new Error('calque absent après normalisation')
  return timeline
}

/** Une série d'équipe à une seule manche (le cas des modes sans manche). */
function equipe(teamId: number, points: Array<[number, number]>) {
  const pts = points.map(([t, v]) => ({ t, v }))
  return { teamId, rounds: [{ round: 0, points: pts }], total: pts }
}

describe('scoreAtFrame — la valeur du palier courant', () => {
  const points = [
    { t: 399, v: 1 },
    { t: 430, v: 2 },
    { t: 656, v: 3 },
  ]

  it('rend 0 avant le premier point : le match a commencé, pas le compteur', () => {
    expect(scoreAtFrame(points, 0)).toBe(0)
    expect(scoreAtFrame(points, 398)).toBe(0)
  })

  it('rend la valeur DÈS le frame du palier, pas au suivant', () => {
    expect(scoreAtFrame(points, 399)).toBe(1)
    expect(scoreAtFrame(points, 430)).toBe(2)
  })

  it('MAINTIENT la dernière valeur entre deux paliers — le flux ne retransmet rien', () => {
    expect(scoreAtFrame(points, 429)).toBe(1)
    expect(scoreAtFrame(points, 655)).toBe(2)
  })

  it('maintient aussi APRÈS le dernier palier : le score final ne s’efface pas', () => {
    expect(scoreAtFrame(points, 4985)).toBe(3)
  })

  it('rend 0 sur une série vide, sans jamais lever', () => {
    expect(scoreAtFrame([], 1000)).toBe(0)
  })
})

describe('teamSeriesFor / teamScoreAtFrame — le camp qui n’a jamais marqué', () => {
  // Témoin CTF `530820e5` (3-0) : UNE seule série publiée pour deux camps.
  const ctf = timelineOf({
    teams: [equipe(0, [[1347, 1], [1808, 2], [4678, 3]])],
    players: null,
  })

  it('rend la série du camp qui marque', () => {
    expect(teamSeriesFor(ctf, 0)?.teamId).toBe(0)
    expect(teamScoreAtFrame(ctf, 0, 1808)).toBe(2)
    expect(teamScoreAtFrame(ctf, 0, 4751)).toBe(3)
  })

  it('rend null pour le camp SANS série — le film se tait, il ne se trompe pas', () => {
    expect(teamSeriesFor(ctf, 1)).toBeNull()
  })

  it('mais son score vaut ZÉRO partout : ne pas marquer est une mesure', () => {
    expect(teamScoreAtFrame(ctf, 1, 0)).toBe(0)
    expect(teamScoreAtFrame(ctf, 1, 4751)).toBe(0)
  })

  it('rend null (et zéro) quand le camp n’est pas identifié', () => {
    expect(teamSeriesFor(ctf, null)).toBeNull()
    expect(teamScoreAtFrame(ctf, null, 4678)).toBe(0)
    expect(teamSeriesFor(undefined, 0)).toBeNull()
  })

  it('traduit le camp du scoreboard en identifiant d’équipe du film', () => {
    expect(teamIdOfSide('t0')).toBe(0)
    expect(teamIdOfSide('t1')).toBe(1)
    expect(teamIdOfSide(null)).toBeNull()
    expect(teamIdOfSide('')).toBeNull()
  })
})

describe('teamRoundScoreAtFrame — le score DANS une manche, qui repart de zéro', () => {
  // Même témoin Oddball `24dbb67d` : team0 en deux manches (100 puis 100), total 200.
  const oddball = timelineOf({
    teams: [
      {
        teamId: 0,
        rounds: [
          { round: 0, points: [{ t: 595, v: 1 }, { t: 2787, v: 100 }] },
          { round: 1, points: [{ t: 3100, v: 1 }, { t: 5060, v: 100 }] },
        ],
        total: [
          { t: 595, v: 1 },
          { t: 2787, v: 100 },
          { t: 3100, v: 101 },
          { t: 5060, v: 200 },
        ],
      },
    ],
    players: null,
  })

  it('rend la valeur DE LA MANCHE, pas le cumul du match', () => {
    expect(teamRoundScoreAtFrame(oddball, 0, 0, 2787)).toBe(100)
    // La manche 2 REPART de zéro : 100 dans la manche là où le total dit 200.
    expect(teamRoundScoreAtFrame(oddball, 0, 1, 5060)).toBe(100)
    expect(teamScoreAtFrame(oddball, 0, 5060)).toBe(200)
  })

  it('rend 0 AVANT le premier palier de la manche (le compteur de manche n’a pas bougé)', () => {
    // f3000 est après la manche 1 mais avant le 1er palier de la manche 2 (f3100).
    expect(teamRoundScoreAtFrame(oddball, 0, 1, 3000)).toBe(0)
    expect(teamRoundScoreAtFrame(oddball, 0, 1, 3100)).toBe(1)
  })

  it('rend 0 pour une manche ABSENTE — le camp n’a pas cette manche', () => {
    expect(teamRoundScoreAtFrame(oddball, 0, 2, 5060)).toBe(0)
  })

  it('rend 0 pour un camp sans série ou non identifié', () => {
    expect(teamRoundScoreAtFrame(oddball, 5, 0, 2787)).toBe(0)
    expect(teamRoundScoreAtFrame(oddball, null, 0, 2787)).toBe(0)
    expect(teamRoundScoreAtFrame(undefined, 0, 0, 2787)).toBe(0)
  })
})

describe('playerCountersAt — publié, ou pas publié : jamais zéro par défaut', () => {
  // Témoin Slayer `000d5950` : 6 joueurs sur 8 portent des compteurs.
  const slayer = timelineOf({
    teams: null,
    players: [
      {
        xuid: '2533274815845110',
        score: { rounds: null, total: [{ t: 1000, v: 500 }, { t: 4408, v: 1520 }] },
        kills: { rounds: null, total: [{ t: 1000, v: 4 }, { t: 4203, v: 12 }] },
        deaths: { rounds: null, total: [{ t: 4796, v: 10 }] },
        assists: { rounds: null, total: [{ t: 4408, v: 6 }] },
      },
    ],
  })

  it('rend les quatre compteurs au frame courant', () => {
    expect(playerCountersAt(slayer, '2533274815845110', 4408)).toEqual({
      score: 1520,
      kills: 12,
      deaths: 0,
      assists: 6,
    })
  })

  it('lit chaque série sur SON dernier palier, pas sur celui d’une autre', () => {
    expect(playerCountersAt(slayer, '2533274815845110', 1500)).toEqual({
      score: 500,
      kills: 4,
      deaths: 0,
      assists: 0,
    })
  })

  it('rend NULL pour un joueur non publié — surtout pas un objet de zéros', () => {
    // Les 2 joueurs du témoin Slayer sans compteurs, et tout un mode (Oddball 0/32).
    expect(playerCountersAt(slayer, '2533274826120416', 4408)).toBeNull()
    expect(playerCountersAt(undefined, '2533274815845110', 4408)).toBeNull()
  })
})

describe('leadChanges — les retournements du match', () => {
  it('n’en compte AUCUN quand un seul camp marque (témoin CTF 3-0)', () => {
    const ctf = timelineOf({ teams: [equipe(0, [[1347, 1], [4678, 3]])], players: null })
    expect(leadChanges(ctf)).toEqual([])
  })

  it('n’en compte AUCUN quand le meneur mène de bout en bout (témoin Slayer)', () => {
    const slayer = timelineOf({
      teams: [
        equipe(0, [[399, 1], [430, 2], [656, 3], [4886, 43]]),
        equipe(1, [[317, 1], [408, 2], [486, 3], [700, 5], [4800, 49], [4908, 50]]),
      ],
      players: null,
    })
    expect(leadChanges(slayer)).toEqual([])
  })

  it('compte le passage devant, et pas la PREMIÈRE prise de tête', () => {
    const t = timelineOf({
      teams: [equipe(0, [[100, 1], [300, 2]]), equipe(1, [[200, 1], [400, 3]])],
      players: null,
    })
    // 100 : t0 prend la tête (première, pas un retournement) — 200 : égalité, suspendue —
    // 300 : t0 repasse devant après l'égalité, or t0 menait déjà : rien — 400 : t1 passe.
    expect(leadChanges(t)).toEqual([{ frame: 400, teamId: 1 }])
  })

  it('compte les TROIS retournements du témoin Oddball', () => {
    const oddball = timelineOf({
      teams: [
        equipe(0, [[595, 1], [653, 4], [1807, 53], [2345, 79], [5000, 200]]),
        equipe(1, [[493, 1], [1807, 54], [2345, 78], [5017, 121]]),
      ],
      players: null,
    })
    expect(leadChanges(oddball)).toEqual([
      { frame: 653, teamId: 0 },
      { frame: 1807, teamId: 1 },
      { frame: 2345, teamId: 0 },
    ])
  })

  it('ignore les camps sans identité — un seul identifié ne « mène » pas', () => {
    const flou = timelineOf({
      teams: [equipe(0, [[100, 1]]), { rounds: null, total: [{ t: 200, v: 5 }] }],
      players: null,
    })
    expect(leadChanges(flou)).toEqual([])
  })
})

describe('filmClockTrusted — la garde d’horloge (P2 de la revue du lot A phase 1)', () => {
  const doc = (coverage: { originResolved?: boolean } | undefined, originMs?: number): ReplayScoreDocument => ({
    coverage,
    originMs,
    scoreTimeline: timelineOf({ teams: [equipe(0, [[100, 1]])], players: null }),
  })

  it('MASQUE le calque quand l’origine n’est ni résolue ni publiée', () => {
    const d = doc({ originResolved: false })
    expect(filmClockTrusted(d)).toBe(false)
    expect(scoreTimelineOf(d)).toBeUndefined()
  })

  it('n’en masque AUCUN quand l’origine est publiée — cas des artefacts de schéma 11', () => {
    // `originResolved` est un booléen non pointeur : un vieil artefact dit `false` alors
    // qu'il porte un `originMs` valide. Le masquer serait perdre un calque juste.
    const d = doc({ originResolved: false }, 3604)
    expect(filmClockTrusted(d)).toBe(true)
    expect(scoreTimelineOf(d)?.teams).toHaveLength(1)
  })

  it('n’en masque aucun quand l’origine EST résolue', () => {
    expect(filmClockTrusted(doc({ originResolved: true }, 3604))).toBe(true)
    expect(filmClockTrusted(doc({ originResolved: true }))).toBe(true)
  })

  it('n’en masque aucun quand le document ne porte pas de couverture du tout', () => {
    expect(filmClockTrusted(doc(undefined, 3604))).toBe(true)
  })

  it('rend undefined quand l’artefact ne porte simplement aucun calque', () => {
    expect(scoreTimelineOf({})).toBeUndefined()
  })
})
