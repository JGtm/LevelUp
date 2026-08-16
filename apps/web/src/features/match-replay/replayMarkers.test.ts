/**
 * replayMarkers.test.ts — CE QUE LE MARQUEUR ÉMET, vérifié sans navigateur.
 *
 * Même outil que le reste du rendu (contexte enregistreur, `test/recordingContext.ts`) :
 * on observe la GÉOMÉTRIE ÉMISE — quelles primitives, combien, avec quels réglages — jamais
 * un pixel. Ce fichier garde les décisions d'habillage du 2026-08-16 :
 *   - le NOM sous le point est cerné AVANT d'être rempli (D4 : lisible du blanc au noir) ;
 *   - le BÂTON n'existe plus (D3 : le calque de visée n'émet plus aucun segment) ;
 *   - la FORME dit l'identité (D5 : losange pour un ami, anneau de plus pour « moi ») ;
 *   - le calque des noms s'éteint (D4 : un BTB à 24 joueurs doit pouvoir le taire) ;
 *   - le STYLE DE LA PLANCHE validée le soir même (§1bis du plan) : plus de halo, traînée
 *     dessinée segment par segment à opacité croissante, croix de mort de taille FIXE, cône
 *     de visée à 0,42 rad de demi-ouverture.
 */
import { describe, expect, it } from 'vitest'

import { drawTracksLayer, type MarkerStyle } from './replayMarkers'
import type { ReplayTrackReady } from './replayNormalize'
import type { PlayerMarkKind } from './playerMarks'
import { count, recordingContext, valuesOf, type CanvasOp } from './test/recordingContext'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 200,
  height: 100,
  pad: 4,
}

/**
 * UNE vie d'UN SEUL point : la traînée a besoin de deux positions pour tracer un segment,
 * donc ce gabarit garantit qu'aucun `lineTo` de traînée ne vient polluer les comptages de
 * forme et de visée. `h` est le regard, la mesure dont le cône se sert.
 */
function singlePointTrack(slot: number, over: Partial<ReplayTrackReady> = {}): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid: 'A',
    startFrame: 0,
    endFrame: 100,
    points: [{ t: 50, x: 5, y: 5, z: 0, h: 90 }],
    ...over,
  }
}

function style(over: Partial<MarkerStyle> = {}): MarkerStyle {
  return {
    colorOfSlot: () => 'rgb(1 2 3)',
    ink: 'rgb(9 9 9)',
    // Frame LOIN du départ de la vie : l'anneau d'apparition ne doit pas s'ajouter aux
    // comptages d'arcs (il vit 0,8 s après le spawn).
    frame: 50,
    timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 },
    // Amplitude verticale connue et joueur AU SOL : aucun anneau d'étage ne s'ajoute aux
    // comptages de formes (une carte plate rendrait, elle, l'étage médian).
    z: { min: 0, max: 10 },
    k: 1,
    showAim: false,
    markOfSlot: () => undefined,
    nameOfSlot: () => 'Spartan',
    showNames: true,
    labelStroke: 'rgb(8 12 18)',
    ...over,
  }
}

/** trace rend les primitives émises par le calque pour une vie et un style donnés. */
function trace(over: Partial<MarkerStyle> = {}, track = singlePointTrack(512)): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawTracksLayer(ctx, [track], VIEW, style(over))
  return ops
}

/** indexOfOp rend le rang du premier appel d'une primitive (-1 si elle n'est pas émise). */
const indexOfOp = (ops: CanvasOp[], op: string): number => ops.findIndex((o) => o.op === op)

describe('étiquette de nom (D4)', () => {
  it('cerne le nom AVANT de le remplir, avec des jonctions rondes', () => {
    const ops = trace()
    const stroke = indexOfOp(ops, 'strokeText')
    const fill = indexOfOp(ops, 'fillText')
    expect(stroke).toBeGreaterThanOrEqual(0)
    expect(fill).toBeGreaterThan(stroke)
    expect(ops[stroke].args[0]).toBe('Spartan')
    expect(ops[fill].args[0]).toBe('Spartan')
    // Le contour est posé APRÈS la mise en place du lineJoin rond : sans lui, les jonctions
    // de lettres font des pointes qui grossissent le nom.
    const joins = ops.filter((o) => o.op === 'set lineJoin' && o.args[0] === 'round')
    expect(joins.length).toBeGreaterThan(0)
    expect(ops.indexOf(joins[joins.length - 1])).toBeLessThan(stroke)
  })

  it('centre le texte sous le point et l aligne par son HAUT (il descend, jamais il ne remonte)', () => {
    const ops = trace()
    expect(ops.some((o) => o.op === 'set textAlign' && o.args[0] === 'center')).toBe(true)
    expect(ops.some((o) => o.op === 'set textBaseline' && o.args[0] === 'top')).toBe(true)
    const fill = ops[indexOfOp(ops, 'fillText')]
    const arc = ops[indexOfOp(ops, 'arc')]
    expect(fill.args[2] as number).toBeGreaterThan(arc.args[1] as number)
  })

  it('n écrit rien quand le calque des noms est éteint', () => {
    const ops = trace({ showNames: false })
    expect(count(ops, 'fillText')).toBe(0)
    expect(count(ops, 'strokeText')).toBe(0)
  })

  it('n écrit rien pour une vie sans propriétaire (aucun nom à donner)', () => {
    const ops = trace({ nameOfSlot: () => null })
    expect(count(ops, 'fillText')).toBe(0)
  })
})

describe('visée (D3/D3prime) — le bâton a disparu, le cône s ouvre à 0,42 rad', () => {
  it('le cône allumé n émet AUCUN segment de droite', () => {
    const ops = trace({ showAim: true })
    // Le cône est un secteur : moveTo + arc + closePath + fill dégradé. Un seul `lineTo`
    // signerait le retour de l'axe supprimé.
    expect(count(ops, 'lineTo')).toBe(0)
    expect(count(ops, 'createRadialGradient')).toBe(1)
  })

  it('ouvre le secteur de 0,42 rad de part et d autre du regard', () => {
    // Marqueur en LOSANGE : il n'émet aucun `arc`, donc le seul du relevé est le secteur.
    const ops = trace({ showAim: true, markOfSlot: () => 'friend' })
    const sector = ops.find((o) => o.op === 'arc')
    expect(sector).toBeDefined()
    const [, , radius, from, to] = sector!.args as number[]
    expect(to - from).toBeCloseTo(0.84, 6)
    // Rayon de la planche « un peu plus prononcé » : 52 px d'écran (k = 1 dans ce test).
    expect(radius).toBeCloseTo(52, 6)
  })
})

describe('formes d identité (D5)', () => {
  const markOf = (kind: PlayerMarkKind | undefined) => () => kind

  it('un marqueur ordinaire n est fait que d arcs', () => {
    const ops = trace({ markOfSlot: markOf(undefined) })
    expect(count(ops, 'lineTo')).toBe(0)
    // Liseré + noyau : deux disques, aucun anneau d'étage (le joueur est au sol) et PLUS DE
    // HALO — la lueur diffuse a été supprimée par la planche du 2026-08-16.
    expect(count(ops, 'arc')).toBe(2)
  })

  it('un AMI est un losange : quatre segments par chemin, liseré et noyau', () => {
    const ops = trace({ markOfSlot: markOf('friend') })
    expect(count(ops, 'lineTo')).toBe(8)
    expect(count(ops, 'closePath')).toBe(2)
    // Aucun disque ne subsiste : le halo qui en dessinait un a disparu avec la planche.
    expect(count(ops, 'arc')).toBe(0)
  })

  it('MOI porte un arc de plus que le marqueur ordinaire — son anneau externe', () => {
    const plain = count(trace({ markOfSlot: markOf(undefined) }), 'arc')
    const mine = count(trace({ markOfSlot: markOf('me') }), 'arc')
    expect(mine).toBe(plain + 1)
  })
})

/** alphaBefore rend l'opacité en vigueur au moment d'une primitive donnée (rang `at`). */
function alphaBefore(ops: CanvasOp[], at: number): number {
  for (let i = at - 1; i >= 0; i--) {
    if (ops[i].op === 'set globalAlpha') return ops[i].args[0] as number
  }
  return 1
}

describe('style de la planche (§1bis)', () => {
  it('dessine la traînée SEGMENT PAR SEGMENT, opacité croissante vers le présent', () => {
    const walking = singlePointTrack(512, {
      points: [
        { t: 40, x: 1, y: 1, z: 0, h: 90 },
        { t: 45, x: 3, y: 3, z: 0, h: 90 },
        { t: 50, x: 5, y: 5, z: 0, h: 90 },
      ],
    })
    const ops = trace({}, walking)
    // Trois positions = deux segments = deux `stroke` (le marqueur, lui, se REMPLIT).
    const strokes = ops.map((o, i) => ({ o, i })).filter((e) => e.o.op === 'stroke')
    expect(strokes).toHaveLength(2)
    const alphas = strokes.map((e) => alphaBefore(ops, e.i))
    expect(alphas[0]).toBeCloseTo(0.355, 6)
    expect(alphas[1]).toBeCloseTo(0.63, 6)
    // Un seul `stroke` global à opacité constante était l'ancien rendu : il ne disait pas le
    // SENS du déplacement.
    expect(alphas[1]).toBeGreaterThan(alphas[0])
    expect(valuesOf(ops, 'lineWidth')).toContain(1.6)
  })

  it('la croix de mort garde sa TAILLE en s estompant (elle ne grandit plus)', () => {
    const dead = singlePointTrack(512, { endFrame: 50 })
    const early = trace({ frame: 55 }, dead)
    const late = trace({ frame: 65 }, dead)
    expect(count(early, 'moveTo')).toBe(2)
    expect(count(early, 'lineTo')).toBe(2)
    const corners = (ops: CanvasOp[]) =>
      ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo').map((o) => o.args)
    expect(corners(late)).toEqual(corners(early))
    // Seule l'opacité bouge : plus la mort est ancienne, plus la croix est pâle.
    const alphaOf = (ops: CanvasOp[]) => alphaBefore(ops, ops.findIndex((o) => o.op === 'stroke'))
    expect(alphaOf(late)).toBeLessThan(alphaOf(early))
    expect(valuesOf(early, 'lineWidth')).toContain(1.6)
  })

  it('marque l étage par un anneau à l ENCRE DU THÈME, pas à la couleur du joueur', () => {
    // Joueur en HAUT d'une carte d'amplitude connue : l'étage le plus élevé porte ses anneaux.
    const ops = trace({ z: { min: 0, max: 10 } }, singlePointTrack(512, {
      points: [{ t: 50, x: 5, y: 5, z: 10, h: 90 }],
    }))
    const strokeStyles = ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(strokeStyles).toContain('rgb(9 9 9)')
    expect(strokeStyles).not.toContain('rgb(1 2 3)')
    expect(valuesOf(ops, 'lineWidth')).toContain(1)
  })
})

describe('couleur par SLOT (D1)', () => {
  it('une vie dont le slot n a pas de couleur n est pas dessinée du tout', () => {
    const ops = trace({ colorOfSlot: () => null })
    expect(count(ops, 'arc')).toBe(0)
    expect(count(ops, 'fillText')).toBe(0)
  })

  it('demande la couleur du SLOT de la vie, jamais de son rang dans le document', () => {
    const asked: number[] = []
    trace({
      colorOfSlot: (slot) => {
        asked.push(slot)
        return 'rgb(1 2 3)'
      },
    }, singlePointTrack(517))
    expect(asked).toEqual([517])
  })
})
