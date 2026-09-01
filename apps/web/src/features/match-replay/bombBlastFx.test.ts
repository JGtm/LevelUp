/**
 * Tests — bombBlastFx (la déflagration d'une bombe d'Assaut, retour du 2026-08-31 : « faudrait
 * un truc bien voyant quand même »).
 *
 * CE QU'ILS PROTÈGENT :
 *  - la GARDE D'HORLOGE : sans origine résolue, `a.t` ne veut rien dire et la liste sort vide —
 *    la même règle que l'onde de capture, et le genre d'oubli qui ne se voit pas à l'écran ;
 *  - UNE SEULE STAT, `bomb_detonations` : c'est la seule que le film réplique en Assaut ;
 *  - AUCUNE POSITION DEVINÉE : une explosion dont l'auteur n'est pas localisable est écartée ;
 *  - CE QUI LA REND VOYANTE, et qui est la demande elle-même : trois couches, une onde deux fois
 *    plus ample que celle d'une capture, et une tenue deux fois plus longue. Un test qui ne
 *    vérifierait que « ça dessine » laisserait ces trois-là se faire raboter sans bruit ;
 *  - MOUVEMENT RÉDUIT : une empreinte fixe, pleine, sans expansion ni éclats.
 */
import { describe, expect, it } from 'vitest'

import {
  BOMB_BLAST_HOLD_FRAMES,
  buildBombBlastFx,
  drawBombBlastFx,
  type BombBlastFx,
} from './bombBlastFx'
import { FLAG_CAPTURE_HOLD_FRAMES } from './flagCaptureFx'
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
  const calls: { method: string; args: unknown[]; alpha: number }[] = []
  const ctx = { globalAlpha: 1, strokeStyle: '', fillStyle: '', lineWidth: 1 } as Record<string, unknown>
  for (const m of ['beginPath', 'arc', 'stroke', 'fill', 'moveTo', 'lineTo']) {
    ctx[m] = (...args: unknown[]) => {
      calls.push({ method: m, args, alpha: ctx.globalAlpha as number })
    }
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, calls }
}

const rayons = (calls: { method: string; args: unknown[] }[]) =>
  calls.filter((c) => c.method === 'arc').map((c) => c.args[2] as number)

const style = (reducedMotion = false) => ({ inkOf: () => INK, reducedMotion })
const BLAST: BombBlastFx = { frame: 10, x: 1, y: 9, xuid: 'A' }

describe('buildBombBlastFx', () => {
  it("pose l'explosion au lieu RELU de son auteur, à l'image de l'action", () => {
    const fx = buildBombBlastFx(
      docWith([{ t: 30, xuid: 'A', stat: 'bomb_detonations', timeMs: 3_000 }]),
      posOfA,
    )
    expect(fx).toEqual([{ frame: 30, x: 1, y: 9, xuid: 'A' }])
  })

  it("HORLOGE NON FIABLE : rien du tout — un effet muet vaut mieux qu'un effet faux", () => {
    const doc = docWith([{ t: 30, xuid: 'A', stat: 'bomb_detonations', timeMs: 3_000 }], {
      coverage: { originResolved: false } as never,
    })
    expect(buildBombBlastFx(doc, posOfA)).toEqual([])
  })

  it('IGNORE toute autre stat', () => {
    const doc = docWith([
      { t: 10, xuid: 'A', stat: 'flag_captures', timeMs: 1_000 },
      { t: 20, xuid: 'A', stat: 'zone_captures', timeMs: 2_000 },
      { t: 40, xuid: 'A', stat: 'bomb_detonations', timeMs: 4_000 },
    ])
    expect(buildBombBlastFx(doc, posOfA).map((f) => f.frame)).toEqual([40])
  })

  it("une explosion dont l'AUTEUR n'est pas localisable est ÉCARTÉE, jamais posée au hasard", () => {
    const doc = docWith([
      { t: 30, xuid: 'INCONNU', stat: 'bomb_detonations', timeMs: 3_000 },
      { t: 40, xuid: 'A', stat: 'bomb_detonations', timeMs: 4_000 },
    ])
    expect(buildBombBlastFx(doc, posOfA).map((f) => f.xuid)).toEqual(['A'])
  })
})

describe('drawBombBlastFx — ce qui la rend VOYANTE', () => {
  it('tient DEUX FOIS plus longtemps que l’onde de capture : un événement rare a ce droit', () => {
    expect(BOMB_BLAST_HOLD_FRAMES).toBe(2 * FLAG_CAPTURE_HOLD_FRAMES)
  })

  it("dessine les TROIS couches à l'instant de l'explosion : éclat, onde, éclats", () => {
    const { ctx, calls } = mockCtx()
    drawBombBlastFx(ctx, [BLAST], VIEW, { frame: 10, hold: BOMB_BLAST_HOLD_FRAMES }, style())
    // L'ÉCLAT est le seul `fill` du tracé ; les ÉCLATS les seuls `moveTo`/`lineTo`.
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(1)
    expect(calls.filter((c) => c.method === 'stroke').length).toBeGreaterThanOrEqual(2)
    expect(calls.filter((c) => c.method === 'moveTo')).toHaveLength(8)
  })

  it("l'ÉCLAT meurt au premier quart, l'onde continue — sinon il empâterait la fin", () => {
    const tot = BOMB_BLAST_HOLD_FRAMES
    const tard = mockCtx()
    drawBombBlastFx(tard.ctx, [BLAST], VIEW, { frame: 10 + tot - 1, hold: tot }, style())
    expect(tard.calls.filter((c) => c.method === 'fill')).toHaveLength(0)
    expect(tard.calls.filter((c) => c.method === 'stroke').length).toBeGreaterThanOrEqual(1)
  })

  it("l'ONDE s'ouvre BIEN PLUS LOIN que celle d'une capture (24 px) : c'est ce qui la distingue", () => {
    const fin = mockCtx()
    drawBombBlastFx(fin.ctx, [BLAST], VIEW, { frame: 10 + BOMB_BLAST_HOLD_FRAMES, hold: BOMB_BLAST_HOLD_FRAMES }, style())
    expect(Math.max(...rayons(fin.calls))).toBeGreaterThan(40)
  })

  it("rien avant l'instant, rien après la fenêtre", () => {
    const avant = mockCtx()
    drawBombBlastFx(avant.ctx, [BLAST], VIEW, { frame: 9, hold: BOMB_BLAST_HOLD_FRAMES }, style())
    expect(avant.calls).toHaveLength(0)
    const apres = mockCtx()
    drawBombBlastFx(apres.ctx, [BLAST], VIEW, { frame: 10 + BOMB_BLAST_HOLD_FRAMES + 1, hold: BOMB_BLAST_HOLD_FRAMES }, style())
    expect(apres.calls).toHaveLength(0)
  })

  it("MOUVEMENT RÉDUIT : une empreinte FIXE et pleine — aucun éclat, aucune expansion", () => {
    const a = mockCtx()
    drawBombBlastFx(a.ctx, [BLAST], VIEW, { frame: 10, hold: BOMB_BLAST_HOLD_FRAMES }, style(true))
    const b = mockCtx()
    drawBombBlastFx(b.ctx, [BLAST], VIEW, { frame: 15, hold: BOMB_BLAST_HOLD_FRAMES }, style(true))
    expect(a.calls.filter((c) => c.method === 'moveTo')).toHaveLength(0)
    // Le rayon ne bouge pas d'une image à l'autre : c'est la définition de « sans mouvement ».
    expect(rayons(b.calls)).toEqual(rayons(a.calls))
    // L'information reste : le disque plein est toujours peint, il ne fait que pâlir.
    expect(b.calls.filter((c) => c.method === 'fill')).toHaveLength(1)
    expect(b.calls[0].alpha).toBeLessThan(a.calls[0].alpha)
  })
})
