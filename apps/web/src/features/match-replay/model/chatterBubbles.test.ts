import { describe, expect, it } from 'vitest'

import {
  CHATTER_FADE_IN_MS,
  CHATTER_FADE_OUT_MS,
  CHATTER_LIFE_MS,
  chatterAlpha,
  liveChatter,
  placeBubbles,
  type Rect,
} from './chatterBubbles'
import type { ChatterLine } from './chatterLines'

describe('chatterAlpha', () => {
  it('fond en 0,15 s, tient, puis sort en 0,5 s ; rien avant ni après sa vie de 2,6 s', () => {
    expect(chatterAlpha(-1, false)).toBe(0)
    expect(chatterAlpha(0, false)).toBe(0)
    expect(chatterAlpha(CHATTER_FADE_IN_MS / 2, false)).toBeCloseTo(0.5)
    expect(chatterAlpha(1000, false)).toBe(1)
    expect(chatterAlpha(CHATTER_LIFE_MS - CHATTER_FADE_OUT_MS / 2, false)).toBeCloseTo(0.5)
    expect(chatterAlpha(CHATTER_LIFE_MS, false)).toBe(0)
    expect(chatterAlpha(CHATTER_LIFE_MS + 1, false)).toBe(0)
  })

  it('sans fondu en mouvement réduit : pleine pendant toute sa vie', () => {
    expect(chatterAlpha(0, true)).toBe(1)
    expect(chatterAlpha(CHATTER_LIFE_MS - 10, true)).toBe(1)
    expect(chatterAlpha(CHATTER_LIFE_MS + 1, true)).toBe(0)
  })
})

function line(frame: number, xuid: string, key = `k${frame}${xuid}`): ChatterLine {
  return { key, frame, xuid, sound: 's', family: 'enemy', text: key, voice: null, count: 1 }
}

describe('liveChatter', () => {
  const FRAME_MS = 100 // 26 images = 2,6 s
  const lines = [line(10, 'a'), line(20, 'a'), line(25, 'b'), line(90, 'a')]

  it('ne montre que ce qui est dit, encore dans sa vie, une bulle par joueur (la plus récente)', () => {
    const live = liveChatter(lines, 30, FRAME_MS, false)
    expect(live.map((l) => l.line.key).sort()).toEqual(['k20a', 'k25b'])
  })

  it("n'a rien avant la première réplique ni longtemps après", () => {
    expect(liveChatter(lines, 5, FRAME_MS, false)).toEqual([])
    expect(liveChatter(lines, 60, FRAME_MS, false)).toEqual([])
  })

  it("donne le même résultat à 0,5× qu'à 4× : le temps est celui du film, pas celui de l'horloge murale", () => {
    // La fonction ne lit que l'image et la durée d'une image : la vitesse de lecture n'y entre pas.
    expect(liveChatter(lines, 30, FRAME_MS, false)).toEqual(liveChatter(lines, 30, FRAME_MS, false))
    expect(liveChatter(lines, 100, FRAME_MS, false).map((l) => l.line.key)).toEqual(['k90a'])
  })

  it('porte le fondu de chaque bulle', () => {
    const [b] = liveChatter([line(10, 'a')], 10, FRAME_MS, false)
    expect(b).toBeUndefined() // âge 0 : opacité 0, pas encore visible
    const [c] = liveChatter([line(10, 'a')], 11, FRAME_MS, false)
    expect(c.alpha).toBe(0.6666666666666666)
  })
})

describe('placeBubbles', () => {
  const CANVAS = { width: 400, height: 300 }
  const overlaps = (a: Rect, b: Rect) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h

  it("pose la bulle au-dessus du pion quand rien ne gêne", () => {
    const [r] = placeBubbles([{ w: 80, h: 24, anchorX: 200, anchorY: 150 }], [], CANVAS)
    expect(r.y + r.h).toBeLessThan(150)
    expect(r.x).toBe(160)
  })

  it('écarte deux bulles qui se recouvriraient', () => {
    const rects = placeBubbles(
      [
        { w: 80, h: 24, anchorX: 200, anchorY: 150 },
        { w: 80, h: 24, anchorX: 205, anchorY: 152 },
      ],
      [],
      CANVAS,
    )
    expect(overlaps(rects[0], rects[1])).toBe(false)
  })

  it('reste dans la toile, même pour un pion au bord', () => {
    const [r] = placeBubbles([{ w: 80, h: 24, anchorX: 2, anchorY: 4 }], [], CANVAS)
    expect(r.x).toBeGreaterThanOrEqual(0)
    expect(r.y).toBeGreaterThanOrEqual(0)
    expect(r.x + r.w).toBeLessThanOrEqual(CANVAS.width)
  })

  it("évite un pion posé là où irait la bulle", () => {
    const marker: Rect = { x: 160, y: 100, w: 80, h: 30 }
    const [r] = placeBubbles([{ w: 80, h: 24, anchorX: 200, anchorY: 150 }], [marker], CANVAS)
    expect(overlaps(r, marker)).toBe(false)
  })
})
