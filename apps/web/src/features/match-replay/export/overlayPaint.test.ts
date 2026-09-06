/**
 * overlayPaint.test.ts — CE QUE LES SURIMPRESSIONS PEIGNENT, sur un contexte espionné.
 *
 * On ne teste pas des pixels (jsdom n'en produit aucun) mais la SUITE D'APPELS : ce qui est
 * peint, dans quel ordre, et avec quelle encre. C'est ce qui verrouille les décisions qui se
 * verraient à l'œil sur un export — l'ordre fond-vers-texte, le tiret dans l'encre atténuée,
 * le texte JAMAIS dans la couleur de camp, l'absence de panneau quand il n'y a rien à dire.
 *
 * La FIDÉLITÉ au DOM, elle, ne se teste pas ici : elle se prononce à l'œil sur le clip, et
 * c'est le gate de recette utilisateur de l'étape E2 du plan.
 */
import { describe, expect, it, vi } from 'vitest'

import {
  neutralStatusStyle,
  paintOverlayPanel,
  type OverlayFonts,
  type OverlayInk,
  type OverlayPanel,
} from './overlayPaint'

const INK: OverlayInk = {
  background: 'rgb(10 10 10)',
  foreground: 'rgb(240 240 240)',
  muted: 'rgb(140 140 140)',
  border: 'rgb(60 60 60)',
  card: 'rgb(24 24 24)',
}
const FONTS: OverlayFonts = { sans: 'Inter, sans-serif', mono: 'JetBrains Mono, monospace' }
const VIEW = { width: 800, height: 400 }

/** Un contexte 2D espionné : chaque appel est enregistré avec l'encre courante. */
function spyContext() {
  const trace: { op: string; args: unknown[]; fill: string; alpha: number }[] = []
  const ctx = {
    fillStyle: '',
    strokeStyle: '',
    globalAlpha: 1,
    font: '',
    letterSpacing: '0px',
    textAlign: '',
    textBaseline: '',
    lineWidth: 0,
    shadowColor: '',
    shadowBlur: 0,
    shadowOffsetY: 0,
    save: vi.fn(),
    restore: vi.fn(),
    beginPath: vi.fn(),
    rect: vi.fn(),
    roundRect: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
    drawImage: vi.fn(),
    measureText: (t: string) => ({ width: t.length * 10 }),
    fillRect: vi.fn(),
    fillText: vi.fn(),
  } as unknown as CanvasRenderingContext2D & { __trace: typeof trace }
  for (const op of ['fillRect', 'fillText', 'drawImage', 'roundRect', 'stroke', 'fill'] as const) {
    const brut = ctx[op] as unknown as ReturnType<typeof vi.fn>
    brut.mockImplementation((...args: unknown[]) => {
      trace.push({
        op,
        args,
        // Les deux encres sont typées `string | CanvasGradient | CanvasPattern` ; ce peintre
        // n'emploie que des chaînes, et le test n'a besoin que de celles-là.
        fill: (op === 'stroke' ? ctx.strokeStyle : ctx.fillStyle) as string,
        alpha: ctx.globalAlpha,
      })
    })
  }
  ctx.__trace = trace
  return ctx
}

const VICTOIRE: OverlayPanel = {
  status: 'Victoire',
  statusStyle: { background: 'rgb(0 90 200)', border: 'rgb(0 120 255)' },
  label: 'Equipe Eagle',
  score: { ally: 50, enemy: 43 },
  veil: true,
}

describe('paintOverlayPanel — l’écran de fin de match', () => {
  it('peint le voile en premier, sur toute la toile, à 70 %', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    const voile = ctx.__trace[0]
    expect(voile.op).toBe('fillRect')
    expect(voile.args).toEqual([0, 0, 800, 400])
    expect(voile.fill).toBe(INK.background)
    expect(voile.alpha).toBeCloseTo(0.7)
  })

  it('met le verdict en CAPITALES, comme le fait `uppercase` dans le DOM', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    const textes = ctx.__trace.filter((e) => e.op === 'fillText').map((e) => e.args[0])
    expect(textes).toContain('VICTOIRE')
    expect(textes).toContain('EQUIPE EAGLE')
  })

  it('écrit le texte en `--foreground`, JAMAIS dans la couleur de camp', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    const verdict = ctx.__trace.find((e) => e.op === 'fillText' && e.args[0] === 'VICTOIRE')
    // Une couleur de camp peut être très claire : le contraste ne se négocie pas.
    expect(verdict?.fill).toBe(INK.foreground)
  })

  it('peint la carte du statut au fond et au bord de la couleur de camp', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    expect(ctx.__trace.find((e) => e.op === 'fill')?.fill).toBe(VICTOIRE.statusStyle.background)
    expect(ctx.__trace.find((e) => e.op === 'stroke')?.fill).toBe(VICTOIRE.statusStyle.border)
  })

  it('met le tiret du score dans l’encre atténuée, et les deux nombres dans l’encre du texte', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    const parNombre = (t: string) =>
      ctx.__trace.find((e) => e.op === 'fillText' && e.args[0] === t)?.fill
    expect(parNombre('50')).toBe(INK.foreground)
    expect(parNombre('43')).toBe(INK.foreground)
    expect(parNombre('-')).toBe(INK.muted)
  })

  it('écrit l’allié À GAUCHE de l’adverse — l’ordre du bandeau', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    const x = (t: string) =>
      ctx.__trace.find((e) => e.op === 'fillText' && e.args[0] === t)?.args[1] as number
    expect(x('50')).toBeLessThan(x('43'))
  })

  it('pose le filigrane SOUS le texte, et à 20 %', () => {
    const ctx = spyContext()
    const logo = {} as CanvasImageSource
    paintOverlayPanel(ctx, VIEW, { ...VICTOIRE, logo }, FONTS, INK)
    const iLogo = ctx.__trace.findIndex((e) => e.op === 'drawImage')
    const iTexte = ctx.__trace.findIndex((e) => e.op === 'fillText')
    expect(iLogo).toBeGreaterThanOrEqual(0)
    expect(iLogo).toBeLessThan(iTexte)
    expect(ctx.__trace[iLogo].alpha).toBeCloseTo(0.2)
  })

  it('ne peint AUCUN filigrane tant que le logo n’est pas chargé', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, VICTOIRE, FONTS, INK)
    // Le panneau n'a jamais eu besoin du logo pour dire comment le match s'est terminé.
    expect(ctx.__trace.some((e) => e.op === 'drawImage')).toBe(false)
  })

  it('centre la colonne verticalement, score compris', () => {
    const hautDuBloc = (p: OverlayPanel) => {
      const c = spyContext()
      paintOverlayPanel(c, VIEW, p, FONTS, INK)
      return c.__trace.find((e) => e.op === 'roundRect')?.args[1] as number
    }
    // Une colonne plus haute commence plus haut : c'est ce que fait `items-center`.
    expect(hautDuBloc(VICTOIRE)).toBeLessThan(hautDuBloc({ ...VICTOIRE, score: null, label: null }))
  })
})

describe('paintOverlayPanel — l’égalité et le message inter-manche', () => {
  it('l’égalité n’emprunte ni logo ni nom d’équipe', () => {
    const ctx = spyContext()
    paintOverlayPanel(
      ctx,
      VIEW,
      { status: 'Egalite', statusStyle: neutralStatusStyle(INK), score: { ally: 50, enemy: 50 }, veil: true },
      FONTS,
      INK,
    )
    expect(ctx.__trace.some((e) => e.op === 'drawImage')).toBe(false)
    expect(ctx.__trace.find((e) => e.op === 'fill')?.fill).toBe(INK.card)
    expect(ctx.__trace.find((e) => e.op === 'stroke')?.fill).toBe(INK.border)
  })

  it('le message inter-manche ne pose PAS de voile', () => {
    const ctx = spyContext()
    paintOverlayPanel(
      ctx,
      VIEW,
      { status: 'Manche 2 terminee', statusStyle: neutralStatusStyle(INK), veil: false },
      FONTS,
      INK,
    )
    // Le DOM n'en pose pas non plus : une manche qui s'achève ne masque pas le terrain.
    expect(ctx.__trace.some((e) => e.op === 'fillRect')).toBe(false)
    expect(ctx.__trace.some((e) => e.op === 'fillText')).toBe(true)
  })

  it('ne peint rien du tout quand il n’y a pas de statut à dire', () => {
    const ctx = spyContext()
    paintOverlayPanel(ctx, VIEW, { status: '   ', statusStyle: neutralStatusStyle(INK), veil: true }, FONTS, INK)
    expect(ctx.__trace).toHaveLength(0)
  })
})
