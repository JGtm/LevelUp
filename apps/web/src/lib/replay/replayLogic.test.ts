import { describe, expect, it } from 'vitest'

import {
  advanceFrame,
  altitudeAt,
  altitudeRatio,
  floorOf,
  footprint,
  formatClock,
  frameToMs,
  framesPerSecond,
  freshness,
  heldReading,
  isAliveAt,
  lastIndexAt,
  msToFrames,
  positionAt,
  sceneBounds,
  trackWindow,
  trailAt,
  canvasToWorld,
  frameBounds,
  layerOffset,
  usefulHeight,
  zoomTowards,
  visibleBounds,
  clampCenter,
  sceneCenter,
  ZOOM_LEVELS,
  worldToCanvas,
} from './replayLogic'
import type { ReplayTrackReady } from './replayNormalize'
import { testReplayDoc as makeDoc } from '../../features/match-replay/test/testDoc'

const pts = [
  { t: 0, x: 0, y: 0 },
  { t: 10, x: 10, y: 20 },
  { t: 20, x: 10, y: 20 },
]

describe('positionAt', () => {
  it('null avant le 1er point', () => expect(positionAt(pts, -1)).toBeNull())
  it('exact au 1er point', () => expect(positionAt(pts, 0)).toEqual({ x: 0, y: 0 }))
  it('interpole entre deux points', () => expect(positionAt(pts, 5)).toEqual({ x: 5, y: 10 }))
  it('maintient la dernière position après la fin', () =>
    expect(positionAt(pts, 99)).toEqual({ x: 10, y: 20 }))
  it('liste vide -> null', () => expect(positionAt([], 3)).toBeNull())
})

describe('advanceFrame', () => {
  it('avance de deltaFrames', () => expect(advanceFrame(0, 2, 100)).toBe(2))
  it('boucle à 0 en fin de rejeu', () => expect(advanceFrame(98, 5, 100)).toBe(0))
  it('frameCount<=1 -> 0', () => expect(advanceFrame(0, 1, 1)).toBe(0))
  it('ne descend pas sous 0', () => expect(advanceFrame(0, -5, 100)).toBe(0))
})

describe('worldToCanvas', () => {
  const b = { minX: 0, minY: 0, maxX: 10, maxY: 10 }
  it('coin monde haut-gauche (0,10) -> haut-gauche canvas (Y inversé)', () => {
    const c = worldToCanvas({ x: 0, y: 10 }, b, 100, 100, 10)
    expect(c.x).toBeCloseTo(10)
    expect(c.y).toBeCloseTo(10)
  })
  it('coin monde bas-droit (10,0) -> bas-droit canvas', () => {
    const c = worldToCanvas({ x: 10, y: 0 }, b, 100, 100, 10)
    expect(c.x).toBeCloseTo(90)
    expect(c.y).toBeCloseTo(90)
  })
})

describe('trailAt', () => {
  it('borne la traînée à la fenêtre et finit à la tête', () => {
    const tr = trailAt(pts, 10, 10)
    expect(tr[0]).toEqual({ x: 0, y: 0 })
    expect(tr[tr.length - 1]).toEqual({ x: 10, y: 20 })
  })
  it('exclut les points hors fenêtre', () => {
    const tr = trailAt(pts, 20, 5)
    expect(tr.every((p) => p.x === 10 && p.y === 20)).toBe(true)
  })
})

// LE PLAFOND PAR CARTE (2026-09-02) : jusqu'où le terrain gagne-t-il à grandir ? Passé le point
// où la largeur devient limitante, un pixel de hauteur de plus n'ajoute plus de carte mais une
// bande vide. Ce plafond ne peut donc pas être une constante — il dépend du ratio de la scène.
describe('usefulHeight', () => {
  const square = { minX: 0, minY: 0, maxX: 10, maxY: 10 }
  const wide = { minX: 0, minY: 0, maxX: 40, maxY: 10 }
  const tall = { minX: 0, minY: 0, maxX: 10, maxY: 40 }

  it('scène carrée : la hauteur utile égale la largeur disponible', () =>
    expect(usefulHeight(square, 900, 24)).toBeCloseTo(900))

  it('scène ALLONGÉE : elle sature bien plus bas — le reste serait du vide', () => {
    const useful = usefulHeight(wide, 900, 24)
    expect(useful).toBeCloseTo((900 - 48) / 4 + 48)
    expect(useful).toBeLessThan(300)
  })

  it('scène ÉTIRÉE en profondeur : elle peut au contraire tout prendre', () =>
    expect(usefulHeight(tall, 900, 24)).toBeGreaterThan(900))

  // LA PROPRIÉTÉ QUI COMPTE : à la hauteur utile, la toile a EXACTEMENT le ratio de la carte.
  // Au-delà, le cadre s'élargirait au-dessus et au-dessous de la scène (cf. frameBounds) : on
  // montrerait du monde vide plutôt que de la carte.
  it('à la hauteur utile, la toile a le ratio exact de la carte', () => {
    for (const bounds of [square, wide, tall]) {
      const useful = usefulHeight(bounds, 900, 24)
      const ratioCarte = (bounds.maxX - bounds.minX) / (bounds.maxY - bounds.minY)
      expect((900 - 48) / (useful - 48)).toBeCloseTo(ratioCarte, 6)
    }
  })

  it('ne rend jamais une hauteur dégénérée, même sur une largeur nulle', () =>
    expect(usefulHeight(wide, 0, 24)).toBeGreaterThan(0))
})

describe('altitudeAt', () => {
  const zpts = [
    { t: 0, x: 0, y: 0, z: 0 },
    { t: 10, x: 0, y: 0, z: 4 },
  ]
  it('interpole le Z', () => expect(altitudeAt(zpts, 5)).toBeCloseTo(2))
  it('z absent -> 0', () => expect(altitudeAt(pts, 5)).toBe(0))
  it('avant le 1er point -> null', () => expect(altitudeAt(zpts, -1)).toBeNull())
})

describe('sceneBounds', () => {
  it('sans géométrie, renvoie les bounds des trajectoires', () => {
    const doc = makeDoc()
    expect(sceneBounds(doc)).toEqual(doc.bounds)
  })
  it('cadre sur l’union trajectoires + fond de carte', () => {
    const doc = makeDoc({ geometryBounds: { minX: -5, minY: 2, maxX: 8, maxY: 30 } })
    expect(sceneBounds(doc)).toMatchObject({ minX: -5, minY: 0, maxX: 10, maxY: 30 })
  })
})

describe('échelle temporelle', () => {
  const doc = makeDoc({ frameIntervalMs: 100 })
  it('1× = 10 frames/s pour un pas de 100 ms', () => expect(framesPerSecond(doc)).toBe(10))
  it('retombe sur la cadence historique sans frameIntervalMs', () =>
    expect(framesPerSecond(makeDoc())).toBe(60))
  it('frameToMs suit le temps réel', () => expect(frameToMs(4985, doc)).toBe(498_500))
  it('msToFrames est l’inverse', () => expect(msToFrames(8000, doc)).toBe(80))
  it('formatClock formate en m:ss', () => {
    expect(formatClock(498_500)).toBe('8:18')
    expect(formatClock(9_000)).toBe('0:09')
    expect(formatClock(-5)).toBe('0:00')
  })
})

describe('fenêtre de vie', () => {
  const track: ReplayTrackReady = { slot: 1, team: -1, points: pts, startFrame: 5, endFrame: 15 }
  it('lit startFrame/endFrame', () => expect(trackWindow(track)).toEqual({ start: 5, end: 15 }))
  it('champs omitempty absents -> 0 et t du dernier point', () =>
    expect(trackWindow({ slot: 1, team: -1, points: pts })).toEqual({ start: 0, end: 20 }))
  it('masque avant la naissance et après la mort', () => {
    expect(isAliveAt(track, 4)).toBe(false)
    expect(isAliveAt(track, 10)).toBe(true)
    expect(isAliveAt(track, 16)).toBe(false)
  })
})

describe('étages', () => {
  it('altitudeRatio normalise et borne', () => {
    expect(altitudeRatio(0, -4, 4)).toBeCloseTo(0.5)
    expect(altitudeRatio(-99, -4, 4)).toBe(0)
    expect(altitudeRatio(99, -4, 4)).toBe(1)
  })
  it('carte plate -> 0,5 (pas d’étage significatif)', () =>
    expect(altitudeRatio(3, 2, 2)).toBe(0.5))
  it('floorOf découpe en 3 tranches, borne haute incluse dans la dernière', () => {
    expect(floorOf(-4, -4, 8)).toBe(0)
    expect(floorOf(0, -4, 8)).toBe(1)
    expect(floorOf(8, -4, 8)).toBe(2)
  })
})

describe('footprint', () => {
  it('sans emprise -> liste vide (rendu en point)', () =>
    expect(footprint({ typeId: 1, x: 0, y: 0 })).toEqual([]))
  it('rectangle non tourné : 4 coins centrés', () => {
    const c = footprint({ typeId: 1, x: 10, y: 20, dx: 2, dy: 4 })
    expect(c).toHaveLength(4)
    expect(c[0].x).toBeCloseTo(9)
    expect(c[0].y).toBeCloseTo(18)
    expect(c[2].x).toBeCloseTo(11)
    expect(c[2].y).toBeCloseTo(22)
  })
  it('yaw 90° échange largeur et profondeur', () => {
    const c = footprint({ typeId: 1, x: 0, y: 0, dx: 2, dy: 4, yaw: 90 })
    expect(c[0].x).toBeCloseTo(2)
    expect(c[0].y).toBeCloseTo(-1)
  })
})

describe('lastIndexAt', () => {
  const p = [
    { t: 0, x: 0, y: 0 },
    { t: 5, x: 0, y: 0 },
    { t: 9, x: 0, y: 0 },
  ]
  it('rend -1 avant le premier point', () => expect(lastIndexAt(p, -1)).toBe(-1))
  it('rend -1 sur une liste vide', () => expect(lastIndexAt([], 3)).toBe(-1))
  it('rend le dernier point <= t', () => {
    expect(lastIndexAt(p, 0)).toBe(0)
    expect(lastIndexAt(p, 4)).toBe(0)
    expect(lastIndexAt(p, 5)).toBe(1)
    expect(lastIndexAt(p, 100)).toBe(2)
  })
})

describe('heldReading', () => {
  // Le flux est différentiel : seuls certains points portent la mesure.
  const p = [
    { t: 0, x: 0, y: 0, sh: 1 },
    { t: 5, x: 0, y: 0 },
    { t: 8, x: 0, y: 0, sh: 0 },
    { t: 12, x: 0, y: 0 },
  ]
  const sh = (q: { sh?: number }) => q.sh

  it('remonte au dernier point qui PORTE la mesure', () => {
    expect(heldReading(p, 6, sh, 20)).toEqual({ value: 1, age: 6 })
  })

  it('publie un ZÉRO — un bouclier brisé est une valeur, pas une absence', () => {
    // C'est le cas que `if (!v)` effacerait, et c'est le plus utile du champ.
    expect(heldReading(p, 8, sh, 20)).toEqual({ value: 0, age: 0 })
    expect(heldReading(p, 13, sh, 20)).toEqual({ value: 0, age: 5 })
  })

  it('rend null au-delà du maintien : une mesure périmée ne s’affiche pas', () => {
    expect(heldReading(p, 30, sh, 10)).toBeNull()
  })

  it('rend null avant toute mesure', () => {
    expect(heldReading(p, -1, sh, 20)).toBeNull()
    expect(heldReading([{ t: 0, x: 0, y: 0 }], 0, sh, 20)).toBeNull()
  })
})

describe('freshness', () => {
  it('une mesure de l’instant est franche', () => expect(freshness(0, 20, 0.62)).toBe(1))
  it('une mesure au bord du maintien est au plancher', () =>
    expect(freshness(20, 20, 0.62)).toBeCloseTo(0.38))
  it('ne descend jamais sous le plancher', () =>
    expect(freshness(1000, 20, 0.62)).toBeCloseTo(0.38))
  it('un maintien nul ne dégrade rien', () => expect(freshness(5, 0, 0.62)).toBe(1))
})

describe('sceneBounds avec un sol reconstruit', () => {
  it('cadre sur la zone JOUÉE, pas sur la structure qui déborde', () => {
    // La structure d'une carte couvre ±250 m là où les joueurs en parcourent 50 : cadrer sur
    // elle réduirait le terrain à un timbre.
    const doc = makeDoc({
      structure: [{ x0: -200, y0: -200, x1: 200, y1: 200, z: 0, zb: -1 }],
      geometryBounds: { minX: -5, minY: 2, maxX: 8, maxY: 30 },
    })
    expect(sceneBounds(doc)).toEqual(doc.bounds)
  })
})

describe('sceneBounds avec un fond de carte posé', () => {
  // LE DÉFAUT CORRIGÉ (2026-08-26, capture utilisateur) : une image posée remplace les props
  // Forge à l'écran (ils sont le `else if` du fond dans ReplayCanvas), mais l'union les
  // gardait au dénominateur du cadre. Le cadre était dimensionné sur de la matière INVISIBLE
  // et la carte se réduisait à un timbre dans un canvas vide.
  const props = { minX: -400, minY: -400, maxX: 400, maxY: 400 }

  it('écarte les props du cadre — la zone jouée seule', () => {
    const doc = makeDoc({ geometryBounds: props })
    expect(sceneBounds(doc, true)).toEqual(doc.bounds)
  })

  it('sans image, les props cadrent encore : ils sont alors le seul fond', () => {
    const doc = makeDoc({ geometryBounds: props })
    expect(sceneBounds(doc, false)).toMatchObject({ minX: -400, maxX: 400, maxY: 400 })
  })

  it('le cadre avec image est stable même quand les props explosent', () => {
    // Le témoin qui MORD : si l'image cessait d'écarter les props, ces deux cadres
    // différeraient d'un facteur 40.
    const serre = sceneBounds(makeDoc({ geometryBounds: { minX: -1, minY: -1, maxX: 1, maxY: 1 } }), true)
    const large = sceneBounds(makeDoc({ geometryBounds: props }), true)
    expect(large).toEqual(serre)
  })
})

// LE ZOOM EST UN CHANGEMENT DE BORNES (2026-09-02) : c'est la décision qui rend le lot petit,
// et ces tests sont ce qui la tient. Si `visibleBounds` cessait de préserver l'aspect ou de
// garder la fenêtre dans la scène, la projection partagée par le dessin ET le survol
// mentirait — un pointeur viserait un autre cadre que celui peint.
describe('visibleBounds — le zoom rétrécit les bornes', () => {
  const scene = { minX: 0, minY: 0, maxX: 100, maxY: 60 }
  const c = sceneCenter(scene)

  it('à zoom 1, la fenêtre EST la scène', () => {
    expect(visibleBounds(scene, 1, c.x, c.y)).toMatchObject(scene)
  })

  it('préserve l aspect de la scène à tous les paliers', () => {
    const ratio = (scene.maxX - scene.minX) / (scene.maxY - scene.minY)
    for (const z of ZOOM_LEVELS) {
      const v = visibleBounds(scene, z, c.x, c.y)
      expect((v.maxX - v.minX) / (v.maxY - v.minY)).toBeCloseTo(ratio, 9)
    }
  })

  it('divise chaque dimension par le facteur', () => {
    const v = visibleBounds(scene, 2, c.x, c.y)
    expect(v.maxX - v.minX).toBeCloseTo(50)
    expect(v.maxY - v.minY).toBeCloseTo(30)
  })

  // ON NE SE DÉPLACE PAS HORS CARTE : un centre absurde est ramené, jamais servi tel quel.
  it('la fenêtre ne sort JAMAIS de la scène, même sur un centre aberrant', () => {
    for (const z of ZOOM_LEVELS) {
      for (const [px, py] of [[-1e6, -1e6], [1e6, 1e6], [0, 60], [100, 0]]) {
        const v = visibleBounds(scene, z, px, py)
        expect(v.minX).toBeGreaterThanOrEqual(scene.minX - 1e-9)
        expect(v.maxX).toBeLessThanOrEqual(scene.maxX + 1e-9)
        expect(v.minY).toBeGreaterThanOrEqual(scene.minY - 1e-9)
        expect(v.maxY).toBeLessThanOrEqual(scene.maxY + 1e-9)
      }
    }
  })

  // À zoom 1 la fenêtre vaut la scène : il n'existe qu'UNE position légale. C'est ce qui
  // désactive la croix directionnelle sans qu'on ait à écrire la règle.
  it('à zoom 1, le centre n a qu une position possible', () => {
    for (const [px, py] of [[-999, 999], [10, 10], [90, 50]]) {
      expect(clampCenter(scene, 1, px, py)).toEqual(c)
    }
  })

  it('l amplitude verticale traverse inchangée — le zoom est plan', () => {
    const withZ = { ...scene, minZ: -3, maxZ: 12 }
    const v = visibleBounds(withZ, 3, c.x, c.y)
    expect(v.minZ).toBe(-3)
    expect(v.maxZ).toBe(12)
  })

  // BAISSER LE ZOOM peut rendre le centre courant illégal : une fenêtre plus large ne tient
  // plus aussi près du bord. Le rebornage doit alors ramener, pas laisser filer.
  it('en dézoomant depuis un coin, le centre se reborne', () => {
    const coin = clampCenter(scene, 3, scene.maxX, scene.maxY)
    const apres = clampCenter(scene, 1.5, coin.x, coin.y)
    expect(apres.x).toBeLessThan(coin.x)
    expect(apres.y).toBeLessThan(coin.y)
    const v = visibleBounds(scene, 1.5, apres.x, apres.y)
    expect(v.maxX).toBeLessThanOrEqual(scene.maxX + 1e-9)
  })

  it('un aller-retour de zoom au centre ne dérive pas', () => {
    let p = clampCenter(scene, 1, c.x, c.y)
    for (const z of [1.5, 2, 3, 2, 1.5, 1]) p = clampCenter(scene, z, p.x, p.y)
    expect(p.x).toBeCloseTo(c.x, 9)
    expect(p.y).toBeCloseTo(c.y, 9)
  })
})

// LA TAILLE DE DESSIN NE DÉPEND PAS DU ZOOM, et c'est ce qui garantit que la MÉMOIRE des quatre
// calques statiques n'en dépend pas non plus : ils cuisent à `view.width x view.height`. La
// crainte d'une mémoire qui enfle avec le grossissement est donc évitée par construction — mais
// elle ne le reste que tant que `visibleBounds` préserve l'aspect. D'où ce test, qui l'épingle
// par sa CONSÉQUENCE plutôt que par sa formule.
describe('la surface a cuire est invariante au zoom', () => {
  const scenes = [
    { minX: 0, minY: 0, maxX: 100, maxY: 60 },
    { minX: -20, minY: 5, maxX: 30, maxY: 95 },
  ]
  it('le ratio de la fenetre ne bouge pas d un palier a l autre', () => {
    for (const scene of scenes) {
      const c = sceneCenter(scene)
      const base = (scene.maxX - scene.minX) / (scene.maxY - scene.minY)
      for (const z of ZOOM_LEVELS) {
        const v = visibleBounds(scene, z, c.x, c.y)
        expect((v.maxX - v.minX) / (v.maxY - v.minY)).toBeCloseTo(base, 6)
      }
    }
  })
  it('usefulHeight rend la meme hauteur a tous les paliers', () => {
    for (const scene of scenes) {
      const c = sceneCenter(scene)
      const base = usefulHeight(scene, 900, 24)
      for (const z of ZOOM_LEVELS) {
        const v = visibleBounds(scene, z, c.x, c.y)
        expect(usefulHeight(v, 900, 24)).toBeCloseTo(base, 6)
      }
    }
  })
})

// LA MOLETTE GROSSIT VERS LE CURSEUR, et ces deux fonctions sont ce qui le permet.
describe('canvasToWorld et zoomTowards', () => {
  const b = { minX: 0, minY: 0, maxX: 100, maxY: 60 }

  // L'ALLER-RETOUR EST L'INVARIANT QUI COMPTE : deux projections écrites séparément finissent
  // toujours par diverger d'un demi-pixel, et un demi-pixel par cran de molette devient un
  // décalage franc en cinq crans.
  it('est l inverse exact de worldToCanvas', () => {
    for (const p of [{ x: 0, y: 0 }, { x: 100, y: 60 }, { x: 37, y: 12.5 }]) {
      const c = worldToCanvas(p, b, 900, 480, 24)
      const back = canvasToWorld(c, b, 900, 480, 24)
      expect(back.x).toBeCloseTo(p.x, 6)
      expect(back.y).toBeCloseTo(p.y, 6)
    }
  })

  it('le point visé reste IMMOBILE à l écran quand le zoom change', () => {
    const c0 = sceneCenter(b)
    const vise = { x: 80, y: 15 }
    // Sa position à l'écran avant le cran...
    const avant = worldToCanvas(vise, visibleBounds(b, 1, c0.x, c0.y), 900, 480, 24)
    // ...et après, une fois le centre repositionné par zoomTowards.
    const c1 = zoomTowards(c0, vise, 1, 2)
    const apres = worldToCanvas(vise, visibleBounds(b, 2, c1.x, c1.y), 900, 480, 24)
    expect(apres.x).toBeCloseTo(avant.x, 6)
    expect(apres.y).toBeCloseTo(avant.y, 6)
  })

  it('viser le centre ne déplace pas le centre', () => {
    const c0 = sceneCenter(b)
    expect(zoomTowards(c0, c0, 1, 3)).toEqual(c0)
  })
})

// LE DÉCALAGE DES CALQUES CUITS pendant un glisser. La propriété qui compte n'est pas la
// formule mais son EXACTITUDE : un déplacement est une translation pure, donc recopier l'image
// décalée doit poser chaque point du monde exactement là où un recuit l'aurait posé.
describe('layerOffset', () => {
  const scene = { minX: 0, minY: 0, maxX: 100, maxY: 60 }
  const cadre = (b: typeof scene) => ({ bounds: b, width: 900, height: 480, pad: 24 })

  it('sans cuisson connue, aucun décalage', () => {
    expect(layerOffset(null, cadre(scene))).toEqual({ x: 0, y: 0 })
  })

  it('cadrage inchangé : décalage nul', () => {
    const v = cadre(scene)
    const off = layerOffset(v, v)
    expect(off.x).toBeCloseTo(0, 9)
    expect(off.y).toBeCloseTo(0, 9)
  })

  // L'INVARIANT : après décalage, un point du monde tombe au même pixel que si l'on avait recuit.
  it('replace un point du monde exactement où un recuit l aurait mis', () => {
    const c = sceneCenter(scene)
    const cuit = cadre(visibleBounds(scene, 2, c.x, c.y))
    const apres = cadre(visibleBounds(scene, 2, c.x + 7, c.y - 4))
    const off = layerOffset(cuit, apres)
    for (const p of [{ x: 40, y: 25 }, { x: 62, y: 33 }]) {
      const dansLeCuit = worldToCanvas(p, cuit.bounds, cuit.width, cuit.height, cuit.pad)
      const recuit = worldToCanvas(p, apres.bounds, apres.width, apres.height, apres.pad)
      expect(dansLeCuit.x + off.x).toBeCloseTo(recuit.x, 6)
      expect(dansLeCuit.y + off.y).toBeCloseTo(recuit.y, 6)
    }
  })

  it('se déplacer vers la droite décale le calque vers la gauche', () => {
    const c = sceneCenter(scene)
    const cuit = cadre(visibleBounds(scene, 2, c.x, c.y))
    const apres = cadre(visibleBounds(scene, 2, c.x + 10, c.y))
    expect(layerOffset(cuit, apres).x).toBeLessThan(0)
  })
})

// LE CADRE — la correction du 2026-09-03. La toile épousait le ratio de la SCÈNE : dès que la
// hauteur était bornée par l'écran, elle devenait plus étroite que le bloc et l'image zoomée se
// coupait à ses bords, pendant que la place d'à côté ne servait à rien.
describe('frameBounds — la fenêtre prend la forme de la toile', () => {
  const scene = { minX: 0, minY: 0, maxX: 100, maxY: 60 }
  const PAD = 24

  /** Le ratio utile d'une toile, marges intérieures déduites. */
  const ratio = (w: number, h: number) => (w - 2 * PAD) / (h - 2 * PAD)

  it('épouse le ratio de la TOILE, quelle que soit la forme de la scène', () => {
    for (const [w, h] of [[900, 480], [1200, 400], [600, 700]]) {
      const f = frameBounds(scene, w, h, PAD)
      expect((f.maxX - f.minX) / (f.maxY - f.minY)).toBeCloseTo(ratio(w, h), 6)
    }
  })

  // IL CONTIENT TOUJOURS LA SCÈNE : le cadre s'ÉLARGIT, il ne rogne jamais. Sinon on perdrait
  // du terrain jouable au premier affichage, avant même d'avoir zoomé.
  it('contient toujours la scène entière', () => {
    for (const [w, h] of [[900, 480], [1200, 400], [600, 700], [300, 300]]) {
      const f = frameBounds(scene, w, h, PAD)
      expect(f.minX).toBeLessThanOrEqual(scene.minX + 1e-9)
      expect(f.maxX).toBeGreaterThanOrEqual(scene.maxX - 1e-9)
      expect(f.minY).toBeLessThanOrEqual(scene.minY + 1e-9)
      expect(f.maxY).toBeGreaterThanOrEqual(scene.maxY - 1e-9)
    }
  })

  it('reste centré sur la scène — on élargit des DEUX côtés', () => {
    const f = frameBounds(scene, 1200, 400, PAD)
    expect((f.minX + f.maxX) / 2).toBeCloseTo((scene.minX + scene.maxX) / 2, 9)
    expect((f.minY + f.maxY) / 2).toBeCloseTo((scene.minY + scene.maxY) / 2, 9)
  })

  // À ratio ÉGAL, le cadre EST la scène : rien n'est ajouté quand il n'y a rien à combler.
  it('toile au ratio de la scène : le cadre vaut la scène', () => {
    const h = (900 - 2 * PAD) * (60 / 100) + 2 * PAD
    const f = frameBounds(scene, 900, h, PAD)
    expect(f.minX).toBeCloseTo(scene.minX, 6)
    expect(f.maxX).toBeCloseTo(scene.maxX, 6)
    expect(f.minY).toBeCloseTo(scene.minY, 6)
    expect(f.maxY).toBeCloseTo(scene.maxY, 6)
  })

  // LE POINT DE LA CORRECTION : zoomé, la fenêtre garde la forme de la toile, donc la remplit.
  // Avec l'ancien cadrage (ratio de la scène) elle restait plus étroite, et l'image se coupait.
  it('zoomée, la fenêtre garde le ratio de la toile — donc la remplit', () => {
    const f = frameBounds(scene, 1200, 400, PAD)
    const c = sceneCenter(f)
    for (const z of ZOOM_LEVELS) {
      const v = visibleBounds(f, z, c.x, c.y)
      expect((v.maxX - v.minX) / (v.maxY - v.minY)).toBeCloseTo(ratio(1200, 400), 6)
    }
  })
})
