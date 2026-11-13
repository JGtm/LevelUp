/**
 * Tests — vehicleDestructionsOf : la lecture de `vehicles[].destruction` dans le document.
 *
 * Ce qu'ils protègent : UNE destruction par véhicule `destroyed` qui n'est pas une pièce montée,
 * datée par `tEnd` ; l'auteur humain par son xuid, l'auteur BOT par le nom de son entrée de roster
 * (jointure `by.i` = `roster[].filmIndex`), jamais un joueur supposé ; les tués avec le véhicule
 * (`killed` seul, sans doublon : l'occupant n'y entre pas) ; et tout champ absent reste absent.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { vehicleDestructionsOf } from './vehicleDestructionSource'

function life(over: Record<string, unknown> = {}) {
  return {
    slot: 7, gen: 0, family: 'Ghost', end: 'destroyed', tEnd: 300, t0: 0, t1: 300, t1max: 300,
    samples: [], rides: [], ...over,
  }
}

function docWith(vehicles: unknown[], roster: unknown[] = []) {
  return testReplayDoc({ frameIntervalMs: 100, vehicles: vehicles as never, roster: roster as never } as never)
}

describe('vehicleDestructionsOf', () => {
  it('un document sans véhicule détruit ne donne rien (rejeu cuit avant le bloc : cas normal et durable)', () => {
    expect(vehicleDestructionsOf(docWith([life({ end: 'unknown', tEnd: undefined })]))).toEqual([])
    expect(vehicleDestructionsOf(docWith([]))).toEqual([])
  })

  it('une destruction complète : instant tEnd, famille, auteur humain et tués', () => {
    const out = vehicleDestructionsOf(
      docWith([
        life({
          destruction: {
            by: { i: 21, xuid: '2534182457688434' },
            cause: 2899431412,
            causeKey: 'hinf_spartan_laser',
            occupant: 'x2',
            killed: ['x2', 'x3'],
          },
        }),
      ]),
    )
    expect(out).toEqual([
      {
        t: 300, family: 'Ghost', authorXuid: '2534182457688434', authorBotName: null, killedXuids: ['x2', 'x3'],
        causeKey: 'hinf_spartan_laser',
      },
    ])
  })

  it('seule la liste `killed` compte : l’occupant n’y est pas ajouté, un tué sans occupant nommé reste ; sans doublon', () => {
    const out = vehicleDestructionsOf(
      docWith([life({ destruction: { by: { i: 1, xuid: 'a' }, occupant: 'x2', killed: ['x3', 'x2'] } })]),
    )
    expect(out[0].killedXuids).toEqual(['x3', 'x2'])
    const occupantSeul = vehicleDestructionsOf(
      docWith([life({ destruction: { by: { i: 1, xuid: 'a' }, occupant: 'x2' } })]),
    )
    expect(occupantSeul[0].killedXuids).toEqual([])
    const doublon = vehicleDestructionsOf(docWith([life({ destruction: { killed: ['x2', 'x2'] } })]))
    expect(doublon[0].killedXuids).toEqual(['x2'])
    const seul = vehicleDestructionsOf(docWith([life({ destruction: { by: { i: 1, xuid: 'a' }, killed: ['x3'] } })]))
    expect(seul[0].killedXuids).toEqual(['x3'])
  })

  it('un destructeur BOT se nomme par le roster (filmIndex), jamais par un xuid', () => {
    const roster = [
      { filmIndex: 5, xuid: '', name: 'Cobra07 [bot]', bot: true },
      { filmIndex: 21, xuid: 'h', name: 'Humain' },
    ]
    const out = vehicleDestructionsOf(
      docWith([life({ destruction: { by: { i: 5, bot: true } } })], roster),
    )
    expect(out).toEqual([
      { t: 300, family: 'Ghost', authorXuid: null, authorBotName: 'Cobra07 [bot]', killedXuids: [], causeKey: null },
    ])
  })

  it('un bot absent du roster, ou sans nom : forme neutre, aucun nom inventé', () => {
    const roster = [{ filmIndex: 6, xuid: '', bot: true }]
    const out = vehicleDestructionsOf(
      docWith(
        [
          life({ slot: 1, destruction: { by: { i: 99, bot: true } } }),
          life({ slot: 2, destruction: { by: { i: 6, bot: true } } }),
        ],
        roster,
      ),
    )
    expect(out.map((d) => [d.authorXuid, d.authorBotName])).toEqual([[null, null], [null, null]])
  })

  it('`by` sans xuid ni drapeau bot : l’auteur n’est pas établi, jamais le nom d’un autre joueur', () => {
    const roster = [{ filmIndex: 3, xuid: 'h', name: 'Humain' }]
    const out = vehicleDestructionsOf(docWith([life({ destruction: { by: { i: 3 } } })], roster))
    expect(out[0].authorXuid).toBeNull()
    expect(out[0].authorBotName).toBeNull()
  })

  it('`destroyed` sans `destruction`, ou `destruction` sans `by` : la ligne existe, sans auteur', () => {
    const out = vehicleDestructionsOf(
      docWith([life({ slot: 1 }), life({ slot: 2, destruction: { cause: 5, occupant: 'x2' } })]),
    )
    expect(out.map((d) => [d.authorXuid, d.authorBotName, d.killedXuids])).toEqual([
      [null, null, []],
      [null, null, []],
    ])
  })

  it('une pièce montée (`part` non vide) ne fait aucune ligne : sa destruction est celle de son porteur', () => {
    const out = vehicleDestructionsOf(
      docWith([life({ part: 'turret', destruction: { by: { i: 1, xuid: 'a' } } }), life({ slot: 8 })]),
    )
    expect(out).toHaveLength(1)
    expect(out[0].t).toBe(300)
  })

  it('une fin `destroyed` sans instant `tEnd`, ou une vie de décor : aucune ligne', () => {
    const out = vehicleDestructionsOf(
      docWith([life({ tEnd: undefined }), life({ slot: 2, scenery: true }), life({ slot: 3, end: 'vanished' })]),
    )
    expect(out).toEqual([])
  })

  it('la clé de la cause se lit telle que le serveur l’a posée ; absente ou vide : nulle, jamais devinée de `cause`', () => {
    const out = vehicleDestructionsOf(
      docWith([
        life({ slot: 1, destruction: { cause: 2899431412, causeKey: 'hinf_spartan_laser' } }),
        life({ slot: 2, destruction: { cause: 2899431412 } }),
        life({ slot: 3, destruction: { cause: 36, causeKey: '' } }),
        life({ slot: 4 }),
      ]),
    )
    expect(out.map((d) => d.causeKey)).toEqual(['hinf_spartan_laser', null, null, null])
  })

  it('une famille absente garde une destruction de famille vide (la ligne la refusera, aucun nom inventé)', () => {
    const out = vehicleDestructionsOf(docWith([life({ family: undefined })]))
    expect(out.map((d) => d.family)).toEqual([''])
  })
})
