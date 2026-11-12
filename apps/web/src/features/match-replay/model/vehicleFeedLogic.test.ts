/**
 * Tests — vehicleFeedLogic : « Véhicules », tiré des épisodes d'occupation `vehicles[].rides[]` et des destructions.
 *
 * Ce qu'ils protègent : un occupant NOMMÉ et un début de montée DATÉ (`src` autre que `gap`) ;
 * un épisode qui chevauche le précédent du même joueur dans le même véhicule est la même
 * montée, pas une seconde ; le décor et les éléments de carte ne sont pas des véhicules.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { vehicleDestructionsOf } from './vehicleDestructionSource'
import { vehicleDestructionEntries, vehicleFeedEntries } from './vehicleFeedLogic'

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

describe('vehicleDestructionEntries — destructions fabriquées', () => {
  const destruction = (over: Record<string, unknown> = {}) => ({
    t: 300, family: 'Ghost', authorXuid: 'x1', authorBotName: null, aboardXuids: [], ...over,
  })

  it('une ligne par destruction, à son instant, au nom de la famille, avec son auteur', () => {
    const out = vehicleDestructionEntries([destruction({ aboardXuids: ['x2'] })], docWith([]), 'fr')
    expect(out).toHaveLength(1)
    expect(out[0].replayMs).toBe(30_000)
    expect(out[0].event).toEqual({
      family: 'vehicles', destroyed: true, xuid: 'x1', botName: null, aboardXuids: ['x2'], name: 'Ghost',
    })
  })

  it('sans auteur connu : l’auteur est nul, la ligne existe quand même', () => {
    const out = vehicleDestructionEntries([destruction({ authorXuid: null })], docWith([]), 'fr')
    expect(out.map((e) => e.event)).toEqual([
      { family: 'vehicles', destroyed: true, xuid: null, botName: null, aboardXuids: [], name: 'Ghost' },
    ])
  })

  it('un destructeur bot porte son nom de roster, pas de xuid', () => {
    const out = vehicleDestructionEntries(
      [destruction({ authorXuid: null, authorBotName: 'Cobra07 [bot]' })],
      docWith([]),
      'fr',
    )
    expect(out.map((e) => e.event)).toEqual([
      { family: 'vehicles', destroyed: true, xuid: null, botName: 'Cobra07 [bot]', aboardXuids: [], name: 'Ghost' },
    ])
  })

  it('le libellé du document, dans la langue du lecteur, prime sur la clé de famille', () => {
    const doc = docWith([], { weird: { en: 'Odd thing', fr: 'Drôle de chose' } })
    const out = vehicleDestructionEntries([destruction({ family: 'weird' })], doc, 'en')
    expect((out[0].event as { name: string }).name).toBe('Odd thing')
  })

  it('écarte une famille vide, le décor et les éléments de carte : pas de nom inventé', () => {
    const doc = docWith([], { turret_auto: { kind: 'map_element', en: 'Auto turret', fr: 'Tourelle' } })
    const out = vehicleDestructionEntries(
      [destruction({ family: '' }), destruction({ family: 'pelican' }), destruction({ family: 'turret_auto' })],
      doc,
      'fr',
    )
    expect(out).toEqual([])
  })

  it('une liste vide ne fait aucune ligne', () => {
    expect(vehicleDestructionEntries([], docWith([]), 'fr')).toEqual([])
  })
})

describe('vehicleFeedEntries — montées et destructions dans un même fil trié', () => {
  it('une destruction s’insère à son instant parmi les montées', () => {
    const doc = docWith([life({ rides: [ride({ t0: 100 }), ride({ t0: 500, t1: 600 })] })])
    const out = vehicleFeedEntries(doc, 'fr', [
      { t: 300, family: 'Warthog', authorXuid: 'x2', authorBotName: null, aboardXuids: ['x1'] },
    ])
    expect(out.map((e) => e.replayMs)).toEqual([10_000, 30_000, 50_000])
    expect(out.map((e) => 'destroyed' in (e.event ?? {}))).toEqual([false, true, false])
  })

  it('sans destruction dans le document (rejeu cuit avant le bloc), le fil ne porte que les montées', () => {
    expect(vehicleDestructionsOf(docWith([life()]))).toEqual([])
    expect(vehicleFeedEntries(docWith([life()]), 'fr')).toHaveLength(1)
  })

  it('le document lu de bout en bout : une vie détruite devient une ligne datée, avec auteur et occupant', () => {
    const doc = docWith([
      life({
        rides: [],
        end: 'destroyed',
        tEnd: 300,
        destruction: { by: { i: 1, xuid: 'x2' }, occupant: 'x1', killed: ['x1'] },
      }),
    ])
    const out = vehicleFeedEntries(doc, 'fr')
    expect(out).toHaveLength(1)
    expect(out[0].replayMs).toBe(30_000)
    expect(out[0].event).toEqual({
      family: 'vehicles', destroyed: true, xuid: 'x2', botName: null, aboardXuids: ['x1'], name: 'Warthog',
    })
  })
})
