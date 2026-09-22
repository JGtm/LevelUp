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
    expect(fx[0].vehicleShot?.mount?.classe).toBe('fixe')
    expect(fx[0].vehicleShot?.family).toBe('ghost')
    expect(fx[0].vehicleShot?.headingDeg).toBe(45)
  })

  /**
   * LA GARDE DU REGISTRE (2026-09-20) : une arme DE JOUEUR tirée depuis un siège de passager
   * garde son propre cap de regard. Le passager vise où il veut, et lui prêter la direction du
   * châssis serait une invention — c'est pourquoi la source véhicule ne lui est PAS attachée.
   */
  it('arme du REGISTRE tirée par un passager : vehicleShot reste null', () => {
    const d = doc({
      vehicles: [
        { slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'ghost', samples: [], rides: [] },
      ],
      weaponLabels: { '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' } },
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: '0xBR', v: 700 }],
    })
    expect(buildShotFx(d, 50)[0].vehicleShot).toBeNull()
  })

  /**
   * LE CORRECTIF DU 2026-09-20. Une arme DE VÉHICULE dont le tag n'est pas dans la table des
   * montages perdait TOUTE la source : plus de véhicule, plus de cap — et comme le bipède
   * embarqué ne réplique plus, l'éclair tombait sur la bouffée ronde sans direction. Mesuré : le
   * cap de regard est lisible pour 1 tir de véhicule sur 241 (`4f77afc1`). La source est désormais
   * gardée, sans montage.
   *
   * LE TÉMOIN A CHANGÉ AU LOT 5.8.3, et il est MEILLEUR : c'était le mortier du Wraith
   * (`121b4009`), qui a désormais un montage mesuré. C'est maintenant `850902ef` — un tag d'arme
   * de véhicule RÉELLEMENT OBSERVÉ dans un document cuit (9 tirs sur `5676a9ba`, §4 D4 du lot 5.8)
   * qu'aucun rapport de RE ne documente. Le cas de test n'est donc plus une hypothèse.
   */
  it('arme DE VÉHICULE sans montage documenté : la source est gardée, montage null', () => {
    const d = doc({
      vehicles: [
        {
          slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'wraith',
          samples: [{ t: 0, x: 5, y: 5, h: 120 }],
          rides: [],
        },
      ],
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: '0x850902EF00000000', v: 700 }],
    })
    const fx = buildShotFx(d, 50)
    expect(fx[0].vehicleShot).not.toBeNull()
    expect(fx[0].vehicleShot?.mount).toBeNull()
    expect(fx[0].vehicleShot?.headingDeg).toBe(120)
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

  /**
   * LOT 5.8.5 — UNE ARME DE JOUEUR TIRÉE D'UN SIÈGE PREND LA VISÉE DE SON TIREUR.
   *
   * 5.2a.5 laissait ces tirs sur leur propre cap de REGARD : la règle était juste, mais ce cap
   * vient de la trajectoire du bipède, qui ne réplique plus une fois embarqué — lisible pour
   * 1 tir sur 241 (`4f77afc1`), 76 tirs concernés. 5.5.2 a établi que la visée de l'épisode du
   * TIREUR (appariée par slot) est son PROPRE regard, et l'utilisateur l'a tranché le 2026-09-21.
   */
  it('arme de JOUEUR d’un siège AVEC visée lue : la source est gardée, pour sa visée', () => {
    const d = doc({
      vehicles: [
        {
          slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'warthog',
          samples: [{ t: 0, x: 5, y: 5, h: 45 }],
          rides: [{ slot: 1, t0: 0, t1: 100, src: 'film', seat: 1, aim: [{ t: 10, h: 200 }] }],
        },
      ],
      weaponLabels: { '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' } },
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: '0xBR', v: 700 }],
    })
    const fx = buildShotFx(d, 50)[0]
    expect(fx.vehicleShot).not.toBeNull()
    expect(fx.vehicleShot?.arme).toBe('joueur')
    expect(fx.vehicleShot?.mount).toBeNull()
    expect(fx.vehicleShot?.shooterHeadingDeg).toBe(200)
    // Le STYLE reste celui du registre : c'est bien une arme de joueur.
    expect(fx.fam).toBe('ballistic')
  })

  it('arme de JOUEUR d’un siège SANS visée lue : la source n’est pas créée (le regard survit)', () => {
    const d = doc({
      vehicles: [
        {
          slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'warthog',
          samples: [{ t: 0, x: 5, y: 5, h: 45 }],
          rides: [{ slot: 1, t0: 0, t1: 100, src: 'film', seat: 1, aim: [] }],
        },
      ],
      weaponLabels: { '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' } },
      shots: [{ slot: 1, t: 10, x: 5, y: 5, w: '0xBR', v: 700 }],
    })
    // Sans lecture de visée, la source n'apporterait rien et le tir perdrait son propre regard :
    // c'est exactement le comportement d'avant le lot, et la règle de 5.2a.5 survit.
    expect(buildShotFx(d, 50)[0].vehicleShot).toBeNull()
  })

  it('une arme DE VÉHICULE se déclare comme telle (le style et la direction en dépendent)', () => {
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
    expect(buildShotFx(d, 50)[0].vehicleShot?.arme).toBe('vehicule')
  })

  /**
   * LOT 5.8.2 — LA SECONDE JOINTURE DE STYLE. Sans elle, une arme de véhicule tombait sur la
   * famille `plain` et la teinte `neutral` (68 % des tirs de véhicule de `4f77afc1`) : un halo
   * gris pâle centré sur un sprite, qui ne se lit pas comme un tir.
   */
  it('une arme DE VÉHICULE prend la famille et la teinte de sa propre table', () => {
    const d = doc({ shots: [{ slot: 1, t: 10, x: 0, y: 0, w: GHOST_WEAP_TAG }] })
    const fx = buildShotFx(d, 50)[0]
    expect(fx.fam).toBe('plasma')
    expect(fx.tint).toBe('plasma_cool')
  })

  it('le REGISTRE garde la main : une arme de joueur ne prend jamais le style d’un véhicule', () => {
    const d = doc({
      weaponLabels: { '0xBR': { en: 'BR75', fr: 'BR75', fx: 'ballistic', tint: 'kinetic' } },
      shots: [{ slot: 1, t: 10, x: 0, y: 0, w: '0xBR' }],
    })
    const fx = buildShotFx(d, 50)[0]
    expect(fx.fam).toBe('ballistic')
    expect(fx.tint).toBe('kinetic')
  })

  it('une arme de véhicule NON documentée garde le rendu neutre, jamais celui d’une voisine', () => {
    const d = doc({ shots: [{ slot: 1, t: 10, x: 0, y: 0, w: '0xDEADBEEF00000000' }] })
    const fx = buildShotFx(d, 50)[0]
    expect(fx.fam).toBe('plain')
    expect(fx.tint).toBe('neutral')
  })
})
