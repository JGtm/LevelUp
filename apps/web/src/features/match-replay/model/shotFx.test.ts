/**
 * shotFx.test.ts — LES RÈGLES DU TIR AVANT DESSIN : d'où vient sa direction, ce qui n'a pas
 * d'éclair, et ce qui reste sans teinte plutôt que d'en emprunter une.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { buildShotFx } from './shotFx'
import { testReplayDoc } from '../test/testDoc'

/** Document 10 Hz : une vie au slot 1, dont le regard n'est transmis qu'à certaines frames. */
function doc(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      {
        slot: 1,
        team: -1,
        xuid: 'A',
        points: [
          { t: 0, x: 0, y: 0, h: 90 },
          { t: 10, x: 1, y: 0 },
          { t: 20, x: 2, y: 0, h: 180 },
        ],
        startFrame: 0,
        endFrame: 60,
      },
    ],
    weaponLabels: {
      '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' },
      '0xSWORD': { en: 'Épée', fr: 'Épée', fx: 'melee' },
      '0xNIL': { en: '?', fr: '?' },
    },
    ...over,
  })
}

describe('buildShotFx', () => {
  it('oriente le tir par le REGARD du tireur, pas par le champ de l’événement', () => {
    // Le tir tombe à la frame 15 : aucune lecture de regard à cet instant même, la dernière
    // connue (frame 0, cap 90) est encore dans la fenêtre de maintien. C'est exactement ce
    // que fait le cône de visée — une seule lecture, une seule règle.
    const fx = buildShotFx(doc({ shots: [{ slot: 1, t: 15, x: 0, y: 0, w: '0xBR' }] }), 50)
    expect(fx).toHaveLength(1)
    expect(fx[0].h).toBe(90)
  })

  it('ne prend PAS le cap de l’événement quand le regard manque : rien plutôt qu’un axe', () => {
    // Fenêtre de maintien d'une seule frame : la lecture de la frame 0 est périmée à 15.
    // Le champ `h` de l'événement, lui, est présent — et il est délibérément ignoré.
    const fx = buildShotFx(doc({ shots: [{ slot: 1, t: 15, x: 0, y: 0, w: '0xBR', h: 42 }] }), 1)
    expect(fx[0].h).toBeNull()
  })

  it('un tir sans tireur identifiable n’a pas de direction, et se dessine quand même', () => {
    const fx = buildShotFx(doc({ shots: [{ slot: 99, t: 5, x: 3, y: 4, w: '0xBR' }] }), 50)
    expect(fx).toHaveLength(1)
    expect(fx[0].h).toBeNull()
    expect(fx[0].x).toBe(3)
  })

  it('la MÊLÉE n’entre pas : un coup de marteau n’a pas d’éclair de bouche', () => {
    const fx = buildShotFx(
      doc({
        shots: [
          { slot: 1, t: 5, x: 0, y: 0, w: '0xSWORD' },
          { slot: 1, t: 6, x: 0, y: 0, w: '0xBR' },
        ],
      }),
      50,
    )
    expect(fx.map((e) => e.fam)).toEqual(['ballistic'])
  })

  it('une arme sans teinte déclarée reste NEUTRE — jamais la teinte d’une voisine', () => {
    const fx = buildShotFx(doc({ shots: [{ slot: 1, t: 5, x: 0, y: 0, w: '0xNIL' }] }), 50)
    expect(fx[0].tint).toBe('neutral')
    expect(fx[0].fam).toBe('plain')
  })

  it('une teinte inconnue du client (document plus récent) retombe sur NEUTRE', () => {
    const fx = buildShotFx(
      doc({
        shots: [{ slot: 1, t: 5, x: 0, y: 0, w: '0xX' }],
        weaponLabels: { '0xX': { en: 'X', fr: 'X', fx: 'plasma', tint: 'antimatiere' } },
      }),
      50,
    )
    expect(fx[0].tint).toBe('neutral')
    expect(fx[0].fam).toBe('plasma')
  })

  it('lit le regard de la vie QUI COUVRE l’instant, pas d’une autre vie du même slot', () => {
    // Le slot de biped est réattribué à chaque réapparition : deux vies portent le slot 1,
    // et le tir tombe dans la seconde. Prendre la première lirait un cap périmé de 30 frames.
    const d = testReplayDoc({
      frameIntervalMs: 100,
      tracks: [
        { slot: 1, team: -1, points: [{ t: 0, x: 0, y: 0, h: 10 }], startFrame: 0, endFrame: 20 },
        { slot: 1, team: -1, points: [{ t: 30, x: 0, y: 0, h: 200 }], startFrame: 30, endFrame: 60 },
      ],
      weaponLabels: { '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' } },
      shots: [{ slot: 1, t: 35, x: 0, y: 0, w: '0xBR' }],
    })
    expect(buildShotFx(d, 50)[0].h).toBe(200)
  })

  it('sans tir, aucun travail : la liste est vide', () => {
    expect(buildShotFx(doc(), 50)).toEqual([])
  })
})

// Le tag `weap` du Ghost tel que le porte réellement `Shot.w` (`vehicleWeaponMounts.ts` :
// `0x` + tag weap 8 hex + 32 bits nuls, vérifié en direct sur le Warthog et le Wasp). Une
// arme de véhicule N'A PAS d'entrée dans `weaponLabels` (ce ne sont pas des armes de joueur).
const GHOST_WEAP_TAG = '0x0001543500000000'

describe('buildShotFx — tirs en véhicule (v), origine au montage plutôt qu’au centre', () => {
  it('v marqué + arme au montage connu (fixe) : vehicleShot porte le montage et le cap', () => {
    const d = doc({
      vehicles: [
        {
          slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'ghost',
          samples: [{ t: 0, x: 5, y: 5, h: 45 }],
          rides: [],
        },
      ],
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: GHOST_WEAP_TAG, v: 700 }],
    })
    const fx = buildShotFx(d, 50)
    expect(fx).toHaveLength(1)
    expect(fx[0].vehicleShot).not.toBeNull()
    expect(fx[0].vehicleShot?.mount.classe).toBe('fixe')
    expect(fx[0].vehicleShot?.family).toBe('ghost')
    expect(fx[0].vehicleShot?.headingDeg).toBe(45)
  })

  it('v marqué mais arme SANS montage connu (arme de joueur tirée par un passager) : vehicleShot est null', () => {
    const d = doc({
      vehicles: [
        { slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'ghost', samples: [], rides: [] },
      ],
      weaponLabels: { '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' } },
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: '0xBR', v: 700 }],
    })
    expect(buildShotFx(d, 50)[0].vehicleShot).toBeNull()
  })

  it('v marqué mais AUCUN véhicule de ce slot dans le document : vehicleShot est null', () => {
    const d = doc({
      vehicles: [],
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: GHOST_WEAP_TAG, v: 700 }],
    })
    expect(buildShotFx(d, 50)[0].vehicleShot).toBeNull()
  })

  it('tir à pied (v absent) : vehicleShot est null, même avec une arme de véhicule connue', () => {
    const d = doc({
      shots: [{ slot: 1, t: 10, x: 0, y: 0, w: GHOST_WEAP_TAG }],
    })
    expect(buildShotFx(d, 50)[0].vehicleShot).toBeNull()
  })
})
