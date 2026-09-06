/**
 * Tests — `tintedIconCanvas` : l'option `composite` (lot A du chantier véhicules, 2026-09-02).
 *
 * Contrat du lot : `'source-in'` (défaut) reste STRICTEMENT INCHANGÉ — aucun call-site actuel
 * (masques HUD des socles d'armes) ne passe `composite`, donc son comportement observé avant
 * ce lot doit rester identique bit à bit. `'multiply'` est le mode nouveau, réservé aux sprites
 * véhicules au trait (fond blanc + traits noirs, cf. `.ai/V7.5/film_re/V4_RAPPORT_SPRITES_2026-08-31.md`
 * §10.1) : Canvas ne respecte pas l'alpha en mode `multiply` (le `fillRect` rend tout le canvas
 * opaque, fond transparent compris) — le second passage `destination-in` qui répare ça est ce
 * que ces tests verrouillent.
 *
 * Comme `gif-hover-thumbnail.test.tsx`, le contexte 2D réel n'existe pas sous jsdom : on
 * monkey-patche `HTMLCanvasElement.prototype.getContext` pour renvoyer un faux contexte qui
 * enregistre les appels (même esprit que `test/recordingContext.ts`, adapté ici pour aussi
 * capturer les affectations de `globalCompositeOperation`, une propriété — pas une méthode).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'

import { drawRotatedSprite, tintedIconCanvas } from './replayDraw'

interface FakeOp {
  op: string
  args: unknown[]
}

function fakeContext() {
  const calls: FakeOp[] = []
  const compositeHistory: string[] = []
  const ctx = {
    translate: vi.fn((...a: unknown[]) => calls.push({ op: 'translate', args: a })),
    rotate: vi.fn((...a: unknown[]) => calls.push({ op: 'rotate', args: a })),
    scale: vi.fn((...a: unknown[]) => calls.push({ op: 'scale', args: a })),
    drawImage: vi.fn((...a: unknown[]) => calls.push({ op: 'drawImage', args: a })),
    fillRect: vi.fn((...a: unknown[]) => calls.push({ op: 'fillRect', args: a })),
    setTransform: vi.fn((...a: unknown[]) => calls.push({ op: 'setTransform', args: a })),
    save: vi.fn((...a: unknown[]) => calls.push({ op: 'save', args: a })),
    restore: vi.fn((...a: unknown[]) => calls.push({ op: 'restore', args: a })),
    set globalCompositeOperation(v: string) {
      compositeHistory.push(v)
      calls.push({ op: 'set globalCompositeOperation', args: [v] })
    },
    get globalCompositeOperation() {
      return compositeHistory[compositeHistory.length - 1] ?? 'source-over'
    },
    set fillStyle(v: string) {
      calls.push({ op: 'set fillStyle', args: [v] })
    },
  }
  return { ctx, calls, compositeHistory }
}

function fakeImage(width = 8, height = 4): HTMLImageElement {
  return { naturalWidth: width, naturalHeight: height } as unknown as HTMLImageElement
}

function stubGetContext(ctx: unknown) {
  HTMLCanvasElement.prototype.getContext = vi.fn(
    () => ctx,
  ) as unknown as typeof HTMLCanvasElement.prototype.getContext
}

describe('tintedIconCanvas — option composite', () => {
  const originalGetContext = HTMLCanvasElement.prototype.getContext

  afterEach(() => {
    HTMLCanvasElement.prototype.getContext = originalGetContext
  })

  it("défaut (composite omis) INCHANGÉ : un seul drawImage, un seul fillRect en 'source-in'", () => {
    const { ctx, calls, compositeHistory } = fakeContext()
    stubGetContext(ctx)
    tintedIconCanvas(fakeImage(), '#ff0000')
    expect(compositeHistory).toEqual(['source-in'])
    expect(calls.filter((c) => c.op === 'drawImage')).toHaveLength(1)
    expect(calls.filter((c) => c.op === 'fillRect')).toHaveLength(1)
    expect(calls.some((c) => c.args[0] === 'destination-in')).toBe(false)
  })

  it("composite: 'source-in' explicite — identique au défaut (même trace)", () => {
    const defaultRun = fakeContext()
    stubGetContext(defaultRun.ctx)
    tintedIconCanvas(fakeImage(), '#ff0000')
    const explicitRun = fakeContext()
    stubGetContext(explicitRun.ctx)
    tintedIconCanvas(fakeImage(), '#ff0000', { composite: 'source-in' })
    expect(explicitRun.calls.map((c) => c.op)).toEqual(defaultRun.calls.map((c) => c.op))
    expect(explicitRun.compositeHistory).toEqual(defaultRun.compositeHistory)
  })

  it("composite: 'multiply' réapplique l'alpha : deux drawImage, un fillRect, puis destination-in", () => {
    const { ctx, calls, compositeHistory } = fakeContext()
    stubGetContext(ctx)
    tintedIconCanvas(fakeImage(), '#00ff00', { composite: 'multiply' })
    expect(compositeHistory).toEqual(['multiply', 'destination-in'])
    expect(calls.filter((c) => c.op === 'drawImage')).toHaveLength(2)
    expect(calls.filter((c) => c.op === 'fillRect')).toHaveLength(1)
    // Le fillRect (la teinte) doit précéder le second drawImage (la réapplication d'alpha) :
    // sinon `destination-in` n'a rien à découper.
    const fillRectIdx = calls.findIndex((c) => c.op === 'fillRect')
    const secondDrawImageIdx = calls.map((c) => c.op).lastIndexOf('drawImage')
    expect(fillRectIdx).toBeLessThan(secondDrawImageIdx)
  })

  it("composite: 'multiply' + mirrored : le second drawImage est précédé du MÊME miroir que le premier", () => {
    const { ctx, calls } = fakeContext()
    stubGetContext(ctx)
    tintedIconCanvas(fakeImage(), '#00ff00', { composite: 'multiply', mirrored: true })
    expect(calls.filter((c) => c.op === 'translate')).toHaveLength(2)
    expect(calls.filter((c) => c.op === 'scale')).toHaveLength(2)
    // Chaque drawImage est précédé (pas forcément immédiatement, `destination-in` s'intercale
    // pour le second) de son propre couple translate/scale ADJACENT, dans le même ordre —
    // le même repère miroir reconstitué avant chaque tracé.
    const ops = calls.map((c) => c.op)
    const firstDraw = ops.indexOf('drawImage')
    const secondDraw = ops.lastIndexOf('drawImage')
    expect(ops.slice(0, firstDraw)).toEqual(['translate', 'scale'])
    const secondMirrorStart = ops.indexOf('translate', firstDraw + 1)
    expect(secondMirrorStart).toBeGreaterThan(firstDraw)
    expect(secondMirrorStart).toBeLessThan(secondDraw)
    expect(ops[secondMirrorStart + 1]).toBe('scale')
    // Rien d'autre qu'un couple translate/scale ne doit précéder le second drawImage après lui :
    // seul `set globalCompositeOperation` ('destination-in') s'intercale entre les deux.
    expect(ops.slice(secondMirrorStart + 2, secondDraw)).toEqual(['set globalCompositeOperation'])
  })

  it('tinted:false ne dessine jamais la teinte, quel que soit composite', () => {
    const { ctx, calls } = fakeContext()
    stubGetContext(ctx)
    tintedIconCanvas(fakeImage(), '#00ff00', { composite: 'multiply', tinted: false })
    expect(calls.filter((c) => c.op === 'fillRect')).toHaveLength(0)
    expect(calls.filter((c) => c.op === 'drawImage')).toHaveLength(1)
  })

  it('sans contexte 2D disponible (jsdom nu) : renvoie un canvas vide, ne jette pas', () => {
    stubGetContext(null)
    const canvas = tintedIconCanvas(fakeImage(), '#00ff00', { composite: 'multiply' })
    expect(canvas).toBeInstanceOf(HTMLCanvasElement)
  })
})

/**
 * Tests — `drawRotatedSprite` (lot C, C2) : la transformation d'écran, sans précédent avant ce
 * lot (cf. l'en-tête de la fonction dans `replayDraw.ts`). Le contexte 2D réel n'existe pas sous
 * jsdom : même fausse instrumentation que `tintedIconCanvas` ci-dessus, complétée de `rotate`,
 * `save` et `restore`.
 *
 * `fakeSprite` PORTE `.width`/`.height`, PAS `.naturalWidth` : l'appelant réel
 * (`vehiclesLayer.ts`) passe la vignette déjà TEINTE par `tintedIconCanvas`, un
 * `HTMLCanvasElement` — même lecture de dimension que `drawPadIcon` (`weaponPadsLayer.ts`), pas
 * celle de `tintedIconCanvas` lui-même (qui lit `naturalWidth` sur l'image SOURCE, avant teinture).
 */
function fakeSprite(width = 20, height = 10): CanvasImageSource {
  return { width, height } as unknown as HTMLCanvasElement
}

describe('drawRotatedSprite', () => {
  it("centre, tourne et met à l'échelle : save, translate(x,y), rotate(angle), scale(s,s), drawImage centré, restore", () => {
    const { ctx, calls } = fakeContext()
    drawRotatedSprite(ctx as unknown as CanvasRenderingContext2D, fakeSprite(20, 10), 100, 50, Math.PI / 2, 2)
    expect(calls.map((c) => c.op)).toEqual(['save', 'translate', 'rotate', 'scale', 'drawImage', 'restore'])
    expect(calls[1].args).toEqual([100, 50])
    expect(calls[2].args).toEqual([Math.PI / 2])
    expect(calls[3].args).toEqual([2, 2])
    // L'image est posée à SA TAILLE NATURELLE (20x10), CENTRÉE sur l'origine déjà mise à
    // l'échelle par `ctx.scale` : c'est ce facteur, pas les dimensions de `drawImage`, qui porte
    // la taille voulue à l'écran.
    expect(calls[4].args).toEqual([expect.anything(), -10, -5, 20, 10])
  })

  it("angle en radians canevas, jamais un cap monde : l'appelant convertit, ce helper ne fait que poser rotate(angle) tel quel", () => {
    const { ctx, calls } = fakeContext()
    const angle = -1.234
    drawRotatedSprite(ctx as unknown as CanvasRenderingContext2D, fakeSprite(), 0, 0, angle, 1)
    expect(calls.find((c) => c.op === 'rotate')?.args).toEqual([angle])
  })

  it('image sans dimension (chargement pas encore abouti) : ne dessine rien, ne jette pas', () => {
    const { ctx, calls } = fakeContext()
    drawRotatedSprite(ctx as unknown as CanvasRenderingContext2D, fakeSprite(0, 0), 10, 10, 0, 1)
    expect(calls).toEqual([])
  })

  it('échelle nulle ou négative : ne dessine rien (un véhicule ne se retourne pas en négatif)', () => {
    const { ctx, calls } = fakeContext()
    drawRotatedSprite(ctx as unknown as CanvasRenderingContext2D, fakeSprite(), 10, 10, 0, 0)
    expect(calls).toEqual([])
    const { ctx: ctx2, calls: calls2 } = fakeContext()
    drawRotatedSprite(ctx2 as unknown as CanvasRenderingContext2D, fakeSprite(), 10, 10, 0, -1)
    expect(calls2).toEqual([])
  })
})
