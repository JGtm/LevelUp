/**
 * vehiclesAim.test.ts — LA VISÉE D'UN OCCUPANT DE VÉHICULE (schéma 39), sans canvas.
 *
 * CE QUE CES TESTS PROTÈGENT. Une régression ici ne se verrait PAS à l'écran : le cône
 * retomberait sur le cap du châssis, c'est-à-dire sur une direction PLAUSIBLE mais fausse de 15,7
 * à 21,8 deg en médiane (q3 39,6-52,9 deg, lot V11). C'est exactement le genre de défaut qu'un
 * visionnage ne rattrape pas — d'où le `measured` du résultat, sur lequel les cas s'accrochent.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleAim } from '@/lib/api/types'

import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import {
  VEHICLE_AIM_HOLD_FRAMES,
  vehicleChassisHeadingAt,
  vehicleOccupantAimAt,
  vehicleRideAimReading,
} from './vehiclesAim'
import { vehicleAimAngle, FAMILLES_ARME_FIXE, VEHICLE_DEFAULT_HEADING_DEG } from './vehiclesLayer'

/** Un véhicule dont le châssis pointe PLEIN EST (cap monde 0°) : le repli est alors l'angle 0. */
function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700,
    gen: 1,
    t0: 0,
    t1: 1000,
    t1max: 1000,
    end: 'inconnue',
    family: 'warthog',
    samples: [{ t: 0, x: 0, y: 0, h: 0 }],
    rides: [],
    ...over,
  }
}

function ride(aim: ReplayVehicleAim[], over: Partial<ReplayVehicleRideReady> = {}): ReplayVehicleRideReady {
  return { t0: 0, t1: 1000, slot: 7, src: 'event', aim, ...over }
}

describe('vehicleRideAimReading — la lecture EN VIGUEUR', () => {
  it('rend la dernière lecture au plus tard à l’image demandée (aucune interpolation)', () => {
    const r = ride([
      { t: 10, h: 10 },
      { t: 14, h: 20 },
      { t: 30, h: 30 },
    ])
    expect(vehicleRideAimReading(r, 14)?.h).toBe(20)
    // ENTRE DEUX LECTURES : la précédente est MAINTENUE, jamais moyennée — interpoler deux caps
    // ferait tourner le cône par le chemin le plus court à travers 0/360 deg.
    expect(vehicleRideAimReading(r, 16)?.h).toBe(20)
  })

  it('rend null AVANT la première lecture, et sur une série vide', () => {
    expect(vehicleRideAimReading(ride([{ t: 10, h: 10 }]), 9)).toBeNull()
    expect(vehicleRideAimReading(ride([]), 50)).toBeNull()
  })

  it('rend null au-delà du MAINTIEN — une visée d’il y a plus d’une seconde n’est plus la sienne', () => {
    const r = ride([{ t: 10, h: 10 }])
    expect(vehicleRideAimReading(r, 10 + VEHICLE_AIM_HOLD_FRAMES)?.h).toBe(10)
    expect(vehicleRideAimReading(r, 10 + VEHICLE_AIM_HOLD_FRAMES + 1)).toBeNull()
  })

  it('SAUTE un point sans cap plutôt que de le lire comme 0° (qui pointerait l’est)', () => {
    const r = ride([{ t: 10, h: 45 }, { t: 12 }])
    expect(vehicleRideAimReading(r, 12)?.h).toBe(45)
  })
})

describe('vehicleOccupantAimAt — la mesure d’abord, le châssis en repli', () => {
  it('la visée MESURÉE l’emporte sur le cap du châssis, et elle porte son élévation', () => {
    const got = vehicleOccupantAimAt(track(), ride([{ t: 50, h: 90, p: -12.5 }]), 50)
    expect(got.measured).toBe(true)
    expect(got.ang).toBeCloseTo(vehicleAimAngle(90), 10)
    expect(got.pitchDeg).toBe(-12.5)
  })

  it('ARTILLEUR ET PASSAGER ont chacun LEUR angle — c’est tout l’objet du lot', () => {
    const t = track()
    const artilleur = vehicleOccupantAimAt(t, ride([{ t: 50, h: 200 }], { slot: 8, seat: 1 }), 50)
    const passager = vehicleOccupantAimAt(t, ride([{ t: 50, h: 300 }], { slot: 9, seat: 2 }), 50)
    expect(artilleur.ang).toBeCloseTo(vehicleAimAngle(200), 10)
    expect(passager.ang).toBeCloseTo(vehicleAimAngle(300), 10)
    expect(artilleur.ang).not.toBeCloseTo(passager.ang, 3)
  })

  it('SANS visée, le repli est le CAP DU CHÂSSIS, à plat — le comportement d’avant la série de visée', () => {
    const got = vehicleOccupantAimAt(track(), ride([]), 50)
    expect(got.measured).toBe(false)
    expect(got.ang).toBeCloseTo(vehicleAimAngle(0), 10) // châssis plein est
    expect(got.pitchDeg).toBe(0)
  })

  it('une élévation ABSENTE se lit « à plat », jamais « inconnue » (contrat de `Point.p`)', () => {
    expect(vehicleOccupantAimAt(track(), ride([{ t: 50, h: 90 }]), 50).pitchDeg).toBe(0)
  })
})
/**
 * vehicleChassisHeadingAt — LE CAP AUQUEL LE CHÂSSIS SE DESSINE (décision de l'utilisateur du
 * 2026-09-20 : « le châssis s'oriente là où l'ARME pointe »).
 *
 * CE QUE CES CAS PROTÈGENT, ET QUI NE SE VOIT PAS À L'OEIL : la règle ne vaut QUE pour les
 * familles à arme fixe. L'appliquer à une famille à tourelle ferait pivoter le châssis à chaque
 * balayage du tireur — un mouvement crédible, et faux, que personne ne saurait attribuer.
 */
describe('vehicleChassisHeadingAt — la visée du conducteur sur les armes fixes, la vélocité sinon', () => {
  /** Le véhicule pointe plein est par sa VÉLOCITÉ (cap 0), et son conducteur vise plein nord. */
  const AIM_NORD = 90
  const conducteur = (aim: ReplayVehicleAim[]) => ride(aim, { seat: 0 })

  it('GHOST occupé (arme fixe) : le cap du châssis EST la visée du conducteur', () => {
    const t = track({ family: 'ghost', rides: [conducteur([{ t: 0, h: AIM_NORD }])] })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(AIM_NORD)
  })

  it('GHOST VIDE : aucun conducteur, donc la vélocité', () => {
    const t = track({ family: 'ghost', rides: [] })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(0)
  })

  it('WARTHOG occupé (famille à TOURELLE) : la vélocité, jamais la visée', () => {
    const t = track({ family: 'warthog', rides: [conducteur([{ t: 0, h: AIM_NORD }])] })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(0)
  })

  it('arme fixe, conducteur présent mais AUCUNE lecture de visée : la vélocité', () => {
    const t = track({ family: 'wraith', rides: [conducteur([])] })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(0)
  })

  it('arme fixe, lecture de visée PÉRIMÉE : la vélocité reprend la main', () => {
    const t = track({ family: 'wraith', rides: [conducteur([{ t: 0, h: AIM_NORD }])] })
    // Au-delà du maintien, la lecture n'est plus en vigueur — même règle que le cône.
    expect(vehicleChassisHeadingAt(t, VEHICLE_AIM_HOLD_FRAMES + 1)).toBe(0)
  })

  it('SEUL LE SIÈGE 0 compte : la visée d’un passager n’oriente pas le châssis', () => {
    const t = track({
      family: 'ghost',
      rides: [ride([{ t: 0, h: AIM_NORD }], { seat: 1 })],
    })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(0)
  })

  it('famille INCONNUE de la table : le repli neutre, c’est-à-dire la vélocité', () => {
    const t = track({ family: 'chassis_futur', rides: [conducteur([{ t: 0, h: AIM_NORD }])] })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(0)
  })

  it('aucun cap de vélocité nulle part : le défaut, nez vers le haut', () => {
    const t = track({ family: 'warthog', samples: [{ t: 0, x: 0, y: 0 }], rides: [] })
    expect(vehicleChassisHeadingAt(t, 0)).toBe(VEHICLE_DEFAULT_HEADING_DEG)
  })

  /**
   * LE CÔNE NE BOUGE PAS. Son repli reste la VÉLOCITÉ nue, et c'est ce que `measured: false`
   * documente : lui donner la visée d'un AUTRE occupant ferait passer une mesure d'autrui pour
   * une approximation de soi.
   */
  it('le repli du CÔNE reste la vélocité, même sur une famille à arme fixe', () => {
    const t = track({ family: 'ghost', rides: [conducteur([{ t: 0, h: AIM_NORD }])] })
    const passager = ride([], { seat: 1 })
    const vu = vehicleOccupantAimAt(t, passager, 0)
    expect(vu.measured).toBe(false)
    expect(vu.ang).toBe(vehicleAimAngle(0))
  })

  it('les dix familles à arme fixe sont celles de la décision, et elles seules', () => {
    expect([...FAMILLES_ARME_FIXE].sort()).toEqual([
      'banshee', 'chopper', 'ghost', 'gungoose', 'mongoose',
      'shade', 'tourelle_fixe', 'tourelle_montee', 'wasp', 'wraith',
    ])
    // Les familles à TOURELLE ne sont PAS une seconde table : elles reçoivent le repli.
    for (const tourelle of [
      'warthog', 'warthog_gauss', 'rockethog', 'razorback',
      'scorpion', 'falcon', 'pelican', 'phantom', 'skiff',
    ]) {
      expect(FAMILLES_ARME_FIXE.has(tourelle), tourelle).toBe(false)
    }
  })
})
