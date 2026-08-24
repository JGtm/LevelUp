/**
 * Tests — deadTimeLogic (le temps mort cumulé d'un joueur).
 *
 * CE QU'ILS PROTÈGENT, EN DEUX FAMILLES.
 *
 * 1. LA DÉFINITION ET SES DEUX BORNES : le temps AVANT la première vie et le temps APRÈS la
 *    dernière ne sont pas du temps mort, et ce sont exactement les deux endroits où un cumul
 *    naïf gonfle. Autour : vies désordonnées, vies qui se chevauchent, débordement de la
 *    fenêtre du match.
 * 2. LE REFUS DE MESURER. Un trou entre deux vies NOMMÉES n'est pas une mort si une vie que le
 *    pont n'a rattachée à personne y vit : le film montre quelqu'un qui court. Ces tests-là
 *    tiennent le `null` — et, tout aussi important, tiennent qu'on ne refuse PAS quand rien ne
 *    le justifie (vie anonyme hors trou, simple contact de bornes) : une ligne muette partout
 *    serait un autre défaut.
 *
 * Les vies entrent par `buildPlayers`, comme à l'écran — sauf le test du tri défensif, qui doit
 * justement contourner ce tri pour éprouver celui du module.
 */
import { describe, expect, it } from 'vitest'

import { deadTimeByPlayer, formatDeadTime } from './deadTimeLogic'
import type { ReplayTrackReady } from './replayNormalize'
import { buildPlayers, type ReplayPlayer } from './rosterLogic'
import { testReplayDoc } from './test/testDoc'

/** Une vie du film : un slot, un propriétaire, une fenêtre [start, end]. */
function life(slot: number, xuid: string, start: number, end: number): ReplayTrackReady {
  return { slot, team: -1, xuid, startFrame: start, endFrame: end, points: [{ t: start, x: 0, y: 0 }] }
}

/** Une vie que le pont slot -> xuid n'a rattachée à PERSONNE — le cas qui invalide un trou. */
function anonymous(slot: number, start: number, end: number): ReplayTrackReady {
  return { slot, team: -1, startFrame: start, endFrame: end, points: [{ t: start, x: 0, y: 0 }] }
}

/**
 * Temps mort du joueur `xuid`, en images — la grandeur que les cas ci-dessous raisonnent, ou
 * `null` quand le module refuse de mesurer. `frameIntervalMs: 1000` fait qu'une image vaut une
 * seconde : les millisecondes rendues se relisent alors directement en images.
 *
 * Joueur absent de la table = -1, une valeur qui ne satisfait aucune attente — surtout pas
 * confondue avec `null`, qui est ici un RÉSULTAT attendu et non une absence.
 */
function deadFrames(
  tracks: ReplayTrackReady[],
  xuid: string,
  frameCount = 1000,
): number | null {
  const doc = testReplayDoc({ frameCount, frameIntervalMs: 1000, tracks })
  const dead = deadTimeByPlayer(buildPlayers(doc, []), doc)
  if (!dead.has(xuid)) return -1
  const ms = dead.get(xuid) ?? null
  return ms === null ? null : ms / 1000
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

  /**
   * LE TRI DÉFENSIF, ÉPROUVÉ POUR DE BON. Le test ci-dessus passe par `buildPlayers`, qui trie
   * déjà : il ne verrait pas la disparition du `.sort` du module. Celui-ci construit le
   * `ReplayPlayer` À LA MAIN, vies en désordre, et c'est le seul de ce fichier à le faire —
   * précisément parce que la structure qu'il fabrique n'existe qu'ici. Sans le tri interne, la
   * couverture partirait de la vie la plus tardive et le total tomberait à 0.
   */
  it('tri INTERNE : des vies désordonnées reçues telles quelles donnent le même total', () => {
    const player: ReplayPlayer = {
      xuid: 'A',
      lives: [life(514, 'A', 360, 500), life(512, 'A', 0, 100), life(513, 'A', 180, 300)],
    }
    const doc = testReplayDoc({ frameCount: 1000, frameIntervalMs: 1000, tracks: [] })
    expect(deadTimeByPlayer([player], doc).get('A')).toBe(140_000)
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

/**
 * LE REFUS DE MESURER — la correction du 2026-08-24 (revue adversariale, constats 1 et 2).
 *
 * Le pont slot -> xuid du film est incomplet sur une partie du corpus. Un trou entre deux vies
 * NOMMÉES peut donc être occupé par une vie réelle que personne ne réclame : mesuré sur les
 * artefacts servis, `64e8adfa` affichait 5:11 de « temps mort » pour flamesamurai dont ~63 %
 * couverts par la trace anonyme du slot 607. On ne rafistole pas ce chiffre, on le refuse.
 */
describe('deadTimeByPlayer — quand le film ne permet pas de conclure', () => {
  it('vie ANONYME DANS un trou : mesure refusée (null), jamais un chiffre rafistolé', () => {
    const tracks = [life(512, 'A', 0, 100), anonymous(900, 120, 160), life(513, 'A', 180, 300)]
    expect(deadFrames(tracks, 'A')).toBeNull()
  })

  it('un seul trou pollué suffit à refuser TOUT le cumul du joueur', () => {
    // Deux trous, un seul occupé : soustraire le douteux rendrait le reste invérifiable.
    const tracks = [
      life(512, 'A', 0, 100),
      life(513, 'A', 180, 300),
      anonymous(900, 320, 340),
      life(514, 'A', 360, 500),
    ]
    expect(deadFrames(tracks, 'A')).toBeNull()
  })

  it('joueur du roster SANS AUCUNE VIE : refus aussi — « 00:00 » dirait « jamais à terre »', () => {
    const doc = testReplayDoc({
      frameCount: 1000,
      frameIntervalMs: 1000,
      roster: [{ xuid: 'C', filmIndex: 0, name: 'Charlie' }],
      tracks: [life(512, 'A', 0, 400)],
    })
    const dead = deadTimeByPlayer(buildPlayers(doc, []), doc)
    expect(dead.has('C')).toBe(true)
    expect(dead.get('C')).toBeNull()
    // Et le joueur qui, lui, a des vies garde sa mesure : le refus n'est pas contagieux.
    expect(dead.get('A')).toBe(0)
  })

  it('le refus est PAR JOUEUR : une vie anonyme hors des trous de B ne l’atteint pas', () => {
    const doc = testReplayDoc({
      frameCount: 1000,
      frameIntervalMs: 1000,
      tracks: [
        life(512, 'A', 0, 100),
        anonymous(900, 120, 160),
        life(513, 'A', 180, 300),
        life(514, 'B', 500, 600),
        life(515, 'B', 700, 800),
      ],
    })
    const dead = deadTimeByPlayer(buildPlayers(doc, []), doc)
    expect(dead.get('A')).toBeNull()
    expect(dead.get('B')).toBe(100_000)
  })

  it('vie anonyme APRÈS la dernière vie nommée : rien à invalider, la mesure tient', () => {
    // La queue n'est pas comptée : une trace anonyme qui y vit ne prouve rien contre le cumul.
    const tracks = [life(512, 'A', 0, 100), life(513, 'A', 180, 300), anonymous(900, 600, 700)]
    expect(deadFrames(tracks, 'A')).toBe(80)
  })

  it('vie anonyme AVANT la première vie nommée : de même, aucun trou concerné', () => {
    const tracks = [anonymous(900, 0, 50), life(512, 'A', 100, 200), life(513, 'A', 280, 400)]
    expect(deadFrames(tracks, 'A')).toBe(80)
  })

  it('simple CONTACT de bornes (intersection nulle) : on ne refuse pas pour rien', () => {
    // Le trou est ]100, 180[. Une trace qui finit à 100 ou démarre à 180 ne dit rien de son
    // intérieur — refuser ici rendrait la ligne muette sur presque tous les matchs.
    const avant = [life(512, 'A', 0, 100), anonymous(900, 50, 100), life(513, 'A', 180, 300)]
    expect(deadFrames(avant, 'A')).toBe(80)
    const apres = [life(512, 'A', 0, 100), anonymous(901, 180, 220), life(513, 'A', 180, 300)]
    expect(deadFrames(apres, 'A')).toBe(80)
  })

  it('une vie anonyme qui ENJAMBE le trou le pollue quand même', () => {
    // Elle commence avant la mort et finit après le retour : l'intersection reste positive.
    const tracks = [life(512, 'A', 0, 100), anonymous(900, 90, 190), life(513, 'A', 180, 300)]
    expect(deadFrames(tracks, 'A')).toBeNull()
  })
})

describe('formatDeadTime — mm:ss, minutes complétées', () => {
  it('un match sans mort s’écrit 00:00', () => {
    expect(formatDeadTime(0)).toBe('00:00')
  })

  it('une mesure REFUSÉE s’écrit d’un tiret, jamais 00:00', () => {
    expect(formatDeadTime(null)).toBe('—')
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
