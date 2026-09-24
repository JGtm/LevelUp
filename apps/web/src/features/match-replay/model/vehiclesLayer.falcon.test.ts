/**
 * vehiclesLayer.falcon.test.ts — LE FALCON N'EST PLUS UNE FAMILLE DE DÉCOR (retours du rejeu, lot
 * M7b, décision utilisateur du 2026-09-24 : « les Pelican c'est toujours du décor ; le Falcon ça
 * dépend »).
 *
 * La décision du 2026-09-02 l'avait rangé dans `FAMILLES_NON_JOUABLES` sur la foi de Falcon de
 * décor vus en visionnage. Il est pilotable en multijoueur (parc du 2026-09-24 : 19 Falcon occupés
 * dans 5 documents) : sa famille ne décide plus rien, et un Falcon de décor se masque VIE PAR VIE
 * par la règle générale du décor de carte (`vehicleIsScenery`, verdict du serveur, lot M7).
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

  it('le Pelican reste une famille de décor : toujours masqué', () => {
    expect(vehicleIsDecor('pelican')).toBe(true)
    expect(vehicleIsHidden(falcon({ family: 'pelican' }))).toBe(true)
  })
})
