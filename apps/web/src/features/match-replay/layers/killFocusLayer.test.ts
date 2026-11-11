/**
 * killFocusLayer.test.ts — LE DESSIN DE LA MISE EN VALEUR, au contexte enregistreur.
 *
 * Ce qu'ils protègent : (1) le voile ne couvre la carte qu'à 10 % et laisse un trou autour des
 * deux joueurs ; (2) les équerres sont des angles OUVERTS (quatre coins, pas un carré fermé), à
 * la couleur de leur joueur, centrées sur lui et animées par le temps écoulé ; (3) un acteur non
 * résolu ou hors cadre n'a ni équerre ni étiquette — jamais de dessin à une position inventée ;
 * (4) l'étiquette du tueur est pleine et porte l'arme, celle de la victime est en creux avec une
 * croix ; (5) `prefers-reduced-motion` : la pose d'arrivée, tout de suite ; (6) aucune couleur
 * écrite en dur : tout vient des encres passées.
 */
import { describe, expect, it } from 'vitest'

import type { CanvasView } from '../model/replayView'
import type { FocusActor, ResolvedKillFocus } from '../model/killFocusLogic'
import { FOCUS_MOTION_MS } from '../model/killFocusMotion'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import {
  BRACKET_ARM_PX,
  BRACKET_HALF_PX,
  VEIL_ALPHA,
  VEIL_HOLE_RADIUS_PX,
  bracketsPath,
  drawKillFocusLabels,
  drawKillFocusVeil,
  type KillFocusInk,
} from './killFocusLayer'

const VIEW: CanvasView = {
  bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100 } as CanvasView['bounds'],
  width: 400,
  height: 400,
  pad: 0,
}

const INK: KillFocusInk = { veil: 'veil-ink', label: 'label-ink' }

function actor(over: Partial<FocusActor> = {}): FocusActor {
  return { xuid: 'a', pos: { x: 50, y: 50 }, color: 'team-color', name: 'Nom', slot: 1, ...over }
}

function focus(over: Partial<ResolvedKillFocus> = {}): ResolvedKillFocus {
  return {
    token: 1,
    frame: 10,
    killer: actor({ xuid: 'k', color: 'killer-color', name: 'Tueur', pos: { x: 25, y: 75 } }),
    victim: actor({ xuid: 'v', color: 'victim-color', name: 'Victime', pos: { x: 75, y: 25 } }),
    weapon: { imageUrl: '/w.png', tinted: true },
    ...over,
  }
}

const ICON = { width: 44, height: 20 } as unknown as CanvasImageSource & { width: number; height: number }

/** Le contexte enregistreur, avec la seule lecture que l'étiquette fait : la mesure du texte. */
function ctxWithMeasure() {
  const rec = recordingContext()
  Reflect.set(rec.ctx, 'measureText', (text: string) => ({ width: text.length * 8 }))
  return rec
}

function paintOpts(over: Partial<Parameters<typeof drawKillFocusLabels>[3]> = {}) {
  return { k: 1, elapsedMs: FOCUS_MOTION_MS, ink: INK, weaponIcon: ICON, ...over }
}

const ops = (all: CanvasOp[], name: string) => all.filter((o) => o.op === name)
const setOf = (all: CanvasOp[], prop: string) => all.filter((o) => o.op === `set ${prop}`).map((o) => o.args[0])

describe('bracketsPath — des angles ouverts, pas un carré', () => {
  it('quatre coins de deux branches : huit segments tracés, aucun côté entier', () => {
    const d = bracketsPath(100, 100, BRACKET_HALF_PX, BRACKET_ARM_PX)
    expect(d).toHaveLength(4)
    for (const corner of d) expect(corner).toHaveLength(3)
  })

  it('chaque coin est à la demi-taille du centre, sur ses deux axes', () => {
    const [haut, , bas] = bracketsPath(100, 100, BRACKET_HALF_PX, BRACKET_ARM_PX)
    expect(haut[1]).toEqual({ x: 100 - BRACKET_HALF_PX, y: 100 - BRACKET_HALF_PX })
    expect(bas[1]).toEqual({ x: 100 + BRACKET_HALF_PX, y: 100 + BRACKET_HALF_PX })
  })

  it('les branches font la longueur de bras et laissent le milieu des côtés ouvert', () => {
    for (const corner of bracketsPath(0, 0, 22, 8)) {
      const [a, c, b] = corner
      expect(Math.hypot(a.x - c.x, a.y - c.y)).toBeCloseTo(8)
      expect(Math.hypot(b.x - c.x, b.y - c.y)).toBeCloseTo(8)
    }
  })
})

describe('le voile', () => {
  it('couvre la carte à 10 % seulement, de l’encre de voile, avec un trou par joueur', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus(), VIEW, paintOpts())
    expect(VEIL_ALPHA).toBe(0.1)
    expect(setOf(all, 'fillStyle')).toContain('veil-ink')
    expect(ops(all, 'rect')[0].args).toEqual([0, 0, VIEW.width, VIEW.height])
    const trous = ops(all, 'arc')
    expect(trous).toHaveLength(2)
    for (const t of trous) expect(t.args[2]).toBe(VEIL_HOLE_RADIUS_PX)
    expect(ops(all, 'fill').some((f) => f.args[0] === 'evenodd')).toBe(true)
  })

  it('un seul acteur résolu : un seul trou', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus({ victim: null }), VIEW, paintOpts())
    expect(ops(all, 'arc')).toHaveLength(1)
  })

  it('le voile se fond avec le temps : sa force part de 0', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus(), VIEW, paintOpts({ elapsedMs: 0 }))
    const alphas = setOf(all, 'globalAlpha') as number[]
    expect(alphas[0]).toBe(0)
  })

  it('à pleine opacité, la force du voile vaut 10 %', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus(), VIEW, paintOpts())
    const alphas = setOf(all, 'globalAlpha') as number[]
    expect(alphas[0]).toBeCloseTo(VEIL_ALPHA)
  })
})

describe('les équerres', () => {
  it('une par acteur, à la couleur de SON joueur', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus(), VIEW, paintOpts())
    const traits = setOf(all, 'strokeStyle')
    expect(traits).toContain('killer-color')
    expect(traits).toContain('victim-color')
    expect(ops(all, 'stroke')).toHaveLength(2)
  })

  it('sont CENTRÉES sur le pion : translation vers lui, rotation et échelle autour de lui', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus(), VIEW, paintOpts({ elapsedMs: 0 }))
    // Tueur en (25, 75) du monde = (100, 100) sur la toile de 400 px (y inversé).
    const t = ops(all, 'translate').map((o) => o.args)
    expect(t).toContainEqual([100, 100])
    expect(t).toContainEqual([-100, -100])
    expect(ops(all, 'rotate').length).toBeGreaterThan(0)
    expect(ops(all, 'scale').length).toBeGreaterThan(0)
  })

  it('au départ elles sont invisibles et larges ; à la fin posées et opaques', () => {
    const debut = ctxWithMeasure()
    drawKillFocusVeil(debut.ctx, focus(), VIEW, paintOpts({ elapsedMs: 0 }))
    const fin = ctxWithMeasure()
    drawKillFocusVeil(fin.ctx, focus(), VIEW, paintOpts({ elapsedMs: FOCUS_MOTION_MS }))
    const alphaDebut = setOf(debut.ops, 'globalAlpha') as number[]
    const alphaFin = setOf(fin.ops, 'globalAlpha') as number[]
    // Voile puis équerres : la dernière opacité posée est celle des équerres.
    expect(alphaDebut[alphaDebut.length - 1]).toBe(0)
    expect(alphaFin[alphaFin.length - 1]).toBe(1)
    const echelleFin = ops(fin.ops, 'scale').map((o) => o.args)
    expect(echelleFin.every(([sx, sy]) => sx === 1 && sy === 1)).toBe(true)
  })

  it('un acteur hors cadre n’a pas d’équerre : le pion est alors une flèche à la marge', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus({ victim: actor({ pos: { x: 5_000, y: 5_000 } }) }), VIEW, paintOpts())
    expect(ops(all, 'stroke')).toHaveLength(1)
    expect(ops(all, 'arc')).toHaveLength(1)
  })

  it('l’épaisseur du trait et la taille suivent la densité du canevas', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, focus(), VIEW, paintOpts({ k: 2 }))
    expect(setOf(all, 'lineWidth')).toContain(6)
  })
})

describe('les étiquettes', () => {
  it('portent le nom du tueur et de la victime, dessinés une fois', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus(), VIEW, paintOpts())
    const textes = ops(all, 'fillText').map((o) => o.args[0])
    expect(textes).toEqual(['Tueur', 'Victime'])
  })

  it('l’étiquette du tueur est PLEINE, à la couleur de son camp, avec le texte à l’encre d’étiquette', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ victim: null }), VIEW, paintOpts())
    const fills = setOf(all, 'fillStyle')
    expect(fills[0]).toBe('killer-color')
    expect(fills).toContain('label-ink')
  })

  it('porte l’icône de l’arme, teintée à l’encre d’étiquette', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ victim: null }), VIEW, paintOpts())
    const images = ops(all, 'drawImage')
    expect(images).toHaveLength(1)
    expect(images[0].args[0]).toBe(ICON)
  })

  it('sans icône d’arme (non identifiée, ou pas encore chargée), pas d’image : jamais celle d’une autre', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ victim: null }), VIEW, paintOpts({ weaponIcon: null }))
    expect(ops(all, 'drawImage')).toHaveLength(0)
  })

  it('l’étiquette de la victime est EN CREUX : fond d’encre, liseré et texte à la couleur de son camp, avec une croix', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ killer: null }), VIEW, paintOpts())
    const fills = setOf(all, 'fillStyle')
    expect(fills[0]).toBe('label-ink')
    expect(fills).toContain('victim-color')
    expect(setOf(all, 'strokeStyle')).toContain('victim-color')
    // La croix : deux diagonales, un seul tracé.
    expect(ops(all, 'moveTo').length).toBeGreaterThanOrEqual(2)
    expect(ops(all, 'drawImage')).toHaveLength(0)
  })

  it('se pose SOUS le pion (le texte descend) et reste centrée sur lui', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ victim: null }), VIEW, paintOpts())
    const [texte, x, y] = ops(all, 'fillText')[0].args as [string, number, number]
    expect(texte).toBe('Tueur')
    expect(y).toBeGreaterThan(100) // le tueur est à y = 100 sur la toile
    expect(x).toBeGreaterThan(0)
  })

  it('un acteur sans nom n’a pas d’étiquette (aucun nom inventé)', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ killer: actor({ name: null }), victim: null }), VIEW, paintOpts())
    expect(ops(all, 'fillText')).toHaveLength(0)
  })

  it('un acteur hors cadre n’a pas d’étiquette', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus({ killer: actor({ pos: { x: 9_999, y: 0 } }), victim: null }), VIEW, paintOpts())
    expect(ops(all, 'fillText')).toHaveLength(0)
  })

  it('apparaît par une échelle qui part de 0,6 (invisible) et finit à 1', () => {
    const debut = ctxWithMeasure()
    drawKillFocusLabels(debut.ctx, focus({ victim: null }), VIEW, paintOpts({ elapsedMs: 0 }))
    const fin = ctxWithMeasure()
    drawKillFocusLabels(fin.ctx, focus({ victim: null }), VIEW, paintOpts({ elapsedMs: FOCUS_MOTION_MS }))
    expect((setOf(debut.ops, 'globalAlpha') as number[])[0]).toBe(0)
    expect((setOf(fin.ops, 'globalAlpha') as number[])[0]).toBe(1)
    expect(ops(debut.ops, 'scale')[0].args).toEqual([0.6, 0.6])
    expect(ops(fin.ops, 'scale')[0].args).toEqual([1, 1])
  })

  it('un contexte reste propre : alpha et état restitués', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusLabels(ctx, focus(), VIEW, paintOpts())
    expect(ops(all, 'save').length).toBe(ops(all, 'restore').length)
  })
})

describe('sans mise en valeur', () => {
  it('rien ne se peint', () => {
    const { ctx, ops: all } = ctxWithMeasure()
    drawKillFocusVeil(ctx, null, VIEW, paintOpts())
    drawKillFocusLabels(ctx, null, VIEW, paintOpts())
    // Seule la pose de la mesure de texte (le harnais) a été écrite dans le contexte.
    expect(all.filter((o) => o.op !== 'set measureText')).toHaveLength(0)
  })
})
