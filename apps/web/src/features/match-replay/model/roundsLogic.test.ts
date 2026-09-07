/**
 * roundsLogic.test.ts — les manches lues dans le calque de score : combien, qui gagne, où
 * elles basculent.
 *
 * LE TÉMOIN est l'Oddball `24dbb67d` du dossier (cf. scoreTimeline.ts) : deux manches
 * gagnées 100/78 puis 100/43 par le même camp, total 200/121. On lui ajoute un troisième
 * témoin à trois manches partagées (allié, adverse, allié) pour éprouver l'ordre des
 * pastilles et le vainqueur par manche.
 */
import { describe, expect, it } from 'vitest'

import { normalizeScoreTimeline, type ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

import {
  activeRoundTransition,
  currentRoundAtFrame,
  roundCount,
  roundDots,
  roundsTally,
  roundTransitions,
} from './roundsLogic'

/** Fabrique un calque normalisé depuis une forme brute — par la frontière du calque. */
function timelineOf(raw: unknown): ReplayScoreTimelineReady {
  const timeline = normalizeScoreTimeline(raw as never)
  if (!timeline) throw new Error('calque absent après normalisation')
  return timeline
}

/** Une manche : numéro et paliers `[frame, valeur]`. */
function manche(round: number, points: Array<[number, number]>) {
  return { round, points: points.map(([t, v]) => ({ t, v })) }
}

/** Une équipe multi-manche : ses manches et son total cumulé. */
function equipe(teamId: number, rounds: ReturnType<typeof manche>[], total: Array<[number, number]>) {
  return { teamId, rounds, total: total.map(([t, v]) => ({ t, v })) }
}

/** Le témoin Oddball : t0 gagne les deux manches (100/78 à f50, 100/43 à f150). */
const ODDBALL = () =>
  timelineOf({
    teams: [
      equipe(
        0,
        [manche(0, [[0, 0], [50, 100]]), manche(1, [[100, 0], [150, 100]])],
        [[0, 0], [50, 100], [150, 200]],
      ),
      equipe(
        1,
        [manche(0, [[0, 0], [50, 78]]), manche(1, [[100, 0], [150, 43]])],
        [[0, 0], [50, 78], [150, 121]],
      ),
    ],
    players: [],
  })

/** Un match à trois manches partagées : t0 gagne la 1re et la 3e, t1 la 2e. */
const BEST_OF_THREE = () =>
  timelineOf({
    teams: [
      equipe(
        0,
        [
          manche(0, [[0, 0], [50, 100]]),
          manche(1, [[100, 0], [150, 40]]),
          manche(2, [[200, 0], [250, 100]]),
        ],
        [[0, 0], [50, 100], [150, 140], [250, 240]],
      ),
      equipe(
        1,
        [
          manche(0, [[0, 0], [50, 60]]),
          manche(1, [[100, 0], [150, 100]]),
          manche(2, [[200, 0], [250, 55]]),
        ],
        [[0, 0], [50, 60], [150, 160], [250, 215]],
      ),
    ],
    players: [],
  })

describe('roundCount — le nombre de manches jouées', () => {
  it('compte deux manches sur le témoin Oddball', () => {
    expect(roundCount(ODDBALL())).toBe(2)
  })

  it('compte trois manches sur un best-of-three', () => {
    expect(roundCount(BEST_OF_THREE())).toBe(3)
  })

  it('rend 0 sans calque', () => {
    expect(roundCount(undefined)).toBe(0)
  })

  it('rend 1 sur un mode à manche unique (Slayer)', () => {
    const slayer = timelineOf({
      teams: [equipe(0, [manche(0, [[0, 0], [500, 43]])], [[0, 0], [500, 43]])],
      players: [],
    })
    expect(roundCount(slayer)).toBe(1)
  })
})

describe('currentRoundAtFrame — la manche EN COURS du match (pas d\'un camp)', () => {
  it('bascule à la borne PARTAGÉE : le premier palier de l\'un OU l\'autre camp', () => {
    // La manche 1 de t0 ne commence qu\'à f130, mais celle de t1 dès f100 : la borne partagée
    // est f100, et les deux barres basculent ensemble dès f110.
    const shared = timelineOf({
      teams: [
        equipe(
          0,
          [manche(0, [[0, 0], [50, 100]]), manche(1, [[130, 0], [200, 100]])],
          [[0, 0], [50, 100], [200, 200]],
        ),
        equipe(
          1,
          [manche(0, [[0, 0], [50, 78]]), manche(1, [[100, 0], [200, 60]])],
          [[0, 0], [50, 78], [200, 138]],
        ),
      ],
      players: [],
    })
    expect(currentRoundAtFrame(shared, 110)).toEqual({ round: 1, index: 2, count: 2 })
  })

  it('rend la manche PRÉCÉDENTE dans la fenêtre inter-manche', () => {
    // f75 : la manche 0 est finie (dernier palier f50), la manche 1 n\'a pas commencé (f100).
    expect(currentRoundAtFrame(ODDBALL(), 75)).toEqual({ round: 0, index: 1, count: 2 })
  })

  it('rend la première manche avant tout début, et le bon compte', () => {
    expect(currentRoundAtFrame(BEST_OF_THREE(), 0)).toEqual({ round: 0, index: 1, count: 3 })
    expect(currentRoundAtFrame(BEST_OF_THREE(), 250)).toEqual({ round: 2, index: 3, count: 3 })
  })

  it('rend {index:1, count:1} sur un mode à manche unique', () => {
    const slayer = timelineOf({
      teams: [equipe(0, [manche(0, [[0, 0], [500, 43]])], [[0, 0], [500, 43]])],
      players: [],
    })
    expect(currentRoundAtFrame(slayer, 500)).toEqual({ round: 0, index: 1, count: 1 })
  })

  it('rend null quand aucune manche n\'est ventilée', () => {
    const vide = timelineOf({ teams: [{ teamId: 0, rounds: [], total: [] }], players: [] })
    expect(currentRoundAtFrame(vide, 100)).toBeNull()
    expect(currentRoundAtFrame(undefined, 100)).toBeNull()
  })
})

describe('roundDots — une pastille par manche, pleine quand la manche est tranchée', () => {
  it('rend deux pastilles, toutes deux gagnées par l\'allié en fin de match', () => {
    const dots = roundDots(ODDBALL(), 0, 1, 200)
    expect(dots).toEqual([
      { round: 0, winner: 'ally' },
      { round: 1, winner: 'ally' },
    ])
  })

  it('déduit le vainqueur PAR MANCHE, pas au total : allié, adverse, allié', () => {
    const dots = roundDots(BEST_OF_THREE(), 0, 1, 300)
    expect(dots.map((d) => d.winner)).toEqual(['ally', 'enemy', 'ally'])
  })

  it('la deuxième manche revient à l\'adverse même si l\'allié gagne le match', () => {
    // t0 mène 240-215 au total, mais la manche 1 finit 40-100 : elle est à l'adverse.
    const dots = roundDots(BEST_OF_THREE(), 0, 1, 300)
    expect(dots[1]).toEqual({ round: 1, winner: 'enemy' })
  })

  it('inverse quand le joueur de la page est dans l\'autre camp', () => {
    const dots = roundDots(BEST_OF_THREE(), 1, 0, 300)
    expect(dots.map((d) => d.winner)).toEqual(['enemy', 'ally', 'enemy'])
  })

  it('une manche NON tranchée reste vide (en cours ou à jouer)', () => {
    // À f120, la manche 0 est close (manche 1 a commencé à f100), la manche 1 est en cours.
    const dots = roundDots(ODDBALL(), 0, 1, 120)
    expect(dots[0]).toEqual({ round: 0, winner: 'ally' })
    expect(dots[1]).toEqual({ round: 1, winner: null })
  })

  it('la dernière manche se remplit quand la lecture atteint son dernier palier', () => {
    // Avant f150 (dernier palier de la manche 1), la manche 1 n'est pas encore tranchée.
    expect(roundDots(ODDBALL(), 0, 1, 149)[1]).toEqual({ round: 1, winner: null })
    expect(roundDots(ODDBALL(), 0, 1, 150)[1]).toEqual({ round: 1, winner: 'ally' })
  })

  it('toutes les pastilles sont vides au coup d\'envoi', () => {
    const dots = roundDots(BEST_OF_THREE(), 0, 1, 0)
    expect(dots.map((d) => d.winner)).toEqual([null, null, null])
  })

  it('se remplit dans l\'ordre au fil de la lecture (jamais un recul)', () => {
    const bo3 = BEST_OF_THREE()
    // f80 : manche 0 close, 1 en cours, 2 à jouer.
    expect(roundDots(bo3, 0, 1, 80).map((d) => d.winner)).toEqual(['ally', null, null])
    // f180 : manche 1 close aussi.
    expect(roundDots(bo3, 0, 1, 180).map((d) => d.winner)).toEqual(['ally', 'enemy', null])
    // f300 : tout est joué.
    expect(roundDots(bo3, 0, 1, 300).map((d) => d.winner)).toEqual(['ally', 'enemy', 'ally'])
  })

  it('se tait sur un mode à manche unique (aucune pastille)', () => {
    const slayer = timelineOf({
      teams: [
        equipe(0, [manche(0, [[0, 0], [500, 43]])], [[0, 0], [500, 43]]),
        equipe(1, [manche(0, [[0, 0], [500, 30]])], [[0, 0], [500, 30]]),
      ],
      players: [],
    })
    expect(roundDots(slayer, 0, 1, 500)).toEqual([])
  })

  it('sans calque : aucune pastille', () => {
    expect(roundDots(undefined, 0, 1, 100)).toEqual([])
  })
})

describe('roundTransitions — les bascules de manche', () => {
  it('une bascule sur le témoin Oddball, à la FIN de la manche 1', () => {
    expect(roundTransitions(ODDBALL())).toEqual([{ endedIndex: 1, frame: 50 }])
  })

  it('deux bascules sur un best-of-three, aux FINS des manches 1 et 2', () => {
    expect(roundTransitions(BEST_OF_THREE())).toEqual([
      { endedIndex: 1, frame: 50 },
      { endedIndex: 2, frame: 150 },
    ])
  })

  // LE CORRECTIF DU 2026-08-29, ÉPINGLÉ : la bascule tombait au DÉBUT de la manche suivante,
  // c'est-à-dire au premier point qu'on y marque — 19 à 34 s après la fin annoncée sur les
  // quatre témoins multi-manches du dossier d'artefacts. Ce cas force un entracte LONG et
  // vérifie que la bascule ne le traverse pas : elle reste collée au dernier palier.
  it("l'entracte ne décale plus la bascule : elle colle au dernier palier de la manche", () => {
    const entracteLong = timelineOf({
      teams: [
        equipe(
          0,
          [manche(0, [[10, 1], [60, 100]]), manche(1, [[330, 1], [400, 100]])],
          [[10, 1], [60, 100], [400, 200]],
        ),
        equipe(
          1,
          [manche(0, [[20, 1], [55, 42]]), manche(1, [[300, 1], [390, 61]])],
          [[20, 1], [55, 42], [390, 103]],
        ),
      ],
      players: [],
    })
    // Fin de la manche 1 : f60 (dernier palier, toutes équipes confondues). Reprise : f300.
    expect(roundTransitions(entracteLong)).toEqual([{ endedIndex: 1, frame: 60 }])
  })

  // LA MÊME BORNE QUE LA PASTILLE, et c'est tout le point du correctif : il n'y a plus qu'un
  // seul instant « fin de manche » dans le rejeu. Si les deux divergeaient à nouveau, le
  // message paraîtrait alors que la pastille est encore vide (ou l'inverse).
  it('partage la borne de la pastille pleine du bandeau', () => {
    const [premiere] = roundTransitions(BEST_OF_THREE())
    expect(roundDots(BEST_OF_THREE(), 0, 1, premiere.frame)[0].winner).toBe('ally')
    expect(roundDots(BEST_OF_THREE(), 0, 1, premiere.frame - 1)[0].winner).toBeNull()
  })

  it('aucune bascule sur un mode à manche unique', () => {
    const slayer = timelineOf({
      teams: [equipe(0, [manche(0, [[0, 0], [500, 43]])], [[0, 0], [500, 43]])],
      players: [],
    })
    expect(roundTransitions(slayer)).toEqual([])
  })

  it('aucune bascule sans calque', () => {
    expect(roundTransitions(undefined)).toEqual([])
  })
})

describe('activeRoundTransition — la fenêtre d\'affichage du message inter-manche', () => {
  const trs = roundTransitions(BEST_OF_THREE())

  it('rend la bascule quand l\'image lue est dans sa fenêtre', () => {
    expect(activeRoundTransition(trs, 50, 60)).toEqual({ endedIndex: 1, frame: 50 })
    expect(activeRoundTransition(trs, 109, 60)).toEqual({ endedIndex: 1, frame: 50 })
  })

  it('rien avant la bascule ni après la fenêtre', () => {
    expect(activeRoundTransition(trs, 49, 60)).toBeNull()
    expect(activeRoundTransition(trs, 110, 60)).toBeNull()
  })

  it('la seconde bascule s\'affiche à son tour', () => {
    expect(activeRoundTransition(trs, 160, 60)).toEqual({ endedIndex: 2, frame: 150 })
  })

  it('retient la PLUS RÉCENTE si deux fenêtres se recouvraient', () => {
    // Fenêtre large : à f170, les deux fenêtres (50 et 150) contiennent l'image.
    expect(activeRoundTransition(trs, 170, 200)).toEqual({ endedIndex: 2, frame: 150 })
  })

  it('aucune bascule active quand il n\'y en a pas', () => {
    expect(activeRoundTransition([], 100, 60)).toBeNull()
  })
})

// ─── roundsTally : le compte de manches en clair, sous l'horloge ────────────

describe('roundsTally', () => {
  it('compte les manches tranchées de chaque camp', () => {
    const tally = roundsTally([
      { round: 0, winner: 'ally' },
      { round: 1, winner: 'enemy' },
      { round: 2, winner: 'ally' },
    ])
    expect(tally).toEqual({ ally: 2, enemy: 1 })
  })

  it('ne compte NI les manches en cours NI les égalités', () => {
    // Une manche sans vainqueur n'appartient à personne : l'attribuer ferait mentir le total.
    const tally = roundsTally([
      { round: 0, winner: 'ally' },
      { round: 1, winner: null },
      { round: 2, winner: null },
    ])
    expect(tally).toEqual({ ally: 1, enemy: 0 })
  })

  it('rend 0-0 sur un mode à manche unique (aucune pastille)', () => {
    expect(roundsTally([])).toEqual({ ally: 0, enemy: 0 })
  })
})
