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
function doc(over: { originMs?: number; frameCount: number; t0FilmMs?: number }) {
  return testReplayDoc({ frameIntervalMs: 100, ...over })
}

/** L'en-tête réduit à ce que la fenêtre lui demande. */
function header(t0Ms: number | undefined, playableSeconds: number | undefined): ReplayWindowHeader {
  return { t0_ms: t0Ms, playable_duration_seconds: playableSeconds }
}

describe('replayWindow — les quatre témoins mesurés', () => {
  it('000d5950 (cas nominal) : le film démarre AVANT le coup d’envoi', () => {
    const w = replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), header(18_465, 478))
    // 18 465 − 3 604 = 14 861 ms => 148,61 images ; le préambule d'une seconde, 10 images plus tôt.
    expect(w).toEqual({ startFrame: 149, leadInFrame: 139, endFrame: 4_929, startMs: 14_861, endMs: 492_861 })
  })

  it('e94163af : le premier paquet arrive APRÈS le T0 — le début se clampe à l’image zéro', () => {
    const w = replayWindow(doc({ originMs: 39_772, frameCount: 1_813 }), header(35_238, 180))
    expect(w).toEqual({ startFrame: 0, leadInFrame: 0, endFrame: 1_755, startMs: 0, endMs: 175_466 })
  })

  it('606d9844 : sans T0, la fin reste cadrée (l’ancrage se compense) et le début vaut zéro', () => {
    const w = replayWindow(doc({ originMs: 6_890, frameCount: 2_342 }), header(undefined, 235))
    expect(w).toEqual({ startFrame: 0, leadInFrame: 0, endFrame: 2_281, startMs: 0, endMs: 228_110 })
  })

  it('64e8adfa (fini au temps) : la fin déclarée, pas le dernier point de score', () => {
    const w = replayWindow(doc({ originMs: 10_516, frameCount: 8_337 }), header(29_069, 810))
    expect(w).toEqual({ startFrame: 186, leadInFrame: 176, endFrame: 8_286, startMs: 18_553, endMs: 828_553 })
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
    expect(w).toEqual({ startFrame: 0, leadInFrame: 0, endFrame: 99, startMs: 0, endMs: 9_900 })
  })
})

/**
 * LE T0 MESURÉ DANS LE FILM PASSE DEVANT CELUI DE L'API (D5 du plan T0-film, 2026-09-02).
 *
 * CE QUE CES CAS PROTÈGENT, et c'est exactement le défaut que le lot corrige : `header.t0_ms`
 * est ESTIMÉ des `first_joined_time`, dégénéré (~0 ms) sur 10 à 15 % des matchs du corpus. Un
 * rejeu ainsi cadré démarre sur le countdown d'avant-match, joueurs figés. Le film, lui, DATE
 * le premier mouvement. La substitution ne se voit à l'exécution qu'à quelques images près —
 * seules des bornes chiffrées disent laquelle des deux sources a servi.
 *
 * LA BORNE DE FIN NE SUIT PAS, et c'est la moitié la moins intuitive de la décision : elle
 * s'appuie sur la compensation `playable_duration_seconds = duration − t0_ms/1000`, faite par
 * le serveur AVEC le t0 de l'API. Le cas « dégénéré » ci-dessous le tient au chiffre près.
 */
describe('replayWindow — le T0 du film prime sur celui de l’API (D5)', () => {
  it('le champ présent décide du début, l’en-tête ne décide plus que de la fin', () => {
    // T0 API dégénéré (0 ms) alors que le coup d'envoi réel tombe à 22 700 ms : sans le film,
    // le rejeu démarrerait 227 images trop tôt, en plein countdown.
    const w = replayWindow(doc({ originMs: 0, frameCount: 3_000, t0FilmMs: 22_700 }), header(0, 200))
    expect(w?.startMs).toBe(22_700)
    expect(w?.startFrame).toBe(227)
    expect(w?.leadInFrame).toBe(217)
    // LA FIN RESTE CELLE DE L'API : 0 + 200 000 − 0. La déplacer de 22,7 s amputerait le match.
    expect(w?.endMs).toBe(200_000)
    expect(w?.endFrame).toBe(2_000)
  })

  it('le champ absent : l’en-tête reprend le début, exactement comme avant ce lot', () => {
    const w = replayWindow(doc({ originMs: 3_604, frameCount: 4_985 }), header(18_465, 478))
    expect(w?.startMs).toBe(14_861)
    expect(w?.startFrame).toBe(149)
  })

  it('les DEUX absents : le début vaut zéro, sans jamais inventer un coup d’envoi', () => {
    const w = replayWindow(doc({ originMs: 6_890, frameCount: 2_342 }), header(undefined, 235))
    expect(w?.startMs).toBe(0)
    expect(w?.startFrame).toBe(0)
    expect(w?.leadInFrame).toBe(0)
  })

  it('un T0 film ANTÉRIEUR à celui de l’API est retenu quand même — c’est lui la mesure', () => {
    const w = replayWindow(doc({ originMs: 0, frameCount: 3_000, t0FilmMs: 12_000 }), header(30_000, 200))
    expect(w?.startMs).toBe(12_000)
    expect(w?.startFrame).toBe(120)
  })
})

/**
 * LE PRÉAMBULE DE LECTURE (`leadInFrame`, D3 user 2026-09-02) — une seconde avant le coup
 * d'envoi, et RIEN d'autre ne bouge. Il ne vit que dans `useReplayPlayback` ; ces cas tiennent
 * sa valeur et son bornage à l'image zéro.
 */
describe('replayWindow — le préambule d’une seconde (D3)', () => {
  it('vaut le coup d’envoi moins une seconde, converti à la cadence du document', () => {
    // 100 ms par image : une seconde = 10 images.
    const w = replayWindow(doc({ originMs: 0, frameCount: 3_000, t0FilmMs: 22_700 }), header(0, 200))
    expect((w?.startFrame ?? 0) - (w?.leadInFrame ?? 0)).toBe(10)
  })

  it('se CLAMPE à l’image zéro quand le coup d’envoi tombe avant la première seconde de film', () => {
    const w = replayWindow(doc({ originMs: 0, frameCount: 3_000, t0FilmMs: 400 }), header(0, 200))
    expect(w?.startFrame).toBe(4)
    expect(w?.leadInFrame).toBe(0)
  })

  it('un début DÉJÀ clampé à zéro n’a pas de préambule à prendre', () => {
    const w = replayWindow(doc({ originMs: 39_772, frameCount: 1_813 }), header(35_238, 180))
    expect(w?.startFrame).toBe(0)
    expect(w?.leadInFrame).toBe(0)
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
