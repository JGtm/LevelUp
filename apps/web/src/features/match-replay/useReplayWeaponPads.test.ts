/**
 * Tests — LA RÉSOLUTION D'UN SOCLE : taille, nom, vignette, pour les DEUX vocabulaires.
 *
 * CE QUE CE FICHIER VERROUILLE, et chaque point est le défaut mesuré le 2026-08-19 :
 *  - un socle de POWER-UP est GRAND. Sa clé (`powerup_overshield`) n'est pas dans
 *    `weaponLabels`, table d'ARMES : la chercher là rendait `undefined`, donc `classic` ;
 *  - il porte un NOM bilingue, jamais sa clé brute — l'infobulle affichait
 *    « powerup_overshield » à l'écran ;
 *  - il porte une VIGNETTE : le masque de HUD livré, title-scopé et teintable ;
 *  - un socle d'ARME ne bouge pas d'un cheveu : mêmes taille, nom et image qu'avant.
 *
 * LES TROIS RÉSOLUTIONS SONT PURES et se testent sans React : le hook ne fait que les
 * emballer dans des `useCallback` (c'est ce que le canvas consomme).
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from './i18n'
import { padIconRefFor, padNameFor, padScaleFor } from './useReplayWeaponPads'
import type { ReplayDocumentReady } from './replayNormalize'

/** Le S7 Sniper tel que le document le sert : clé canonique, libellé bilingue, masque. */
const SNIPER = '0x0A1992BC'
/** Une famille d'arme que le titre ne catalogue pas : ni clé, ni libellé, ni image. */
const INCONNUE = '0xD7915565'
const POWERUP = 'powerup_overshield'
const CAMO = 'powerup_camo'

const LABELS: ReplayDocumentReady['weaponLabels'] = {
  [SNIPER]: {
    en: 'S7 Sniper',
    fr: 'S7 Sniper',
    key: 'hinf_s7_sniper',
    img: '/static/weapons-assets/halo_infinite/jeu/contour-05.png',
    tinted: true,
  },
  '0x2B1824D5': { en: 'BR75', fr: 'BR75', key: 'hinf_br75', img: '/x.png', tinted: true },
}

describe('padScaleFor — la taille, quel que soit le vocabulaire de la clé', () => {
  it('un POWER-UP de socle est GRAND, sans passer par le catalogue d’armes', () => {
    expect(padScaleFor(POWERUP, LABELS)).toBe('power')
    expect(padScaleFor(CAMO, LABELS)).toBe('power')
    // Et il l'est même quand le document ne sert AUCUN catalogue (film sans arme nommée).
    expect(padScaleFor(POWERUP, undefined)).toBe('power')
  })

  it('un socle d’ARME garde exactement la règle d’avant', () => {
    expect(padScaleFor(SNIPER, LABELS)).toBe('power')
    expect(padScaleFor('0x2B1824D5', LABELS)).toBe('classic')
    expect(padScaleFor(INCONNUE, LABELS)).toBe('classic')
  })
})

describe('padNameFor — ce que l’infobulle écrit', () => {
  it('un POWER-UP est nommé dans les deux langues, JAMAIS par sa clé brute', () => {
    for (const locale of ['fr', 'en'] as const) {
      for (const key of [POWERUP, CAMO]) {
        const nom = padNameFor(key, LABELS, REPLAY_TEXT[locale], locale)
        expect(nom, `${key} sans nom en ${locale}`).toBeTruthy()
        expect(nom, `${key} affiche sa clé brute en ${locale}`).not.toContain('powerup_')
      }
    }
    expect(padNameFor(POWERUP, LABELS, REPLAY_TEXT.fr, 'fr')).toBe('Surbouclier')
    expect(padNameFor(POWERUP, LABELS, REPLAY_TEXT.en, 'en')).toBe('Overshield')
    expect(padNameFor(CAMO, LABELS, REPLAY_TEXT.fr, 'fr')).toBe('Camouflage actif')
    expect(padNameFor(CAMO, LABELS, REPLAY_TEXT.en, 'en')).toBe('Active camouflage')
  })

  it('une ARME garde le libellé du document', () => {
    expect(padNameFor(SNIPER, LABELS, REPLAY_TEXT.fr, 'fr')).toBe('S7 Sniper')
  })

  it('une arme hors catalogue garde son identifiant — c’est VOULU, et ça n’a pas bougé', () => {
    expect(padNameFor(INCONNUE, LABELS, REPLAY_TEXT.fr, 'fr')).toBe(INCONNUE)
  })
})

describe('padIconRefFor — quelle image, et d’où elle vient', () => {
  it('un POWER-UP prend le masque de HUD livré, title-scopé et teintable', () => {
    expect(padIconRefFor(POWERUP, LABELS, 'halo_infinite')).toEqual({
      url: '/static/weapons-assets/halo_infinite/hud/Overshield.png',
      tinted: true,
    })
    expect(padIconRefFor(CAMO, LABELS, 'halo_infinite')).toEqual({
      url: '/static/weapons-assets/halo_infinite/hud/ActiveCamouflage.png',
      tinted: true,
    })
  })

  it('le slug vient de l’appelant — aucun titre écrit en dur', () => {
    expect(padIconRefFor(POWERUP, LABELS, 'un_autre_titre')?.url).toContain('/un_autre_titre/')
  })

  it('une ARME garde la vignette du document', () => {
    expect(padIconRefFor(SNIPER, LABELS, 'halo_infinite')).toEqual({
      url: '/static/weapons-assets/halo_infinite/jeu/contour-05.png',
      tinted: true,
    })
  })

  it('sans image, RIEN — le calque posera son glyphe neutre, jamais l’icône d’un voisin', () => {
    expect(padIconRefFor(INCONNUE, LABELS, 'halo_infinite')).toBeNull()
    expect(padIconRefFor(SNIPER, undefined, 'halo_infinite')).toBeNull()
  })
})
