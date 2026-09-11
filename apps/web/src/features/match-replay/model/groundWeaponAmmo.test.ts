/**
 * Tests — groundWeaponAmmo : LE CHOIX DE LA SOURCE, ET TOUT CE QUE LE REPLI REFUSE DE DIRE.
 *
 * DEUX SOURCES depuis le lot 6.10 (2026-09-11). L'EXACTE, `groundWeapons[].ammo`, est lue sur
 * l'objet à l'instant du lâcher et passe TOUJOURS en premier ; le repli DATÉ du lot 6.6 — la
 * dernière lecture d'inventaire du lâcheur, en retard de 9 s en médiane — ne sert que lorsque
 * l'exacte manque, ce qui est le cas majoritaire.
 *
 * Ce fichier verrouille les deux règles de choix, puis les cinq refus qui empêchent le repli
 * de mentir :
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
import { REPLAY_TEXT } from '../i18n/i18n'
import {
  GROUND_WEAPON_AMMO_MAX_AGE_MS,
  groundWeaponAmmoAt,
  groundWeaponAmmoLine,
} from './groundWeaponAmmo'

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
    expect(r).toEqual({ kind: 'dated', mag: 7, res: 14, ageMs: 10_000 })
  })

  it('préfère les munitions EXACTES de l’objet à la lecture d’inventaire du lâcheur', () => {
    // Le document porte les DEUX : l'inventaire dit 7/14 à l'image 30, l'objet dit 3/14 au
    // lâcher. C'est l'objet qui gagne — et la lecture rendue n'a pas d'âge, parce qu'elle
    // n'est pas en retard.
    const r = groundWeaponAmmoAt(docNominal(), lachee({ ammo: { mag: 3, res: 14 } }))
    expect(r).toEqual({ kind: 'exact', mag: 3, res: 14 })
  })

  it('rend les munitions EXACTES d’une arme `spawned`, que le repli ne pouvait pas servir', () => {
    const spawned = lachee({ origin: 'spawned', dropper: -1, ammo: { mag: 36, res: 108 } })
    expect(groundWeaponAmmoAt(docNominal(), spawned)).toEqual({ kind: 'exact', mag: 36, res: 108 })
  })

  it('n’invente rien : un chargeur EXACT à zéro se dit, et se dit comme exact', () => {
    // Une arme lâchée vide est une information — pas une absence de lecture.
    expect(groundWeaponAmmoAt(docNominal(), lachee({ ammo: { mag: 0, res: 0 } }))).toEqual({
      kind: 'exact',
      mag: 0,
      res: 0,
    })
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

describe('groundWeaponAmmoLine — la phrase dit la NATURE de la lecture', () => {
  it('écrit la lecture EXACTE sans « ≈ » et sans âge, en FR comme en EN', () => {
    const exacte = { kind: 'exact', mag: 12, res: 24 } as const
    const fr = groundWeaponAmmoLine(REPLAY_TEXT.fr, exacte)
    const en = groundWeaponAmmoLine(REPLAY_TEXT.en, exacte)
    for (const phrase of [fr, en]) {
      expect(phrase).toContain('12')
      expect(phrase).toContain('24')
      // LE POINT DU TEST : ni approximation, ni datation — la valeur est mesurée au lâcher.
      expect(phrase).not.toContain('≈')
      expect(phrase).not.toMatch(/\d\s*s\b/)
    }
    expect(fr).not.toEqual(en)
  })

  it('écrit la lecture DATÉE avec son « ≈ » et son âge, en FR comme en EN', () => {
    const datee = { kind: 'dated', mag: 12, res: 24, ageMs: 9_000 } as const
    for (const t of [REPLAY_TEXT.fr, REPLAY_TEXT.en]) {
      const phrase = groundWeaponAmmoLine(t, datee)
      expect(phrase).toContain('≈')
      expect(phrase).toContain('9.0')
    }
  })

  it('la forme sans réserve reste réservée à la lecture DATÉE', () => {
    // L'exacte porte toujours ses deux champs ; seule la datée peut manquer de réserve.
    const phrase = groundWeaponAmmoLine(REPLAY_TEXT.fr, {
      kind: 'dated',
      mag: 12,
      res: null,
      ageMs: 9_000,
    })
    expect(phrase).toContain('≈')
    expect(phrase).not.toContain('réserve')
  })
})
