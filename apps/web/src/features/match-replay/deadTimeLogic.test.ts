/**
 * Tests — deadTimeLogic (le temps mort cumulé d'un joueur).
 *
 * CE QU'ILS PROTÈGENT : les deux bornes de la définition. Le temps AVANT la première vie et le
 * temps APRÈS la dernière ne sont pas du temps mort, et ce sont exactement les deux endroits où
 * un cumul naïf gonfle. Le reste tient la robustesse d'entrée : vies désordonnées, vies qui se
 * chevauchent, débordement de la fenêtre du match.
 *
 * Les vies entrent par `buildPlayers`, comme à l'écran : un test qui fabriquerait des
 * `ReplayPlayer` à la main éprouverait une structure que la page ne construit jamais.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayTrack } from '@/lib/api/types'

import { deadTimeByPlayer, formatDeadTime } from './deadTimeLogic'
import { buildPlayers } from './rosterLogic'
import { testReplayDoc } from './test/testDoc'

/** Une vie du film : un slot, un propriétaire, une fenêtre [start, end]. */
function life(slot: number, xuid: string, start: number, end: number): ReplayTrack {
  return {
    slot,
    team: -1,
    xuid,
    startFrame: start,
    endFrame: end,
    points: [{ t: start, x: 0, y: 0 }],
  }
}

/**
 * Temps mort du joueur `xuid`, en images — la grandeur que les cas ci-dessous raisonnent.
 * `frameIntervalMs: 1000` fait qu'une image vaut une seconde : les millisecondes rendues par
 * le module se relisent alors directement en images, sans arithmétique dans le test.
 */
function deadFrames(tracks: ReplayTrack[], xuid: string, frameCount = 1000): number {
  const doc = testReplayDoc({ frameCount, frameIntervalMs: 1000, tracks })
  // Joueur absent de la table = -1 : une valeur qui ne peut satisfaire aucune attente
  // ci-dessous, plutôt qu'un zéro qui se confondrait avec « aucune mort ».
  const ms = deadTimeByPlayer(buildPlayers(doc, []), doc).get(xuid) ?? -1000
  return ms / 1000
}

describe('deadTimeByPlayer — la définition et ses deux bornes', () => {
  it('une seule vie : aucun temps mort, et c’est une MESURE (0), pas une lacune', () => {
    expect(deadFrames([life(512, 'A', 0, 400)], 'A')).toBe(0)
  })

  it('cas nominal : deux trous entre trois vies, et rien d’autre', () => {
    // Vies [0,100], [180,300], [360,500] : trous de 80 puis 60 images.
    const trous = deadFrames(
      [life(512, 'A', 0, 100), life(513, 'A', 180, 300), life(514, 'A', 360, 500)],
      'A',
    )
    expect(trous).toBe(140)
  })

  it('AUCUN intervalle de tête : entrer en jeu à la 200e image ne coûte rien', () => {
    // Un joueur qui rejoint en cours de partie n'était pas mort avant d'exister.
    expect(deadFrames([life(512, 'A', 200, 400)], 'A')).toBe(0)
  })

  it('AUCUN intervalle de queue : mort sans retour (fin de match, abandon) n’accumule rien', () => {
    // Vie unique close à 100 dans un match de 1000 images : les 900 restantes ne comptent pas.
    // C'est la règle qui distingue « il est mort et n'est pas revenu » de « il a passé quinze
    // minutes à terre » — le film ne dit pas la seconde.
    expect(deadFrames([life(512, 'A', 0, 100)], 'A')).toBe(0)
    // Et avec un retour PUIS une mort finale : seul le trou daté des deux côtés compte.
    expect(deadFrames([life(512, 'A', 0, 100), life(513, 'A', 180, 300)], 'A')).toBe(80)
  })

  it('deux vies CONTIGUËS (fin = départ suivant) : trou nul', () => {
    expect(deadFrames([life(512, 'A', 0, 100), life(513, 'A', 100, 200)], 'A')).toBe(0)
  })

  it('vies désordonnées en entrée : le tri est refait ici, le total ne bouge pas', () => {
    const ordre = [life(514, 'A', 360, 500), life(512, 'A', 0, 100), life(513, 'A', 180, 300)]
    expect(deadFrames(ordre, 'A')).toBe(140)
  })

  it('vies qui se CHEVAUCHENT : aucun trou inventé sous une vie couvrante', () => {
    // [0,400] couvre [100,200] : la troisième vie démarre à 300, donc toujours sous la
    // couverture de la première. Comparer chaque vie à la seule précédente compterait ici
    // un trou de 100 images pendant lesquelles le joueur était vivant.
    const chevauche = [life(512, 'A', 0, 400), life(513, 'A', 100, 200), life(514, 'A', 300, 350)]
    expect(deadFrames(chevauche, 'A')).toBe(0)
  })

  it('borné à la fenêtre du match : une vie qui déborde `frameCount` est ramenée dedans', () => {
    // Le film publie 200 images ; la seconde vie court jusqu'à 260. Le trou reste [100,180].
    expect(deadFrames([life(512, 'A', 0, 100), life(513, 'A', 180, 260)], 'A', 200)).toBe(80)
    // Et un départ hors fenêtre est ramené à la dernière image : le trou se ferme à 199.
    expect(deadFrames([life(512, 'A', 0, 100), life(513, 'A', 500, 600)], 'A', 200)).toBe(99)
  })

  it('chaque joueur a son propre cumul, et les vies anonymes n’entrent chez personne', () => {
    const doc = testReplayDoc({
      frameCount: 1000,
      frameIntervalMs: 1000,
      tracks: [
        life(512, 'A', 0, 100),
        life(513, 'A', 200, 300),
        life(514, 'B', 0, 500),
        { ...life(515, 'A', 600, 700), xuid: undefined },
      ],
    })
    const dead = deadTimeByPlayer(buildPlayers(doc, []), doc)
    // A : un seul trou [100,200] — la trace SANS xuid (caméra, spectateur) n'ouvre pas de
    // second trou chez lui, elle n'appartient à personne.
    expect(dead.get('A')).toBe(100_000)
    expect(dead.get('B')).toBe(0)
  })

  it('sans échelle temporelle, la cadence de repli de la page s’applique (60 images / s)', () => {
    // Artefact ancien sans `frameIntervalMs` : l'axe T est un index, et `frameToMs` retombe
    // sur 60 images par seconde — la même règle que le chronomètre du rejeu.
    const doc = testReplayDoc({
      frameCount: 1000,
      tracks: [life(512, 'A', 0, 100), life(513, 'A', 160, 300)],
    })
    expect(deadTimeByPlayer(buildPlayers(doc, []), doc).get('A')).toBe(1000)
  })
})

describe('formatDeadTime — mm:ss, minutes complétées', () => {
  it('un match sans mort s’écrit 00:00', () => {
    expect(formatDeadTime(0)).toBe('00:00')
  })

  it('complète les minutes ET les secondes pour que la colonne s’aligne', () => {
    expect(formatDeadTime(9_000)).toBe('00:09')
    expect(formatDeadTime(64_000)).toBe('01:04')
    expect(formatDeadTime(544_000)).toBe('09:04')
    expect(formatDeadTime(727_000)).toBe('12:07')
  })

  it('tronque à la seconde écoulée et ne rend jamais de durée négative', () => {
    expect(formatDeadTime(1_999)).toBe('00:01')
    expect(formatDeadTime(-5_000)).toBe('00:00')
  })

  it('au-delà de l’heure, les minutes débordent plutôt que de mentir', () => {
    expect(formatDeadTime(3_723_000)).toBe('62:03')
  })
})
