/**
 * scoreBannerLogic.test.ts — ce que le bandeau affiche, et ce qu'il refuse d'afficher.
 *
 * Les cas reprennent les trois témoins re-cuits du calque de score : Slayer `000d5950`
 * (43/50, deux séries), CTF `530820e5` (3-0, UNE seule série publiée — le piège qui a fait
 * réécrire ce module) et Oddball `24dbb67d` (200/121 en deux manches). Les refus, eux, sont
 * la moitié utile : un bandeau qui se rend quand il ne sait pas est pire qu'un bandeau absent.
 */
import { describe, expect, it } from 'vitest'

import { normalizeScoreTimeline, type ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { readScoreBanner } from './scoreBannerLogic'

/** Fabrique un calque normalisé depuis une forme brute — par la frontière du calque. */
function timelineOf(raw: unknown): ReplayScoreTimelineReady {
  const timeline = normalizeScoreTimeline(raw as never)
  if (!timeline) throw new Error('calque absent après normalisation')
  return timeline
}

/** Une série d'équipe à manche unique (le cas des modes sans manche). */
function equipe(teamId: number, points: Array<[number, number]>) {
  const pts = points.map(([t, v]) => ({ t, v }))
  return { teamId, rounds: [{ round: 0, points: pts }], total: pts }
}

/** Deux lignes de scoreboard : le camp et l'identité suffisent à ce module. */
function board(sides: Array<[string, string | null]>): Array<Pick<MatchScoreboardRow, 'xuid' | 'team_side'>> {
  return sides.map(([xuid, team_side]) => ({ xuid, team_side }))
}

/** L'index de camps : `true` = du côté du joueur de la page. */
function allies(entries: Array<[string, boolean]>): ReadonlyMap<string, { ally: boolean }> {
  return new Map(entries.map(([xuid, ally]) => [xuid, { ally }]))
}

/** Le témoin Slayer : t0 mène 43, t1 suit à 30 au frame 500. */
const SLAYER = () =>
  timelineOf({
    teams: [
      equipe(0, [
        [0, 0],
        [100, 20],
        [500, 43],
      ]),
      equipe(1, [
        [0, 0],
        [100, 15],
        [500, 30],
      ]),
    ],
    players: [],
  })

const SB_2V2 = board([
  ['moi', 't0'],
  ['eux', 't1'],
])
const ALLY_T0 = allies([
  ['moi', true],
  ['eux', false],
])

describe('readScoreBanner — les deux camps au frame lu', () => {
  it('met le camp du joueur de la page à gauche, avec son score au frame', () => {
    const r = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 500)
    expect(r).not.toBeNull()
    expect(r?.ally).toMatchObject({ teamId: 0, score: 43 })
    expect(r?.enemy).toMatchObject({ teamId: 1, score: 30 })
  })

  it('inverse les côtés quand le joueur de la page est dans t1', () => {
    const r = readScoreBanner(
      SLAYER(),
      SB_2V2,
      allies([
        ['moi', false],
        ['eux', true],
      ]),
      500,
    )
    expect(r?.ally.teamId).toBe(1)
    expect(r?.ally.score).toBe(30)
    expect(r?.enemy.teamId).toBe(0)
  })

  it('TIQUE : le score lu au frame 100 n\'est pas celui de la fin', () => {
    expect(readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 100)?.ally.score).toBe(20)
    expect(readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 500)?.ally.score).toBe(43)
  })

  it('rend 0 avant le premier palier, sans rien inventer', () => {
    const r = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 0)
    expect(r?.ally.score).toBe(0)
    expect(r?.enemy.score).toBe(0)
  })
})

describe('readScoreBanner — le remplissage sur la cible de victoire (le score final du vainqueur)', () => {
  it('rapporte chaque barre au score FINAL du vainqueur, pas au camp d\'en face au frame lu', () => {
    // À mi-match (20-15), la version relative remplissait la barre du meneur : ici les DEUX
    // disent leur chemin vers la cible (43), et aucune n'est pleine avant qu'elle soit atteinte.
    const mid = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 100)
    expect(mid?.ally.fill).toBeCloseTo(20 / 43, 6)
    expect(mid?.enemy.fill).toBeCloseTo(15 / 43, 6)
    const end = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 500)
    expect(end?.ally.fill).toBe(1)
    expect(end?.enemy.fill).toBeCloseTo(30 / 43, 6)
  })

  it('préfère la cible PUBLIÉE par l\'artefact : au chrono, le vainqueur ne finit pas plein', () => {
    // Le témoin Slayer arrêté au chrono à 43-30, avec la cible du mode (50) publiée.
    const withTarget = timelineOf({
      targetScore: 50,
      teams: [
        equipe(0, [
          [0, 0],
          [500, 43],
        ]),
        equipe(1, [
          [0, 0],
          [500, 30],
        ]),
      ],
      players: [],
    })
    const r = readScoreBanner(withTarget, SB_2V2, ALLY_T0, 500)
    expect(r?.ally.fill).toBeCloseTo(43 / 50, 6)
    expect(r?.enemy.fill).toBeCloseTo(30 / 50, 6)
  })

  it('ne se vide JAMAIS en cours de lecture : le dénominateur est constant', () => {
    let prevAlly = -1
    let prevEnemy = -1
    for (const frame of [0, 100, 300, 500]) {
      const r = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, frame)
      expect(r?.ally.fill).toBeGreaterThanOrEqual(prevAlly)
      expect(r?.enemy.fill).toBeGreaterThanOrEqual(prevEnemy)
      prevAlly = r?.ally.fill ?? 0
      prevEnemy = r?.enemy.fill ?? 0
    }
  })

  it('laisse les DEUX barres vides à 0-0 (et ne divise pas par zéro)', () => {
    const r = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 0)
    expect(r?.ally.fill).toBe(0)
    expect(r?.enemy.fill).toBe(0)
  })

  it('borne le remplissage à [0,1] quel que soit l\'écart', () => {
    const r = readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 500)
    for (const side of [r?.ally, r?.enemy]) {
      expect(side?.fill).toBeGreaterThanOrEqual(0)
      expect(side?.fill).toBeLessThanOrEqual(1)
    }
  })
})

describe('readScoreBanner — le témoin CTF 3-0 : un camp sans série vaut zéro', () => {
  /** Le camp qui n'a jamais marqué n'émet AUCUNE série : le calque ne publie que t0. */
  const CTF = () => timelineOf({ teams: [equipe(0, [[200, 1], [400, 3]])], players: [] })

  it('rend quand même le bandeau, avec 0 pour le camp muet', () => {
    const r = readScoreBanner(CTF(), SB_2V2, ALLY_T0, 400)
    expect(r).not.toBeNull()
    expect(r?.ally.score).toBe(3)
    expect(r?.enemy.score).toBe(0)
  })

  it('vide la barre du camp muet et remplit celle du marqueur', () => {
    const r = readScoreBanner(CTF(), SB_2V2, ALLY_T0, 400)
    expect(r?.ally.fill).toBe(1)
    expect(r?.enemy.fill).toBe(0)
  })
})

describe('readScoreBanner — les manches', () => {
  /** Le témoin Oddball : deux manches, total 200/121. */
  const ODDBALL = () =>
    timelineOf({
      teams: [
        {
          teamId: 0,
          rounds: [
            { round: 0, points: [[0, 0], [50, 100]].map(([t, v]) => ({ t, v })) },
            { round: 1, points: [[100, 0], [150, 100]].map(([t, v]) => ({ t, v })) },
          ],
          total: [[0, 0], [50, 100], [150, 200]].map(([t, v]) => ({ t, v })),
        },
        equipe(1, [
          [0, 0],
          [50, 78],
          [150, 121],
        ]),
      ],
      players: [],
    })

  it('annonce la manche en cours quand le mode en a plusieurs', () => {
    expect(readScoreBanner(ODDBALL(), SB_2V2, ALLY_T0, 50)?.round).toEqual({ index: 1, count: 2 })
    expect(readScoreBanner(ODDBALL(), SB_2V2, ALLY_T0, 150)?.round).toEqual({ index: 2, count: 2 })
  })

  it('affiche le TOTAL du match, jamais la valeur de la manche', () => {
    // Manche 2 en est à 100 ; le total, lui, dit 200 (l'écart du contresens : 100 points).
    expect(readScoreBanner(ODDBALL(), SB_2V2, ALLY_T0, 150)?.ally.score).toBe(200)
  })

  it('se tait sur un mode à manche unique (l\'indicateur répéterait le total)', () => {
    expect(readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 500)?.round).toBeNull()
  })
})

describe('readScoreBanner — les pastilles de manche, dans le camp du joueur de la page', () => {
  /** Oddball à deux manches, les DEUX équipes ayant leurs manches ventilées (t0 gagne 100/78 puis 100/43). */
  const ODDBALL_SPLIT = () =>
    timelineOf({
      teams: [
        {
          teamId: 0,
          rounds: [
            { round: 0, points: [[0, 0], [50, 100]].map(([t, v]) => ({ t, v })) },
            { round: 1, points: [[100, 0], [150, 100]].map(([t, v]) => ({ t, v })) },
          ],
          total: [[0, 0], [50, 100], [150, 200]].map(([t, v]) => ({ t, v })),
        },
        {
          teamId: 1,
          rounds: [
            { round: 0, points: [[0, 0], [50, 78]].map(([t, v]) => ({ t, v })) },
            { round: 1, points: [[100, 0], [150, 43]].map(([t, v]) => ({ t, v })) },
          ],
          total: [[0, 0], [50, 78], [150, 121]].map(([t, v]) => ({ t, v })),
        },
      ],
      players: [],
    })

  it('remplit les deux pastilles au camp allié en fin de match', () => {
    expect(readScoreBanner(ODDBALL_SPLIT(), SB_2V2, ALLY_T0, 200)?.dots).toEqual([
      { round: 0, winner: 'ally' },
      { round: 1, winner: 'ally' },
    ])
  })

  it('inverse le vainqueur des pastilles quand le joueur de la page est dans t1', () => {
    const r = readScoreBanner(
      ODDBALL_SPLIT(),
      SB_2V2,
      allies([
        ['moi', false],
        ['eux', true],
      ]),
      200,
    )
    expect(r?.dots).toEqual([
      { round: 0, winner: 'enemy' },
      { round: 1, winner: 'enemy' },
    ])
  })

  it('ne rend aucune pastille sur un mode à manche unique', () => {
    expect(readScoreBanner(SLAYER(), SB_2V2, ALLY_T0, 500)?.dots).toEqual([])
  })
})

describe('readScoreBanner — ce que le bandeau REFUSE d\'afficher', () => {
  it('FFA (aucun camp au scoreboard) : pas de bandeau', () => {
    const ffa = board([
      ['a', null],
      ['b', null],
      ['c', null],
    ])
    expect(readScoreBanner(SLAYER(), ffa, allies([['a', true]]), 500)).toBeNull()
  })

  it('trois camps : pas de bandeau (deux barres ne peuvent en dire trois)', () => {
    const sb = board([
      ['moi', 't0'],
      ['eux', 't1'],
      ['autre', 't2'],
    ])
    expect(readScoreBanner(SLAYER(), sb, ALLY_T0, 500)).toBeNull()
  })

  it('un seul camp : pas de bandeau', () => {
    const sb = board([
      ['moi', 't0'],
      ['coequipier', 't0'],
    ])
    expect(readScoreBanner(SLAYER(), sb, ALLY_T0, 500)).toBeNull()
  })

  it('calque absent (artefact antérieur au schéma 12) : pas de bandeau', () => {
    expect(readScoreBanner(undefined, SB_2V2, ALLY_T0, 500)).toBeNull()
  })

  it('calque SANS aucun camp : pas de bandeau — « 0 — 0 » serait une mesure inventée', () => {
    const vide = timelineOf({ teams: [], players: [] })
    expect(readScoreBanner(vide, SB_2V2, ALLY_T0, 500)).toBeNull()
  })

  it('aucun joueur reconnu : pas de côté, donc pas de bandeau', () => {
    expect(readScoreBanner(SLAYER(), SB_2V2, undefined, 500)).toBeNull()
    expect(readScoreBanner(SLAYER(), SB_2V2, allies([['inconnu', true]]), 500)).toBeNull()
  })

  it('scoreboard contradictoire (les deux camps alliés) : pas de bandeau', () => {
    const r = readScoreBanner(
      SLAYER(),
      SB_2V2,
      allies([
        ['moi', true],
        ['eux', true],
      ]),
      500,
    )
    expect(r).toBeNull()
  })
})

describe('readScoreBanner — un seul camp reconnu suffit à nommer l\'autre', () => {
  it('déduit le camp adverse quand seul l\'allié est reconnu', () => {
    const r = readScoreBanner(SLAYER(), SB_2V2, allies([['moi', true]]), 500)
    expect(r?.ally.teamId).toBe(0)
    expect(r?.enemy.teamId).toBe(1)
  })

  it('déduit le camp allié quand seul l\'adverse est reconnu', () => {
    const r = readScoreBanner(SLAYER(), SB_2V2, allies([['eux', false]]), 500)
    expect(r?.ally.teamId).toBe(0)
    expect(r?.enemy.teamId).toBe(1)
  })
})
