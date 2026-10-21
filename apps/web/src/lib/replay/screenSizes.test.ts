/**
 * screenSizes.test.ts — LE PLAFOND DE TAILLE SUIT LA TOILE (contrôles v8, 2026-10-21).
 *
 * Les véhicules ont été agrandis de 20 % (plancher d'échelle) pour la lisibilité, pour TOUTE
 * toile. Sur un petit écran, les plus longs prenaient une place disproportionnée : leur plafond
 * doux est désormais borné par le côté court de la toile — sans rien changer sur grand écran.
 */
import { describe, expect, it } from 'vitest'

import {
  PION_SCREEN_PX,
  VEHICLE_CEIL_CANVAS_RATIO,
  VEHICLE_CEIL_MIN_PX,
  VEHICLE_SOFT_CEIL_PX,
  screenLengthPx,
  vehicleCeilPx,
} from './screenSizes'

/** Une grande carte (échelle sous le plancher) : c'est le plancher qui sert, comme sur Flood Gulch. */
const GRANDE_CARTE_PX_PAR_M = 3.5

describe('vehicleCeilPx — le plafond doux suit le côté court de la toile', () => {
  it('SANS EFFET SUR GRAND ÉCRAN : à partir de 514 px de côté court, le plafond est l’historique', () => {
    for (const cote of [514, 600, 800, 1080, 4000]) {
      expect(vehicleCeilPx(cote)).toBeCloseTo(VEHICLE_SOFT_CEIL_PX, 6)
    }
    expect(VEHICLE_CEIL_CANVAS_RATIO * 514).toBeGreaterThanOrEqual(VEHICLE_SOFT_CEIL_PX)
  })

  it('sur petit écran, il vaut 12 % du côté court', () => {
    expect(vehicleCeilPx(360)).toBeCloseTo(0.12 * 360, 9)
    expect(vehicleCeilPx(420)).toBeCloseTo(0.12 * 420, 9)
  })

  it('jamais sous quatre pions, et une toile non mesurée garde le plafond historique', () => {
    expect(vehicleCeilPx(50)).toBe(VEHICLE_CEIL_MIN_PX)
    expect(VEHICLE_CEIL_MIN_PX).toBe(4 * PION_SCREEN_PX)
    for (const inconnue of [0, -10, Number.NaN]) {
      expect(vehicleCeilPx(inconnue)).toBe(VEHICLE_SOFT_CEIL_PX)
    }
  })

  it('croît avec la toile (monotone)', () => {
    let prec = 0
    for (let cote = 100; cote <= 800; cote += 20) {
      const c = vehicleCeilPx(cote)
      expect(c).toBeGreaterThanOrEqual(prec)
      prec = c
    }
  })
})

describe('screenLengthPx avec le plafond de toile', () => {
  const SCORPION_MM = 3880
  const MONGOOSE_MM = 1280

  it('SANS plafond fourni (ou celui d’un grand écran), le rendu est strictement celui d’avant', () => {
    const avant = screenLengthPx(SCORPION_MM, GRANDE_CARTE_PX_PAR_M)
    expect(screenLengthPx(SCORPION_MM, GRANDE_CARTE_PX_PAR_M, vehicleCeilPx(1080))).toBe(avant)
    expect(screenLengthPx(SCORPION_MM, GRANDE_CARTE_PX_PAR_M, VEHICLE_SOFT_CEIL_PX)).toBe(avant)
  })

  it('sur petit écran, un long véhicule est borné — sans jamais dépasser son rendu de grand écran', () => {
    const grand = screenLengthPx(SCORPION_MM, GRANDE_CARTE_PX_PAR_M)
    const petit = screenLengthPx(SCORPION_MM, GRANDE_CARTE_PX_PAR_M, vehicleCeilPx(360))
    expect(petit).toBeLessThan(grand)
    // Borné : le plafond de toile plus la compression en racine carrée de l'excédent.
    expect(petit).toBeLessThanOrEqual(vehicleCeilPx(360) + Math.sqrt(grand - vehicleCeilPx(360)))
  })

  it('un véhicule court sous le plafond ne bouge pas, et l’ordre des tailles est conservé', () => {
    const ceil = vehicleCeilPx(360)
    const mongooseGrand = screenLengthPx(MONGOOSE_MM, GRANDE_CARTE_PX_PAR_M)
    expect(mongooseGrand).toBeLessThan(ceil)
    expect(screenLengthPx(MONGOOSE_MM, GRANDE_CARTE_PX_PAR_M, ceil)).toBe(mongooseGrand)
    expect(screenLengthPx(SCORPION_MM, GRANDE_CARTE_PX_PAR_M, ceil)).toBeGreaterThan(
      screenLengthPx(MONGOOSE_MM, GRANDE_CARTE_PX_PAR_M, ceil),
    )
  })

  it('un plafond plus bas ne rend jamais plus gros', () => {
    for (const mm of [500, 1280, 2220, 3880, 11220, 100000]) {
      expect(screenLengthPx(mm, GRANDE_CARTE_PX_PAR_M, vehicleCeilPx(360))).toBeLessThanOrEqual(
        screenLengthPx(mm, GRANDE_CARTE_PX_PAR_M),
      )
    }
  })
})
