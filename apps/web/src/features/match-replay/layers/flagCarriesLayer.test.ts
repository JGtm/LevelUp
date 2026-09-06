/**
 * Tests — flagCarriesLayer (la vie des drapeaux de CTF, schéma 15).
 *
 * CE QU'ILS PROTÈGENT :
 *  - les QUATRE états rendent ce que l'artefact dit, et rien de plus : `carried` sur le porteur
 *    relu image par image, `carried_open` CREUX (l'incertitude est visible), `dropped` à sa
 *    position publiée, `home` à la base ;
 *  - CE QUI EST HORS DE SA BASE CLIGNOTE, et rien d'autre (retour du 2026-08-27) : les trois
 *    états sortis battent, `home` et le rappel d'absence restent fixes, `prefers-reduced-motion`
 *    éteint le battement sans éteindre le glyphe ;
 *  - le GLYPHE REPOSE SUR UN LISERÉ à l'encre du fond, tracé dessous et plus épais, qui suit son
 *    opacité et laisse le fanion creux lisible ;
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
  flagBlinkAlpha,
  flagPointAt,
  flagSpanAt,
  FLAG_HIT_RADIUS,
  FLAG_STATES,
  homeAnchorOf,
  type FlagCarriesInput,
} from './flagCarriesLayer'
import type { ReplayFlagCarryReady } from '../model/replayNormalize'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 480 + 48,
  height: 480 + 48,
  pad: 24,
}

const INK = '#123456'
/** L'encre du FOND, celle du liseré : servie par l'appelant, jamais choisie par le calque. */
const OUTLINE = '#fedcba'

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
 * contexte (opacité, encres, épaisseur, jointure) AU MOMENT de chaque appel — sans quoi un test
 * ne lirait que la dernière valeur posée, et ni l'atténuation ni le liseré ne se vérifieraient.
 */
function mockCtx() {
  const calls: {
    method: string
    args: unknown[]
    alpha: number
    stroke: string
    fill: string
    width: number
    join: string
  }[] = []
  const ctx = {
    globalAlpha: 1,
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    lineJoin: 'miter',
    beginPath: () => {},
    moveTo: () => {},
    lineTo: () => {},
    closePath: () => {},
    arc: () => {},
    rect: () => {},
    clip: () => {},
    save: () => {},
    restore: () => {},
    fill: () => {},
    stroke: () => {},
    fillText: () => {},
    strokeText: () => {},
  }
  for (const m of ['beginPath', 'moveTo', 'lineTo', 'closePath', 'arc', 'rect', 'clip', 'save', 'restore', 'fill', 'stroke', 'fillText', 'strokeText']) {
    ;(ctx as unknown as Record<string, unknown>)[m] = (...args: unknown[]) => {
      calls.push({
        method: m,
        args,
        alpha: ctx.globalAlpha,
        stroke: ctx.strokeStyle,
        fill: ctx.fillStyle,
        width: ctx.lineWidth,
        join: ctx.lineJoin,
      })
    }
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, calls }
}

/** Un calque dont le porteur est TOUJOURS localisable (position servie par le test). */
function layerWith(pos: { x: number; y: number } | null, reducedMotion = false): FlagCarriesInput {
  return {
    style: { colorOfTeam: () => INK, outline: OUTLINE, reducedMotion },
    posOf: () => pos,
  }
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
        outline: OUTLINE,
        reducedMotion: false,
      },
      posOf: () => ({ x: 5, y: 5 }),
    }
    drawFlagCarries(ctx, layer, [FLAG_0, FLAG_1], VIEW, 15)
    expect(vus).toEqual([0, 1])
    // Aucune encre n'échappe aux DEUX tokens résolus par l'appelant : celui de l'équipe pour le
    // glyphe, celui du fond pour son liseré. Une troisième valeur serait une couleur en dur.
    for (const c of calls) {
      if (c.method === 'fill') expect(c.fill).toBe(INK)
      if (c.method === 'stroke') expect([INK, OUTLINE]).toContain(c.stroke)
    }
  })

  it('À LA BASE : un seul glyphe, PLEIN — la base n\'est pas doublée d\'un rappel d\'absence', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, 5)
    // Un glyphe = quatre tracés : hampe et fanion du LISERÉ, puis hampe et fanion du glyphe.
    expect(calls.filter((c) => c.method === 'beginPath')).toHaveLength(4)
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(1)
  })

  it('AILLEURS QU\'À LA BASE : la base garde un glyphe ATTÉNUÉ (drapeau absent)', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 15)
    // Deux glyphes : la base CREUSE et atténuée, puis le drapeau porté, PLEIN. Cinq chemins pour
    // le creux (le liseré de son fanion ouvre en plus le chemin composé de son écrêtage), quatre
    // pour le plein.
    expect(calls.filter((c) => c.method === 'beginPath')).toHaveLength(9)
    const fills = calls.filter((c) => c.method === 'fill')
    expect(fills).toHaveLength(1)
    const ghostStroke = calls.filter((c) => c.method === 'stroke')
    expect(ghostStroke.some((c) => c.alpha < 0.5)).toBe(true)
    // Depuis le 2026-08-27, l'opacité du glyphe VIVANT est celle de son clignotement : ce qui le
    // distingue du rappel d'absence est sa FORME (fanion plein contre fanion creux), pas une
    // opacité plus forte. Il n'est en revanche jamais éteint — la borne basse tient.
    expect(fills[0].alpha).toBeGreaterThanOrEqual(0.35)
  })

  it('LE GLYPHE REPOSE SUR UN LISERÉ à l\'encre du FOND, tracé AVANT lui et plus épais', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, 5)
    const strokes = calls.filter((c) => c.method === 'stroke')
    // Hampe du liseré, fanion du liseré, hampe du glyphe (le fanion plein part en `fill`).
    expect(strokes.map((c) => c.stroke)).toEqual([OUTLINE, OUTLINE, INK])
    for (const s of strokes) {
      if (s.stroke === OUTLINE) expect(s.width).toBeGreaterThan(strokes[2].width)
    }
    // La pointe du fanion est un angle aigu : sans jointure ronde, le liseré y pousse une
    // aiguille bien plus longue que le débord demandé.
    expect(strokes[1].join).toBe('round')
  })

  it('LE LISERÉ SUIT L\'OPACITÉ du glyphe — un rappel d\'absence n\'est pas cerné d\'un trait franc', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 15)
    // La base atténuée est le premier glyphe tracé : son liseré porte la MÊME opacité que lui.
    const base = calls.filter((c) => c.method === 'stroke').slice(0, 4)
    expect(base.map((c) => c.stroke)).toEqual([OUTLINE, OUTLINE, INK, INK])
    expect(new Set(base.map((c) => c.alpha)).size).toBe(1)
  })

  it('FANION CREUX : le liseré est ÉCRÊTÉ à l\'extérieur — le creux n\'est PAS comblé', () => {
    // Ce que ce test remplace (revue R1 du 2026-08-27, P1) : l'ancienne version n'assertait que
    // l'ORDRE des tracés, et un liseré de 8 px l'aurait passée sans broncher. Un trait de canvas
    // est centré sur son chemin : sans écrêtage, sa moitié intérieure comble le creux — le seul
    // signal qui reste à `carried_open` depuis que l'atténuation lui a été retirée.
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_1], VIEW, 15)
    const seq = calls.map((c) => c.method)
    const iClip = seq.indexOf('clip')
    expect(iClip, "le liseré du fanion creux n'est pas écrêté").toBeGreaterThan(-1)
    // La règle de remplissage FAIT le travail : `evenodd` retire l'intérieur du fanion (entouré
    // de DEUX sous-chemins) de la région de dessin. En `nonzero`, l'écrêtage ne servirait à rien.
    expect(calls[iClip].args[0]).toBe('evenodd')
    // Le chemin d'écrêtage est COMPOSÉ — l'emprise du glyphe, puis le fanion — et il est ouvert
    // après le `save`, sinon la région resterait posée sur les calques suivants.
    const avant = seq.slice(0, iClip)
    expect(avant.lastIndexOf('save')).toBeGreaterThan(-1)
    expect(avant.lastIndexOf('rect')).toBeGreaterThan(avant.lastIndexOf('save'))
    expect(avant.lastIndexOf('beginPath')).toBeGreaterThan(avant.lastIndexOf('save'))
    expect(avant.lastIndexOf('closePath')).toBeGreaterThan(avant.lastIndexOf('rect'))
    // Puis : le trait de liseré, PUIS la fermeture de l'écrêtage — et tout cela AVANT la moindre
    // goutte d'encre d'équipe.
    const apres = seq.slice(iClip)
    const iStroke = apres.indexOf('stroke')
    const iRestore = apres.indexOf('restore')
    expect(iStroke).toBeGreaterThan(-1)
    expect(iRestore).toBeGreaterThan(iStroke)
    expect(calls[iClip + iStroke].stroke).toBe(OUTLINE)
    expect(calls[iClip + iStroke].width).toBeCloseTo(4.8, 10)
    const premiereEncre = calls.findIndex((c) => c.method === 'stroke' && c.stroke === INK)
    expect(premiereEncre).toBeGreaterThan(iClip + iRestore)
    // Rien n'est rempli : le fanion reste creux, c'est tout le propos.
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(0)
  })

  it('FANION PLEIN : AUCUN écrêtage — son remplissage recouvre déjà le débord intérieur', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, 5)
    const seq = calls.map((c) => c.method)
    expect(seq).not.toContain('clip')
    expect(seq).not.toContain('save')
    expect(seq).not.toContain('rect')
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(1)
  })

  it('CHAQUE écrêtage est REFERMÉ — une région laissée ouverte rognerait les calques suivants', () => {
    const { ctx, calls } = mockCtx()
    // Deux glyphes creux à cette image : la base vide et le drapeau à fin non datée.
    drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_1], VIEW, 15)
    const seq = calls.map((c) => c.method)
    expect(seq.filter((m) => m === 'save')).toHaveLength(2)
    expect(seq.filter((m) => m === 'clip')).toHaveLength(2)
    expect(seq.filter((m) => m === 'restore')).toHaveLength(2)
    expect(seq.lastIndexOf('restore')).toBeGreaterThan(seq.lastIndexOf('clip'))
  })

  it('la JOINTURE ne fuit pas vers les calques suivants (comme l\'opacité)', () => {
    // Les deux voies du liseré, y compris celle qui passe par un save/restore : l'état du
    // contexte doit être rendu propre dans les deux cas.
    for (const carry of [FLAG_0, FLAG_1]) {
      const { ctx } = mockCtx()
      drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [carry], VIEW, 15)
      expect((ctx as unknown as { lineJoin: string }).lineJoin).toBe('miter')
      expect(ctx.globalAlpha).toBe(1)
    }
  })

  it('`carried_open` est CREUX là où `carried` est plein — la réserve se voit à la FORME', () => {
    const porte = mockCtx()
    drawFlagCarries(porte.ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, 15)
    const ouvert = mockCtx()
    drawFlagCarries(ouvert.ctx, layerWith({ x: 5, y: 5 }), [FLAG_1], VIEW, 15)
    // `carried` remplit son fanion ; `carried_open` ne fait que le cerner. Depuis le 2026-08-27
    // l'opacité ne les sépare PLUS : les deux sont hors base, donc les deux clignotent — c'est
    // le creux, et lui seul, qui porte la fin non datée.
    const fills = porte.calls.filter((c) => c.method === 'fill')
    const cernes = ouvert.calls.filter((c) => c.method === 'stroke')
    expect(fills).toHaveLength(1)
    expect(ouvert.calls.filter((c) => c.method === 'fill')).toHaveLength(0)
    // Les deux glyphes VIVANTS (dernier tracé de chaque) battent à la MÊME opacité : si
    // `carried_open` avait gardé son atténuation d'avant, les deux valeurs différeraient.
    expect(cernes[cernes.length - 1].alpha).toBeCloseTo(fills[0].alpha, 10)
  })

  it('AU SOL : le drapeau CLIGNOTE — son opacité change avec l\'image', () => {
    const alphas = new Set<number>()
    for (const frame of [20, 22, 25, 27]) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, frame)
      alphas.add(calls.filter((c) => c.method === 'fill')[0].alpha)
    }
    expect(alphas.size).toBeGreaterThan(1)
  })

  it('PORTÉ : le drapeau clignote AUSSI — c\'est « hors de sa base » qui bat, pas « au sol »', () => {
    const alphas = new Set<number>()
    for (const frame of [12, 14, 17, 19]) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, frame)
      alphas.add(calls.filter((c) => c.method === 'fill')[0].alpha)
    }
    expect(alphas.size).toBeGreaterThan(1)
  })

  it('À LA BASE : aucun clignotement — le repos est un repère, pas un événement', () => {
    const alphas = new Set<number>()
    for (const frame of [0, 2, 5, 7] as const) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith(null), [FLAG_0], VIEW, frame)
      alphas.add(calls.filter((c) => c.method === 'fill')[0].alpha)
    }
    expect(alphas.size).toBe(1)
  })

  it('MOUVEMENT RÉDUIT : le clignotement devient une opacité CONSTANTE et PLEINE', () => {
    const alphas = new Set<number>()
    for (const frame of [20, 22, 25, 27]) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith(null, true), [FLAG_0], VIEW, frame)
      alphas.add(calls.filter((c) => c.method === 'fill')[0].alpha)
    }
    expect(alphas.size).toBe(1)
    expect([...alphas][0]).toBeCloseTo(0.95, 10)
  })

  it('LE RAPPEL D\'ABSENCE À LA BASE ne clignote pas, lui — il reste atténué et fixe', () => {
    const alphas = new Set<number>()
    for (const frame of [12, 14, 17, 19]) {
      const { ctx, calls } = mockCtx()
      drawFlagCarries(ctx, layerWith({ x: 5, y: 5 }), [FLAG_0], VIEW, frame)
      // La base est le PREMIER glyphe tracé : son liseré ouvre la liste des tracés.
      alphas.add(calls.filter((c) => c.method === 'stroke')[0].alpha)
    }
    expect(alphas.size).toBe(1)
    expect([...alphas][0]).toBeLessThan(0.5)
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

  it('LE RAYON DE SURVOL A SUIVI L\'ÉLARGISSEMENT du 2026-08-27 (échelle 1,45)', () => {
    // Centre du glyphe porté : (264 + 6 ; 264 − 2 − 13·1,45/2). À 15 px de là, le point était
    // HORS du rayon à l'échelle 1,2 (12 × 1,2 = 14,4) et doit être dedans maintenant (17,4).
    expect(FLAG_HIT_RADIUS).toBeGreaterThan(15)
    const hit = flagAt([FLAG_0], layerWith({ x: 5, y: 5 }), VIEW, 15, {
      x: 270 + 15,
      y: 264 - 2 - (13 * 1.45) / 2,
    })
    expect(hit).not.toBeNull()
    expect(hit?.now.state).toBe('carried')
  })
})

describe('flagBlinkAlpha (le clignotement hors base, 2026-08-27)', () => {
  /** Les trois états HORS BASE : ce sont eux, et eux seuls, qui battent. */
  const DEHORS = ['carried', 'carried_open', 'dropped'] as const

  it('les TROIS états hors base clignotent : leur opacité dépend de l\'image', () => {
    for (const s of DEHORS) {
      const vues = new Set([0, 2, 5, 7].map((f) => flagBlinkAlpha(s, f, false)))
      expect(vues.size, `${s} ne bouge pas`).toBeGreaterThan(1)
    }
  })

  it('`home` est STABLE et PLEIN — insensible à l\'image', () => {
    for (const f of [0, 1, 3, 7, 40, 1_000]) expect(flagBlinkAlpha('home', f, false)).toBe(0.95)
  })

  it('un état INCONNU suit le comportement « présent » : plein et fixe, jamais clignotant', () => {
    // Un artefact plus récent que ce code peut nommer un état qu'il ne connaît pas ; le faire
    // battre affirmerait « il est sorti », ce que rien ne dit.
    for (const f of [0, 3, 5, 8]) expect(flagBlinkAlpha('teleported', f, false)).toBe(0.95)
  })

  it('MOUVEMENT RÉDUIT : les quatre états rendent la même opacité pleine, sans battement', () => {
    for (const s of [...DEHORS, 'home'] as const) {
      for (const f of [0, 2, 5, 7, 13]) expect(flagBlinkAlpha(s, f, true)).toBe(0.95)
    }
  })

  it('les bornes [0,35 ; 0,95] ne sont JAMAIS franchies — un glyphe éteint serait introuvable', () => {
    for (const s of DEHORS) {
      for (let f = 0; f <= 120; f += 1) {
        const a = flagBlinkAlpha(s, f, false)
        expect(a, `${s} à l'image ${f}`).toBeGreaterThanOrEqual(0.35)
        expect(a, `${s} à l'image ${f}`).toBeLessThanOrEqual(0.95)
      }
    }
    // Et le battement PARCOURT TOUT l'intervalle, sur des images ENTIÈRES : c'est ce que le
    // cosinus achète (un sinus s'arrêterait à 0,36 et 0,94 — cf. BLINK_PERIOD_FRAMES).
    const vues = Array.from({ length: 120 }, (_, f) => flagBlinkAlpha('dropped', f, false))
    expect(Math.min(...vues)).toBeCloseTo(0.35, 10)
    expect(Math.max(...vues)).toBeCloseTo(0.95, 10)
  })

  it('LA PÉRIODE EST CELLE ATTENDUE : deux images à une demi-période d\'écart s\'opposent', () => {
    // Période de 10 images (≈ 1 s au pas de 100 ms) : 0 et 5 sont à une demi-période l'une de
    // l'autre, donc aux deux bouts de la course — l'écart doit être franc, pas cosmétique.
    const haut = flagBlinkAlpha('dropped', 0, false)
    const bas = flagBlinkAlpha('dropped', 5, false)
    expect(Math.abs(haut - bas)).toBeGreaterThan(0.5)
    // Une période PLEINE ramène la même valeur : c'est ce qui fait un cycle, pas une dérive.
    expect(flagBlinkAlpha('dropped', 10, false)).toBeCloseTo(haut, 10)
    expect(flagBlinkAlpha('dropped', 15, false)).toBeCloseTo(bas, 10)
  })
})

describe('FLAG_STATES', () => {
  it('couvre exactement les quatre états du contrat, sans doublon', () => {
    expect([...FLAG_STATES].sort()).toEqual(['carried', 'carried_open', 'dropped', 'home'])
  })
})
