/**
 * Tests — flagCaptureFx (l'onde de choc d'une capture de drapeau, retour du 2026-08-27).
 *
 * CE QU'ILS PROTÈGENT :
 *  - la GARDE D'HORLOGE : sans origine résolue, `a.t` ne veut rien dire et la liste sort vide —
 *    la même règle que les pulses d'objectif, et le genre d'oubli qui ne se voit pas à l'écran
 *    (l'onde s'ouvrirait simplement au mauvais moment, ce qui se lit comme juste) ;
 *  - UNE SEULE STAT : `flag_captures`. Les autres actions de la famille `flag_` sont déjà
 *    lisibles sur le glyphe vivant ; les reprendre ici referait le substitut qu'on a retiré ;
 *  - AUCUNE POSITION DEVINÉE : une capture dont l'auteur n'est pas localisable est écartée ;
 *  - la FENÊTRE D'ÂGE : rien avant l'instant, deux anneaux pendant, rien après ;
 *  - MOUVEMENT RÉDUIT : un seul anneau, sans expansion, qui s'éteint quand même.
 */
import { describe, expect, it } from 'vitest'

import {
  buildFlagCaptureFx,
  drawFlagCaptureFx,
  FLAG_CAPTURE_HOLD_FRAMES,
  type FlagCaptureFx,
} from './flagCaptureFx'
import { testReplayDoc } from './test/testDoc'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 480 + 48,
  height: 480 + 48,
  pad: 24,
}

const INK = 'team-ink'

/** Le joueur A immobile en (1,9) : sa position est relue à toute image de sa vie. */
const TRACKS = [
  {
    slot: 1,
    team: 0,
    xuid: 'A',
    points: [
      { t: 0, x: 1, y: 9 },
      { t: 100, x: 1, y: 9 },
    ],
    startFrame: 0,
    endFrame: 100,
  },
]

/** La relecture de position du hook, réduite ici à ce que le test veut voir. */
const posOfA = (xuid: string) => (xuid === 'A' ? { x: 1, y: 9 } : null)

function docWith(objectives: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    tracks: TRACKS as never,
    objectives: objectives as never,
    ...over,
  })
}

/** Contexte enregistreur : jsdom n'implémente pas getContext (patron du calque voisin). */
function mockCtx() {
  const calls: { method: string; args: unknown[]; alpha: number; stroke: string; width: number }[] = []
  const ctx = { globalAlpha: 1, strokeStyle: '', lineWidth: 1 } as Record<string, unknown>
  for (const m of ['beginPath', 'arc', 'stroke']) {
    ctx[m] = (...args: unknown[]) => {
      calls.push({
        method: m,
        args,
        alpha: ctx.globalAlpha as number,
        stroke: String(ctx.strokeStyle),
        width: ctx.lineWidth as number,
      })
    }
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, calls }
}

/** Les rayons demandés à `arc`, dans l'ordre du tracé. */
function rayons(calls: { method: string; args: unknown[] }[]): number[] {
  return calls.filter((c) => c.method === 'arc').map((c) => c.args[2] as number)
}

describe('buildFlagCaptureFx', () => {
  it("pose la capture au lieu RELU de son auteur, à l'image de l'action", () => {
    const fx = buildFlagCaptureFx(docWith([{ t: 30, xuid: 'A', stat: 'flag_captures', timeMs: 3_000 }]), posOfA)
    expect(fx).toHaveLength(1)
    expect(fx[0]).toEqual({ frame: 30, x: 1, y: 9, xuid: 'A' })
  })

  it("HORLOGE NON FIABLE : rien du tout — un effet muet vaut mieux qu'un effet faux", () => {
    // `originResolved: false` ET aucun `originMs` : le recalage Go n'a pas eu lieu, les frames
    // publiées sont décalées d'un écart inconnu.
    const doc = docWith([{ t: 30, xuid: 'A', stat: 'flag_captures', timeMs: 3_000 }], {
      coverage: { originResolved: false } as never,
    })
    expect(buildFlagCaptureFx(doc, posOfA)).toEqual([])
  })

  it("un artefact ANCIEN (drapeau à false, mais `originMs` publié) garde son onde", () => {
    // Le booléen n'est pas un pointeur : un schéma 11 servi tel quel dit `false` alors qu'il
    // porte une origine valide. La garde exige LES DEUX conditions.
    const doc = docWith([{ t: 30, xuid: 'A', stat: 'flag_captures', timeMs: 3_000 }], {
      coverage: { originResolved: false } as never,
      originMs: 3_604,
    })
    expect(buildFlagCaptureFx(doc, posOfA)).toHaveLength(1)
  })

  it('IGNORE toute autre stat, y compris les autres actions de la famille `flag_`', () => {
    const doc = docWith([
      { t: 10, xuid: 'A', stat: 'flag_grabs', timeMs: 1_000 },
      { t: 20, xuid: 'A', stat: 'flag_returns', timeMs: 2_000 },
      { t: 25, xuid: 'A', stat: 'flag_capture_assists', timeMs: 2_500 },
      { t: 30, xuid: 'A', stat: 'zone_captures', timeMs: 3_000 },
      { t: 40, xuid: 'A', stat: 'flag_captures', timeMs: 4_000 },
    ])
    expect(buildFlagCaptureFx(doc, posOfA).map((f) => f.frame)).toEqual([40])
  })

  it("une capture dont l'AUTEUR n'est pas localisable est ÉCARTÉE, jamais posée au hasard", () => {
    const doc = docWith([
      { t: 30, xuid: 'INCONNU', stat: 'flag_captures', timeMs: 3_000 },
      { t: 40, xuid: 'A', stat: 'flag_captures', timeMs: 4_000 },
    ])
    expect(buildFlagCaptureFx(doc, posOfA).map((f) => f.xuid)).toEqual(['A'])
  })

  it("un film sans action d'objectif ne construit rien", () => {
    expect(buildFlagCaptureFx(docWith([]), posOfA)).toEqual([])
  })
})

describe('drawFlagCaptureFx', () => {
  const FX: FlagCaptureFx[] = [{ frame: 30, x: 5, y: 5, xuid: 'A' }]
  const style = (reducedMotion = false) => ({ inkOf: () => INK, reducedMotion })
  const win = (frame: number) => ({ frame, hold: FLAG_CAPTURE_HOLD_FRAMES })

  it("AVANT l'instant de la capture : rien n'est tracé", () => {
    const { ctx, calls } = mockCtx()
    drawFlagCaptureFx(ctx, FX, VIEW, win(29), style())
    expect(calls).toHaveLength(0)
  })

  it("APRÈS la fenêtre : rien non plus — l'effet ne traîne pas", () => {
    const { ctx, calls } = mockCtx()
    drawFlagCaptureFx(ctx, FX, VIEW, win(30 + FLAG_CAPTURE_HOLD_FRAMES + 1), style())
    expect(calls).toHaveLength(0)
  })

  it('PENDANT : DEUX anneaux concentriques, à l\'encre servie par l\'appelant', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCaptureFx(ctx, FX, VIEW, win(32), style())
    const rs = rayons(calls)
    expect(rs).toHaveLength(2)
    // Le second TRAÎNE derrière le premier : c'est ce décalage qui donne le sens de propagation.
    expect(rs[1]).toBeLessThan(rs[0])
    for (const c of calls) if (c.method === 'stroke') expect(c.stroke).toBe(INK)
  })

  it("L'ONDE S'OUVRE ET S'ÉTEINT : le rayon croît, l'opacité et l'épaisseur décroissent", () => {
    const lu = (frame: number) => {
      const { ctx, calls } = mockCtx()
      drawFlagCaptureFx(ctx, FX, VIEW, win(frame), style())
      const stroke = calls.find((c) => c.method === 'stroke')!
      return { r: rayons(calls)[0], alpha: stroke.alpha, width: stroke.width }
    }
    const tot = lu(31)
    const tard = lu(35)
    expect(tard.r).toBeGreaterThan(tot.r)
    expect(tard.alpha).toBeLessThan(tot.alpha)
    expect(tard.width).toBeLessThan(tot.width)
    // À la toute fin de la fenêtre, il ne reste rien à voir.
    expect(lu(36).alpha).toBeCloseTo(0, 10)
  })

  it('MOUVEMENT RÉDUIT : UN SEUL anneau, de rayon CONSTANT, qui s\'éteint quand même', () => {
    const lu = (frame: number) => {
      const { ctx, calls } = mockCtx()
      drawFlagCaptureFx(ctx, FX, VIEW, win(frame), style(true))
      return { rs: rayons(calls), alpha: calls.find((c) => c.method === 'stroke')!.alpha }
    }
    const tot = lu(31)
    const tard = lu(35)
    expect(tot.rs).toHaveLength(1)
    expect(tard.rs).toEqual(tot.rs)
    expect(tard.alpha).toBeLessThan(tot.alpha)
  })

  it("l'opacité est remise à 1 en sortie — le calque suivant ne peint pas en transparence", () => {
    const { ctx } = mockCtx()
    drawFlagCaptureFx(ctx, FX, VIEW, win(32), style())
    expect(ctx.globalAlpha).toBe(1)
  })

  it('aucun rayon NÉGATIF au premier instant : `arc` lèverait plutôt que de dessiner', () => {
    const { ctx, calls } = mockCtx()
    drawFlagCaptureFx(ctx, FX, VIEW, win(30), style())
    for (const r of rayons(calls)) expect(r).toBeGreaterThan(0)
  })
})
