/**
 * playerNowReadings.test.ts — LES LECTURES DU BLOC « À L'INSTANT », sans rendu.
 *
 * Elles prouvent que le bloc ne mesure rien : chaque grandeur est celle que la fiche de la colonne lit
 * (mêmes fonctions), et ce qui n'est pas lu est absent — jamais un zéro.
 */
import { describe, expect, it } from 'vitest'

import { buildPlayers } from '@/lib/replay/rosterLogic'
import { vitalityPresence } from '@/lib/replay/playerState'

import { testReplayDoc } from '../test/testDoc'
import { gaugePct, playerNowReadings, readingAgeTitle } from './playerNowReadings'

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
  tracks: [
    {
      slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100,
      points: [{ t: 0, x: 0, y: 0, sh: 1, hp: 1 }, { t: 40, x: 0, y: 0, sh: 0.55, hp: 0.5 }],
    },
    { slot: 513, team: -1, xuid: 'A', startFrame: 160, endFrame: 200, points: [{ t: 160, x: 0, y: 0, sh: 1, hp: 1 }] },
  ],
  weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' }, '0xCCCC': { fr: 'Épée', en: 'Sword', fx: 'melee' } },
  loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xCCCC'] }],
  inventory: [{ t: 0, slot: 512, d: 1, am: [{ mag: 20, res: 60 }, { gauge: 0.25 }], g: [] }],
})
const PLAYER = buildPlayers(DOC, [])[0]
const PRESENCE = vitalityPresence(DOC)

const lire = (frame: number) => playerNowReadings({ player: PLAYER, doc: DOC, frame, presence: PRESENCE, locale: 'fr' })

describe('playerNowReadings', () => {
  it('une fraction du plein devient un pour-cent entier borné', () => {
    expect(gaugePct(0.555)).toBe(56)
    expect(gaugePct(1.4)).toBe(100)
    expect(gaugePct(-0.2)).toBe(0)
    expect(gaugePct(0)).toBe(0)
  })

  it('la vitalité est la dernière mesure de la vie, avec son âge ; avant toute mesure, le plein d’apparition', () => {
    expect(lire(10).shield).toEqual({ pct: 100, age: 10 })
    expect(lire(50).shield).toEqual({ pct: 55, age: 10 })
    expect(lire(50).health).toEqual({ pct: 50, age: 10 })
  })

  it('l’arme en main d’abord : le sélecteur apparie l’arme et son emplacement d’inventaire', () => {
    const { weapons } = lire(50)
    // D = 1 : l'épée (emplacement 1, arme à charge) est en main ; le fusil est rangé avec SON compteur.
    expect(weapons.map((w) => [w.id, w.place])).toEqual([['0xCCCC', 'hand'], ['0xAAAA', 'stowed']])
    expect(weapons[0].ammo).toEqual({ kind: 'charge', pct: 75 })
    expect(weapons[1].ammo).toEqual({ kind: 'count', mag: 20, res: 60 })
  })

  it('le nom vient du catalogue du document ; sans vignette au catalogue, pas d’icône', () => {
    expect(lire(50).weapons.map((w) => w.name)).toEqual(['Épée', 'Fusil'])
    expect(lire(50).weapons.every((w) => w.icon === null)).toBe(true)
  })

  it('sans sélecteur lu, aucune arme n’est dite en main et aucune munition n’est affirmée', () => {
    const sansSelecteur = testReplayDoc({
      frameIntervalMs: 100,
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
      tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] }],
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, am: [{ mag: 1 }, { mag: 2 }], g: [] }],
    })
    const r = playerNowReadings({
      player: buildPlayers(sansSelecteur, [])[0], doc: sansSelecteur, frame: 10, presence: vitalityPresence(sansSelecteur), locale: 'fr',
    })
    expect(r.weapons.map((w) => [w.place, w.ammo])).toEqual([[null, null], [null, null]])
  })

  it('des compteurs de grenade non lus (tableau vide) ne font aucune rangée', () => {
    expect(lire(10).grenades).toEqual([])
  })

  it('mort à l’instant lu : éliminé, sans jauge ni arme ; avant toute vie, pas éliminé', () => {
    const mort = lire(120)
    expect(mort.eliminated).toBe(true)
    expect([mort.shield, mort.health, mort.weapons, mort.ability]).toEqual([null, null, [], null])
    expect(mort.state.respawnFrame).toBe(160)
  })

  it('un document sans vitalité publiée ne donne aucune jauge', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
      tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] }],
    })
    const r = playerNowReadings({
      player: buildPlayers(doc, [])[0], doc, frame: 10, presence: vitalityPresence(doc), locale: 'fr',
    })
    expect([r.shield, r.health]).toEqual([null, null])
  })

  it('l’âge d’une lecture se dit, et une lecture à venir se dit comme telle', () => {
    expect(readingAgeTitle('lu il y a', 'lu dans', 85, DOC)).toBe('lu il y a 8.5 s')
    expect(readingAgeTitle('lu il y a', 'lu dans', -30, DOC)).toBe('lu dans 3.0 s')
  })
})
