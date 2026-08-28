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
    showTrail: true,
    selfInk: 'rgb(4 4 4)',
    deathInk: 'rgb(5 5 5)',
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
  it('aucun segment ne PART du marqueur : le bâton reste supprimé', () => {
    const ops = trace({ showAim: true })
    // La règle D3 porte sur l'AXE — un trait issu du point. Depuis le schéma 13 le calque
    // émet un segment (le tick d'élévation), mais il vit à la POINTE du cône : aucun `lineTo`
    // ne doit suivre un `moveTo` posé sur le centre du marqueur. Le compter, comme avant,
    // aurait interdit le tick ; vérifier son ORIGINE interdit ce que D3 interdisait vraiment.
    const centre = ops.find((o) => o.op === 'arc')!.args as number[]
    ops.forEach((o, i) => {
      if (o.op !== 'lineTo') return
      const prev = ops[i - 1]
      if (prev?.op !== 'moveTo') return
      const [mx, my] = prev.args as number[]
      expect(Math.hypot(mx - centre[0], my - centre[1])).toBeGreaterThan(1)
    })
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

/**
 * L'ÉLÉVATION (schéma 13) : le cône raccourcit, le tick dit le sens.
 *
 * `AIM_LENGTH` (52) reste la longueur d'une visée à plat ; l'élévation la multiplie par
 * `max(0,35 ; cos p)`. Le rayon du secteur est donc l'observable, et le tick — un segment posé
 * à la pointe — porte le signe que le cosinus, étant pair, ne peut pas porter.
 */
describe('élévation de visée (schéma 13)', () => {
  /** viséeInclinée : une vie d'un point, cap plein est (h = 0), élévation donnée. */
  const inclinee = (p?: number) =>
    singlePointTrack(512, { points: [{ t: 50, x: 5, y: 5, z: 0, h: 360, ...(p === undefined ? {} : { p }) }] })

  /** rayonDuCone : le 3e argument du seul `arc` du relevé (marqueur losange = zéro autre arc). */
  const rayonDuCone = (p?: number): number => {
    const ops = trace({ showAim: true, markOfSlot: () => 'friend' }, inclinee(p))
    return (ops.find((o) => o.op === 'arc')!.args as number[])[2]
  }

  it('garde sa pleine longueur à plat, et quand l artefact ne porte pas d élévation', () => {
    expect(rayonDuCone(0)).toBeCloseTo(52, 6)
    // Artefact antérieur au schéma 13 : `p` absent se lit « à plat », jamais « inconnu ».
    expect(rayonDuCone(undefined)).toBeCloseTo(52, 6)
  })

  it('raccourcit du COSINUS de l élévation — 60° = la moitié', () => {
    expect(rayonDuCone(60)).toBeCloseTo(26, 6)
    // Le cosinus est PAIR : plonger de 60° raccourcit exactement autant que viser 60° en l'air.
    expect(rayonDuCone(-60)).toBeCloseTo(rayonDuCone(60), 6)
  })

  it('s arrête au plancher de 35 % à la verticale (le marqueur doit rester lisible)', () => {
    expect(rayonDuCone(90)).toBeCloseTo(52 * 0.35, 6)
    expect(rayonDuCone(-90)).toBeCloseTo(52 * 0.35, 6)
    // 80° tomberait à 9 px sans plancher : c'est lui qui tient, pas le cosinus.
    expect(rayonDuCone(80)).toBeCloseTo(52 * 0.35, 6)
  })

  /**
   * tick : le segment d'élévation, en distances au centre du marqueur.
   *
   * Marqueur ORDINAIRE ici (pas le losange qui sert au rayon) : le disque n'émet aucun
   * `lineTo`, donc le seul du relevé est le tick, et les seuls `moveTo` sont celui du secteur
   * — posé sur le centre — puis celui du tick.
   */
  const tick = (p?: number): { from: number; to: number } | null => {
    const ops = trace({ showAim: true }, inclinee(p))
    const centre = ops.find((o) => o.op === 'moveTo')!.args as number[]
    const i = ops.findIndex((o) => o.op === 'lineTo')
    if (i < 0) return null
    const [mx, my] = ops[i - 1].args as number[]
    const [lx, ly] = ops[i].args as number[]
    return {
      from: Math.hypot(mx - centre[0], my - centre[1]),
      to: Math.hypot(lx - centre[0], ly - centre[1]),
    }
  }

  it('vers le HAUT, le tick sort du cône ; vers le BAS, il rentre dedans', () => {
    const haut = tick(30)!
    expect(haut.from).toBeCloseTo(52 * Math.cos(Math.PI / 6), 6)
    expect(haut.to).toBeCloseTo(haut.from + 6, 6)

    const bas = tick(-30)!
    // Même longueur de cône (cosinus pair), sens opposé : c'est TOUT ce qui distingue les deux.
    expect(bas.from).toBeCloseTo(haut.from, 6)
    expect(bas.to).toBeCloseTo(bas.from - 6, 6)
  })

  it('ne dessine AUCUN tick quand la visée se lit à plat', () => {
    expect(tick(0)).toBeNull()
    expect(tick(undefined)).toBeNull()
    // Zone morte : sous 2°, le cône n'a pas bougé (cos 2° = 0,9994) et un trait affirmerait un
    // sens que l'œil ne peut pas vérifier — il changerait de camp à chaque image.
    expect(tick(1.9)).toBeNull()
    expect(tick(-1.9)).toBeNull()
    expect(tick(2.1)).not.toBeNull()
  })

  it('prend l élévation du MÊME instant que le cap, jamais une plus ancienne', () => {
    // Le cap est frais (t = 50), l'élévation ne vit que sur un point BIEN plus vieux : la
    // publier reviendrait à poser une plongée périmée sur une visée à plat actuelle.
    const perimee = singlePointTrack(512, {
      points: [
        { t: 10, x: 5, y: 5, z: 0, h: 360, p: -70 },
        { t: 50, x: 5, y: 5, z: 0, h: 360 },
      ],
    })
    const ops = trace({ showAim: true }, perimee)
    const secteur = ops.find((o) => o.op === 'arc' && (o.args as number[]).length === 5)!
    expect((secteur.args as number[])[2]).toBeCloseTo(52, 6)
    // Marqueur ORDINAIRE : il n emet aucun segment, donc zero `lineTo` = zero tick.
    expect(count(ops, 'lineTo')).toBe(0)
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

  // V1 (2026-08-18) : l'anneau externe unique est devenu un DOUBLE CONTOUR posé sur un HALO —
  // trois arcs de plus, parce que la couleur ne doit jamais porter seule l'identité.
  it('MOI porte TROIS arcs de plus que le marqueur ordinaire — double contour et halo', () => {
    const plain = count(trace({ markOfSlot: markOf(undefined) }), 'arc')
    const mine = count(trace({ markOfSlot: markOf('me') }), 'arc')
    expect(mine).toBe(plain + 3)
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
  })

  /**
   * A1 (2026-08-18) — LA CROIX EST PLUS PETITE, PLUS ÉPAISSE, ET TOUJOURS ROUGE.
   *
   * Les trois tiennent ensemble : réduire la croix sans l'épaissir la ferait disparaître, et
   * lui laisser la couleur d'équipe la ferait lire comme un marqueur de joueur. Ce test épingle
   * les trois d'un coup, sur la même croix.
   */
  it('la croix de mort est plus petite, plus épaisse, et à l ENCRE DE MORT', () => {
    const dead = singlePointTrack(512, { endFrame: 50 })
    const ops = trace({ frame: 55 }, dead)
    expect(valuesOf(ops, 'lineWidth')).toContain(2.6)
    expect(valuesOf(ops, 'lineWidth')).not.toContain(1.6)
    // Demi-taille : l'écart entre les deux extrémités d'une diagonale vaut 2 x 3,6 px.
    const xs = ops
      .filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
      .map((o) => Number(o.args[0]))
    expect(Math.abs(xs[1] - xs[0])).toBeCloseTo(7.2, 6)
    // ROUGE, jamais la couleur d'équipe du défunt.
    const strokeStyles = ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(strokeStyles).toContain('rgb(5 5 5)')
    expect(strokeStyles).not.toContain('rgb(1 2 3)')
  })

  it('une vie SANS couleur de slot ne dessine aucune croix non plus', () => {
    const dead = singlePointTrack(512, { endFrame: 50 })
    const ops = trace({ frame: 55, colorOfSlot: () => null }, dead)
    // Le calque pose ses jointures avant de boucler : aucune PRIMITIVE de tracé ne suit.
    expect(count(ops, 'stroke')).toBe(0)
    expect(count(ops, 'moveTo')).toBe(0)
  })

  /**
   * A1 (2026-08-18) — L'ANNEAU D'ÉTAGE PREND LA COULEUR DU PION.
   *
   * Il était à l'encre du thème pour ne pas faire dire deux choses à la couleur ; à l'écran,
   * cette neutralité le détachait de son point. C'est le NOMBRE d'anneaux qui dit la hauteur,
   * et lui seul : la couleur ne dit toujours que le camp.
   */
  it('marque l étage par un anneau à la COULEUR DU PION', () => {
    // Joueur en HAUT d'une carte d'amplitude connue : l'étage le plus élevé porte ses anneaux.
    const ops = trace({ z: { min: 0, max: 10 } }, singlePointTrack(512, {
      points: [{ t: 50, x: 5, y: 5, z: 10, h: 90 }],
    }))
    const strokeStyles = ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(strokeStyles).toContain('rgb(1 2 3)')
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

  it('résout l identité à une image BORNÉE à la fenêtre de la vie, jamais à l image courante', () => {
    // Une vie [0,50] dessinée à l'image 55 (croix de mort, 5 frames après la fin). Un slot de
    // biped est réattribué entre manches : à l'image courante il peut être LIBRE ou repris par
    // un AUTRE joueur — y résoudre effacerait la croix (null) ou lui prêterait une autre teinte.
    // Le double ci-dessous ne rend une couleur que DANS la fenêtre de la vie ; la croix ne se
    // dessine donc que si la porte a été interrogée à une image bornée à [start, end].
    const dead = singlePointTrack(512, { startFrame: 0, endFrame: 50 })
    const askedFrames: number[] = []
    const ops = trace({
      frame: 55,
      colorOfSlot: (_slot, frame) => {
        askedFrames.push(frame)
        return frame <= 50 ? 'rgb(1 2 3)' : null
      },
    }, dead)
    // La croix EST dessinée, à l'encre de mort — la porte a donc reçu une image dans la vie.
    expect(count(ops, 'moveTo')).toBeGreaterThan(0)
    expect(ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toContain('rgb(5 5 5)')
    // CONTRE-ÉPREUVE : jamais l'image courante 55 (hors de la vie), toujours dans [0, 50].
    expect(askedFrames).not.toContain(55)
    expect(askedFrames.every((f) => f >= 0 && f <= 50)).toBe(true)
  })
})

/**
 * V1 (retour utilisateur du 2026-08-18) — LA TRAÎNÉE EST UNE OPTION, et LE JOUEUR DE LA PAGE
 * se distingue par sa FORME avant sa couleur.
 *
 * Ce qui est vérifié ici est exactement ce qui a été promis : éteindre la traînée n'enlève
 * QUE la traînée ; le marqueur « moi » porte DEUX anneaux et un halo, tous trois à l'encre
 * dédiée, et son NOYAU garde la couleur d'ÉQUIPE — sans quoi la couleur porterait seule.
 */
describe('traînée en option et joueur de la page (V1, 2026-08-18)', () => {
  /** Une vie de trois points : de quoi tracer deux segments de traînée. */
  const moving = (): ReplayTrackReady => ({
    slot: 512,
    team: -1,
    xuid: 'A',
    startFrame: 0,
    endFrame: 100,
    points: [
      { t: 48, x: 4, y: 5, z: 0 },
      { t: 49, x: 4.5, y: 5, z: 0 },
      { t: 50, x: 5, y: 5, z: 0, h: 90 },
    ],
  })

  it('traînée allumée : la polyligne est tracée segment par segment', () => {
    const ops = trace({ showTrail: true }, moving())
    expect(count(ops, 'lineTo')).toBeGreaterThan(0)
  })

  it('traînée éteinte : plus AUCUN segment, et le marqueur reste dessiné', () => {
    const off = trace({ showTrail: false }, moving())
    expect(count(off, 'lineTo')).toBe(0)
    // Le point, lui, est toujours là (liseré + noyau) : on n'a éteint que la trace.
    expect(count(off, 'arc')).toBeGreaterThan(0)
    expect(count(off, 'fillText')).toBe(1)
  })

  it('joueur de la page : DEUX anneaux et un halo, à l encre dédiée', () => {
    const mine = trace({
      markOfSlot: (): PlayerMarkKind => 'me',
      selfInk: 'rgb(7 7 7)',
      showNames: false,
    })
    // Le double contour : deux `stroke` à l'encre dédiée, de rayons DIFFÉRENTS.
    const strokeStyles = mine.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(strokeStyles.filter((s) => s === 'rgb(7 7 7)').length).toBeGreaterThanOrEqual(1)
    const rayons = mine.filter((o) => o.op === 'arc').map((o) => o.args[2] as number)
    expect(new Set(rayons).size).toBeGreaterThanOrEqual(3)
    // Le halo : un dégradé radial, que seul le joueur de la page émet.
    expect(count(mine, 'createRadialGradient')).toBe(1)
  })

  it('LA COULEUR N EST JAMAIS SEULE : le noyau du joueur de la page garde sa couleur d équipe', () => {
    const mine = trace({ markOfSlot: (): PlayerMarkKind => 'me', selfInk: 'rgb(7 7 7)' })
    expect(mine.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])).toContain('rgb(1 2 3)')
  })

  it('un joueur ORDINAIRE n a ni double anneau ni halo', () => {
    const autre = trace({ markOfSlot: () => undefined, selfInk: 'rgb(7 7 7)' })
    expect(count(autre, 'createRadialGradient')).toBe(0)
    expect(autre.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).not.toContain('rgb(7 7 7)')
  })
})
