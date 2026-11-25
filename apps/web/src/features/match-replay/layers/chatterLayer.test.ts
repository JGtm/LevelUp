import { describe, expect, it } from 'vitest'

import type { LiveChatter } from '../model/chatterBubbles'
import type { ChatterLine } from '../model/chatterLines'
import type { CanvasView } from '../model/replayView'
import { drawChatter, wrapText, type ChatterPaint } from './chatterLayer'

/** Un contexte témoin : il note le texte écrit, les opacités posées et les tracés fermés (`closes` : bulle et haut-parleur). */
function recorder() {
  const texts: string[] = []
  let speakers = 0
  const alphas: number[] = []
  const state: Record<string, unknown> = {}
  const ctx = new Proxy(state, {
    get(target, prop: string) {
      if (prop === 'measureText') return (s: string) => ({ width: s.length * 6 })
      if (prop === 'fillText') return (s: string) => texts.push(s)
      if (prop === 'closePath') return () => {
        speakers += 1
      }
      if (prop === 'globalAlpha') return target.globalAlpha ?? 1
      if (typeof target[prop] !== 'undefined') return target[prop]
      return () => {}
    },
    set(target, prop: string, value) {
      target[prop] = value
      if (prop === 'globalAlpha') alphas.push(value as number)
      return true
    },
  }) as unknown as CanvasRenderingContext2D
  return { ctx, texts, alphas, closes: () => speakers }
}

const VIEW: CanvasView = { bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100 }, width: 400, height: 400, pad: 0 }

function line(xuid: string, text: string, voice: string | null): ChatterLine {
  return { key: xuid, frame: 0, xuid, sound: 's', family: 'enemy', text, voice, count: 1 }
}

function paint(over: Partial<ChatterPaint> & { live: readonly LiveChatter[] }): ChatterPaint {
  return {
    posOf: (xuid) => (xuid === 'absent' ? null : { x: 50, y: 50 }),
    colorOf: () => '#abcdef',
    frame: 5,
    showBubble: true,
    showIcon: true,
    ink: { fill: '#fff', text: '#000' },
    fallbackColor: '#888',
    k: 1,
    ...over,
  }
}

const live = (xuid: string, text: string, voice: string | null, alpha = 1): LiveChatter => ({
  line: line(xuid, text, voice),
  ageMs: 500,
  alpha,
})

describe('wrapText', () => {
  const measure = (s: string) => s.length * 6
  it('coupe aux espaces sous la largeur donnée', () => {
    expect(wrapText(measure, 'Ennemi aperçu à proximité', 85)).toEqual(['Ennemi aperçu', 'à proximité'])
  })
  it("ne garde que trois lignes et finit la dernière par une ellipse", () => {
    const out = wrapText(measure, 'a bb cc dd ee ff gg hh ii jj kk ll', 30)
    expect(out).toHaveLength(3)
    expect(out[2].endsWith('…')).toBe(true)
  })
  it('un mot plus long que la largeur reste entier sur sa ligne', () => {
    expect(wrapText(measure, 'Extraordinairement long', 30)).toEqual(['Extraordinairement', 'long'])
  })
})

describe('drawChatter', () => {
  it('écrit le texte de la bulle de chaque joueur qui parle', () => {
    const { ctx, texts } = recorder()
    drawChatter(ctx, VIEW, paint({ live: [live('a', 'Ennemi', null), live('b', 'Boucliers', null)] }))
    expect(texts.sort()).toEqual(['Boucliers', 'Ennemi'])
  })

  it('applique le fondu de chaque réplique', () => {
    const { ctx, alphas } = recorder()
    drawChatter(ctx, VIEW, paint({ live: [live('a', 'Ennemi', null, 0.4)] }))
    expect(alphas).toContain(0.4)
  })

  it("ne dessine rien quand les deux affichages sont éteints", () => {
    const { ctx, texts, closes } = recorder()
    drawChatter(ctx, VIEW, paint({ live: [live('a', 'Ennemi', 'v')], showBubble: false, showIcon: false }))
    expect(texts).toEqual([])
    expect(closes()).toBe(0)
  })

  it("n'écrit rien quand seule l'icône est demandée, et ne dessine l'icône que pour une réplique qui a une voix", () => {
    const sans = recorder()
    drawChatter(sans.ctx, VIEW, paint({ live: [live('a', 'Ennemi', null)], showBubble: false }))
    expect(sans.closes()).toBe(0)
    const avec = recorder()
    drawChatter(avec.ctx, VIEW, paint({ live: [live('a', 'Ennemi', 'v')], showBubble: false }))
    expect(avec.texts).toEqual([])
    expect(avec.closes()).toBeGreaterThan(0)
  })

  it("une bulle sans voix ne porte pas de haut-parleur : l'icône ne promet pas un son qui n'existe pas", () => {
    const sans = recorder()
    drawChatter(sans.ctx, VIEW, paint({ live: [live('a', 'Ennemi', null)] }))
    const avec = recorder()
    drawChatter(avec.ctx, VIEW, paint({ live: [live('a', 'Ennemi', 'v')] }))
    expect(avec.closes()).toBeGreaterThan(sans.closes())
  })

  it("l'icône éteinte, la bulle d'une réplique avec voix n'a pas de haut-parleur", () => {
    const a = recorder()
    drawChatter(a.ctx, VIEW, paint({ live: [live('a', 'Ennemi', 'v')], showIcon: false }))
    const b = recorder()
    drawChatter(b.ctx, VIEW, paint({ live: [live('a', 'Ennemi', 'v')], showIcon: true }))
    expect(b.closes()).toBeGreaterThan(a.closes())
  })

  it("saute un joueur dont le film ne donne pas la position à cette image", () => {
    const { ctx, texts } = recorder()
    drawChatter(ctx, VIEW, paint({ live: [live('absent', 'Fantôme', null), live('a', 'Ennemi', null)] }))
    expect(texts).toEqual(['Ennemi'])
  })
})
