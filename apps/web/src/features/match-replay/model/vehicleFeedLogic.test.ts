/**
 * Tests — vehicleFeedLogic : « Véhicules », tiré des épisodes d'occupation `vehicles[].rides[]`.
 *
 * Ce qu'ils protègent : un occupant NOMMÉ et un début de montée DATÉ (`src` autre que `gap`) ;
 * un épisode qui chevauche le précédent du même joueur dans le même véhicule est la même
 * montée, pas une seconde ; le décor et les éléments de carte ne sont pas des véhicules.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { vehicleFeedEntries } from './vehicleFeedLogic'

const FRAME_MS = 100

function ride(over: Record<string, unknown> = {}) {
  return { t0: 100, t1: 200, slot: 1, xuid: 'x1', src: 'event', aim: [], ...over }
}

function life(over: Record<string, unknown> = {}) {
  return {
    slot: 7, gen: 0, family: 'Warthog', end: 'unknown', t0: 0, t1: 900, t1max: 900,
    samples: [], rides: [ride()], ...over,
  }
}

function docWith(vehicles: unknown[], vehicleLabels?: Record<string, unknown>) {
  return testReplayDoc({
    frameIntervalMs: FRAME_MS,
    vehicles: vehicles as never,
    ...(vehicleLabels ? { vehicleLabels: vehicleLabels as never } : {}),
  } as never)
}

describe('vehicleFeedEntries', () => {
  it('une ligne par montée nommée et datée, au nom de la famille', () => {
    const out = vehicleFeedEntries(docWith([life()]), 'fr')
    expect(out).toHaveLength(1)
    expect(out[0].replayMs).toBe(10_000)
    expect(out[0].event).toEqual({ family: 'vehicles', xuid: 'x1', name: 'Warthog' })
  })

  it('écarte un occupant inconnu et un début non daté (`gap`)', () => {
    const out = vehicleFeedEntries(
      docWith([life({ rides: [ride({ xuid: '' }), ride({ src: 'gap', xuid: 'x2' })] })]),
      'fr',
    )
    expect(out).toEqual([])
  })

  it('un épisode qui chevauche le précédent du même joueur dans le même véhicule n’est pas une seconde montée', () => {
    const rides = [ride({ t0: 100, t1: 200 }), ride({ t0: 150, t1: 260, seat: 1 }), ride({ t0: 400, t1: 500 })]
    const out = vehicleFeedEntries(docWith([life({ rides })]), 'fr')
    expect(out.map((e) => e.replayMs)).toEqual([10_000, 40_000])
  })

  it('le même joueur dans deux véhicules différents fait deux lignes', () => {
    const out = vehicleFeedEntries(
      docWith([
        life({ slot: 7, rides: [ride({ t0: 100, t1: 200 })] }),
        life({ slot: 8, family: 'Ghost', rides: [ride({ t0: 120, t1: 220 })] }),
      ]),
      'fr',
    )
    expect(out.map((e) => (e.event as { name: string }).name)).toEqual(['Warthog', 'Ghost'])
  })

  it('le libellé du document, dans la langue du lecteur, prime sur la clé de famille', () => {
    const out = vehicleFeedEntries(
      docWith([life({ family: 'weird' })], { weird: { en: 'Odd thing', fr: 'Drôle de chose' } }),
      'fr',
    )
    expect((out[0].event as { name: string }).name).toBe('Drôle de chose')
  })

  it('ignore le décor, les éléments de carte, et les vies déclarées décor par le serveur', () => {
    const out = vehicleFeedEntries(
      docWith(
        [
          life({ slot: 1, family: 'pelican' }),
          life({ slot: 2, family: 'turret_auto' }),
          life({ slot: 3, scenery: true }),
        ],
        { turret_auto: { kind: 'map_element', en: 'Auto turret', fr: 'Tourelle automatique' } },
      ),
      'fr',
    )
    expect(out).toEqual([])
  })

  it('une famille non résolue (vide) ne fait aucune ligne : pas de nom inventé', () => {
    expect(vehicleFeedEntries(docWith([life({ family: '' })]), 'fr')).toEqual([])
  })
})
