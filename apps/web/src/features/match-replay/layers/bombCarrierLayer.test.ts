import { describe, expect, it, vi } from 'vitest'

import { bombCarrierActiveAt, bombGroundAt, drawBombCarrier, type BombCarrierInput } from './bombCarrierLayer'
import { type CanvasView } from '../model/replayView'
import type { ReplayBombCarry } from '../../../lib/replay/replayNormalize'
import { OFFSCREEN_MARGIN_PX } from '../model/edgeClamp'

const carry = (over: Partial<ReplayBombCarry>): ReplayBombCarry => ({
  xuid: '2533274806055812',
  t0: 0,
  t1: 100,
  closed: true,
  ...over,
})

describe('bombCarrierActiveAt', () => {
  it('rend les portages qui couvrent l’image, bornes incluses', () => {
    const a = carry({ xuid: 'a', t0: 0, t1: 50 })
    const b = carry({ xuid: 'b', t0: 60, t1: 100 })
    expect(bombCarrierActiveAt([a], 0).map((c) => c.xuid)).toEqual(['a'])
    expect(bombCarrierActiveAt([a], 50).map((c) => c.xuid)).toEqual(['a'])
    expect(bombCarrierActiveAt([a], 51)).toEqual([])
    expect(bombCarrierActiveAt([a, b], 70).map((c) => c.xuid)).toEqual(['b'])
  })
})

describe('bombGroundAt', () => {
  const a = carry({ xuid: 'a', t0: 0, t1: 50 })
  const b = carry({ xuid: 'b', t0: 200, t1: 300 })

  it('pose la bombe au sol après un lâcher, jusqu’à la prise suivante', () => {
    // Entre les deux portages : au sol, au dernier point du lâcheur `a`.
    expect(bombGroundAt([a, b], [], 100)?.xuid).toBe('a')
    // Pendant un portage : jamais au sol.
    expect(bombGroundAt([a, b], [], 30)).toBeNull()
    expect(bombGroundAt([a, b], [], 250)).toBeNull()
    // À l'instant du lâcher (t1) : le porteur l'a encore — pas de sol.
    expect(bombGroundAt([a, b], [], 50)).toBeNull()
  })

  it('coupe le sol à l’explosion : après elle, la bombe n’existe plus', () => {
    // Explosion à 80 : au sol jusqu'à 80 inclus exclu, plus rien ensuite.
    expect(bombGroundAt([a, b], [80], 70)?.xuid).toBe('a')
    expect(bombGroundAt([a, b], [80], 80)).toBeNull()
    expect(bombGroundAt([a, b], [80], 150)).toBeNull()
  })

  it('sans horloge de confiance (explosions null), aucun sol n’est dessiné', () => {
    expect(bombGroundAt([a, b], null, 100)).toBeNull()
  })

  it('un portage OUVERT ne met jamais la bombe au sol : personne ne l’a lâchée', () => {
    const open = carry({ xuid: 'a', t0: 0, t1: 50, closed: false })
    expect(bombGroundAt([open], [], 100)).toBeNull()
  })
})

describe('drawBombCarrier', () => {
  const view: CanvasView = {
    bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
    width: 200,
    height: 200,
    pad: 0,
  }
  const makeCtx = () => {
    const fill = vi.fn()
    const arc = vi.fn()
    const ctx = {
      beginPath: vi.fn(), arc, rect: vi.fn(), moveTo: vi.fn(),
      quadraticCurveTo: vi.fn(), fill, stroke: vi.fn(), lineCap: 'butt',
    } as unknown as CanvasRenderingContext2D
    return { ctx, fill, arc }
  }

  it('ne dessine pas un porteur non localisable (aucune position propre à inventer)', () => {
    const { ctx, fill } = makeCtx()
    const layer: BombCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      posOf: () => null,
    }
    drawBombCarrier(ctx, layer, [carry({ t0: 0, t1: 100 })], [], view, 10)
    expect(fill).not.toHaveBeenCalled()
  })

  it('dessine la bombe sur la position relue du porteur courant', () => {
    const { ctx, fill } = makeCtx()
    const layer: BombCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      posOf: () => ({ x: 50, y: 50 }),
    }
    drawBombCarrier(ctx, layer, [carry({ t0: 0, t1: 100 })], [], view, 10)
    // Le glyphe partagé remplit la silhouette UNE fois pour UNE bombe.
    expect(fill).toHaveBeenCalledTimes(1)
  })

  it('au sol, relit la position du lâcheur À L’INSTANT DU LÂCHER, pas à l’image courante', () => {
    const { ctx, fill } = makeCtx()
    const posOf = vi.fn(() => ({ x: 50, y: 50 }))
    const layer: BombCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      posOf,
    }
    drawBombCarrier(ctx, layer, [carry({ xuid: 'a', t0: 0, t1: 50 })], [], view, 120)
    expect(fill).toHaveBeenCalledTimes(1)
    expect(posOf).toHaveBeenCalledWith('a', 50)
  })

  /**
   * BORNAGE HORS CADRE (plan escouade hors cadre, chantier B, phase B3, décision D3,
   * 2026-09-10). Vue : bornes 0..100, toile 200x200, `pad: 0` — 1 unité monde = 2 px canvas,
   * `scaleOf(view)` sans ambiguïté.
   */
  it('PORTÉE ET HORS CADRE : la bombe est PLAQUÉE À LA MARGE, jamais projetée hors toile', () => {
    const { ctx, arc } = makeCtx()
    const layer: BombCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      // Monde (1000, 25) -> canvas brut (2000, 150) : très au-delà de la toile (200x200) sur
      // l'axe X seul, pour garder un test à un seul axe borné.
      posOf: () => ({ x: 1000, y: 25 }),
    }
    drawBombCarrier(ctx, layer, [carry({ t0: 0, t1: 100 })], [], view, 10)
    // Le corps de la silhouette (`bombSilhouette`) trace son premier `arc` au centre du glyphe.
    const [cx] = arc.mock.calls[0] as number[]
    expect(cx).toBeCloseTo(view.width - OFFSCREEN_MARGIN_PX, 5)
    expect(cx).toBeLessThan(2000)
  })

  it('AU SOL, hors cadre : reste à sa position PROJETÉE, pas bornée (hors périmètre D3)', () => {
    // Une bombe au sol n'a plus de PORTEUR (D3 ne borne que les porteurs) : c'est un objet de
    // carte comme un socle d'arme, hors du périmètre tranché pour ce lot.
    const { ctx, arc } = makeCtx()
    const posOf = vi.fn(() => ({ x: 1000, y: 25 }))
    const layer: BombCarrierInput = { style: { ink: '#fff', outline: '#000', reducedMotion: true }, posOf }
    drawBombCarrier(ctx, layer, [carry({ xuid: 'a', t0: 0, t1: 50 })], [], view, 120)
    const [cx] = arc.mock.calls[0] as number[]
    expect(cx).toBeCloseTo(2000, 5)
  })
})
