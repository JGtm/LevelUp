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

import { drawShotEffect, familyOf, type ShotFamily, type ShotShape } from './shotEffects'
import { count, recordingContext, valuesOf, type CanvasOp } from './test/recordingContext'


// Le contexte enregistreur vit dans `test/recordingContext.ts` depuis le 2026-08-15 :
// l'éclair de bouche (muzzleFlash.test.ts) en a besoin du même, et une deuxième copie
// aurait appelé la troisième.

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
    // ordre. Ce qui les sépare est le POIDS du trait (2,2 px contre 1, recalage POC 2.2) et
    // son opacité — rien d'autre. Le sujet est consigné : « huit formes » est exact au sens des
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
    // Largeur recalée sur le POC (2.2) — lot 2 item 2.2, avec les deux horloges (éclat
    // au carré, trait en puissance 1,5).
    expect(valuesOf(ops, 'lineWidth')).toEqual([2.2])
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
    // Largeurs recalées sur le POC (6,5 / 2) — lot 2 item 2.2.
    expect(valuesOf(ops, 'lineWidth')).toEqual([6.5, 2])
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

  it('mêlée : deux arcs courts, et AUCUN éclair de bouche — le geste n’est pas un tir', () => {
    const ops = trace('melee')
    expect(count(ops, 'lineTo')).toBe(0)
    expect(count(ops, 'fill')).toBe(0)
    // Deux arcs concentriques (recalage POC, lot 2 item 2.2) ; SANS extrémité réelle
    // (`target` absent : un tir), l'arc de liaison ne se dessine JAMAIS.
    expect(count(ops, 'arc')).toBe(2)
    expect(count(ops, 'quadraticCurveTo')).toBe(0)
    const arc = ops.find((o) => o.op === 'arc')!
    const [, , , a0, a1] = arc.args as number[]
    expect(a1 - a0).toBeCloseTo(1.8) // ±0,9 rad : un arc, pas un cercle
  })

  it('mêlée à PORTÉE (mort, victime sous 8 m) : l’arc de liaison rejoint la victime', () => {
    const ops = trace('melee', { target: true, meleeLink: true })
    expect(count(ops, 'quadraticCurveTo')).toBe(1)
    // Hors de portée, même sur une mort orientée : pas de liaison — relier un marteau à
    // une victime à 20 m affirmerait un contact qui n'a pas eu lieu.
    const loin = trace('melee', { target: true, meleeLink: false })
    expect(count(loin, 'quadraticCurveTo')).toBe(0)
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

  it('la famille vient du DOCUMENT, et une famille hors rendu reste sobre', () => {
    // Depuis le lot 3.2 la famille est publiée par l'artefact (`weaponLabels[id].fx`),
    // résolue depuis les mappings du titre : le rendu ne connaît plus une seule arme.
    expect(familyOf('needles')).toBe('needles')
    const connue = JSON.stringify(trace(familyOf('needles')).map((o) => o.op))
    const inconnue = JSON.stringify(trace(familyOf('famille-inventee-9000')).map((o) => o.op))
    expect(connue).not.toBe(inconnue)
  })
})

