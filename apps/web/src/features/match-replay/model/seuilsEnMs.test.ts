/**
 * seuilsEnMs.test.ts — CHAQUE SEUIL TEMPOREL DU REJEU EST UNE DURÉE, la même à 100 ms et à 33 ms
 * par image.
 *
 * POURQUOI. Un seuil écrit « 12 images » vaut 1,2 s à 100 ms par image et 0,4 s à 33 ms : changer
 * la cadence du rejeu raccourcirait en silence chaque effet, chaque tolérance de son, chaque
 * maintien de visée. Ces tests tiennent, pour chaque site, deux choses : à 100 ms la décision est
 * celle d'avant la conversion (valeur historique en images), et à 33 ms elle porte sur la même
 * durée réelle (à une image près, la grille ne tombant pas juste).
 *
 * Le garde-rail `seuilsEnMs.guard.test.ts` interdit le retour d'une constante `*_FRAMES`.
 */
import { describe, expect, it } from 'vitest'

import { carriedGlyphAlpha, CARRIED_PULSE_MAX, CARRIED_PULSE_MIN, CARRIED_PULSE_PERIOD_MS } from '../layers/carriedGlyphPulse'
import { BLINK_PERIOD_MS, flagBlinkAlpha } from '../layers/flagCarriesLayer'
import { BOMB_BLAST_HOLD_MS, bombBlastHoldFrames, drawBombBlastFx, type BombBlastFx } from '../layers/bombBlastFx'
import { FLAG_CAPTURE_HOLD_MS, drawFlagCaptureFx, flagCaptureHoldFrames, type FlagCaptureFx } from '../layers/flagCaptureFx'
import { PROJECTILE_TAIL_MS, drawProjectilesLayer } from '../layers/replayProjectiles'
import { frameToMs, msToFrames } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayProjectileReady } from '../../../lib/replay/replayNormalize'
import { recordingContext } from '../test/recordingContext'
import { headingTowardNext, HEADING_INTERP_MAX_SPAN_MS } from './headingInterp'
import { VEHICLE_AIM_HOLD_MS, vehicleRideAimReading } from './vehiclesAim'
import { skullSoundEvents, SKULL_SOUND_STEMS } from '../sound/skullSound'
import { zoneSoundEvents, ZONE_SOUND_STEMS } from '../sound/zoneSound'
import { weaponChangeSoundEvents } from '../sound/weaponChangeSound'

/** Les deux grilles : l'historique, et 30 images par seconde. */
const PAS = [100, 33] as const

const docAuPas = (pas: number) => ({ frameIntervalMs: pas }) as unknown as ReplayDocumentReady

/** La plus grande durée d'une fenêtre fermée de `n` images : ne dépasse jamais `ms`. */
const PLUS_GRAND_NOMBRE_ENTIER = (ms: number, pas: number) => Math.floor(ms / pas)

describe('durées des effets : la même durée réelle à 100 et à 33 ms', () => {
  it('à 100 ms, les valeurs historiques en images', () => {
    const d = docAuPas(100)
    expect(bombBlastHoldFrames(d)).toBe(12)
    expect(flagCaptureHoldFrames(d)).toBe(6)
    expect(msToFrames(PROJECTILE_TAIL_MS, d)).toBe(7)
    expect(msToFrames(HEADING_INTERP_MAX_SPAN_MS, d)).toBe(10)
    expect(msToFrames(VEHICLE_AIM_HOLD_MS, d)).toBe(10)
    expect(msToFrames(CARRIED_PULSE_PERIOD_MS, d)).toBe(26)
    expect(msToFrames(BLINK_PERIOD_MS, d)).toBe(10)
  })

  it('un document sans pas redonne les anciens nombres d images (pas historique de 100 ms)', () => {
    const sansPas = {} as unknown as ReplayDocumentReady
    expect(bombBlastHoldFrames(sansPas)).toBe(12)
    expect(flagCaptureHoldFrames(sansPas)).toBe(6)
  })

  it.each(PAS)("la déflagration et l'onde de capture durent la même durée réelle à %i ms", (pas) => {
    const d = docAuPas(pas)
    for (const [hold, ms] of [
      [bombBlastHoldFrames(d), BOMB_BLAST_HOLD_MS],
      [flagCaptureHoldFrames(d), FLAG_CAPTURE_HOLD_MS],
    ] as const) {
      expect(frameToMs(hold, d)).toBeCloseTo(ms, 6)
    }
    // Un événement rare tient deux fois l'onde de capture, sur toute grille.
    expect(BOMB_BLAST_HOLD_MS).toBe(2 * FLAG_CAPTURE_HOLD_MS)
  })

  it.each(PAS)('la déflagration est peinte jusqu à sa durée, pas une image de plus, à %i ms', (pas) => {
    const d = docAuPas(pas)
    const fx: BombBlastFx = { frame: 10, x: 5, y: 5, xuid: 'A' }
    const vue = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 528, height: 528, pad: 24 }
    const peint = (age: number) => {
      const { ctx, ops } = recordingContext()
      drawBombBlastFx(ctx, [fx], vue, { frame: 10 + age, hold: bombBlastHoldFrames(d) }, { inkOf: () => 'x', reducedMotion: false })
      return ops.some((o) => o.op === 'stroke')
    }
    const derniere = PLUS_GRAND_NOMBRE_ENTIER(BOMB_BLAST_HOLD_MS, pas)
    expect(peint(derniere)).toBe(true)
    expect(peint(derniere + 1)).toBe(false)
  })

  it.each(PAS)("l'onde de capture est peinte jusqu à sa durée, pas une image de plus, à %i ms", (pas) => {
    const d = docAuPas(pas)
    const fx: FlagCaptureFx = { frame: 10, x: 5, y: 5, xuid: 'A' }
    const vue = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 528, height: 528, pad: 24 }
    const peint = (age: number) => {
      const { ctx, ops } = recordingContext()
      drawFlagCaptureFx(ctx, [fx], vue, { frame: 10 + age, hold: flagCaptureHoldFrames(d) }, { inkOf: () => 'x', reducedMotion: false })
      return ops.some((o) => o.op === 'stroke')
    }
    const derniere = PLUS_GRAND_NOMBRE_ENTIER(FLAG_CAPTURE_HOLD_MS, pas)
    expect(peint(derniere)).toBe(true)
    expect(peint(derniere + 1)).toBe(false)
  })

  it.each(PAS)('la traînée d un projectile dure la même durée réelle à %i ms', (pas) => {
    const d = docAuPas(pas)
    const pr = { t0: 0, p: [[0, 1, 1], [5, 2, 2]] } as unknown as ReplayProjectileReady
    const vue = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 528, height: 528, pad: 24 }
    const tailFrames = msToFrames(PROJECTILE_TAIL_MS, d)
    const peint = (frame: number) => {
      const { ctx, ops } = recordingContext()
      drawProjectilesLayer(ctx, [pr], vue, { frame, tailFrames }, 'c')
      return ops.some((o) => o.op === 'stroke')
    }
    // Le vol finit à l'image 5 : visible jusqu'à 5 + tailFrames inclus.
    const derniere = 5 + Math.floor(tailFrames)
    expect(peint(derniere)).toBe(true)
    expect(peint(derniere + 1)).toBe(false)
    expect(frameToMs(Math.floor(tailFrames), d)).toBeLessThanOrEqual(PROJECTILE_TAIL_MS)
  })
})

describe('périodes des glyphes : la même période réelle à 100 et à 33 ms', () => {
  it.each(PAS)('la pulsation d un glyphe porté se referme après sa période à %i ms', (pas) => {
    const periodeImages = CARRIED_PULSE_PERIOD_MS / pas
    for (const f of [0, 3, 11.5]) {
      expect(carriedGlyphAlpha(false, f + periodeImages, false, pas)).toBeCloseTo(
        carriedGlyphAlpha(false, f, false, pas),
        9,
      )
    }
    const vu = Array.from({ length: 400 }, (_, f) => carriedGlyphAlpha(false, f, false, pas))
    expect(Math.min(...vu)).toBeGreaterThanOrEqual(CARRIED_PULSE_MIN - 1e-9)
    expect(Math.max(...vu)).toBeLessThanOrEqual(CARRIED_PULSE_MAX + 1e-9)
  })

  it('à 100 ms la pulsation est celle d avant : 26 images', () => {
    const milieu = (CARRIED_PULSE_MIN + CARRIED_PULSE_MAX) / 2
    // Sinus : milieu à la phase 0, sommet au quart de période (6,5 images), creux aux trois quarts.
    expect(carriedGlyphAlpha(false, 0, false, 100)).toBeCloseTo(milieu, 12)
    expect(carriedGlyphAlpha(false, 6.5, false, 100)).toBeCloseTo(CARRIED_PULSE_MAX, 12)
    expect(carriedGlyphAlpha(false, 19.5, false, 100)).toBeCloseTo(CARRIED_PULSE_MIN, 12)
    expect(carriedGlyphAlpha(false, 13, false, 100)).toBeCloseTo(milieu, 12)
  })

  it.each(PAS)('le clignotement du drapeau porté se referme après sa période à %i ms', (pas) => {
    const periodeImages = BLINK_PERIOD_MS / pas
    for (const f of [0, 2, 7.25]) {
      expect(flagBlinkAlpha('carried', f + periodeImages, false, pas)).toBeCloseTo(
        flagBlinkAlpha('carried', f, false, pas),
        9,
      )
    }
  })

  it('à 100 ms le clignotement est celui d avant : 10 images, cosinus', () => {
    // 0,35 .. 0,95 : les extrema tombent sur des images entières (0 et 5).
    expect(flagBlinkAlpha('carried', 0, false, 100)).toBeCloseTo(0.95, 10)
    expect(flagBlinkAlpha('carried', 5, false, 100)).toBeCloseTo(0.35, 10)
  })
})

describe('visée et cap des véhicules : une seconde réelle à 100 et à 33 ms', () => {
  it.each(PAS)('le cap tourne vers la lecture suivante jusqu à une seconde, pas au-delà, à %i ms', (pas) => {
    const maxSpan = msToFrames(HEADING_INTERP_MAX_SPAN_MS, docAuPas(pas))
    const limite = Math.floor(maxSpan)
    const pts = (ecart: number) => [{ t: 0, h: 0 }, { t: ecart, h: 90 }]
    // Au bord : on tourne (le cap à mi-chemin est entre les deux lectures).
    const milieu = headingTowardNext(pts(limite), 0, limite / 2, maxSpan)
    expect(milieu).toBeGreaterThan(0)
    expect(milieu).toBeLessThan(90)
    // Une image au-delà : le cap reste tenu à la dernière lecture.
    expect(headingTowardNext(pts(limite + 1), 0, (limite + 1) / 2, maxSpan)).toBe(0)
  })

  it.each(PAS)('une lecture de visée reste en vigueur une seconde, pas une image de plus, à %i ms', (pas) => {
    const ride = { aim: [{ t: 10, h: 10 }] } as unknown as Parameters<typeof vehicleRideAimReading>[0]
    const limite = Math.floor(VEHICLE_AIM_HOLD_MS / pas)
    expect(vehicleRideAimReading(ride, 10 + limite, pas)?.h).toBe(10)
    expect(vehicleRideAimReading(ride, 10 + limite + 1, pas)).toBeNull()
  })
})

describe('sons : les tolérances sont des durées', () => {
  const crane = (pas: number, carries: unknown[], lives: unknown[]) =>
    ({
      frameIntervalMs: pas,
      originMs: 0,
      coverage: { originResolved: true },
      skullCarries: carries,
      objectiveObjects: lives.map((l) => ({ ...(l as object), pts: [], family: 'skull', fr: '', en: '' })),
    }) as unknown as ReplayDocumentReady

  it.each(PAS)('crâne : une prise suit un repos à 1,2 s près, pas au-delà, à %i ms', (pas) => {
    const ecart = Math.floor(1200 / pas)
    const prise = (gap: number) => {
      const d = crane(pas, [{ t0: 100 + gap, t1: 5000, closed: true, xuid: 'a' }], [{ t0: 100, t1: 100 }])
      return skullSoundEvents(d).map((e) => e.stem)
    }
    expect(prise(ecart)).toContain(SKULL_SOUND_STEMS.taken)
    expect(prise(ecart + 1)).toContain(SKULL_SOUND_STEMS.pickup)
  })

  it.each(PAS)('crâne : il retombe au sol dans les 6 s qui suivent le lâcher, pas au-delà, à %i ms', (pas) => {
    const ecart = Math.floor(6000 / pas)
    const lache = (gap: number) => {
      const d = crane(pas, [{ t0: 10, t1: 200, closed: true, xuid: 'a' }], [{ t0: 200 + gap, t1: 200 + gap + 50 }])
      return skullSoundEvents(d).map((e) => e.stem)
    }
    expect(lache(ecart)).toContain(SKULL_SOUND_STEMS.dropped)
    expect(lache(ecart + 1)).toContain(SKULL_SOUND_STEMS.despawn)
  })

  it.each(PAS)('zone : le propriétaire d arrivée s ouvre dans les 2 s qui suivent la rampe, pas au-delà, à %i ms', (pas) => {
    const ecart = Math.floor(2000 / pas)
    const rampe = (gap: number) =>
      zoneSoundEvents(
        {
          frameIntervalMs: pas,
          zoneStates: [
            {
              spans: [
                { t0: 0, t1: 30 + gap - 1, owner: null, active: false },
                { t0: 30 + gap, t1: 5000, owner: 1, active: false },
              ],
              gauge: [
                { t: 10, v: 0.1 },
                { t: 20, v: 0.4 },
                { t: 30, v: 0.9 },
              ],
            },
          ],
        } as unknown as ReplayDocumentReady,
        1,
      ).map((e) => e.stem)
    expect(rampe(ecart)).toEqual([ZONE_SOUND_STEMS.capturing.ally])
    expect(rampe(ecart + 1)).toEqual([ZONE_SOUND_STEMS.contested])
  })

  it.each(PAS)('arme : un ramassage natif et un `taken` à 500 ms près sont le même geste, à %i ms', (pas) => {
    const ecart = Math.floor(500 / pas)
    const geste = (gap: number) =>
      weaponChangeSoundEvents({
        frameIntervalMs: pas,
        weaponChanges: [{ t: 10, slot: 512, kind: 'taken', w: '1111aaaa' }],
        equipmentChanges: [],
        pickups: [{ t: 10 + gap, slot: 512, kind: 'weapon', w: '1111aaaa', class: 0 }],
      } as unknown as ReplayDocumentReady)
    expect(geste(ecart)).toHaveLength(1)
    expect(geste(ecart + 1)).toHaveLength(2)
  })
})
