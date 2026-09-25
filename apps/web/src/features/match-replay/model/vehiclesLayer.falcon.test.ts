/**
 * vehiclesLayer.falcon.test.ts — LE FALCON N'EST PLUS UNE FAMILLE DE DÉCOR (retours du rejeu, lot
 * M7b, décision utilisateur du 2026-09-24 : « les Pelican c'est toujours du décor ; le Falcon ça
 * dépend »).
 *
 * La décision du 2026-09-02 l'avait rangé dans `FAMILLES_NON_JOUABLES` sur la foi des Falcon de
 * décor de Behemoth (`0d76e8f1` : 0,3-0,8 m/s sur toute leur vie). Il est pilotable en
 * multijoueur : sa famille ne refuse plus ses occupants. Seul le verdict de décor de carte du
 * serveur (`vehicleIsScenery`, lot M7 : pose SEULE) le masque — et AUCUN Falcon du parc ne le
 * remplit : un Falcon qui plane sans occupant (profil de Behemoth) est DESSINÉ. C'est le changement
 * visible du lot, soumis à l'utilisateur (revue adverse RR-M7b-02) ; ce test le fige pour qu'une
 * décision contraire se voie.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleCanEmbark, vehicleIsDecor, vehicleIsHidden } from './vehiclesLayer'

function falcon(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700, gen: 1, t0: 0, t1: 1000, t1max: 1000, end: 'inconnue',
    family: 'falcon', samples: [], rides: [], ...over,
  }
}

describe('le Falcon suit la règle générale du décor (lot M7b)', () => {
  it('un Falcon se dessine et embarque son occupant, comme tout véhicule pilotable', () => {
    expect(vehicleIsDecor('falcon')).toBe(false)
    expect(vehicleIsHidden(falcon())).toBe(false)
    expect(vehicleCanEmbark(falcon())).toBe(true)
  })

  it('un Falcon de DÉCOR DE CARTE (verdict du serveur) reste masqué et n’embarque personne', () => {
    const decor = falcon({ scenery: true })
    expect(vehicleIsHidden(decor)).toBe(true)
    expect(vehicleCanEmbark(decor)).toBe(false)
  })

  it('un Falcon qui PLANE sans jamais être occupé (profil de Behemoth) est DESSINÉ : aucune règle ne le décide', () => {
    const planant = falcon({
      samples: [
        { t: 0, x: -146, y: 27.4, z: 8.5 },
        { t: 500, x: -147.3, y: 25.4, z: 10.8 },
      ],
    })
    expect(vehicleIsHidden(planant)).toBe(false)
  })

  it('le Pelican reste une famille de décor : toujours masqué', () => {
    expect(vehicleIsDecor('pelican')).toBe(true)
    expect(vehicleIsHidden(falcon({ family: 'pelican' }))).toBe(true)
  })
})
