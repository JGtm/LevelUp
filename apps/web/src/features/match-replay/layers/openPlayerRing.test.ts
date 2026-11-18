/**
 * Tests — le pion cerclé de la fiche ouverte : la position relue (vivant seulement, position du bipède) et
 * l'anneau tracé (deux traits au même centre : liseré sombre puis anneau clair, aux cotes nommées).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { COTES_PION_CERCLE } from '../model/cotesFenetreFiche'
import { projectTo, type CanvasView } from '../model/replayView'
import { count, recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { drawOpenPlayerRing, openPlayerPosAt, type OpenPlayerRing } from './openPlayerRing'

const VIEW: CanvasView = { bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 }, width: 200, height: 200, pad: 0 }

/** Un joueur dont la vie couvre les images 10 à 50 et qui avance de (20, 20) à (60, 60). */
function joueur(): { doc: ReturnType<typeof testReplayDoc>; player: ReplayPlayer } {
  const doc = testReplayDoc({
    frameCount: 100,
    bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
    roster: [{ xuid: 'X1', filmIndex: 0, name: 'Un', team: 0 }],
    tracks: [
      {
        slot: 512, team: 0, xuid: 'X1', startFrame: 10, endFrame: 50,
        points: [{ t: 10, x: 20, y: 20, sh: 1, hp: 1 }, { t: 50, x: 60, y: 60, sh: 1, hp: 1 }],
      },
    ],
  })
  const lives = doc.tracks.filter((t) => t.xuid === 'X1')
  return { doc, player: { xuid: 'X1', team: 0, lives } }
}

describe('openPlayerPosAt', () => {
  it('vivant : la position du pion à l’image', () => {
    const { doc, player } = joueur()
    const at = openPlayerPosAt(doc, player)
    expect(at(10)).toEqual({ x: 20, y: 20 })
    expect(at(30)).toEqual({ x: 40, y: 40 })
    expect(at(50)).toEqual({ x: 60, y: 60 })
  })

  it('avant sa vie, après sa vie : aucune position (pas d’anneau sur une croix de mort)', () => {
    const { doc, player } = joueur()
    const at = openPlayerPosAt(doc, player)
    expect(at(9)).toBeNull()
    expect(at(51)).toBeNull()
  })

  it('un joueur sans vie : jamais de position', () => {
    const { doc, player } = joueur()
    expect(openPlayerPosAt(doc, { ...player, lives: [] })(30)).toBeNull()
  })
})

describe('drawOpenPlayerRing', () => {
  const ring = (posAt: OpenPlayerRing['posAt']): OpenPlayerRing => ({ posAt, ink: 'clair', outline: 'fond' })

  it('trace le liseré puis l’anneau au centre projeté du pion, au rayon de la cote mis à l’échelle', () => {
    const { ctx, ops } = recordingContext()
    drawOpenPlayerRing(ctx, ring(() => ({ x: 40, y: 40 })), VIEW, 30, 2)
    const centre = projectTo(VIEW, { x: 40, y: 40 })
    const arcs = ops.filter((o) => o.op === 'arc')
    expect(arcs).toHaveLength(2)
    for (const arc of arcs) expect(arc.args).toEqual([centre.x, centre.y, COTES_PION_CERCLE.rayonPx * 2, 0, Math.PI * 2])
    expect(ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toEqual(['fond', 'clair'])
    expect(ops.filter((o) => o.op === 'set lineWidth').map((o) => o.args[0])).toEqual([
      (COTES_PION_CERCLE.anneauPx + 2 * COTES_PION_CERCLE.lisereExtraPx) * 2,
      COTES_PION_CERCLE.anneauPx * 2,
    ])
    expect(count(ops, 'stroke')).toBe(2)
  })

  it('sans position à l’image : aucun appel de dessin', () => {
    const { ctx, ops } = recordingContext()
    drawOpenPlayerRing(ctx, ring(() => null), VIEW, 30, 1)
    expect(ops).toEqual([])
  })
})
