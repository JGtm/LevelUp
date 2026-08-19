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
 *
 * DEPUIS LE 2026-08-19, ce fichier verrouille aussi LE CROISEMENT avec le catalogue de la
 * carte (`crossedWeaponPads`) : la position vient du fichier de carte, la présence reste
 * celle du match, un socle du film que le catalogue ignore reste dessiné, un emplacement
 * que le film ne confirme pas n'apparaît jamais — et le tout est posé DÈS L'IMAGE 0.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from './i18n'
import { worldToCanvas } from './replayLogic'
import type { ReplayDocumentReady, ReplayWeaponPadReady } from './replayNormalize'
import { count, recordingContext } from './test/recordingContext'
import {
  crossedWeaponPads,
  padIconRefFor,
  padNameFor,
  padScaleFor,
} from './useReplayWeaponPads'
import { drawWeaponPadsLayer, padStateAt, type PadStyle } from './weaponPadsLayer'

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

// --- LE CROISEMENT AVEC LE CATALOGUE DE LA CARTE (2026-08-19) ---------------------------

/** Un socle du match, tel que l'artefact le publie une fois normalisé. */
function socle(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return {
    x: -9.74,
    y: 0,
    z: 22.4,
    weapon: SNIPER,
    spawns: [0],
    presence: [{ t0: 0, tLow: 100, tHigh: 120 }],
    ...over,
  }
}

/** Le calque croisé tel que la reponse le porte : positions du fichier de carte. */
function croisement(
  pads: { x: number; y: number; z?: number; pad: number }[],
  catalogN = pads.length,
): ReplayDocumentReady['mapWeaponPads'] {
  return { pads, catalogN }
}

/** Le cadrage des tests de calque : 10 m de cote sur 100 px. */
const VUE = { bounds: { minX: -20, minY: -20, maxX: 20, maxY: 20 }, width: 200, height: 200, pad: 0 }

const STYLE: PadStyle = {
  ink: 'encre',
  fill: 'remplissage',
  outline: 'contour',
  iconOf: () => null,
  scaleOf: () => 'power',
  countdownLabel: (s) => `${Math.ceil(s)} s`,
}

describe('crossedWeaponPads — la position vient du catalogue, la presence reste celle du match', () => {
  it('un socle confirme est POSE a la position du fichier de carte', () => {
    const pads = [socle()]
    const out = crossedWeaponPads(pads, croisement([{ x: -9.738, y: -0.003, z: 22.403, pad: 0 }]))
    expect(out).toHaveLength(1)
    expect(out[0].x).toBe(-9.738)
    expect(out[0].y).toBe(-0.003)
    expect(out[0].z).toBe(22.403)
    // ET RIEN D'AUTRE NE BOUGE : la presence est la mesure du match, pas celle du fichier.
    expect(out[0].presence).toBe(pads[0].presence)
    expect(out[0].spawns).toBe(pads[0].spawns)
    expect(out[0].weapon).toBe(pads[0].weapon)
    expect(padStateAt(out[0], 0)).toBe(padStateAt(pads[0], 0))
    expect(padStateAt(out[0], 110)).toBe(padStateAt(pads[0], 110))
  })

  it('un socle du film que le catalogue ignore reste dessine, a SA position', () => {
    const pads = [socle(), socle({ x: 100, y: 100, weapon: INCONNUE })]
    const out = crossedWeaponPads(pads, croisement([{ x: -9.738, y: -0.003, z: 22.403, pad: 0 }], 17))
    expect(out).toHaveLength(2)
    expect(out[1].x).toBe(100)
    expect(out[1].y).toBe(100)
    expect(out[1]).toBe(pads[1])
  })

  it('un emplacement NON confirme n’arrive jamais : rien ne le fabrique cote client', () => {
    const pads = [socle()]
    // Le serveur n'envoie que des emplacements confirmes ; meme une reference hors bornes
    // (reponse abimee, artefact desynchronise) ne doit RIEN ajouter.
    const out = crossedWeaponPads(pads, croisement([{ x: 0.257, y: 0, z: 21.36, pad: 7 }], 17))
    expect(out).toHaveLength(1)
    expect(out[0]).toBe(pads[0])
  })

  it('deux emplacements ne deplacent pas le meme socle deux fois — la premiere citation gagne', () => {
    const out = crossedWeaponPads(
      [socle()],
      croisement([
        { x: 1, y: 1, z: 1, pad: 0 },
        { x: 2, y: 2, z: 2, pad: 0 },
      ]),
    )
    expect(out).toHaveLength(1)
    expect(out[0].x).toBe(1)
  })

  it('sans croisement, la liste du film TELLE QUELLE — le repli est le comportement d’avant', () => {
    const pads = [socle(), socle({ x: 5.16, y: 0 })]
    expect(crossedWeaponPads(pads, undefined)).toBe(pads)
    expect(crossedWeaponPads(pads, croisement([]))).toBe(pads)
  })

  it('un z absent au catalogue laisse celui du match, jamais zero', () => {
    const out = crossedWeaponPads([socle({ z: 22.4 })], croisement([{ x: 1, y: 2, pad: 0 }]))
    expect(out[0].z).toBe(22.4)
  })
})

describe('DES LA PREMIERE IMAGE — ce que le calque pose a l’image 0', () => {
  it('tous les socles croises sont poses des l’image 0, aux positions du CATALOGUE', () => {
    const pads = [socle(), socle({ x: 5.16, y: 0, z: 26.5 })]
    const croises = crossedWeaponPads(
      pads,
      croisement(
        [
          { x: -9.738, y: -0.003, z: 22.403, pad: 0 },
          { x: 5.16, y: -0.003, z: 26.501, pad: 1 },
        ],
        17,
      ),
    )
    const { ops, ctx } = recordingContext()
    drawWeaponPadsLayer(ctx, croises, VUE, { frame: 0, frameMs: 100, k: 1 }, STYLE)
    // Un point par socle a l’image 0 (l’arc du point ; le glyphe neutre en pose un second
    // quand l’etat n’est pas vide — d’ou le compte par socle et non le compte brut).
    const arcs = ops.filter((o) => o.op === 'arc').map((o) => o.args as number[])
    expect(arcs.length).toBeGreaterThanOrEqual(2)
    for (const [i, spot] of [
      { x: -9.738, y: -0.003 },
      { x: 5.16, y: -0.003 },
    ].entries()) {
      const c = worldToCanvas(spot, VUE.bounds, VUE.width, VUE.height, VUE.pad)
      const trouve = arcs.some(([px, py]) => Math.abs(px - c.x) < 0.01 && Math.abs(py - c.y) < 0.01)
      expect(trouve, `socle ${i} absent de l’image 0 a la position du catalogue`).toBe(true)
    }
  })

  it('AUCUN socle a l’image 0 quand le match n’en publie aucun — le temoin Super Fiesta', () => {
    const { ops, ctx } = recordingContext()
    // Cliffhanger porte dix-sept emplacements au fichier ; en Super Fiesta le film n’en
    // sert aucun, donc la reponse ne porte pas de croisement et il n’y a RIEN a dessiner.
    drawWeaponPadsLayer(ctx, crossedWeaponPads([], undefined), VUE, { frame: 0, frameMs: 100, k: 1 }, STYLE)
    expect(count(ops, 'arc')).toBe(0)
    expect(count(ops, 'drawImage')).toBe(0)
  })
})
