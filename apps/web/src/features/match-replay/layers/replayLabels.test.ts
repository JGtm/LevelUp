/**
 * replayLabels.test.ts : le nom sous le pion, et les icônes de titre qui le précèdent.
 *
 * Le contexte enregistreur mesure le texte à six pixels par caractère : la mise en page (icône,
 * compte, nom, centrés sous le pion) se lit alors sur les abscisses émises.
 */
import { describe, expect, it } from 'vitest'

import { recordingContext, type CanvasOp } from '../test/recordingContext'
import { TITLE_ICON_GAP_PX, TITLE_RUN_GAP_PX, type LabelTitle } from './antagonistMarks'
import { drawNameLabel } from './replayLabels'

const CHAR = 6
const C = { x: 100, y: 50 }
const EDGE = 5
const STYLE = { k: 1, labelStroke: 'encre-sombre' }

/** Une icône déjà teintée : 12 de large et de haut au corps, 2 de contour de chaque côté. */
const icon = { image: { tag: 'icone' } as unknown as CanvasImageSource, wPx: 16, hPx: 16 }

const titre = (count: number, over: Partial<LabelTitle> = {}): LabelTitle => ({
  icon,
  innerWPx: 12,
  count,
  alpha: 1,
  scale: 1,
  ...over,
})

function peint(name: string, titles?: readonly LabelTitle[], style = STYLE) {
  const { ops, ctx } = recordingContext({ charWidth: CHAR })
  drawNameLabel(ctx, C, name, titles ? { ...style, titles } : style, 'equipe', EDGE)
  return ops
}

const de = (ops: CanvasOp[], op: string) => ops.filter((o) => o.op === op)
const textes = (ops: CanvasOp[]) => de(ops, 'fillText').map((o) => o.args[0])

describe('drawNameLabel sans titre : comportement d’avant', () => {
  it('écrit le nom centré sous le pion, sans mesurer ni poser d’image', () => {
    const ops = peint('Brasier')
    expect(de(ops, 'drawImage')).toEqual([])
    expect(de(ops, 'measureText')).toEqual([])
    expect(ops.find((o) => o.op === 'set textAlign')?.args).toEqual(['center'])
    expect(de(ops, 'fillText')[0].args.slice(0, 2)).toEqual(['Brasier', C.x])
  })

  it('une liste vide est un nom seul', () => {
    expect(peint('Brasier', [])).toEqual(peint('Brasier'))
  })
})

describe('drawNameLabel avec titres : « icône compte nom »', () => {
  it('pose l’icône, puis le compte, puis le nom, dans cet ordre', () => {
    const ops = peint('Brasier', [titre(3)])
    const ordre = ops.filter((o) => o.op === 'drawImage' || o.op === 'fillText').map((o) => (o.op === 'drawImage' ? 'icone' : o.args[0]))
    expect(ordre).toEqual(['icone', '3', 'Brasier'])
  })

  it('contourne le compte comme le nom, avant de le remplir', () => {
    const ops = peint('Brasier', [titre(3)])
    const traits = de(ops, 'strokeText').map((o) => o.args[0])
    expect(traits).toEqual(['3', 'Brasier'])
    const sequence = ops.filter((o) => o.op === 'strokeText' || o.op === 'fillText').map((o) => `${o.op}:${o.args[0]}`)
    expect(sequence).toEqual(['strokeText:3', 'fillText:3', 'strokeText:Brasier', 'fillText:Brasier'])
  })

  it('centre l’ensemble sous le pion', () => {
    const ops = peint('Brasier', [titre(3)])
    const dessin = de(ops, 'drawImage')[0]
    const [, ix] = dessin.args as [unknown, number, number, number, number]
    const gauche = ix + 2 // le corps de l’icône commence après son contour (16 - 12) / 2
    const nom = de(ops, 'fillText').find((o) => o.args[0] === 'Brasier')!
    const droite = (nom.args[1] as number) + 'Brasier'.length * CHAR
    expect((gauche + droite) / 2).toBeCloseTo(C.x, 6)
    expect(ops.filter((o) => o.op === 'set textAlign').map((o) => o.args[0])).not.toContain('center')
  })

  it('espace l’icône, le compte et le nom de leurs écarts', () => {
    const ops = peint('Brasier', [titre(3)])
    const ix = (de(ops, 'drawImage')[0].args as number[])[1] + 2
    const compte = de(ops, 'fillText').find((o) => o.args[0] === '3')!
    const nom = de(ops, 'fillText').find((o) => o.args[0] === 'Brasier')!
    expect(compte.args[1]).toBeCloseTo(ix + 12 + TITLE_ICON_GAP_PX, 6)
    expect(nom.args[1]).toBeCloseTo((compte.args[1] as number) + CHAR + TITLE_RUN_GAP_PX, 6)
  })

  it('deux titres : deux icônes, némésis puis souffre-douleur, devant un seul nom', () => {
    const ops = peint('Brasier', [titre(3), titre(12)])
    expect(de(ops, 'drawImage')).toHaveLength(2)
    expect(textes(ops)).toEqual(['3', '12', 'Brasier'])
  })

  it('l’icône d’un titre qui n’est pas encore chargée garde sa place : le nom ne bouge pas à son arrivée', () => {
    const chargee = peint('Brasier', [titre(3)])
    const attente = peint('Brasier', [titre(3, { icon: null })])
    expect(de(attente, 'drawImage')).toEqual([])
    expect(de(attente, 'fillText').map((o) => o.args)).toEqual(de(chargee, 'fillText').map((o) => o.args))
  })

  it('l’opacité d’un titre qui s’efface ne touche que son icône et son compte', () => {
    const ops = peint('Brasier', [titre(3, { alpha: 0.4 })])
    const alphas = ops.filter((o) => o.op === 'set globalAlpha').map((o) => o.args[0])
    expect(alphas).toContain(0.4)
    const iNom = ops.findIndex((o) => o.op === 'fillText' && o.args[0] === 'Brasier')
    const dernierAvantNom = ops.slice(0, iNom).filter((o) => o.op === 'set globalAlpha').pop()
    expect(dernierAvantNom?.args).toEqual([1])
  })

  it('une icône qui pulse grossit autour de son centre sans pousser le nom', () => {
    const repos = peint('Brasier', [titre(3)])
    const pulse = peint('Brasier', [titre(3, { scale: 1.5 })])
    const a = de(repos, 'drawImage')[0].args as number[]
    const b = de(pulse, 'drawImage')[0].args as number[]
    expect(b[3]).toBeCloseTo(a[3] * 1.5, 6)
    expect(b[4]).toBeCloseTo(a[4] * 1.5, 6)
    expect(b[1] + b[3] / 2).toBeCloseTo(a[1] + a[3] / 2, 6)
    expect(b[2] + b[4] / 2).toBeCloseTo(a[2] + a[4] / 2, 6)
    expect(de(pulse, 'fillText').map((o) => o.args)).toEqual(de(repos, 'fillText').map((o) => o.args))
  })

  it('tout suit la densité de l’écran', () => {
    const un = peint('Brasier', [titre(3)])
    const deux = peint('Brasier', [titre(3)], { ...STYLE, k: 2 })
    const a = de(un, 'drawImage')[0].args as number[]
    const b = de(deux, 'drawImage')[0].args as number[]
    expect(b[3]).toBeCloseTo(a[3] * 2, 6)
  })

  it('sans encre de contour, le compte et le nom ne sont pas contournés', () => {
    const ops = peint('Brasier', [titre(3)], { k: 1, labelStroke: '' })
    expect(de(ops, 'strokeText')).toEqual([])
    expect(textes(ops)).toEqual(['3', 'Brasier'])
  })

  it('le nom reste sous le bord du pion', () => {
    const sans = peint('Brasier')
    const avec = peint('Brasier', [titre(3)])
    const ySans = de(sans, 'fillText')[0].args[2] as number
    const yNom = (de(avec, 'fillText').find((o) => o.args[0] === 'Brasier')!.args[2]) as number
    expect(yNom).toBeCloseTo(ySans, 6)
    const yIcone = (de(avec, 'drawImage')[0].args as number[])[2]
    // Le corps de l’icône (sans sa marge de contour transparente) est sous le bord du pion.
    expect(yIcone + 2).toBeGreaterThan(C.y + EDGE)
  })
})
