import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from './replayNormalize'
import {
  ZONE_SOUND_STEMS,
  ZONE_TICK_PERIOD_MS,
  zoneSoundEvents,
} from './zoneSound'

/**
 * zoneSound.test.ts — les trois règles des sons d'état de zone, une par une.
 *
 * CE QUE CES TESTS ÉPINGLENT EN PRIORITÉ, ce sont les SILENCES. Un son d'objectif qui part au
 * mauvais moment ou du mauvais camp annonce un gain quand on perd une base : la règle du
 * chantier est que le rejeu se TAIT plutôt que de deviner. Trois silences sont donc testés
 * nommément — camp allié non résolu, rampe de jauge sans changement de propriétaire, et tics
 * en Roi de la colline.
 */

/** Un document minimal : seuls le pas de grille et les états de zone comptent ici. */
function doc(zoneStates: unknown[]): ReplayDocumentReady {
  return { frameIntervalMs: 100, zoneStates } as unknown as ReplayDocumentReady
}

/** Une zone tenue par `owner` de `t0` à `t1`. */
function span(t0: number, t1: number, owner: number | null, active = false) {
  return { t0, t1, owner, active }
}

describe('capture en cours — la jauge dit QUAND, le propriétaire d arrivée dit QUI', () => {
  it('une rampe suivie d une prise sonne au DÉBUT de la rampe, du camp qui prend', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [
          { t: 10, v: 0.1 },
          { t: 20, v: 0.4 },
          { t: 30, v: 0.9 },
        ],
      },
    ])
    const evs = zoneSoundEvents(d, 1)
    expect(evs).toEqual([{ ms: 1000, stem: ZONE_SOUND_STEMS.capturing.ally }])
  })

  it('le camp est celui du NOUVEAU propriétaire, pas de l ancien', () => {
    const d = doc([
      {
        spans: [span(0, 49, 1), span(50, 200, 0)],
        gauge: [
          { t: 10, v: 0.2 },
          { t: 30, v: 0.8 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 1000, stem: ZONE_SOUND_STEMS.capturing.enemy },
    ])
  })

  it('MUET quand la rampe retombe sans changer le propriétaire — c est le cas « contestée »', () => {
    const d = doc([
      {
        spans: [span(0, 200, 0)],
        gauge: [
          { t: 10, v: 0.2 },
          { t: 30, v: 0.8 },
          { t: 40, v: 0 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  it('MUET sans camp allié résolu : le rejeu ne devine jamais un camp', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [
          { t: 10, v: 0.1 },
          { t: 30, v: 0.9 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, null)).toEqual([])
  })

  it('un seul point de jauge n est pas une rampe', () => {
    const d = doc([{ spans: [span(0, 49, null), span(50, 200, 1)], gauge: [{ t: 10, v: 0.9 }] }])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })
})

describe('tic de score — un par seconde tant qu un camp tient TOUTES les zones', () => {
  it('trois zones au même camp : un tic par seconde, du bon côté', () => {
    const d = doc([
      { spans: [span(0, 30, 1)] },
      { spans: [span(0, 30, 1)] },
      { spans: [span(0, 30, 1)] },
    ])
    const evs = zoneSoundEvents(d, 1)
    // 0 à 30 frames à 100 ms = 3 s : les tics de 0, 1000, 2000 et 3000 ms.
    expect(evs.map((e) => e.ms)).toEqual([0, ZONE_TICK_PERIOD_MS, 2000, 3000])
    expect(new Set(evs.map((e) => e.stem))).toEqual(new Set([ZONE_SOUND_STEMS.tick.ally]))
  })

  it('une seule zone au camp adverse suffit à faire taire les tics', () => {
    const d = doc([
      { spans: [span(0, 30, 1)] },
      { spans: [span(0, 30, 0)] },
      { spans: [span(0, 30, 1)] },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  it('une zone NEUTRE fait taire les tics', () => {
    const d = doc([{ spans: [span(0, 30, 1)] }, { spans: [span(0, 30, null)] }])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  it('MUET en Roi de la colline — le marqueur `active` est la garde', () => {
    const d = doc([
      { spans: [span(0, 30, 1, true)] },
      { spans: [span(0, 30, 1, true)] },
    ])
    expect(zoneSoundEvents(d, 1).every((e) => e.stem === ZONE_SOUND_STEMS.newZone)).toBe(true)
  })

  it('MUET sur une zone unique : une zone n est pas une domination', () => {
    expect(zoneSoundEvents(doc([{ spans: [span(0, 30, 1)] }]), 1)).toEqual([])
  })
})

describe('nouvelle colline — chaque déplacement, jamais le premier intervalle', () => {
  it('deux collines successives ne sonnent qu une fois', () => {
    const d = doc([
      { spans: [span(0, 100, null, true)] },
      { spans: [span(101, 200, null, true)] },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 10100, stem: ZONE_SOUND_STEMS.newZone },
    ])
  })

  it('sonne même sans camp allié : le déplacement n affirme aucun camp', () => {
    const d = doc([
      { spans: [span(0, 100, null, true)] },
      { spans: [span(101, 200, null, true)] },
    ])
    expect(zoneSoundEvents(d, null)).toHaveLength(1)
  })

  it('une seule colline ne sonne pas — ce n est pas un déplacement', () => {
    expect(zoneSoundEvents(doc([{ spans: [span(0, 200, null, true)] }]), 1)).toEqual([])
  })
})

describe('cas dégénérés', () => {
  it('aucun état de zone : aucun son', () => {
    expect(zoneSoundEvents(doc([]), 1)).toEqual([])
  })

  it('jauge absente : aucune capture en cours, mais les tics restent possibles', () => {
    const d = doc([{ spans: [span(0, 10, 1)] }, { spans: [span(0, 10, 1)] }])
    const evs = zoneSoundEvents(d, 1)
    expect(evs.every((e) => e.stem === ZONE_SOUND_STEMS.tick.ally)).toBe(true)
    expect(evs.length).toBeGreaterThan(0)
  })
})
