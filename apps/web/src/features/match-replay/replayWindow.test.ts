/**
 * Tests — replayWindow (la fenêtre de gameplay) et displayClockMs (l'horloge affichée).
 *
 * LES QUATRE TÉMOINS SONT DES MESURES, pas des exemples inventés : `originMs`, `frameCount`
 * et l'intervalle viennent des artefacts servis sous `data/cache/replays/halo_infinite/`, et
 * `t0_ms` / `playable_duration_seconds` de l'en-tête de leur Match View (relevé du
 * 2026-08-26). Ils couvrent les quatre situations que la formule doit tenir : le cas nominal,
 * le film qui commence APRÈS le coup d'envoi, le match sans T0, et le match terminé au temps.
 *
 * Ce que ces tests protègent en pratique : un cadrage faux ne casse rien à l'exécution — il
 * ampute silencieusement le début ou la fin d'un rejeu. Seules des bornes chiffrées le voient.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from './test/testDoc'
import { displayClockMs, replayWindow, type ReplayWindowHeader } from './replayWindow'

/** Un document au pas de 100 ms — la cadence de tous les artefacts servis aujourd'hui. */
function doc(over: { originMs?: number; frameCount: number }) {
  return testReplayDoc({ frameIntervalMs: 100, ...over })
}

/** L'en-tête réduit à ce que la fenêtre lui demande. */
function header(t0Ms: number | undefined, playableSeconds: number | undefined): ReplayWindowHeader {
  return { t0_ms: t0Ms, playable_duration_seconds: playableSeconds }
}

describe('replayWindow — les quatre témoins mesurés', () => {
  it('000d5950 (cas nominal) : le film démarre AVANT le coup d’envoi', () => {
    const w = replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), header(18_465, 478))
    // 18 465 − 3 604 = 14 861 ms => 148,61 images
    expect(w).toEqual({ startFrame: 149, endFrame: 4_929, startMs: 14_861, endMs: 492_861 })
  })

  it('e94163af : le premier paquet arrive APRÈS le T0 — le début se clampe à l’image zéro', () => {
    const w = replayWindow(doc({ originMs: 39_772, frameCount: 1_813 }), header(35_238, 180))
    expect(w).toEqual({ startFrame: 0, endFrame: 1_755, startMs: 0, endMs: 175_466 })
  })

  it('606d9844 : sans T0, la fin reste cadrée (l’ancrage se compense) et le début vaut zéro', () => {
    const w = replayWindow(doc({ originMs: 6_890, frameCount: 2_342 }), header(undefined, 235))
    expect(w).toEqual({ startFrame: 0, endFrame: 2_281, startMs: 0, endMs: 228_110 })
  })

  it('64e8adfa (fini au temps) : la fin déclarée, pas le dernier point de score', () => {
    const w = replayWindow(doc({ originMs: 10_516, frameCount: 8_337 }), header(29_069, 810))
    expect(w).toEqual({ startFrame: 186, endFrame: 8_286, startMs: 18_553, endMs: 828_553 })
  })

  it('la queue post-match est bien rognée sur les quatre témoins', () => {
    const temoins: Array<[number, number, number, number]> = [
      [3_604, 4_985, 18_465, 478],
      [39_772, 1_813, 35_238, 180],
      [6_890, 2_342, 0, 235],
      [10_516, 8_337, 29_069, 810],
    ]
    for (const [originMs, frameCount, t0, playable] of temoins) {
      const w = replayWindow(doc({ originMs, frameCount }), header(t0, playable))
      expect(w, `témoin d'origine ${originMs}`).not.toBeNull()
      // Entre 5 et 7 secondes de film restent APRÈS la fin déclarée : c'est la queue mesurée.
      const queueMs = (frameCount - 1 - (w?.endFrame ?? 0)) * 100
      expect(queueMs).toBeGreaterThanOrEqual(5_000)
      expect(queueMs).toBeLessThanOrEqual(7_000)
    }
  })
})

describe('replayWindow — sans donnée, pas de cadrage (D-A3)', () => {
  it('artefact sans origine (schéma < 4) : null', () => {
    expect(replayWindow(doc({ frameCount: 4_985 }), header(18_465, 478))).toBeNull()
  })

  it('en-tête sans durée jouable : null', () => {
    expect(replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), header(18_465, undefined))).toBeNull()
  })

  it('en-tête absent : null', () => {
    expect(replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), null)).toBeNull()
    expect(replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), undefined)).toBeNull()
  })

  it('bornes incohérentes (fin avant début) : null plutôt qu’une fenêtre d’une image', () => {
    // Durée jouable de 2 s alors que le film commence 60 s après le coup d'envoi.
    expect(replayWindow(doc({ originMs: 60_000, frameCount: 900 }), header(0, 2))).toBeNull()
  })

  it('film d’une seule image : null', () => {
    expect(replayWindow(doc({ originMs: 0, frameCount: 1 }), header(0, 300))).toBeNull()
  })

  it('film plus court que la fin déclarée : la fin se borne à la dernière image du film', () => {
    // 300 s annoncées, 100 images de film (10 s) : la fenêtre ne peut pas sortir du film.
    const w = replayWindow(doc({ originMs: 0, frameCount: 100 }), header(0, 300))
    expect(w).toEqual({ startFrame: 0, endFrame: 99, startMs: 0, endMs: 9_900 })
  })
})

describe('displayClockMs — l’horloge affichée se recale sur le gameplay (D-A2)', () => {
  const w = replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), header(18_465, 478))

  it('le coup d’envoi se lit 0:00', () => {
    expect(displayClockMs(14_861, w)).toBe(0)
  })

  it('une minute de jeu plus tard, 60 000 ms', () => {
    expect(displayClockMs(74_861, w)).toBe(60_000)
  })

  it('un instant d’avant le coup d’envoi tombe au plancher, jamais en négatif', () => {
    expect(displayClockMs(0, w)).toBe(0)
    expect(displayClockMs(1_000, w)).toBe(0)
  })

  it('sans fenêtre, l’identité : l’axe du film reste affiché tel quel', () => {
    expect(displayClockMs(14_861, null)).toBe(14_861)
  })
})
