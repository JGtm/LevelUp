/**
 * Tests — pickupFeedLogic : « Armes et bonus », tiré de `doc.padPickups[]`.
 *
 * Ce qu'ils protègent : seules les prises que le serveur a DATÉES (`t`) ET NOMMÉES (`xuid`)
 * font une ligne — une prise sans date ou sans ramasseur est un fait que le film ne donne pas,
 * jamais une ligne devinée. Le nom vient du résolveur de socle de l'appelant.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { pickupFeedEntries } from './pickupFeedLogic'

const FRAME_MS = 100

const DEFAULT_PADS = [
  { x: 0, y: 0, weapon: '0x0000AAAA', spawns: [], presence: [] },
  { x: 1, y: 1, weapon: 'powerup_overshield', spawns: [], presence: [] },
]

function docWith(padPickups: unknown[], weaponPads: unknown[] = DEFAULT_PADS) {
  return testReplayDoc({
    frameIntervalMs: FRAME_MS,
    padPickups: padPickups as never,
    weaponPads: weaponPads as never,
  } as never)
}

const nameOf = (w: string) => (w === 'powerup_overshield' ? 'Surbouclier' : `arme ${w}`)

describe('pickupFeedEntries', () => {
  it('une ligne par prise datée et nommée, au nom du socle', () => {
    const out = pickupFeedEntries(
      docWith([
        { pad: 1, t: 200, tLow: 150, tHigh: 250, xuid: 'x2' },
        { pad: 0, t: 100, tLow: 50, tHigh: 150, xuid: 'x1' },
      ]),
      nameOf,
    )
    expect(out.map((e) => e.replayMs)).toEqual([10_000, 20_000])
    expect(out.map((e) => e.event)).toEqual([
      { family: 'pickups', xuid: 'x1', name: 'arme 0x0000AAAA', nature: 'classic' },
      { family: 'pickups', xuid: 'x2', name: 'Surbouclier', nature: 'powerup' },
    ])
  })

  it('la nature suit la règle des socles : arme spéciale par la clé canonique du titre, râtelier sinon', () => {
    const doc = testReplayDoc({
      frameIntervalMs: FRAME_MS,
      padPickups: [
        { pad: 0, t: 100, tLow: 50, tHigh: 150, xuid: 'x1' },
        { pad: 1, t: 200, tLow: 150, tHigh: 250, xuid: 'x2' },
        { pad: 2, t: 300, tLow: 250, tHigh: 350, xuid: 'x3' },
      ],
      weaponPads: [
        { x: 0, y: 0, weapon: '0x0000AAAA', spawns: [], presence: [] },
        { x: 1, y: 1, weapon: '0x0000BBBB', spawns: [], presence: [] },
        { x: 2, y: 2, weapon: 'powerup_camo', spawns: [], presence: [] },
      ],
      weaponLabels: {
        '0x0000AAAA': { fr: 'Sniper', en: 'Sniper', key: 'hinf_s7_sniper' },
        '0x0000BBBB': { fr: 'Commando', en: 'Commando', key: 'hinf_vk78_commando' },
      },
    } as never)
    expect(pickupFeedEntries(doc, nameOf).map((e) => (e.event as { nature: string }).nature)).toEqual([
      'power',
      'classic',
      'powerup',
    ])
  })

  it('écarte une prise sans date, sans ramasseur, ou sur un socle inconnu', () => {
    const out = pickupFeedEntries(
      docWith([
        { pad: 0, tLow: 50, tHigh: 150, xuid: 'x1' },
        { pad: 0, t: 100, tLow: 50, tHigh: 150, xuid: null },
        { pad: 9, t: 100, tLow: 50, tHigh: 150, xuid: 'x1' },
      ]),
      nameOf,
    )
    expect(out).toEqual([])
  })

  it('un document sans prise rend une liste vide', () => {
    expect(pickupFeedEntries(docWith([]), nameOf)).toEqual([])
  })
})
