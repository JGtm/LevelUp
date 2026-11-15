/**
 * aliveBalanceLogic.test.ts — la supériorité numérique : joueurs en vie de l'équipe du joueur
 * moins ceux d'en face, bots compris.
 *
 * L'horloge des cas : 100 ms par image, 101 images (10 s de film), origine et coup d'envoi à
 * zéro — l'axe du gameplay vaut `image x 100`, calculé à la main. Un cas porte une horloge
 * décalée (origine à 5 s, coup d'envoi à 2 s) pour que la conversion ms -> image soit éprouvée.
 */
import { describe, expect, it } from 'vitest'

import {
  ALIVE_BALANCE_STEP_MS,
  aliveBalanceAt,
  buildAliveBalance,
  type AliveBalanceInput,
} from './aliveBalanceLogic'
import { matchClock } from './matchClock'
import type { ReplayTrackReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'

/** Une vie : fenêtre `[start, end]` en images (bornes comprises, cf. `isAliveAt`). */
function vie(start: number, end: number, slot = 0): ReplayTrackReady {
  return { slot, startFrame: start, endFrame: end, points: [] } as unknown as ReplayTrackReady
}

function joueur(
  xuid: string,
  team: number | undefined,
  lives: ReplayTrackReady[],
  bot = false,
): ReplayPlayer {
  return { xuid, team, lives, bot: bot || undefined }
}

function horloge(frameCount = 101, originMs = 0, t0Ms = 0) {
  return matchClock({ originMs, frameIntervalMs: 100, frameCount }, { t0_ms: t0Ms })
}

function input(over: Partial<AliveBalanceInput> = {}): AliveBalanceInput {
  return {
    players: [
      joueur('a1', 0, [vie(0, 100)]),
      joueur('a2', 0, [vie(0, 50), vie(70, 100)]),
      joueur('e1', 1, [vie(0, 100)]),
      joueur('e2', 1, [vie(0, 30)]),
    ],
    allyTeam: 0,
    clock: horloge(),
    ...over,
  }
}

describe('buildAliveBalance — le calcul', () => {
  it('compte, à chaque instant, les joueurs en vie de chaque camp et leur écart', () => {
    const b = buildAliveBalance(input())!
    const at = (ms: number) => b.points.find((p) => p.ms === ms)!
    // 0 s : tout le monde vit.
    expect(at(0)).toMatchObject({ ally: 2, opponent: 2 })
    // 4 s (image 40) : e2 est mort depuis l'image 30, a2 vit encore (fin 50).
    expect(at(4000)).toMatchObject({ ally: 2, opponent: 1 })
    // 6 s (image 60) : a2 est mort (fin 50), pas encore réapparu (début 70).
    expect(at(6000)).toMatchObject({ ally: 1, opponent: 1 })
    // 8 s (image 80) : a2 est revenu.
    expect(at(8000)).toMatchObject({ ally: 2, opponent: 1 })
  })

  it('échantillonne toutes les secondes par défaut, de 0 à la fin du match incluse', () => {
    expect(ALIVE_BALANCE_STEP_MS).toBe(1000)
    const b = buildAliveBalance(input())!
    expect(b.points.map((p) => p.ms)).toEqual([
      0, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000,
    ])
    expect(b.endMs).toBe(10_000)
  })

  it('le pas est un paramètre en millisecondes, indépendant du pas du document', () => {
    const b = buildAliveBalance(input({ stepMs: 2500 }))!
    // 0, 2,5 s, 5 s, 7,5 s, puis la fin (10 s) — jamais le pas des images (100 ms).
    expect(b.points.map((p) => p.ms)).toEqual([0, 2500, 5000, 7500, 10_000])
  })

  it('lit les images sur l’horloge du GAMEPLAY : origine du film et coup d’envoi décalés', () => {
    // Le film commence 3 s avant le coup d'envoi (origine 5 s, coup d'envoi 8 s) : l'instant
    // de gameplay 0 est l'image 30.
    const b = buildAliveBalance(
      input({
        clock: horloge(101, 5_000, 8_000),
        players: [
          joueur('a1', 0, [vie(30, 100)]),
          joueur('e1', 1, [vie(0, 29), vie(50, 100)]),
        ],
      }),
    )!
    const at = (ms: number) => b.points.find((p) => p.ms === ms)!
    // 0 s de gameplay = image 30 : e1 vient de mourir (fin 29), pas encore revenu.
    expect(at(0)).toMatchObject({ ally: 1, opponent: 0 })
    // 2 s de gameplay = image 50 : e1 est revenu.
    expect(at(2000)).toMatchObject({ ally: 1, opponent: 1 })
  })

  it('COMPTE LES BOTS : un bot en vie est un joueur en vie (règle « les bots comptent partout »)', () => {
    const b = buildAliveBalance(
      input({
        players: [
          joueur('a1', 0, [vie(0, 100)]),
          joueur('bot:Zed', 0, [vie(0, 100)], true),
          joueur('e1', 1, [vie(0, 100)]),
        ],
      }),
    )!
    expect(b.points[0]).toMatchObject({ ally: 2, opponent: 1 })
  })

  it('un joueur à plusieurs vies compte UNE fois à un instant donné', () => {
    // Deux fenêtres qui se recouvrent d'une image : le joueur n'est pas « deux joueurs ».
    const b = buildAliveBalance(
      input({
        players: [joueur('a1', 0, [vie(0, 50), vie(50, 100)]), joueur('e1', 1, [vie(0, 100)])],
      }),
    )!
    expect(b.points.find((p) => p.ms === 5000)).toMatchObject({ ally: 1, opponent: 1 })
  })

  it('une vie sans équipe au film n’entre dans aucun camp : rien n’est deviné', () => {
    const b = buildAliveBalance(
      input({
        players: [
          joueur('a1', 0, [vie(0, 100)]),
          joueur('inconnu', undefined, [vie(0, 100)]),
          joueur('e1', 1, [vie(0, 100)]),
        ],
      }),
    )!
    expect(b.points[0]).toMatchObject({ ally: 1, opponent: 1 })
  })

  it('plus de deux camps : l’écart se prend contre la MEILLEURE équipe adverse, et le dit', () => {
    const b = buildAliveBalance(
      input({
        players: [
          joueur('a1', 0, [vie(0, 100)]),
          joueur('b1', 1, [vie(0, 100)]),
          joueur('b2', 1, [vie(0, 100)]),
          joueur('c1', 2, [vie(0, 100)]),
        ],
      }),
    )!
    expect(b.multiTeam).toBe(true)
    expect(b.points[0]).toMatchObject({ ally: 1, opponent: 2 })
  })

  it('deux camps : multiTeam vaut false', () => {
    expect(buildAliveBalance(input())!.multiTeam).toBe(false)
  })
})

describe('buildAliveBalance — quand il n’y a rien à dire', () => {
  it('rend null sans horloge : sans origine, l’instant d’une image est un inconnu', () => {
    expect(buildAliveBalance(input({ clock: null }))).toBeNull()
    expect(buildAliveBalance(input({ clock: undefined }))).toBeNull()
  })

  it('rend null sans camp du joueur : la référence est absente du film ou sans équipe', () => {
    expect(buildAliveBalance(input({ allyTeam: null }))).toBeNull()
  })

  it('rend null sans équipe d’en face', () => {
    expect(
      buildAliveBalance(input({ players: [joueur('a1', 0, [vie(0, 100)])] })),
    ).toBeNull()
  })

  it('rend null quand un des deux côtés n’a AUCUNE vie : le film ne dit rien, pas « 0 en vie »', () => {
    expect(
      buildAliveBalance(
        input({ players: [joueur('a1', 0, [vie(0, 100)]), joueur('e1', 1, [])] }),
      ),
    ).toBeNull()
  })

  it('rend null quand le film s’arrête avant le coup d’envoi', () => {
    expect(buildAliveBalance(input({ clock: horloge(11, 0, 5_000) }))).toBeNull()
  })

  it('rend null avec un pas non positif', () => {
    expect(buildAliveBalance(input({ stepMs: 0 }))).toBeNull()
  })
})

describe('aliveBalanceAt — le point tenu à un instant', () => {
  const b = buildAliveBalance(input())!

  it('rend le dernier point au plus tard à l’instant lu : l’effectif ATTEND entre deux relevés', () => {
    expect(aliveBalanceAt(b, 4_999)).toMatchObject({ ms: 4000, ally: 2, opponent: 1 })
    expect(aliveBalanceAt(b, 6_000)).toMatchObject({ ms: 6000, ally: 1, opponent: 1 })
  })

  it('borne aux extrémités : avant 0, le premier point ; après la fin, le dernier', () => {
    expect(aliveBalanceAt(b, -300).ms).toBe(0)
    expect(aliveBalanceAt(b, 99_999).ms).toBe(10_000)
  })
})
