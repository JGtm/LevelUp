/**
 * replayExportPlan.test.ts — l'échantillonnage de l'export, sans navigateur ni encodeur.
 *
 * Ce qui se joue ici est la DURÉE du clip et le nombre d'images qu'il faudra peindre : deux
 * nombres qu'on ne peut pas vérifier à l'œil sur un fichier vidéo, et qui décident pourtant de
 * la barre de progression, de la mémoire consommée et de la fin du clip.
 */
import { describe, expect, it } from 'vitest'

import {
  buildExportPlan,
  clampExportBounds,
  defaultExportBounds,
  exportProgressPct,
} from './replayExportPlan'
import { testReplayDoc } from './test/testDoc'

/** Un film à 20 images par seconde : 50 ms par image, 200 images, soit 10 s de match. */
const DOC = testReplayDoc({ frameIntervalMs: 50, frameCount: 200 })

describe('defaultExportBounds — la plage proposée', () => {
  it('est la fenêtre de GAMEPLAY quand elle est connue', () => {
    const w = { startFrame: 20, endFrame: 180, startMs: 1000, endMs: 9000 }
    expect(defaultExportBounds(DOC, w)).toEqual({ startFrame: 20, endFrame: 180 })
  })

  it('retombe sur le film entier sans cadrage', () => {
    // Moins juste (le countdown y est), mais c'est tout ce que le document permet d'affirmer.
    expect(defaultExportBounds(DOC, null)).toEqual({ startFrame: 0, endFrame: 199 })
  })
})

describe('clampExportBounds — deux champs remplis par un humain', () => {
  const domaine = { startFrame: 20, endFrame: 180 }

  it('laisse une plage valide intacte', () => {
    expect(clampExportBounds({ startFrame: 50, endFrame: 100 }, domaine)).toEqual({
      startFrame: 50,
      endFrame: 100,
    })
  })

  it('remet dans l’ordre des bornes croisées', () => {
    expect(clampExportBounds({ startFrame: 120, endFrame: 60 }, domaine)).toEqual({
      startFrame: 60,
      endFrame: 120,
    })
  })

  it('ramène dans le match ce qui en sort', () => {
    expect(clampExportBounds({ startFrame: -500, endFrame: 9999 }, domaine)).toEqual({
      startFrame: 20,
      endFrame: 180,
    })
  })
})

describe('buildExportPlan — la suite des images', () => {
  it('échantillonne à la cadence du FICHIER, pas à celle du film', () => {
    // 2 s de match (frames 0 à 40 d'un film à 20 im/s) exportées à 30 im/s : 60 pas + la borne.
    const plan = buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30)
    expect(plan.durationMs).toBe(2000)
    expect(plan.frames).toHaveLength(61)
    expect(plan.fps).toBe(30)
  })

  it('avance d’un pas FRACTIONNAIRE quand les deux cadences ne tombent pas juste', () => {
    const plan = buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30)
    // 1/30 s = 33,33 ms = 0,667 image d'un film à 50 ms. Arrondir saccaderait le mouvement.
    expect(plan.frames[1]).toBeCloseTo(0.6667, 3)
    expect(plan.frames[3]).toBeCloseTo(2, 3)
  })

  it('finit EXACTEMENT sur la borne de fin', () => {
    // L'écran de fin de match ne se peint QU'À cette borne : la manquer, c'est livrer un clip
    // de fin de match sans son écran de fin.
    const plan = buildExportPlan({ startFrame: 10, endFrame: 47 }, DOC, 30)
    expect(plan.frames[plan.frames.length - 1]).toBe(47)
  })

  it('part de la borne de début', () => {
    expect(buildExportPlan({ startFrame: 10, endFrame: 47 }, DOC, 30).frames[0]).toBe(10)
  })

  it('rend UNE image sur une plage vide', () => {
    // Un arrêt sur image reste un résultat ; un fichier vide se téléchargerait quand même.
    const plan = buildExportPlan({ startFrame: 30, endFrame: 30 }, DOC, 30)
    expect(plan.frames).toEqual([30])
    expect(plan.durationMs).toBe(0)
  })

  it('un document SANS échelle de temps garde l’axe de repli, et s’exporte quand même', () => {
    // Sans `frameIntervalMs`, `frameToMs` retombe sur la cadence historique (60 im/s) : l'axe
    // n'est plus le temps réel du match, mais il reste un axe — l'export n'a pas à le refuser.
    const sansEchelle = testReplayDoc({ frameCount: 200 })
    const plan = buildExportPlan({ startFrame: 0, endFrame: 199 }, sansEchelle, 30)
    expect(plan.frames[0]).toBe(0)
    // Un pas de 1/30 s vaut 2 images d'un axe à 60 im/s.
    expect(plan.frames[1]).toBeCloseTo(2, 6)
    expect(plan.frames[plan.frames.length - 1]).toBe(199)
  })

  it('la durée annoncée est celle de la PLAGE, jamais celle du calcul', () => {
    const plan = buildExportPlan({ startFrame: 20, endFrame: 180 }, DOC, 30)
    expect(plan.durationMs).toBe(8000)
  })
})

describe('exportProgressPct — la barre ne ment pas', () => {
  it('rend la part parcourue', () => {
    expect(exportProgressPct(25, 100)).toBe(25)
  })

  it('ne dépasse jamais 100 %, ni ne descend sous 0', () => {
    expect(exportProgressPct(200, 100)).toBe(100)
    expect(exportProgressPct(-5, 100)).toBe(0)
  })

  it('rend 0 plutôt qu’une division par zéro', () => {
    expect(exportProgressPct(0, 0)).toBe(0)
  })
})

describe('buildExportPlan — le MAINTIEN de la dernière image', () => {
  it('n’ajoute rien quand on ne le demande pas', () => {
    const sans = buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30)
    expect(sans.holdMs).toBe(0)
    expect(sans.frames[sans.frames.length - 1]).toBe(40)
  })

  it('répète la dernière image le temps demandé', () => {
    // Sans ce maintien, l'écran de fin de match — qui ne se peint QU'À la borne — durait une
    // seule image, soit 1/30 s : un éclair imperceptible.
    const sans = buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30)
    const avec = buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30, 3000)
    expect(avec.frames).toHaveLength(sans.frames.length + 90)
    expect(avec.holdMs).toBe(3000)
  })

  it('les images de maintien sont TOUTES la borne de fin', () => {
    const avec = buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30, 1000)
    expect(avec.frames.slice(-30)).toEqual(Array.from({ length: 30 }, () => 40))
  })

  it('la durée annoncée reste celle de la PLAGE, maintien non compris', () => {
    // Le dialogue annonce « Plage exportée » : le maintien est du rab, pas de la plage.
    expect(buildExportPlan({ startFrame: 0, endFrame: 40 }, DOC, 30, 3000).durationMs).toBe(2000)
  })
})
