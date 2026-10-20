import { describe, expect, it } from 'vitest'

import { edgeMarkFor } from './edgeClamp'
import type { CanvasView } from './replayView'

/**
 * Cadrage de test : bornes 0..100 des deux côtés, toile 100x100 sans marge de projection
 * (`pad: 0`) — `scaleOf(view)` vaut alors exactement 1 pixel par mètre, ce qui rend les
 * assertions de distance lisibles sans calcul caché.
 */
const VIEW: CanvasView = {
  bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100, minZ: 0, maxZ: 0 },
  width: 100,
  height: 100,
  pad: 0,
}

const MARGE = 10

describe('edgeMarkFor — la géométrie pure du bornage hors cadre', () => {
  it('rend null pour un point DEDANS (même tout contre la marge)', () => {
    expect(edgeMarkFor({ x: 50, y: 50 }, VIEW, MARGE, 1)).toBeNull()
    expect(edgeMarkFor({ x: 10, y: 10 }, VIEW, MARGE, 1)).toBeNull()
    expect(edgeMarkFor({ x: 90, y: 90 }, VIEW, MARGE, 1)).toBeNull()
  })

  it('au bord DROIT : x = w - marge, et l’angle pointe à droite (0 rad)', () => {
    // Monde (150, 50) -> canvas (150, 50) avec ce cadrage : hors toile à droite, dans la
    // bande verticale utile.
    const mark = edgeMarkFor({ x: 150, y: 50 }, VIEW, MARGE, 1)
    expect(mark).not.toBeNull()
    expect(mark!.at.x).toBeCloseTo(VIEW.width - MARGE, 6)
    expect(mark!.at.y).toBeCloseTo(50, 6)
    expect(mark!.angle).toBeCloseTo(0, 6)
  })

  it('au bord GAUCHE : x = marge, et l’angle pointe à gauche (± PI rad)', () => {
    const mark = edgeMarkFor({ x: -50, y: 50 }, VIEW, MARGE, 1)
    expect(mark).not.toBeNull()
    expect(mark!.at.x).toBeCloseTo(MARGE, 6)
    expect(Math.abs(mark!.angle)).toBeCloseTo(Math.PI, 6)
  })

  it('dans un COIN : les deux axes sont bornés, et l’angle vise la diagonale', () => {
    // Monde (150, 150) -> canvas (150, -50) : hors toile à droite ET par le haut (+Y monde
    // est vers le haut, donc une grande ordonnée monde sort par le HAUT du canevas).
    const mark = edgeMarkFor({ x: 150, y: 150 }, VIEW, MARGE, 1)
    expect(mark).not.toBeNull()
    expect(mark!.at.x).toBeCloseTo(VIEW.width - MARGE, 6)
    expect(mark!.at.y).toBeCloseTo(MARGE, 6)
    // Coin haut-droit, à 45° exactement ici (60 px d'écart sur les deux axes) : -PI/4.
    expect(mark!.angle).toBeCloseTo(-Math.PI / 4, 6)
  })

  it('inversion de Y respectée : un point au SUD du monde sort par le BAS du cadre', () => {
    // Monde (50, -100) -> canvas (50, 200) avec ce cadrage : +Y monde est vers le haut, donc
    // une ordonnée monde très NÉGATIVE se projette bien en bas du canevas (grand y canvas).
    const mark = edgeMarkFor({ x: 50, y: -100 }, VIEW, MARGE, 1)
    expect(mark).not.toBeNull()
    expect(mark!.at.y).toBeCloseTo(VIEW.height - MARGE, 6)
    expect(mark!.angle).toBeCloseTo(Math.PI / 2, 6)
  })

  it('la distance rendue est en MÈTRES : pixels d’écart / scaleOf(view)', () => {
    // scaleOf(VIEW) = 1 px/m ici : 60 px d'écart doivent rendre exactement 60 m.
    const mark = edgeMarkFor({ x: 150, y: 50 }, VIEW, MARGE, 1)
    expect(mark!.distanceM).toBeCloseTo(60, 6)

    // Bornes deux fois plus SERRÉES sur la même toile : `scaleOf` double (2 px/m). Le monde
    // (75, 25) se projette au MÊME point canvas (150, 50) que le cas ci-dessus — même écart de
    // 60 px au bord — mais doit rendre MOITIÉ moins de mètres : preuve que la conversion passe
    // par `scaleOf(view)`, pas par un compte de pixels nu recopié d'un cadrage à l'autre.
    const serre: CanvasView = {
      bounds: { minX: 0, maxX: 50, minY: 0, maxY: 50, minZ: 0, maxZ: 0 },
      width: 100,
      height: 100,
      pad: 0,
    }
    const serreMark = edgeMarkFor({ x: 75, y: 25 }, serre, MARGE, 1)
    expect(serreMark!.at).toEqual(mark!.at)
    expect(serreMark!.distanceM).toBeCloseTo(30, 6)
  })

  it('`echelle` grandit la marge d’écran (marge = margeEcran * echelle)', () => {
    // Monde (50, 85) -> canvas (50, 15) avec ce cadrage (+Y monde vers le haut : une grande
    // ordonnée monde sort par le HAUT du canevas, petite ordonnée canvas). À échelle 1,
    // marge = 10 : 15 est DEDANS ([10, 90]). À échelle 4, marge = 40 : 15 < 40, le point sort
    // par le haut — un point dedans à échelle 1 doit sortir à échelle 4, borné à la marge.
    expect(edgeMarkFor({ x: 50, y: 85 }, VIEW, MARGE, 1)).toBeNull()
    const mark = edgeMarkFor({ x: 50, y: 85 }, VIEW, MARGE, 4)
    expect(mark).not.toBeNull()
    expect(mark!.at.y).toBeCloseTo(MARGE * 4, 6)
  })
})
