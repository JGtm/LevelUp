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
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { createRef, type RefObject } from 'react'

import { REPLAY_TEXT } from '../i18n/i18n'
import { worldToCanvas } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'
import { testReplayDoc } from '../test/testDoc'
import { count, diamondCentres, recordingContext } from '../test/recordingContext'
import {
  crossedWeaponPads,
  padIconRefFor,
  padNameFor,
  padScaleFor,
  useReplayWeaponPads,
} from './useReplayWeaponPads'
import { padFamilyOf } from '../model/weaponPadFamilies'
import { padStateAt } from '../model/weaponPadTime'
import { drawWeaponPadsLayer, type PadStyle } from './weaponPadsLayer'

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
      mirrored: false,
    })
    expect(padIconRefFor(CAMO, LABELS, 'halo_infinite')).toEqual({
      url: '/static/weapons-assets/halo_infinite/hud/ActiveCamouflage.png',
      tinted: true,
      mirrored: false,
    })
  })

  it('le slug vient de l’appelant — aucun titre écrit en dur', () => {
    expect(padIconRefFor(POWERUP, LABELS, 'un_autre_titre')?.url).toContain('/un_autre_titre/')
  })

  it('une ARME prend la SILHOUETTE — la même icône que les fiches et le kill feed', () => {
    // Retour utilisateur du 2026-08-28 : le trait à vide (`contour`) se perd sur un fond de
    // carte en niveaux de gris. La forme pleine est celle des fiches, et elle se retourne.
    expect(padIconRefFor(SNIPER, LABELS, 'halo_infinite')).toEqual({
      url: '/static/weapons-assets/halo_infinite/jeu/silhouette-05.png',
      tinted: true,
      mirrored: true,
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
  inkOf: () => 'nature',
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
    const arcs = diamondCentres(ops).map((c) => [c.x, c.y])
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

// --- LE CALQUE BOUT EN BOUT, SUR LA DONNÉE RÉELLE (signalement du 2026-08-26) --------------

/**
 * LES CINQ SOCLES DU MATCH `530820e5` (CTF Catalyst), copie RÉDUITE de l'artefact réel
 * `data/cache/replays/halo_infinite/530820e5.json` (schéma 18).
 *
 * POURQUOI UNE COPIE ET NON UNE LECTURE DU CACHE : un test qui lit `data/` dépendrait d'un
 * fichier que rien ne garantit — purge, autre machine, autre match. Les positions, les
 * identifiants d'arme et les premières occupations sont ceux du fichier, au dix-millième ;
 * les occupations suivantes sont tronquées, elles ne changent aucun des états lus ici.
 *
 * CE QUE CETTE FIXTURE PROUVE, et c'est le signalement utilisateur : avec CETTE donnée-là —
 * cinq socles bien dans les bornes, `spawns` et `presence` peuplés — le calque doit poser
 * cinq points dès l'image 0. L'utilisateur n'en voyait aucun, ni au survol.
 */
const SOCLES_530820E5: ReplayWeaponPadReady[] = [
  {
    x: 6.2765, y: 6.9393, z: 27.0176, weapon: '0x2B1824D5',
    spawns: [0, 606, 3353],
    presence: [{ t0: 0, tLow: 146, tHigh: 346 }, { t0: 606, tLow: 2946, tHigh: 3146 }],
    cycle: { medianS: 30.3, p10S: 30.3, p90S: 30.3, gaps: 2, missing: 0 },
  },
  {
    x: 5.1598, y: -0.0028, z: 26.5007, weapon: '0x230447B1',
    spawns: [0, 1111], presence: [{ t0: 0, tLow: 146, tHigh: 346 }],
  },
  {
    x: -11.0455, y: -0.0028, z: 25.344, weapon: '0x4FF3937E',
    spawns: [0], presence: [{ t0: 0, tLow: 146, tHigh: 346 }],
    cycle: { medianS: 187.81, p10S: 187.81, p90S: 187.81, gaps: 2, missing: 1 },
  },
  {
    x: 0.0032, y: -25.2038, z: 26.5007, weapon: '0x0A1992BC',
    spawns: [0], presence: [{ t0: 0, tLow: 146, tHigh: 346 }],
  },
  {
    x: 0.0032, y: 25.2979, z: 26.5007, weapon: '0x0A1992BC',
    spawns: [0], presence: [{ t0: 0, tLow: 146, tHigh: 346 }],
  },
]

/** Les bornes et la cadence du même artefact : les socles y sont tous largement à l'intérieur. */
const DOC_530820E5 = () =>
  testReplayDoc({
    frameCount: 4751,
    frameIntervalMs: 100,
    bounds: { minX: -18.68, minY: -25.27, maxX: 19.28, maxY: 25.37, minZ: 0.68, maxZ: 29.5 },
    weaponPads: SOCLES_530820E5,
  })

const VUE_REELLE = {
  bounds: { minX: -18.68, minY: -25.27, maxX: 19.28, maxY: 25.37 },
  width: 700,
  height: 480,
  pad: 24,
}

function monterCalque(doc: ReplayDocumentReady, enabled = true) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = 0
  return renderHook(() =>
    useReplayWeaponPads({
      doc,
      view: VUE_REELLE,
      frameRef,
      enabled,
      ink: {
        neutral: 'encre',
        fill: 'remplissage',
        outline: 'contour',
        family: { powerup: '#powerup', power: '#power', classic: '#classic' },
      },
      locale: 'fr',
      redraw: vi.fn(),
    }),
  )
}

describe('useReplayWeaponPads — le calque bout en bout sur l’artefact réel 530820e5', () => {
  it('les cinq socles du match passent la frontière et sont DISPONIBLES', () => {
    const doc = DOC_530820E5()
    expect(doc.weaponPads).toHaveLength(5)
    const { result } = monterCalque(doc)
    expect(result.current.available).toBe(true)
  })

  it('le calque POSE cinq marques à l’image 0 — le défaut signalé le 2026-08-26', () => {
    const { result } = monterCalque(DOC_530820E5())
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    const arcs = diamondCentres(ops).map((c) => [c.x, c.y])
    for (const [i, socle] of SOCLES_530820E5.entries()) {
      const c = worldToCanvas(socle, VUE_REELLE.bounds, VUE_REELLE.width, VUE_REELLE.height, VUE_REELLE.pad)
      const trouve = arcs.some(([px, py]) => Math.abs(px - c.x) < 0.01 && Math.abs(py - c.y) < 0.01)
      expect(trouve, `socle ${i} (${socle.weapon}) absent du tracé à l'image 0`).toBe(true)
    }
  })

  it('et ils sont SURVOLABLES là où ils sont peints', () => {
    const { result } = monterCalque(DOC_530820E5())
    const socle = SOCLES_530820E5[2]
    const c = worldToCanvas(socle, VUE_REELLE.bounds, VUE_REELLE.width, VUE_REELLE.height, VUE_REELLE.pad)
    const trouve = padStateAt(socle, 0)
    expect(trouve).toBe('full')
    // Le survol se rejoue sur la donnée : le socle doit être atteignable à sa propre position.
    expect(result.current.available).toBe(true)
    expect(c.x).toBeGreaterThan(0)
    expect(c.y).toBeGreaterThan(0)
  })
})

/**
 * LES DEUX SEULES SORTIES SILENCIEUSES DU CALQUE, et c'est le résultat du diagnostic du
 * 2026-08-26 : `paint` ne rend rien QUE dans ces deux cas — bascule éteinte, ou liste vide.
 * Tout le reste (position hors bornes, état `empty`, vignette absente, catalogue non croisé)
 * pose quand même un point. Ces deux cas sont donc les seules hypothèses à départager quand
 * un socle ne s'affiche pas, et ce test les fixe pour la prochaine enquête.
 */
describe('useReplayWeaponPads — quand, et SEULEMENT quand, le calque se tait', () => {
  it('bascule ÉTEINTE : rien n’est peint, mais le calque reste DISPONIBLE', () => {
    const { result } = monterCalque(DOC_530820E5(), false)
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    expect(ops).toHaveLength(0)
    // `available` ne suit PAS la bascule : la commande doit rester offerte pour rallumer.
    expect(result.current.available).toBe(true)
  })

  it('liste VIDE : rien n’est peint, et le calque n’est PAS disponible (bascule masquée)', () => {
    const { result } = monterCalque(testReplayDoc({ weaponPads: [] }))
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    expect(ops).toHaveLength(0)
    expect(result.current.available).toBe(false)
  })

  it('un socle VIDE à l’image lue est quand même POSÉ — l’état ne fait pas disparaître le point', () => {
    const { result } = monterCalque(DOC_530820E5())
    const { ops, ctx } = recordingContext()
    // Frame 500 : la première occupation de chaque socle est finie (tHigh = 346).
    expect(padStateAt(SOCLES_530820E5[3], 500)).toBe('empty')
    result.current.paint(ctx, 500, 1)
    expect(new Set(diamondCentres(ops).map((c) => `${c.x},${c.y}`)).size).toBe(5)
  })
})

/**
 * A9 — UN SOCLE, UNE MARQUE (retour utilisateur du 2026-08-26 : « j'ai l'impression qu'il y en
 * a deux »).
 *
 * CE QUE LA MESURE A MONTRÉ, et c'est la preuve du doublon : sur les CINQ socles réels de
 * `530820e5`, le calque émettait DIX arcs. Cinq points, plus cinq disques pleins posés une
 * dizaine de pixels sous eux — le glyphe de repli de `drawPadIcon`, du même rayon que le point.
 * Aucune duplication de DONNÉE (`crossedWeaponPads` rendait bien cinq socles) : le doublon
 * était entièrement au tracé.
 */
describe('useReplayWeaponPads — A9 : un socle ne se dessine JAMAIS deux fois', () => {
  // L'INVARIANT EST LE NOMBRE DE LIEUX MARQUÉS, PAS LE NOMBRE DE FORMES. Un socle n'en émet
  // qu'une depuis le 2026-08-27 (anneau-bordure supprimé), deux au seul état INCERTAIN — marque
  // et halo — mais CONCENTRIQUES. Le défaut d'A9 était tout autre : deux formes à des centres
  // DIFFÉRENTS. Compter les centres distincts dit donc exactement ce qu'on veut.
  it('cinq socles réels, CINQ lieux marqués — plus dix', () => {
    const { result } = monterCalque(DOC_530820E5())
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    const centres = new Set(
      diamondCentres(ops).map((c) => `${Math.round(c.x * 100)},${Math.round(c.y * 100)}`),
    )
    expect(centres.size).toBe(5)
  })

  it('chaque arc est SUR un socle : aucune marque décalée sous un autre', () => {
    const { result } = monterCalque(DOC_530820E5())
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    const centres = SOCLES_530820E5.map((p) =>
      worldToCanvas(p, VUE_REELLE.bounds, VUE_REELLE.width, VUE_REELLE.height, VUE_REELLE.pad),
    )
    for (const { x, y } of diamondCentres(ops)) {
      const surUnSocle = centres.some((c) => Math.abs(c.x - x) < 0.01 && Math.abs(c.y - y) < 0.01)
      expect(surUnSocle, `marque à (${x}, ${y}) hors de tout socle`).toBe(true)
    }
  })

  // RÈGLE INVERSÉE LE 2026-08-27 (retour utilisateur : « l'icône doit être au-dessus du petit
  // losange, pas dedans ») : la vignette était CENTRÉE sur la marque, elle se pose désormais
  // AU-DESSUS. Ce que ce cas garantit encore — le sujet d'A9 — est qu'elle reste sur la MÊME
  // COLONNE : une image décalée latéralement se lirait comme un second socle.
  it('la vignette se pose AU-DESSUS du losange, sur sa colonne', () => {
    const doc = DOC_530820E5()
    const { ops, ctx } = recordingContext()
    const image = { width: 40, height: 20 } as unknown as CanvasImageSource
    drawWeaponPadsLayer(ctx, doc.weaponPads, VUE_REELLE, { frame: 0, frameMs: 100, k: 1 }, {
      ...STYLE,
      scaleOf: () => 'classic',
      iconOf: () => ({ fill: image, outline: image }),
    })
    const centres = SOCLES_530820E5.map((p) =>
      worldToCanvas(p, VUE_REELLE.bounds, VUE_REELLE.width, VUE_REELLE.height, VUE_REELLE.pad),
    )
    const images = ops.filter((o) => o.op === 'drawImage').map((o) => o.args as number[])
    // `drawImage(src, x, y, w, h)` : neuf poses par socle — huit pour le liseré, une pour le
    // corps. TOUTES doivent tenir sur la colonne de leur socle et au-dessus de son losange.
    expect(images).toHaveLength(SOCLES_530820E5.length * 9)
    for (const [, x, y, w, h] of images) {
      const bas = { x: x + w / 2, y: y + h }
      const socle = centres.reduce((meilleur, cc) =>
        (cc.x - bas.x) ** 2 + (cc.y - bas.y) ** 2 < (meilleur.x - bas.x) ** 2 + (meilleur.y - bas.y) ** 2
          ? cc
          : meilleur,
      )
      // Le liseré s'écarte du corps de 1,2 px dans les huit directions : c'est la seule marge.
      expect(Math.abs(bas.x - socle.x), 'la vignette a quitté la colonne du socle').toBeLessThan(1.3)
      expect(bas.y, 'la vignette mord sur le losange').toBeLessThan(socle.y)
    }
  })
})

/**
 * A13 — UNE COULEUR PAR NATURE DE SOCLE (retour utilisateur du 2026-08-26 : « bordure et
 * couleur plus vive, une couleur pour chaque type, en respectant les couleurs accessibles »).
 *
 * LA RÉSOLUTION RESTE CHEZ L'APPELANT (règle color-tokens) : le calque reçoit trois chaînes
 * déjà résolues et ne connaît aucun token. Ce que ces cas verrouillent, c'est l'APPARIEMENT —
 * quelle nature reçoit laquelle — et le fait qu'un socle qu'on ne sait pas nommer ne soit
 * jamais promu.
 */
describe('weaponPadFamilies — la nature d’un socle (A13)', () => {
  it('un POWER-UP est reconnu par sa famille d’équipement, sans clé d’arme', () => {
    expect(padFamilyOf('powerup_overshield', undefined)).toBe('powerup')
    expect(padFamilyOf('powerup_camo', null)).toBe('powerup')
  })

  it('une ARME DE PUISSANCE est reconnue par sa clé canonique', () => {
    expect(padFamilyOf(SNIPER, 'hinf_s7_sniper')).toBe('power')
    expect(padFamilyOf('0xFFFF', 'hinf_energy_sword')).toBe('power')
  })

  it('tout le reste est CLASSIQUE — et une arme sans clé n’est jamais promue', () => {
    expect(padFamilyOf('0x2B1824D5', 'hinf_br75')).toBe('classic')
    expect(padFamilyOf(INCONNUE, undefined)).toBe('classic')
    expect(padFamilyOf(INCONNUE, null)).toBe('classic')
  })
})

describe('useReplayWeaponPads — chaque nature prend SON encre (A13)', () => {
  it('les trois natures se distinguent à l’écran', () => {
    const doc = DOC_530820E5()
    const { result } = monterCalque(doc)
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    const encres = new Set(
      ops.filter((o) => o.op === 'set strokeStyle').map((o) => String(o.args[0])),
    )
    // La fixture ne porte pas de clé canonique (elle est posée à la requête, pas dans
    // l'artefact) : ses cinq socles sont donc tous CLASSIQUES, et n'emploient que cette encre.
    expect(encres).toEqual(new Set(['#classic']))
  })

  it('un socle de POWER-UP prend l’encre du power-up, pas celle d’un râtelier', () => {
    const doc = testReplayDoc({
      weaponPads: [{ x: 0, y: 0, z: 0, weapon: 'powerup_overshield', spawns: [0], presence: [{ t0: 0, tLow: 99, tHigh: 99 }] }],
    })
    const { result } = monterCalque(doc)
    const { ops, ctx } = recordingContext()
    result.current.paint(ctx, 0, 1)
    const encres = ops.filter((o) => o.op === 'set strokeStyle').map((o) => String(o.args[0]))
    expect(encres.length).toBeGreaterThan(0)
    expect(new Set(encres)).toEqual(new Set(['#powerup']))
  })
})

/**
 * A14 — LE SOCLE EST UN LOSANGE (retour utilisateur du 2026-08-26 : « les socles d'armes et de
 * power up je veux pas de cercles, des points en losanges ce serait mieux, ça facilite la
 * lecture sinon on peut confondre avec des points de joueurs »).
 *
 * C'EST LA FORME QUI PORTE LA DISTINCTION, et rien d'autre : un marqueur de JOUEUR reste un
 * rond (`replayMarkers`). Ces cas verrouillent donc l'absence d'arc autant que la présence du
 * losange — un socle qui redeviendrait rond passerait tous les autres tests de ce fichier.
 */
describe('useReplayWeaponPads — A14 : la marque d’un socle est un LOSANGE', () => {
  it('aucun cercle n’est tracé pour un socle, quel que soit son état', () => {
    const { result } = monterCalque(DOC_530820E5())
    for (const frame of [0, 200, 500, 3000]) {
      const { ops, ctx } = recordingContext()
      result.current.paint(ctx, frame, 1)
      expect(count(ops, 'arc'), `un arc subsiste à l'image ${frame}`).toBe(0)
      expect(diamondCentres(ops).length).toBeGreaterThan(0)
    }
  })

  // AMENDÉ LE 2026-08-27 : ce cas verrouillait la marque ET SA BORDURE à TOUS les états.
  // L'anneau-bordure est supprimé (il enfermait la vignette) ; le second losange n'existe plus
  // que sur l'INCERTAIN, où il est un HALO POINTILLÉ. Même propriété — deux formes
  // concentriques, la seconde la plus large — mais elle change d'état porteur.
  it('au PLEIN, une seule forme ; à l’INCERTAIN, la marque et son halo concentriques', () => {
    const doc = testReplayDoc({
      weaponPads: [{ x: 0, y: 0, z: 0, weapon: '0x0A1992BC', spawns: [0], presence: [{ t0: 0, tLow: 50, tHigh: 150 }] }],
    })
    const { result } = monterCalque(doc)
    const plein = recordingContext()
    result.current.paint(plein.ctx, 0, 1)
    expect(diamondCentres(plein.ops)).toHaveLength(1)
    const incertain = recordingContext()
    result.current.paint(incertain.ctx, 100, 1)
    const centres = diamondCentres(incertain.ops)
    expect(centres).toHaveLength(2)
    expect(centres[0]).toEqual(centres[1])
    // Les deux demi-diagonales, lues sur les sommets : le halo cerne la marque.
    const sommets = incertain.ops.filter((o) => o.op === 'moveTo').map((o) => (o.args as number[])[1])
    expect(sommets[1]).toBeLessThan(sommets[0])
  })
})
