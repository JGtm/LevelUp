/**
 * antagonistMarks.test.ts : les icônes de titre teintées (cache, contour), la pulsation et la
 * mise en page « icône, compte, nom » partagée par le nom sous le pion et la flèche hors cadre.
 *
 * Aucun pixel : le contexte enregistreur note les appels de dessin, par toile (la teinte se fait
 * sur des toiles hors écran, que `getContext` rend une à une).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { count, recordingContext, type CanvasOp } from '../test/recordingContext'
import {
  createTitleIcons,
  drawTitlePulse,
  pulseScale,
  titleFade,
  TITLE_ICON_HEIGHT_PX,
  TITLE_ICON_URL,
  TITLE_OUTLINE_PX,
  titleInnerWidthPx,
  type TitleDraw,
} from './antagonistMarks'

/** Une image chargée : seules ses dimensions naturelles servent, la teinte lit le contexte enregistreur. */
const image = (w: number, h: number) => ({ naturalWidth: w, naturalHeight: h }) as unknown as HTMLImageElement
const NEMESIS = image(714, 670)
const VICTIM = image(1099, 1369)

let toiles: Map<HTMLCanvasElement, CanvasOp[]>

beforeEach(() => {
  toiles = new Map()
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function (this: HTMLCanvasElement) {
    const { ops, ctx } = recordingContext()
    if (!toiles.has(this)) toiles.set(this, ops)
    return ctx
  } as never)
})
afterEach(() => vi.restoreAllMocks())

const opsDe = (toile: unknown): CanvasOp[] => toiles.get(toile as HTMLCanvasElement) ?? []

describe('les images des titres', () => {
  it('sont les PNG noirs du dépôt, un par rôle', () => {
    expect(TITLE_ICON_URL).toEqual({ nemesis: '/icons/nemesis-black.png', victim: '/icons/victim-black.png' })
  })
})

describe('createTitleIcons : une icône par (rôle, couleur, contour, taille, densité)', () => {
  it('rend null tant que l’image du rôle n’est pas chargée', () => {
    const icons = createTitleIcons()
    expect(icons.iconOf('nemesis', 'rgb(1 2 3)', 'rgb(0 0 0)', 1)).toBeNull()
    icons.setSource('victim', VICTIM)
    expect(icons.iconOf('nemesis', 'rgb(1 2 3)', 'rgb(0 0 0)', 1)).toBeNull()
    expect(icons.iconOf('victim', 'rgb(1 2 3)', 'rgb(0 0 0)', 1)).not.toBeNull()
  })

  it('garde le rapport de l’image : 12 px de haut, la largeur suit', () => {
    const icons = createTitleIcons()
    icons.setSource('nemesis', NEMESIS)
    icons.setSource('victim', VICTIM)
    const n = icons.iconOf('nemesis', 'c', 'o', 1)
    const v = icons.iconOf('victim', 'c', 'o', 1)
    expect(titleInnerWidthPx('nemesis')).toBeCloseTo((TITLE_ICON_HEIGHT_PX * 714) / 670, 6)
    expect(titleInnerWidthPx('victim')).toBeCloseTo((TITLE_ICON_HEIGHT_PX * 1099) / 1369, 6)
    // Le contour déborde de l'icône de chaque côté ; le reste de la toile est le corps de l'icône.
    const pad = Math.ceil(TITLE_OUTLINE_PX)
    const image = n?.image as HTMLCanvasElement
    expect(image.height).toBe(TITLE_ICON_HEIGHT_PX + 2 * pad)
    expect(image.width).toBe(Math.round(titleInnerWidthPx('nemesis')) + 2 * pad)
    expect(n?.hPx).toBe(TITLE_ICON_HEIGHT_PX + 2 * pad)
    void v
  })

  it('sert la même toile pour les mêmes réglages, une autre pour une couleur, une taille ou une densité différente', () => {
    const icons = createTitleIcons()
    icons.setSource('nemesis', NEMESIS)
    const a = icons.iconOf('nemesis', 'rouge', 'noir', 1)
    expect(icons.iconOf('nemesis', 'rouge', 'noir', 1)).toBe(a)
    expect(icons.iconOf('nemesis', 'bleu', 'noir', 1)).not.toBe(a)
    expect(icons.iconOf('nemesis', 'rouge', 'blanc', 1)).not.toBe(a)
    const dense = icons.iconOf('nemesis', 'rouge', 'noir', 2)
    expect(dense).not.toBe(a)
    expect((dense?.image as HTMLCanvasElement).height).toBe(2 * TITLE_ICON_HEIGHT_PX + 2 * Math.ceil(2 * TITLE_OUTLINE_PX))
    // La taille en pixels d'ÉCRAN ne dépend (presque) pas de la densité : seul l'arrondi au pixel diffère.
    expect(Math.abs(dense!.hPx - a!.hPx)).toBeLessThanOrEqual(1)
  })

  it('recommence avec l’image quand elle change (autre document, rechargement)', () => {
    const icons = createTitleIcons()
    icons.setSource('nemesis', NEMESIS)
    const a = icons.iconOf('nemesis', 'rouge', 'noir', 1)
    icons.setSource('nemesis', image(714, 670))
    expect(icons.iconOf('nemesis', 'rouge', 'noir', 1)).not.toBe(a)
  })

  it('teint le masque à la couleur du joueur, le cerne de l’encre du contour, et pose la couleur par-dessus', () => {
    const icons = createTitleIcons()
    icons.setSource('nemesis', NEMESIS)
    const icon = icons.iconOf('nemesis', 'rouge', 'noir', 1)!
    const ops = opsDe(icon.image)
    const dessins = ops.filter((o) => o.op === 'drawImage')
    // Douze silhouettes du contour autour, une seule de la couleur, posée en dernier au centre.
    expect(dessins).toHaveLength(13)
    const pad = Math.ceil(TITLE_OUTLINE_PX)
    expect(dessins[12].args.slice(1, 3)).toEqual([pad, pad])
    // Les silhouettes sont des toiles teintes : la couleur est un remplissage `source-in`.
    const teintes = [...toiles.entries()].flatMap(([, o]) =>
      o.filter((x) => x.op === 'set fillStyle').map((x) => x.args[0]),
    )
    expect(teintes).toContain('rouge')
    expect(teintes).toContain('noir')
  })

  it('un contour vide (variable de thème absente) ne pose aucune silhouette de contour', () => {
    const icons = createTitleIcons()
    icons.setSource('nemesis', NEMESIS)
    const icon = icons.iconOf('nemesis', 'rouge', '', 1)!
    expect(opsDe(icon.image).filter((o) => o.op === 'drawImage')).toHaveLength(1)
  })

  it('l’image est réduite par paliers : un PNG de 1369 px ne se réduit pas à 12 px d’un coup', () => {
    const icons = createTitleIcons()
    icons.setSource('victim', VICTIM)
    icons.iconOf('victim', 'rouge', 'noir', 1)
    const reductions = [...toiles.values()].flatMap((o) => o.filter((x) => x.op === 'drawImage'))
    // Au moins un palier intermédiaire avant la taille finale, en plus des 13 pose du contour et de la couleur.
    expect(reductions.length).toBeGreaterThan(13 + 2)
  })
})

describe('la pulsation', () => {
  it('l’icône grossit puis revient exactement à sa taille', () => {
    expect(pulseScale(0)).toBe(1)
    expect(pulseScale(0.5)).toBeGreaterThan(1.2)
    expect(pulseScale(0.25)).toBeGreaterThan(1)
    expect(pulseScale(1)).toBe(1)
    expect(pulseScale(1.7)).toBe(1)
  })

  it('l’icône de l’ancien tenant s’efface de 1 à 0', () => {
    expect(titleFade(0)).toBe(1)
    expect(titleFade(0.5)).toBeCloseTo(0.5, 6)
    expect(titleFade(1)).toBe(0)
  })

  const marque = (pulse: number | null): TitleDraw => ({ titles: [], pulse, color: 'rouge', k: 1 })

  it('deux anneaux à la couleur du joueur autour du pion à mi-course', () => {
    const { ops, ctx } = recordingContext()
    drawTitlePulse(ctx, { x: 50, y: 40 }, marque(0.5), 6)
    expect(count(ops, 'arc')).toBe(2)
    expect(ops.filter((o) => o.op === 'arc').every((o) => o.args[0] === 50 && o.args[1] === 40)).toBe(true)
    expect(ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toEqual(['rouge', 'rouge'])
    // Les anneaux partent du bord du pion et s'en éloignent.
    const rayons = ops.filter((o) => o.op === 'arc').map((o) => o.args[2] as number)
    expect(Math.min(...rayons)).toBeGreaterThan(6)
  })

  it('le second anneau part un peu après le premier', () => {
    const { ops, ctx } = recordingContext()
    drawTitlePulse(ctx, { x: 0, y: 0 }, marque(0.1), 6)
    expect(count(ops, 'arc')).toBe(1)
  })

  it('aucun anneau au repos, ni sans titre', () => {
    const { ops, ctx } = recordingContext()
    drawTitlePulse(ctx, { x: 0, y: 0 }, marque(null), 6)
    drawTitlePulse(ctx, { x: 0, y: 0 }, undefined, 6)
    expect(ops).toEqual([])
  })

  it('les anneaux s’éteignent en grandissant et laissent l’opacité à 1', () => {
    const { ops, ctx } = recordingContext()
    drawTitlePulse(ctx, { x: 0, y: 0 }, marque(0.7), 6)
    const opacites = ops.filter((o) => o.op === 'set globalAlpha').map((o) => o.args[0] as number)
    expect(opacites.slice(0, -1).every((a) => a > 0 && a < 1)).toBe(true)
    expect(opacites[opacites.length - 1]).toBe(1)
  })
})
