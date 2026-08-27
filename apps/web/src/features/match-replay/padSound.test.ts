import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from './replayNormalize'
import { familleDe, PAD_PICKUP_SOUND_STEM, padPickupSoundEvents } from './padSound'

/**
 * padSound.test.ts — le ramassage sur socle, et surtout ses BORNES.
 *
 * La règle est une HEURISTIQUE assumée (le premier tir d'une arme de socle prouve qu'on l'a
 * ramassée), pas une mesure du film. Ce qui doit donc être épinglé, c'est ce qu'elle refuse de
 * faire : sonner sans catalogue de socles, sonner deux fois pour le même joueur, sonner pour
 * une arme qui n'est sur aucun socle.
 */

function doc(weaponPads: unknown[], shots: unknown[]): ReplayDocumentReady {
  return { frameIntervalMs: 100, weaponPads, shots } as unknown as ReplayDocumentReady
}

const SOCLE = '0x84BD29ED'
const TIR_SOCLE = '0x84BD29ED42C9679F'
const TIR_AUTRE = '0xB619D84A42C9679F'

describe('familleDe — la clé de jointure, calculée UNE fois', () => {
  it('garde les 32 bits de poids fort, en minuscules', () => {
    expect(familleDe('0x84BD29ED42C9679F')).toBe('0x84bd29ed')
  })

  it('rend la chaîne telle quelle si elle est trop courte pour porter une famille', () => {
    expect(familleDe('0x84')).toBe('0x84')
  })
})

describe('ramassage sur socle — un son au premier tir de l arme du socle', () => {
  it('sonne à la frame du tir, pas à celle du socle', () => {
    const d = doc([{ weapon: SOCLE }], [{ t: 300, slot: 512, w: TIR_SOCLE }])
    expect(padPickupSoundEvents(d)).toEqual([{ ms: 30000, stem: PAD_PICKUP_SOUND_STEM }])
  })

  it('UNE SEULE FOIS par joueur et par famille : garder l arme ne la re-annonce pas', () => {
    const d = doc(
      [{ weapon: SOCLE }],
      [
        { t: 300, slot: 512, w: TIR_SOCLE },
        { t: 310, slot: 512, w: TIR_SOCLE },
        { t: 900, slot: 512, w: TIR_SOCLE },
      ],
    )
    expect(padPickupSoundEvents(d)).toHaveLength(1)
  })

  it('deux joueurs sur la même famille sonnent chacun une fois', () => {
    const d = doc(
      [{ weapon: SOCLE }],
      [
        { t: 300, slot: 512, w: TIR_SOCLE },
        { t: 400, slot: 514, w: TIR_SOCLE },
      ],
    )
    expect(padPickupSoundEvents(d).map((e) => e.ms)).toEqual([30000, 40000])
  })

  it('une arme qui n est sur AUCUN socle ne sonne pas', () => {
    const d = doc([{ weapon: SOCLE }], [{ t: 300, slot: 512, w: TIR_AUTRE }])
    expect(padPickupSoundEvents(d)).toEqual([])
  })

  it('MUET sans catalogue de socles : sans lui, tous les tirs sonneraient', () => {
    expect(padPickupSoundEvents(doc([], [{ t: 300, slot: 512, w: TIR_SOCLE }]))).toEqual([])
  })

  it('MUET quand les socles n ont pas de famille lisible', () => {
    expect(
      padPickupSoundEvents(doc([{ weapon: null }], [{ t: 300, slot: 512, w: TIR_SOCLE }])),
    ).toEqual([])
  })

  it('les événements sortent dans l ordre du temps, quel que soit celui des tirs', () => {
    const d = doc(
      [{ weapon: SOCLE }],
      [
        { t: 900, slot: 514, w: TIR_SOCLE },
        { t: 300, slot: 512, w: TIR_SOCLE },
      ],
    )
    expect(padPickupSoundEvents(d).map((e) => e.ms)).toEqual([30000, 90000])
  })
})
