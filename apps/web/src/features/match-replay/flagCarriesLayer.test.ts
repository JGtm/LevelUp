/**
 * Tests — flagCarriesLayer (la vie des drapeaux de CTF, schéma 15).
 *
 * CE QU'ILS PROTÈGENT :
 *  - les QUATRE états rendent ce que l'artefact dit, et rien de plus : `carried` sur le porteur
 *    relu image par image, `carried_open` ATTÉNUÉ (l'incertitude est visible), `dropped` à sa
 *    position avec une respiration, `home` à la base ;
 *  - la BASE porte un état présent / absent — un drapeau ailleurs laisse un glyphe atténué
 *    derrière lui, un drapeau chez lui n'en laisse pas deux ;
 *  - la couleur passe TOUJOURS par `colorOfTeam`, jamais par une valeur écrite ici ;
 *  - un film non-CTF (aucun drapeau publié) ne trace AUCUNE primitive ;
 *  - le calque n'écrit jamais de texte, comme ses deux voisins.
 */
import { describe, expect, it, vi } from 'vitest'

import {
  drawFlagCarries,
  flagAt,
  flagPointAt,
  flagSpanAt,
  FLAG_STATES,
  homeAnchorOf,
  type FlagCarriesInput,
} from './flagCarriesLayer'
import type { ReplayFlagCarryReady } from './replayNormalize'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 480 + 48,
  height: 480 + 48,
  pad: 24,
}

const INK = '#123456'

/** Un drapeau d'équipe 0 : à la base, porté par A, lâché, puis rentré. */
const FLAG_0: ReplayFlagCarryReady = {
  team: 0,
  spans: [
    { state: 'home', t0: 0, t1: 9, xuid: null, x: 1, y: 9 },
    { state: 'carried', t0: 10, t1: 19, xuid: 'A', x: 2, y: 8 },
    { state: 'dropped', t0: 20, t1: 29, xuid: null, x: 5, y: 5 },
    { state: 'home', t0: 30, t1: 39, xuid: null, x: 1, y: 9 },
  ],
}

/** Un drapeau d'équipe 1 dont le portage n'est fermé par RIEN : borne haute assumée. */
const FLAG_1: ReplayFlagCarryReady = {
  team: 1,
  spans: [
    { state: 'home', t0: 0, t1: 9, xuid: null, x: 9, y: 1 },
    { state: 'carried_open', t0: 10, t1: 99, xuid: 'B', x: 8, y: 2 },
  ],
}

/**
 * Contexte canvas enregistreur : jsdom n'implémente pas getContext. Il capture l'ÉTAT du
 * contexte (opacité, encres) AU MOMENT de chaque appel — sans quoi un test ne lirait que la
 * dernière valeur posée, et l'atténuation ne se vérifierait pas.
 */
function mockCtx() {
  const calls: { method: string; args: unknown[]; alpha: number; stroke: string; fill: string }[] = []
  const ctx = {
    globalAlpha: 1,
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    beginPath: () => {},
    moveTo: () => {},
    lineTo: () => {},
    closePath: () => {},
    arc: () => {},
    fill: () => {},
    stroke: () => {},
    fillText: () => {},
    strokeText: () => {},
  }
  for (const m of ['beginPath', 'moveTo', 'lineTo', 'closePath', 'arc', 'fill', 'stroke', 'fillText', 'strokeText']) {
    ;(ctx as unknown as Record<string, unknown>)[m] = (...args: unknown[]) => {
      calls.push({ method: m, args, alpha: ctx.globalAlpha, stroke: ctx.strokeStyle, fill: ctx.fillStyle })
    }
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, calls }
}

/** Un calque dont le porteur est TOUJOURS localisable (position servie par le test). */
function layerWith(pos: { x: number; y: number } | null, reducedMotion = false): FlagCarriesInput {
  return { style: { colorOfTeam: () => INK, reducedMotion }, posOf: () => pos }
}

describe('flagSpanAt', () => {
  it("rend l'état qui couvre l'image, avec son équipe, son porteur et son début", () => {
    const now = flagSpanAt(FLAG_0, 15)
    expect(now).not.toBeNull()
    expect(now?.state).toBe('carried')
    expect(now?.xuid).toBe('A')
    expect(now?.team).toBe(0)
    expect(now?.t0).toBe(10)
  })

  it('rend null quand aucun intervalle ne couvre l\'image — rien n\'est affirmé par défaut', () => {
    expect(flagSpanAt(FLAG_0, 40)).toBeNull()
    expect(flagSpanAt({ team: 0, spans: [] }, 0)).toBeNull()
  })

  it('les bornes sont INCLUSES des deux côtés (t0 et t1 appartiennent au span)', () => {
    expect(flagSpanAt(FLAG_0, 10)?.state).toBe('carried')
    expect(flagSpanAt(FLAG_0, 19)?.state).toBe('carried')
    expect(flagSpanAt(FLAG_0, 9)?.state).toBe('home')
  })
})

describe('homeAnchorOf', () => {
  it('rend la position de la BASE, celle des intervalles `home`', () => {
    expect(homeAnchorOf(FLAG_0)).toEqual({ x: 1, y: 9 })
    expect(homeAnchorOf(FLAG_1)).toEqual({ x: 9, y: 1 })
  })

  it("rend null quand le drapeau n'est jamais rentré — une base inconnue n'est pas un zéro", () => {
    expect(homeAnchorOf({ team: 0, spans: [FLAG_0.spans[1]] })).toBeNull()
  })
})

describe('flagPointAt', () => {
  it('un drapeau PORTÉ se lit sur la position RELUE de son porteur, pas sur celle du span', () => {
    const now = flagSpanAt(FLAG_0, 15)!
    expect(flagPointAt(now, 15, () => ({ x: 7, y: 3 }))).toEqual({ x: 7, y: 3 })
  })

  it('porteur non localisable : REPLI sur la position mesurée du span, jamais de disparition', () => {
    const now = flagSpanAt(FLAG_0, 15)!
    expect(flagPointAt(now, 15, () => null)).toEqual({ x: 2, y: 8 })
  })

  it('au sol et à la base : la position du span, sans relecture de joueur', () => {
    const posOf = vi.fn(() => ({ x: 0, y: 0 }))
    expect(flagPointAt(flagSpanAt(FLAG_0, 25)!, 25, posOf)).toEqual({ x: 5, y: 5 })
    expect(flagPointAt(flagSpanAt(FLAG_0, 35)!, 35, posOf)).toEqual({ x: 1, y: 9 })
    expect(posOf).not.toHaveBeenCalled()
  })
})

describe('drawFlagCarries', () => {
  it("n'écrit JAMAIS de texte — comme le calque statique et l'état des zones", () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0, FLAG_1], VIEW, 15)
    expect(calls.filter((c) => c.method === 'fillText' || c.method === 'strokeText')).toHaveLength(0)
  })

  it('un film NON-CTF (aucun drapeau publié) ne trace AUCUNE primitive', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [], VIEW, 15)
    expect(calls).toHaveLength(0)
  })

  it("un drapeau sans état à cette image n'est pas dessiné", () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 40)
    expect(calls).toHaveLength(0)
  })

  it('LA COULEUR VIENT DE `colorOfTeam`, appelée avec l\'équipe du drapeau', () => {
    const vus: number[] = []
    const { ctx, calls } = mockCtx()
    const layer: FlagCarriesInput = {
      style: {
        colorOfTeam: (team) => {
          vus.push(team)
          return INK
        },
        reducedMotion: false,
      },
      posOf: () => ({ x: 5, y: 5 }),
    }
    drawFlagCarries(ctx, layer, [FLAG_0, FLAG_1], VIEW, 15)
    expect(vus).toEqual([0, 1])
    // Aucune encre n'échappe au token résolu par l'appelant.
    for (const c of calls) {
      if (c.method === 'fill') expect(c.fill).toBe(INK)
      if (c.method === 'stroke') expect(c.stroke).toBe(INK)
    }
  })

  it('À LA BASE : un seul glyphe, PLEIN — la base n\'est pas doublée d\'un rappel d\'absence', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, 5)
    // Un glyphe = deux tracés (hampe puis fanion) : ici une seule paire.
    expect(calls.filter((c) => c.method === 'beginPath')).toHaveLength(2)
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(1)
  })

  it('AILLEURS QU\'À LA BASE : la base garde un glyphe ATTÉNUÉ (drapeau absent)', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 15)
    // Deux glyphes : la base creuse et atténuée, puis le drapeau porté, plein.
    expect(calls.filter((c) => c.method === 'beginPath')).toHaveLength(4)
    const fills = calls.filter((c) => c.method === 'fill')
    expect(fills).toHaveLength(1)
    const ghostStroke = calls.filter((c) => c.method === 'stroke')
    expect(ghostStroke.some((c) => c.alpha < 0.5)).toBe(true)
    expect(fills[0].alpha).toBeGreaterThan(0.9)
  })

  it('`carried_open` est ATTÉNUÉ et CREUX là où `carried` est plein — la réserve se voit', () => {
    const porte = mockCtx()
    drawFlagCarries(porte.ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 15)
    const ouvert = mockCtx()
    drawFlagCarries(ouvert.ctx, layerWith({ x: 5, y: 5 }), [FLAG_1], VIEW, 15)
    // `carried` remplit son fanion ; `carried_open` ne fait que le cerner.
    expect(porte.calls.filter((c) => c.method === 'fill')).toHaveLength(1)
    expect(ouvert.calls.filter((c) => c.method === 'fill')).toHaveLength(0)
    const alphaOuvert = Math.max(...ouvert.calls.map((c) => c.alpha))
    const alphaPorte = Math.max(...porte.calls.map((c) => c.alpha))
    expect(alphaOuvert).toBeLessThan(alphaPorte)
  })

  it('AU SOL : le drapeau RESPIRE — son opacité change avec l\'image', () => {
    const alphas = new Set<number>()
    for (const frame of [20, 23, 26, 29]) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, frame)
      alphas.add(calls.filter((c) => c.method === 'fill')[0].alpha)
    }
    expect(alphas.size).toBeGreaterThan(1)
  })

  it('MOUVEMENT RÉDUIT : la respiration devient une opacité CONSTANTE', () => {
    const alphas = new Set<number>()
    for (const frame of [20, 23, 26, 29]) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith(null, true), [FLAG_0], VIEW, frame)
      alphas.add(calls.filter((c) => c.method === 'fill')[0].alpha)
    }
    expect(alphas.size).toBe(1)
  })

  it("l'opacité est remise à 1 en sortie — le calque suivant ne peint pas en transparence", () => {
    const { ctx } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 15)
    expect(ctx.globalAlpha).toBe(1)
  })
})

describe('flagAt (survol)', () => {
  it('trouve le drapeau sous le point, et rend le point du GLYPHE pour y poser l\'infobulle', () => {
    const layer = layerWith({ x: 5, y: 5 })
    // VIEW cadre [0,10]² sur 480 px utiles : 48 px/m, Y inversé. (5,5) -> (264, 264).
    const hit = flagAt([FLAG_0], layer, VIEW, 15, { x: 264 + 6, y: 264 - 2 - 6 })
    expect(hit).not.toBeNull()
    expect(hit?.now.state).toBe('carried')
    expect(hit?.at.x).toBeCloseTo(270, 5)
  })

  it('rend null loin de tout glyphe — aucun survol au jugé', () => {
    expect(flagAt([FLAG_0], layerWith({ x: 5, y: 5 }), VIEW, 15, { x: 10, y: 10 })).toBeNull()
  })

  it("un drapeau sans état à cette image n'est pas survolable", () => {
    expect(flagAt([FLAG_0], layerWith(null), VIEW, 40, { x: 264, y: 264 })).toBeNull()
  })
})

describe('FLAG_STATES', () => {
  it('couvre exactement les quatre états du contrat, sans doublon', () => {
    expect([...FLAG_STATES].sort()).toEqual(['carried', 'carried_open', 'dropped', 'home'])
  })
})
