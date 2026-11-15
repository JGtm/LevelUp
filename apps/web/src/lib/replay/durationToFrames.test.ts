/**
 * Conversion d'une DURÉE en images quand le document ne publie pas son pas (`frameIntervalMs` absent, artefacts
 * d'avant l'échelle temporelle) : elle se lit au pas HISTORIQUE de 100 ms, ce qui rend exactement les nombres
 * d'images que portaient les seuils avant leur conversion en millisecondes. L'horloge (`frameToMs`, `msToFrames`,
 * `framesPerSecond`) garde sa cadence de repli : seul le passage durée -> images change.
 */
import { describe, expect, it } from 'vitest'

import {
  durationFrameMs,
  durationToFrames,
  frameToMs,
  framesPerSecond,
  HISTORICAL_FRAME_INTERVAL_MS,
  msToFrames,
} from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

const sansPas = {} as unknown as ReplayDocumentReady
const auPas = (pas: number) => ({ frameIntervalMs: pas }) as unknown as ReplayDocumentReady

describe('pas historique pour la conversion des durées', () => {
  it('le pas historique est de 100 ms, une seule constante', () => {
    expect(HISTORICAL_FRAME_INTERVAL_MS).toBe(100)
  })

  it('sans pas, une durée se convertit au pas historique : les anciens nombres d images', () => {
    // Durées des seuils convertis par l'étape B, et le nombre d'images qu'elles valaient.
    const anciens: Array<[number, number]> = [
      [1200, 12], // déflagration
      [600, 6], // onde de capture
      [700, 7], // traînée des projectiles
      [1000, 10], // interpolation de cap, maintien de visée, clignotement
      [2600, 26], // pulsation
      [500, 5], // ramassage d'arme
      [6000, 60], // retombée
    ]
    for (const [ms, images] of anciens) expect(durationToFrames(ms, sansPas)).toBe(images)
    expect(durationFrameMs(sansPas)).toBe(100)
  })

  it('avec un pas, la durée suit le pas du document', () => {
    expect(durationToFrames(700, auPas(100))).toBe(7)
    expect(durationToFrames(700, auPas(33))).toBeCloseTo(700 / 33, 9)
    expect(durationFrameMs(auPas(33))).toBe(33)
  })

  it('l horloge ne change pas : le repli de lecture reste 60 images par seconde', () => {
    expect(framesPerSecond(sansPas)).toBe(60)
    expect(frameToMs(60, sansPas)).toBe(1000)
    expect(msToFrames(1000, sansPas)).toBe(60)
  })
})
