/**
 * Tests — groundWeaponAmmo : LA JOINTURE DE REPLI, ET TOUT CE QU'ELLE REFUSE DE DIRE.
 *
 * Le lot 6.6 a MESURÉ que les munitions ne sont PAS sur l'objet
 * (`.ai/V7.5/RAPPORT_MUNITIONS_OBJET_2026-09-10.md`). Il ne reste que la dernière lecture
 * d'inventaire du lâcheur AVANT le lâcher — une lecture d'image-clé, en retard de 9 s en
 * médiane. Ce fichier verrouille les cinq refus qui empêchent cette lecture de mentir :
 *
 *  1. rien pour une arme `spawned` : elle n'a pas de lâcheur mesuré, donc pas d'inventaire ;
 *  2. rien quand l'arme lâchée n'est pas à l'emplacement lu : `am[i]` est indexé par
 *     `loadouts[].w`, un décalage servirait les munitions de l'AUTRE arme ;
 *  3. rien quand l'emplacement n'a pas de chargeur (arme à jauge) : publier 0 affirmerait
 *     « chargeur vide » ;
 *  4. rien quand la lecture est PLUS VIEILLE que le seuil, et rien quand elle est À VENIR ;
 *  5. la jointure de clé se fait sur la FORME NORMALISÉE (`weaponLabelKeyOf`) : l'artefact
 *     écrit `groundWeapons[].w` en minuscules nues et `loadouts[].w` en `0x` majuscules.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayGroundWeapon } from '@/lib/api/types'

import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { testReplayDoc } from '../test/testDoc'
import { GROUND_WEAPON_AMMO_MAX_AGE_MS, groundWeaponAmmoAt } from './groundWeaponAmmo'

/** Une vie couvrant [start, end] sur un slot — même patron que inventoryReading.test.ts. */
function track(slot: number, start: number, end: number): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid: 'A',
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

/** L'arme lâchée du cas nominal : famille `0a1992bc`, lâchée par le slot 512 à l'image 40. */
function lachee(over: Partial<ReplayGroundWeapon> = {}): ReplayGroundWeapon {
  return {
    t0: 40,
    t1: 80,
    t1max: 80,
    x: 1,
    y: 1,
    w: '0a1992bc',
    origin: 'dropped',
    dropper: 512,
    end: 'open',
    picker: -1,
    ...over,
  }
}

/** Le document nominal : une image de 1 000 ms, une lecture à l'image 30, deux emplacements. */
function docNominal(over: Parameters<typeof testReplayDoc>[0] = {}) {
  return testReplayDoc({
    frameIntervalMs: 1000,
    tracks: [track(512, 0, 100)],
    loadouts: [{ t: 30, slot: 512, w: ['0xB533957E', '0x0A1992BC'] }],
    inventory: [{ t: 30, slot: 512, am: [{ mag: 30, res: 90 }, { mag: 7, res: 14 }] }],
    ...over,
  })
}

describe('groundWeaponAmmoAt — la lecture retenue', () => {
  it('rend le chargeur de L’EMPLACEMENT de l’arme lâchée, et l’ÂGE de la lecture', () => {
    const r = groundWeaponAmmoAt(docNominal(), lachee())
    // L'emplacement 1 porte `0x0A1992BC` : 7 balles, pas les 30 de l'emplacement 0.
    expect(r).toEqual({ mag: 7, res: 14, ageMs: 10_000 })
  })

  it('joint sur la forme NORMALISÉE de la clé, jamais sur l’écriture brute', () => {
    // Les deux écritures de l'artefact désignent la même famille ; une jointure exacte
    // rendrait null (c'est le défaut de jointure du lot 6.5, mesuré sur les 64 artefacts).
    expect(groundWeaponAmmoAt(docNominal(), lachee({ w: '0A1992BC' }))?.mag).toBe(7)
  })

  it('rend null pour une arme `spawned` — aucun lâcheur, donc aucun inventaire', () => {
    expect(groundWeaponAmmoAt(docNominal(), lachee({ origin: 'spawned', dropper: -1 }))).toBeNull()
  })

  it('rend null quand l’arme lâchée n’est pas à l’emplacement lu', () => {
    expect(groundWeaponAmmoAt(docNominal(), lachee({ w: 'deadbeef' }))).toBeNull()
  })

  it('rend null quand l’emplacement n’a pas de chargeur (arme à jauge)', () => {
    const d = docNominal({
      inventory: [{ t: 30, slot: 512, am: [{ mag: 30 }, { gauge: 0.5 }] }],
    })
    expect(groundWeaponAmmoAt(d, lachee())).toBeNull()
  })

  it('rend null quand la lecture est plus VIEILLE que le seuil', () => {
    // Le seuil est en millisecondes ; à 1 000 ms l'image, il vaut autant de frames.
    const trop = GROUND_WEAPON_AMMO_MAX_AGE_MS / 1000 + 1
    const d = docNominal({
      tracks: [track(512, 0, 200)],
      loadouts: [{ t: 0, slot: 512, w: ['0xB533957E', '0x0A1992BC'] }],
      inventory: [{ t: 0, slot: 512, am: [{ mag: 30 }, { mag: 7 }] }],
    })
    expect(groundWeaponAmmoAt(d, lachee({ t0: trop }))).toBeNull()
    // MUTATION : la borne est un `>` sur l'âge. Exactement AU seuil, la lecture passe encore.
    expect(groundWeaponAmmoAt(d, lachee({ t0: trop - 1 }))?.mag).toBe(7)
  })

  it('rend null quand la seule lecture est À VENIR — on n’affirme pas l’avenir au passé', () => {
    const d = docNominal({
      loadouts: [{ t: 60, slot: 512, w: ['0x0A1992BC'] }],
      inventory: [{ t: 60, slot: 512, am: [{ mag: 7 }] }],
    })
    expect(groundWeaponAmmoAt(d, lachee())).toBeNull()
  })

  it('rend null sans lecture d’inventaire du tout', () => {
    expect(groundWeaponAmmoAt(docNominal({ inventory: [] }), lachee())).toBeNull()
  })

  it('rend null quand aucun loadout ne date du MÊME instant que la lecture', () => {
    // `am[i]` est indexé « dans l'ordre de Loadout.W » du MÊME relevé : un loadout d'un autre
    // instant peut porter d'autres armes dans un autre ordre.
    expect(groundWeaponAmmoAt(docNominal({ loadouts: [] }), lachee())).toBeNull()
  })
})
