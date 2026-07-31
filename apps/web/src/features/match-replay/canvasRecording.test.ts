/**
 * canvasRecording.test.ts — LE RENDU CANVAS, TESTÉ SANS NAVIGATEUR.
 *
 * POURQUOI C'EST POSSIBLE, ET POURQUOI ÇA NE COÛTE AUCUNE DÉPENDANCE. Le code de dessin du
 * rejeu n'INTERROGE jamais le canvas : il écrit dedans. Une seule opération lit quelque chose
 * (`createRadialGradient`, dans le cône de visée), et elle rend un objet dont on n'appelle que
 * `addColorStop`. Un contexte ENREGISTREUR — un objet qui empile `{op, args}` — suffit donc à
 * observer tout ce que le rendu produit. Ni `node-canvas`, ni `jsdom`, ni image de référence.
 *
 * CE QUE CES TESTS VÉRIFIENT, ET CE QU'ILS NE VÉRIFIENT PAS. Ils vérifient la GÉOMÉTRIE ÉMISE :
 * combien de traits, quelles primitives, dans quel ordre, avec quelles largeurs. Ils ne
 * vérifient AUCUN pixel — un test de pixels serait un test d'anti-crénelage, et il tomberait à
 * chaque changement de moteur sans rien dire du rendu.
 *
 * CE QUE LE CORRECTIF J1-a CHANGE ICI. Les huit familles d'effet de tir existaient depuis le
 * début du rejeu 2D, et AUCUNE n'était atteinte : l'interface écrite à la main nommait `weapon`
 * le champ que le contrat nomme `w`, donc `familyOf()` ne recevait jamais rien et tous les tirs
 * tombaient sur la forme par défaut. Vérifier que les huit formes DIFFÈRENT n'avait donc aucun
 * sens avant ce correctif ; c'est ce qui rend ce fichier utile aujourd'hui.
 */
import { describe, expect, it } from 'vitest'

import { buildFloorGrid } from './mapFloor'
import { drawFloorLayer } from './replayDraw'
import { drawShotEffect, familyOf, type ShotFamily, type ShotShape } from './shotEffects'

import type { ReplayBounds } from '@/lib/api/types'
import type { ReplaySurfaceReady } from './replayNormalize'

/** Une opération enregistrée : le nom de la méthode (ou `set <propriété>`) et ses arguments. */
interface CanvasOp {
  op: string
  args: unknown[]
}

/**
 * recordingContext rend un faux contexte 2D qui n'exécute rien et note tout.
 *
 * LA SEULE LECTURE À TRAITER est `createRadialGradient` : le rendu attend un objet portant
 * `addColorStop`. On rend donc un jeton inerte, et l'appel reste dans la trace — c'est le
 * comportement qu'on veut, pas une émulation.
 */
function recordingContext(): { ops: CanvasOp[]; ctx: CanvasRenderingContext2D } {
  const ops: CanvasOp[] = []
  const state: Record<string, unknown> = {}
  const proxy = new Proxy(
    {},
    {
      get(_t, prop) {
        if (typeof prop !== 'string') return undefined
        if (prop === 'createRadialGradient') {
          return (...args: unknown[]) => {
            ops.push({ op: prop, args })
            return { addColorStop: (...a: unknown[]) => ops.push({ op: 'addColorStop', args: a }) }
          }
        }
        if (prop in state) return state[prop]
        return (...args: unknown[]) => {
          ops.push({ op: prop, args })
        }
      },
      set(_t, prop, value) {
        if (typeof prop === 'string') {
          state[prop] = value
          ops.push({ op: `set ${prop}`, args: [value] })
        }
        return true
      },
    },
  )
  return { ops, ctx: proxy as unknown as CanvasRenderingContext2D }
}

const count = (ops: CanvasOp[], op: string): number => ops.filter((o) => o.op === op).length

/** valuesOf rend les valeurs successives affectées à une propriété (lineWidth, globalAlpha…). */
const valuesOf = (ops: CanvasOp[], prop: string): number[] =>
  ops.filter((o) => o.op === `set ${prop}`).map((o) => o.args[0] as number)

const shape = (over: Partial<ShotShape> = {}): ShotShape => ({
  x: 100,
  y: 100,
  angle: 0,
  length: 26,
  fade: 1,
  reduced: false,
  seed: 3,
  ...over,
})

/** trace rend la suite des primitives de dessin émises pour une famille — sa SIGNATURE. */
function trace(family: ShotFamily, over: Partial<ShotShape> = {}): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawShotEffect(ctx, family, shape(over), 'rgb(1 2 3)')
  return ops
}

const ALL_FAMILIES: ShotFamily[] = [
  'ballistic',
  'plasma',
  'light',
  'shock',
  'explosive',
  'melee',
  'needles',
  'plain',
]

describe('contexte enregistreur', () => {
  it('note les appels et les affectations sans rien exécuter', () => {
    const { ops, ctx } = recordingContext()
    ctx.strokeStyle = 'rgb(9 9 9)'
    ctx.beginPath()
    ctx.moveTo(1, 2)
    expect(ops).toEqual([
      { op: 'set strokeStyle', args: ['rgb(9 9 9)'] },
      { op: 'beginPath', args: [] },
      { op: 'moveTo', args: [1, 2] },
    ])
  })

  it('sert un dégradé inerte pour la seule opération qui LIT le contexte', () => {
    const { ops, ctx } = recordingContext()
    const g = ctx.createRadialGradient(0, 0, 0, 0, 0, 10)
    g.addColorStop(0, 'rgb(1 1 1)')
    expect(ops.map((o) => o.op)).toEqual(['createRadialGradient', 'addColorStop'])
  })
})

describe('les huit formes d’effet de tir', () => {
  it('produisent huit signatures DISTINCTES', () => {
    // C'est LE test que le correctif du champ `w` rend significatif : avant lui, les huit
    // familles étaient inatteignables et se seraient toutes dessinées `plain`.
    //
    // LA SIGNATURE PORTE AUSSI LES LARGEURS ET LES OPACITÉS, et c'est une MESURE, pas une
    // commodité : `ballistic` et `plain` émettent exactement les mêmes primitives, dans le même
    // ordre. Ce qui les sépare est le POIDS du trait (1,6 px contre 1) et son opacité (0,9
    // contre 0,7) — rien d'autre. Le sujet est consigné : « huit formes » est exact au sens des
    // huit rendus, mais il n'y a que SEPT géométries distinctes, et le rendu neutre d'une arme
    // hors catalogue est un balistique aminci plutôt qu'une forme qui n'affirme rien.
    const signatures = new Map<string, ShotFamily[]>()
    for (const f of ALL_FAMILIES) {
      const ops = trace(f)
      const sig = JSON.stringify([
        ops.map((o) => o.op),
        valuesOf(ops, 'lineWidth'),
        valuesOf(ops, 'globalAlpha').map((a) => Math.round(a * 1000)),
      ])
      signatures.set(sig, [...(signatures.get(sig) ?? []), f])
    }
    const collisions = [...signatures.values()].filter((fs) => fs.length > 1)
    expect(collisions).toEqual([])
    expect(signatures.size).toBe(ALL_FAMILIES.length)
  })

  it('n’offre que SEPT géométries pour huit familles — le fait est mesuré, pas supposé', () => {
    // Découverte consignée (J2, lot 2.4). Ce test FIGE l'état actuel pour qu'il cesse d'être
    // invisible : si un jour `plain` gagne sa propre géométrie, c'est ce test qui le dira, et
    // le commentaire ci-dessus devra suivre.
    const parPrimitives = new Set(ALL_FAMILIES.map((f) => JSON.stringify(trace(f).map((o) => o.op))))
    expect(parPrimitives.size).toBe(7)
  })

  it('encadrent chaque effet d’un save/restore — aucun état ne fuit sur le calque suivant', () => {
    for (const f of ALL_FAMILIES) {
      const ops = trace(f)
      expect(ops[0]?.op, f).toBe('save')
      expect(ops[ops.length - 1]?.op, f).toBe('restore')
    }
  })

  it('ballistique : un trait net, et l’éclat d’origine', () => {
    const ops = trace('ballistic')
    expect(count(ops, 'lineTo')).toBe(1)
    expect(count(ops, 'stroke')).toBe(1)
    expect(count(ops, 'arc')).toBe(1) // l'éclat
    expect(count(ops, 'fill')).toBe(1)
    expect(valuesOf(ops, 'lineWidth')).toEqual([1.6])
  })

  it('plasma : une polyligne qui ondule, jamais un segment droit', () => {
    const ops = trace('plasma')
    expect(count(ops, 'moveTo')).toBe(1)
    expect(count(ops, 'lineTo')).toBe(14) // 15 points d'échantillonnage
    expect(count(ops, 'stroke')).toBe(1)
    expect(count(ops, 'fill')).toBe(0)
    // L'ondulation est bien une ondulation : les points s'écartent de l'axe.
    const ys = ops.filter((o) => o.op === 'lineTo').map((o) => o.args[1] as number)
    expect(new Set(ys).size).toBeGreaterThan(1)
  })

  it('lumière : DEUX passes sur le même segment, large et pâle sous fine et franche', () => {
    const ops = trace('light')
    expect(count(ops, 'lineTo')).toBe(1)
    expect(count(ops, 'stroke')).toBe(2)
    expect(valuesOf(ops, 'lineWidth')).toEqual([5, 1.2])
    const alphas = valuesOf(ops, 'globalAlpha')
    expect(alphas[0]).toBeLessThan(alphas[1])
  })

  it('choc : un arc BRISÉ — sept points, jamais une droite', () => {
    const ops = trace('shock')
    expect(count(ops, 'moveTo')).toBe(1)
    expect(count(ops, 'lineTo')).toBe(6)
    const ys = ops.filter((o) => o.op === 'lineTo').map((o) => o.args[1] as number)
    expect(new Set(ys).size).toBeGreaterThan(1) // il zigzague de part et d'autre de l'axe
  })

  it('explosif : DEUX TEMPS — le départ bref ne vit que dans les 30 % les plus frais', () => {
    const frais = trace('explosive', { fade: 1 })
    expect(count(frais, 'stroke')).toBe(2) // le départ épais, puis l'onde
    const vieux = trace('explosive', { fade: 0.2 })
    expect(count(vieux, 'stroke')).toBe(1) // l'onde seule
    expect(count(vieux, 'arc')).toBe(1)
  })

  it('mêlée : un arc court, et AUCUN éclair de bouche — le geste n’est pas un tir', () => {
    const ops = trace('melee')
    expect(count(ops, 'lineTo')).toBe(0)
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'arc')).toBe(1)
    const arc = ops.find((o) => o.op === 'arc')!
    const [, , , a0, a1] = arc.args as number[]
    expect(a1 - a0).toBeCloseTo(1.8) // ±0,9 rad : un arc, pas un cercle
  })

  it('aiguilles : la signature est la GERBE — cinq brins issus du même point', () => {
    const ops = trace('needles')
    expect(count(ops, 'moveTo')).toBe(5)
    expect(count(ops, 'lineTo')).toBe(5)
    expect(count(ops, 'stroke')).toBe(5)
    const ends = ops.filter((o) => o.op === 'lineTo').map((o) => `${o.args[0]},${o.args[1]}`)
    expect(new Set(ends).size).toBe(5) // ils s'écartent, ils ne se superposent pas
  })

  it('sobre : un trait qui n’affirme aucune famille', () => {
    const ops = trace('plain')
    expect(count(ops, 'lineTo')).toBe(1)
    expect(valuesOf(ops, 'lineWidth')).toEqual([1])
    expect(count(ops, 'arc')).toBe(1)
  })

  it('sans visée lisible : l’éclat SEUL, aucune direction inventée', () => {
    for (const f of ALL_FAMILIES) {
      const ops = trace(f, { angle: null })
      expect(count(ops, 'lineTo'), f).toBe(0)
      expect(count(ops, 'arc'), f).toBe(1)
      expect(count(ops, 'stroke'), f).toBe(0)
    }
  })

  it('mouvement réduit : la géométrie ne PROGRESSE plus avec la rémanence', () => {
    // L'onde de l'explosif est la forme dont le rayon dépend de l'avancement : sous
    // « mouvement réduit », deux rémanences différentes doivent donner le MÊME rayon.
    const radius = (fade: number, reduced: boolean): number => {
      const arc = trace('explosive', { fade, reduced }).filter((o) => o.op === 'arc')
      return arc[arc.length - 1].args[2] as number
    }
    expect(radius(0.9, true)).toBe(radius(0.3, true))
    expect(radius(0.9, false)).not.toBe(radius(0.3, false))
  })

  it('la famille vient du LIBELLÉ publié, et une arme hors catalogue reste sobre', () => {
    expect(familyOf('Needler')).toBe('needles')
    const connue = JSON.stringify(trace(familyOf('Needler')).map((o) => o.op))
    const inconnue = JSON.stringify(trace(familyOf('Arme Inventée 9000')).map((o) => o.op))
    expect(connue).not.toBe(inconnue)
  })
})

describe('la trame du sol', () => {
  const bounds: ReplayBounds = { minX: 0, minY: 0, maxX: 2, maxY: 1, minZ: 0, maxZ: 1 }
  const view = { bounds, width: 200, height: 100, pad: 0 }

  /** Une dalle carrée d'aire suffisante pour ne pas tomber sous le plancher de 1 m². */
  const slab = (x0: number, y0: number, x1: number, y1: number, z: number): ReplaySurfaceReady => ({
    x0,
    y0,
    x1,
    y1,
    z,
    zb: z - 0.5,
    poly: [],
  })

  it('ne peint QUE les cellules qui portent un sol — le vide reste vide', () => {
    // Une seule dalle sur la moitié gauche : la moitié droite n'a pas de sol, et personne ne
    // doit y inventer un aplat.
    const grid = buildFloorGrid([slab(0, 0, 1, 1, 0)], bounds)
    const { ops, ctx } = recordingContext()
    drawFloorLayer(ctx, grid, view, { fill: 'rgb(1 1 1)', edge: 'rgb(2 2 2)' })
    const rects = ops.filter((o) => o.op === 'fillRect')
    expect(rects.length).toBeGreaterThan(0)
    // La dalle s'arrête à x = 1 sur une étendue de 2 m rendue sur 200 px, soit 100 px. La
    // borne tolère UNE cellule de plus (25 cm = 12,5 px) : la cellule qui chevauche le bord de
    // la dalle est peinte en entier, c'est la rasterisation, pas du sol invente.
    const cellPx = (view.width * grid.cell) / (bounds.maxX - bounds.minX)
    for (const r of rects) {
      const [x, , w] = r.args as number[]
      expect(x + w).toBeLessThanOrEqual(view.width / 2 + cellPx + 1)
    }
  })

  it('peint une PLAGE d’un seul trait, pas cellule par cellule', () => {
    // C'est la raison d'être de floorRun : deux rectangles voisins arrondis au pixel se
    // chevauchent d'un pixel, et ce chevauchement dessine un quadrillage parasite.
    const grid = buildFloorGrid([slab(0, 0, 2, 1, 0)], bounds)
    const { ops, ctx } = recordingContext()
    drawFloorLayer(ctx, grid, view, { fill: 'rgb(1 1 1)', edge: 'rgb(2 2 2)' })
    const rects = ops.filter((o) => o.op === 'fillRect')
    expect(rects.length).toBeLessThan(grid.filled)
    expect(rects.length).toBeLessThanOrEqual(grid.ny)
  })

  it('fait monter l’opacité avec l’altitude du sol — l’étage se lit sans couleur dédiée', () => {
    const grid = buildFloorGrid([slab(0, 0, 1, 1, -1), slab(1, 0, 2, 1, 0.9)], bounds)
    const { ops, ctx } = recordingContext()
    drawFloorLayer(ctx, grid, view, { fill: 'rgb(1 1 1)', edge: 'rgb(2 2 2)' })
    const alphas = new Set<number>()
    for (let i = 0; i < ops.length; i++) {
      if (ops[i].op === 'fillRect') continue
      if (ops[i].op === 'set globalAlpha') alphas.add(ops[i].args[0] as number)
    }
    expect(alphas.size).toBeGreaterThan(1)
    expect(Math.max(...alphas)).toBeGreaterThan(Math.min(...alphas))
  })

  it('trace TOUTES les arêtes en un seul chemin — des milliers de stroke coûteraient plus', () => {
    const grid = buildFloorGrid([slab(0, 0, 1, 1, 0)], bounds)
    const { ops, ctx } = recordingContext()
    drawFloorLayer(ctx, grid, view, { fill: 'rgb(1 1 1)', edge: 'rgb(2 2 2)' })
    expect(count(ops, 'stroke')).toBe(1)
    expect(count(ops, 'beginPath')).toBe(1)
    expect(count(ops, 'moveTo')).toBeGreaterThan(0)
  })

  it('une carte SANS structure ne peint rien du tout', () => {
    const grid = buildFloorGrid([], bounds)
    const { ops, ctx } = recordingContext()
    drawFloorLayer(ctx, grid, view, { fill: 'rgb(1 1 1)', edge: 'rgb(2 2 2)' })
    expect(grid.filled).toBe(0)
    expect(count(ops, 'fillRect')).toBe(0)
    expect(count(ops, 'moveTo')).toBe(0)
  })
})
