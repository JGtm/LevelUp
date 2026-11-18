// @vitest-environment node
import { describe, expect, it } from 'vitest'

import { CUT_ALPHA_ABOVE, CUT_ALPHA_BELOW } from './constants'
import { ReliefEngine, type ReliefRenderRequest } from './engine'
import { isReliefError } from './errors'
import { lineCount } from './lineSet'
import { fourBlocksScene, reliefData, reliefImage } from './reliefScene.testutil'
import { handleReliefRequest } from './workerHandler'

const BASE = '#3f8c4f'
const request = (over: Partial<ReliefRenderRequest> = {}): ReliefRenderRequest => ({
  turns: 0,
  cut: null,
  base: BASE,
  lines: false,
  masks: false,
  ...over,
})
const px = (col: number) => 5 * 40 + col

function loaded(): ReliefEngine {
  const engine = new ReliefEngine()
  engine.load(fourBlocksScene())
  return engine
}

describe('ReliefEngine.load', () => {
  it('rend la taille de la carte et dit si elle a un maillage', () => {
    expect(new ReliefEngine().load(fourBlocksScene())).toEqual({ width: 40, height: 10, hasMesh: true })
    const none = fourBlocksScene({ degradations: ['sans-maillage'], rects: [], levels: [] })
    expect(new ReliefEngine().load(none).hasMesh).toBe(false)
  })

  it('refuse une image qui n’est pas de la taille du calage', () => {
    const { data } = fourBlocksScene()
    const err = (() => {
      try {
        new ReliefEngine().load({ data, image: reliefImage(10, 10, () => ({ z: 0 })) })
      } catch (e) {
        return e
      }
    })()
    expect(isReliefError(err) && err.code).toBe('image-size')
  })

  it('rend not-loaded avant tout chargement ou après dispose', () => {
    const engine = new ReliefEngine()
    expect(() => engine.render(request())).toThrow(/aucune carte/)
    const ready = loaded()
    ready.dispose()
    expect(() => ready.render(request())).toThrow(/aucune carte/)
  })
})

describe('ReliefEngine.render sans coupe (« Tout »)', () => {
  it('teinte les pixels de la carte en pleine opacité, laisse le reste transparent', () => {
    const engine = new ReliefEngine()
    const { data, image } = fourBlocksScene()
    image.flags[px(3)] &= ~1 // un pixel connu hors de la zone
    engine.load({ data, image })
    const frame = engine.render(request())
    expect(frame.cut).toBeNull()
    expect(frame.image!.length).toBe(40 * 10 * 4)
    expect(frame.image![px(5) * 4 + 3]).toBe(255)
    expect(frame.image![px(3) * 4 + 3]).toBe(0)
    expect(frame.alpha).toBeNull()
    expect(frame.planAlpha).toBeNull()
  })

  it('rend les traits du fichier, en copie, sans murs', () => {
    const { data, image } = fourBlocksScene()
    const engine = new ReliefEngine()
    engine.load({ data, image })
    const frame = engine.render(request({ lines: true }))
    expect(Array.from(frame.lines!.steps.points)).toEqual(Array.from(data.lines.steps.points))
    expect(frame.lines!.steps.points).not.toBe(data.lines.steps.points)
    expect(lineCount(frame.lines!.walls)).toBe(0)
    expect(engine.render(request()).lines).toBeNull()
  })

  it('une teinte différente change l’image, pas la forme', () => {
    const engine = loaded()
    const a = engine.render(request({ base: '#3f8c4f' })).image!
    const b = engine.render(request({ base: '#7559b5' })).image!
    expect(Array.from(a)).not.toEqual(Array.from(b))
    for (let p = 0; p < 400; p++) expect(a[p * 4 + 3]).toBe(b[p * 4 + 3])
  })

  it('sans base, n’a pas d’image : seuls les masques et les traits sont demandés', () => {
    const frame = loaded().render(request({ base: null, cut: 2.2, masks: true }))
    expect(frame.image).toBeNull()
    expect(frame.alpha).not.toBeNull()
  })

  it('refuse une couleur de base illisible avec un code distinct', () => {
    expect(() => loaded().render(request({ base: 'pas une couleur' }))).toThrow(/invalide/)
    try {
      loaded().render(request({ base: 'x' }))
    } catch (e) {
      expect(isReliefError(e) && e.code).toBe('tint-invalid')
    }
  })

  it('l’orientation change l’ombrage (la lumière reste en haut à gauche de l’écran)', () => {
    const engine = new ReliefEngine()
    const data = reliefData({ width: 20, height: 20 })
    engine.load({ data, image: reliefImage(20, 20, (i) => ({ z: i * 0.2 })) })
    const a = engine.render(request({ turns: 0 })).image!
    const b = engine.render(request({ turns: 2 })).image!
    expect(Array.from(a)).not.toEqual(Array.from(b))
  })
})

describe('ReliefEngine.render avec coupe', () => {
  it('arrondit la hauteur de coupe au quart d’unité et la rend', () => {
    expect(loaded().render(request({ cut: 2.2 })).cut).toBe(2.25)
    expect(loaded().render(request({ cut: -0.01 })).cut).toBe(0)
  })

  it('rend les masques des fonds Rendu et Plan, au format de la coupe', () => {
    const frame = loaded().render(request({ cut: 2.2, masks: true }))
    expect(frame.alpha![px(15)]).toBe(255)
    expect(frame.alpha![px(5)]).toBe(Math.round(CUT_ALPHA_BELOW * 255))
    expect(frame.planAlpha![px(25)]).toBe(Math.round(CUT_ALPHA_BELOW * 255))
    expect(frame.image![px(15) * 4 + 3]).toBe(255)
    expect(frame.image![px(5) * 4 + 3]).toBe(Math.round(CUT_ALPHA_BELOW * 255))
    expect(loaded().render(request({ cut: 0, masks: true })).alpha![px(15)]).toBe(Math.round(CUT_ALPHA_ABOVE * 255))
  })

  it('sans demande de masques, n’en rend pas', () => {
    const frame = loaded().render(request({ cut: 2.2 }))
    expect(frame.alpha).toBeNull()
    expect(frame.planAlpha).toBeNull()
  })

  it('filtre les ruptures et les courbes par la coupe, et ajoute les murs de la bande', () => {
    const frame = loaded().render(request({ cut: 2.2, lines: true }))
    const steps = frame.lines!.steps
    expect(lineCount(steps)).toBe(1)
    expect(steps.points[0]).toBeCloseTo(9.75, 4)
    expect(steps.points[steps.points.length - 2]).toBeCloseTo(29.25, 4)
    expect(lineCount(frame.lines!.contours)).toBe(1) // la colonne 15 est dans l'étage
    expect(lineCount(frame.lines!.walls)).toBeGreaterThan(0)
    const atGround = loaded().render(request({ cut: 0, lines: true }))
    expect(lineCount(atGround.lines!.contours)).toBe(0)
  })

  it('les tableaux rendus sont à l’appelant : les modifier n’abîme pas le cache du moteur', () => {
    const engine = loaded()
    const first = engine.render(request({ cut: 2.2, masks: true }))
    first.alpha![0] = 7
    first.planAlpha![0] = 7
    first.image![3] = 9
    const second = engine.render(request({ cut: 2.2, masks: true }))
    expect(second.alpha![0]).not.toBe(7)
    expect(second.planAlpha![0]).not.toBe(7)
    expect(second.image![3]).not.toBe(9)
  })

  it('une carte sans maillage ignore la coupe : résultat de « Tout »', () => {
    const engine = new ReliefEngine()
    engine.load(fourBlocksScene({ degradations: ['sans-maillage'], rects: [], levels: [] }))
    const frame = engine.render(request({ cut: 2.2, masks: true, lines: true }))
    expect(frame.cut).toBeNull()
    expect(frame.alpha).toBeNull()
    expect(lineCount(frame.lines!.steps)).toBe(1)
  })

  it('charger une autre carte jette l’état de la précédente', () => {
    const engine = loaded()
    engine.render(request({ cut: 2.2 }))
    engine.load(fourBlocksScene({ steps: [] }))
    expect(lineCount(engine.render(request({ lines: true })).lines!.steps)).toBe(0)
  })
})

describe('handleReliefRequest : le traitement du worker, sans worker', () => {
  it('charge, rend et libère, et rend la liste des buffers à transférer', () => {
    const engine = new ReliefEngine()
    const load = handleReliefRequest(engine, { id: 1, op: 'load', input: fourBlocksScene() })
    expect(load.response).toEqual({ id: 1, ok: true, op: 'load', info: { width: 40, height: 10, hasMesh: true } })
    expect(load.transfer).toEqual([])
    const render = handleReliefRequest(engine, { id: 2, op: 'render', request: request({ cut: 2.2, lines: true, masks: true }) })
    expect(render.response.ok && render.response.op).toBe('render')
    // image, alpha, planAlpha, puis les trois ensembles de traits (deux buffers chacun).
    expect(render.transfer).toHaveLength(3 + 6)
    expect(new Set(render.transfer).size).toBe(render.transfer.length)
    const cloned = structuredClone(render.response, { transfer: render.transfer })
    expect(cloned.ok).toBe(true)
    expect(render.transfer.every((b) => b.byteLength === 0)).toBe(true)
    expect(handleReliefRequest(engine, { id: 3, op: 'dispose' }).response).toEqual({ id: 3, ok: true, op: 'dispose' })
  })

  it('rend l’erreur dans la réponse plutôt que de la lever, avec son code', () => {
    const engine = new ReliefEngine()
    const { response, transfer } = handleReliefRequest(engine, { id: 9, op: 'render', request: request() })
    expect(response).toMatchObject({ id: 9, ok: false, error: { code: 'not-loaded' } })
    expect(transfer).toEqual([])
  })
})
