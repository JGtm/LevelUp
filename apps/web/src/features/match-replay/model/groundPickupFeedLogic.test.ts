/**
 * Tests — groundPickupFeedLogic : les armes RAMASSÉES AU SOL de « Armes et bonus ».
 *
 * Ce qu'ils protègent : une ligne n'existe que si le document DATE le ramassage (fin `pickup`,
 * exacte), NOMME le ramasseur (le slot du ramasseur occupe une vie à xuid à cet instant),
 * IDENTIFIE l'arme (le libellé du document la nomme) ET si l'arme n'est PAS une arme de base — la
 * règle des armes de départ du calque (dotation de naissance, `startingWeaponKeys`). Une arme de
 * base ramassée n'apparaît pas ; sans dotation de naissance, ou avec des départs aléatoires,
 * la base n'est pas établie : aucune ligne plutôt qu'un classement deviné.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { eventFeedEntry } from './feedEventLines'
import { groundPickupFeedEntries } from './groundPickupFeedLogic'

const FRAME_MS = 100

/** Famille de départ (AR) et familles « spéciales » : les écritures des deux sources diffèrent. */
const START = '0x0000AAAA'
const SNIPER = '0x0000BBBB'
const HAMMER = '0x0000CCCC'

const LABELS = {
  [START]: { en: 'Assault Rifle', fr: 'Fusil d’assaut' },
  [SNIPER]: { en: 'Sniper', fr: 'Sniper' },
  [HAMMER]: { en: 'Hammer', fr: 'Marteau' },
}

const nameOf = (w: string) => `arme ${w}`

function life(slot: number, xuid: string, startFrame = 0, endFrame = 500) {
  return {
    slot, xuid, team: -1, startFrame, endFrame,
    points: [{ t: startFrame, x: 0, y: 0 }, { t: endFrame, x: 1, y: 1 }],
  }
}

function ground(over: Record<string, unknown> = {}) {
  return {
    t0: 100, t1: 200, t1max: 200, x: 0, y: 0, w: 'bbbb'.padStart(8, '0'),
    origin: 'dropped', dropper: 3, end: 'pickup', picker: 1, ...over,
  }
}

function docWith(items: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({
    frameIntervalMs: FRAME_MS,
    tracks: [life(1, 'x1'), life(2, 'x2')] as never,
    loadouts: [{ slot: 1, src: 'birth', k: [0], w: [START] }] as never,
    weaponLabels: LABELS as never,
    groundWeapons: items as never,
    ...over,
  } as never)
}

describe('groundPickupFeedEntries — ce qui fait une ligne', () => {
  it('une arme spéciale ramassée au sol : une ligne, à l’instant du ramassage, au nom du ramasseur', () => {
    const out = groundPickupFeedEntries(docWith([ground()]), nameOf, [])
    expect(out).toHaveLength(1)
    expect(out[0].replayMs).toBe(20_000)
    expect(out[0].event).toEqual({ family: 'pickups', xuid: 'x1', name: 'arme 0000bbbb', nature: 'classic', ground: true })
  })

  it('la nature de l’arme ramassée suit la clé canonique du titre : arme spéciale si elle est au registre', () => {
    const labels = { ...LABELS, [SNIPER]: { ...LABELS[SNIPER], key: 'hinf_s7_sniper' } }
    const out = groundPickupFeedEntries(docWith([ground()], { weaponLabels: labels }), nameOf, [])
    expect((out[0].event as { nature: string }).nature).toBe('power')
  })

  it('une arme de base (dotation de naissance) ramassée au sol n’apparaît pas', () => {
    const base = ground({ w: '0000aaaa', origin: 'spawned', dropper: -1 })
    expect(groundPickupFeedEntries(docWith([base]), nameOf, [])).toEqual([])
    expect(groundPickupFeedEntries(docWith([{ ...base, origin: 'dropped', dropper: 3 }]), nameOf, [])).toEqual([])
  })

  it('une arme de base et une arme spéciale côte à côte : seule la spéciale reste', () => {
    const out = groundPickupFeedEntries(
      docWith([ground({ w: '0000aaaa' }), ground({ w: '0000cccc', t1: 300, t1max: 300, picker: 2 })]),
      nameOf,
      [],
    )
    expect(out.map((e) => e.event)).toEqual([
      { family: 'pickups', xuid: 'x2', name: 'arme 0000cccc', nature: 'classic', ground: true },
    ])
  })

  it('écarte une arme jamais reprise ou dont la reprise n’est pas datée (fins `seen` et `open`)', () => {
    const out = groundPickupFeedEntries(
      docWith([ground({ end: 'seen', t1: 200, t1max: 400, picker: -1 }), ground({ end: 'open', picker: -1 })]),
      nameOf,
      [],
    )
    expect(out).toEqual([])
  })

  it('écarte un ramassage sans ramasseur : slot -1, slot sans vie à cet instant, vie sans xuid (bot)', () => {
    const docBots = docWith([ground({ picker: -1 }), ground({ picker: 9 }), ground({ picker: 4 })], {
      tracks: [life(1, 'x1'), { ...life(4, ''), bot: 'Bot [bot]' }, life(2, 'x2', 0, 150)],
    })
    expect(groundPickupFeedEntries(docBots, nameOf, [])).toEqual([])
    // Le slot 2 n'a plus de vie à l'image 200 : le ramasseur ne se devine pas.
    expect(groundPickupFeedEntries(docWith([ground({ picker: 2 })], { tracks: [life(2, 'x2', 0, 150)] }), nameOf, [])).toEqual([])
  })

  it('un slot réattribué entre deux vies : le ramasseur est celui de la vie qui couvre l’instant', () => {
    const doc = docWith([ground({ t1: 300, t1max: 300 })], {
      tracks: [life(1, 'x1', 0, 150), life(1, 'x9', 250, 500)],
    })
    expect(groundPickupFeedEntries(doc, nameOf, []).map((e) => (e.event as { xuid: string }).xuid)).toEqual(['x9'])
  })

  it('écarte une arme que le document ne nomme pas : rien n’est écrit sous un identifiant brut', () => {
    expect(groundPickupFeedEntries(docWith([ground({ w: '0000dddd' })]), nameOf, [])).toEqual([])
  })
})

describe('groundPickupFeedEntries — la base doit être établie', () => {
  it('sans dotation de naissance : aucune ligne (on ne peut pas dire qu’une arme n’est pas de base)', () => {
    expect(groundPickupFeedEntries(docWith([ground()], { loadouts: [] }), nameOf, [])).toEqual([])
    expect(
      groundPickupFeedEntries(
        docWith([ground()], { loadouts: [{ slot: 1, src: 'kf', k: [0], w: [START] }] as never }),
        nameOf,
        [],
      ),
    ).toEqual([])
  })

  it('départs aléatoires : pas de niveau de base, aucune ligne', () => {
    const doc = docWith([ground()], { weaponTiers: { randomStarts: true } as never })
    expect(groundPickupFeedEntries(doc, nameOf, [])).toEqual([])
  })

  it('départs non aléatoires déclarés : la règle s’applique comme sans le bloc', () => {
    const doc = docWith([ground()], { weaponTiers: { randomStarts: false } as never })
    expect(groundPickupFeedEntries(doc, nameOf, [])).toHaveLength(1)
  })
})

describe('groundPickupFeedEntries — pas de doublon avec une prise sur socle', () => {
  const pad = (ms: number, xuid: string, name: string) =>
    eventFeedEntry(`u-${xuid}-${ms}`, ms, { family: 'pickups', xuid, name, nature: 'classic' })

  it('écarte la ligne du même joueur, même arme, à moins de cinq images d’une prise de socle', () => {
    const out = groundPickupFeedEntries(docWith([ground()]), nameOf, [pad(20_300, 'x1', 'arme 0000bbbb')])
    expect(out).toEqual([])
  })

  it('garde la ligne si l’arme, le joueur ou l’instant diffèrent', () => {
    const doc = docWith([ground()])
    expect(groundPickupFeedEntries(doc, nameOf, [pad(20_000, 'x2', 'arme 0000bbbb')])).toHaveLength(1)
    expect(groundPickupFeedEntries(doc, nameOf, [pad(20_000, 'x1', 'Surbouclier')])).toHaveLength(1)
    expect(groundPickupFeedEntries(doc, nameOf, [pad(20_600, 'x1', 'arme 0000bbbb')])).toHaveLength(1)
  })

  it('les lignes rendues sont triées par instant', () => {
    const out = groundPickupFeedEntries(
      docWith([ground({ t1: 400, t1max: 400 }), ground({ w: '0000cccc', t1: 100, t1max: 100, picker: 2 })]),
      nameOf,
      [],
    )
    expect(out.map((e) => e.replayMs)).toEqual([10_000, 40_000])
  })

  it('un document sans arme au sol rend une liste vide', () => {
    expect(groundPickupFeedEntries(docWith([]), nameOf, [])).toEqual([])
  })
})
