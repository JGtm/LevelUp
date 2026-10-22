/**
 * Tests — padPickupDating : LA FENÊTRE D'UNE OCCUPATION COLLAPSE À L'INSTANT EXACT QUAND LE
 * SERVEUR L'A DATÉE (`padPickups[].t`), ARME COMME POWER-UP.
 *
 * Chaque cas est une clause du lot detonations (item 2, 2026-10-22) :
 *  - une occupation DATÉE voit `tLow`/`tHigh` collapser sur l'instant publié, et porte le
 *    ramasseur ;
 *  - `padStateAt` lit alors PLEIN jusqu'à cet instant, VIDE dès lui — plus d'« incertain » ;
 *  - une occupation SANS ramassage correspondant (pad/tLow/tHigh différents) traverse
 *    INCHANGÉE ;
 *  - une occupation que `padPickups` connaît mais sans `t` (non datée) traverse INCHANGÉE ;
 *  - référence préservée quand rien ne change (aucun socle recréé).
 */
import { describe, expect, it } from 'vitest'

import { applyPadPickupDating, type DatedPadPresence } from './padPickupDating'
import { padOccupancyAt, padStateAt } from './weaponPadTime'
import type { ReplayDocumentReady, ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'

/** Un socle : une occupation [100, 200), comme padPresenceRefine.test.ts. */
function pad(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return {
    x: 0,
    y: 0,
    weapon: 'powerup_overshield',
    spawns: [0],
    presence: [{ t0: 0, tLow: 100, tHigh: 200 }],
    ...over,
  }
}

type PadPickups = ReplayDocumentReady['padPickups']

describe('applyPadPickupDating', () => {
  it('une occupation datée collapse sa fenêtre sur l instant publié et porte le ramasseur', () => {
    const pads = [pad()]
    const padPickups: PadPickups = [{ pad: 0, tLow: 100, tHigh: 200, t: 137, xuid: '2535470853746269' }]
    const out = applyPadPickupDating(pads, padPickups)
    const occ = padOccupancyAt(out[0], 137) as DatedPadPresence | null
    expect(occ?.tLow).toBe(137)
    expect(occ && occ.tHigh > occ.tLow).toBe(true)
    expect(occ?.pickupXuid).toBe('2535470853746269')
    // L'« INCERTAIN » SE RÉDUIT À UNE FRACTION DE FRAME (PAD_BETWEEN_FRAMES = 1/64), invisible
    // en pratique — même garantie que le raffinement par proximité, pour la même raison
    // (`tHigh > tLow` strictement, sinon le sentinel « jamais vidé » reprendrait la main) :
    // plein jusqu'à l'instant daté, vide dès la frame suivante.
    expect(padStateAt(out[0], 136)).toBe('full')
    expect(padStateAt(out[0], 138)).toBe('empty')
  })

  it('une occupation sans ramassage correspondant (pad/fenêtre différents) traverse inchangée', () => {
    const pads = [pad()]
    const padPickups: PadPickups = [{ pad: 1, tLow: 100, tHigh: 200, t: 137, xuid: '111' }]
    const out = applyPadPickupDating(pads, padPickups)
    expect(out).toBe(pads) // référence préservée : rien n a bougé
  })

  it('un padPickup sans `t` (non daté) ne touche rien', () => {
    const pads = [pad()]
    const padPickups: PadPickups = [{ pad: 0, tLow: 100, tHigh: 200, xuid: '111' }]
    const out = applyPadPickupDating(pads, padPickups)
    expect(out).toBe(pads)
  })

  it('aucun padPickups : référence préservée', () => {
    const pads = [pad()]
    expect(applyPadPickupDating(pads, [])).toBe(pads)
  })

  it('un socle de power-up se date EXACTEMENT comme un socle d arme : même mécanisme', () => {
    const pads = [pad({ weapon: '0x0A1992BC' }), pad({ weapon: 'powerup_camo' })]
    const padPickups: PadPickups = [
      { pad: 0, tLow: 100, tHigh: 200, t: 150, xuid: '1' },
      { pad: 1, tLow: 100, tHigh: 200, t: 162, xuid: '2' },
    ]
    const out = applyPadPickupDating(pads, padPickups)
    expect(padOccupancyAt(out[0], 150)?.tLow).toBe(150)
    expect(padOccupancyAt(out[1], 162)?.tLow).toBe(162)
  })
})
