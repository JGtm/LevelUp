/**
 * Tests — thrusterDashFx (la poussée du propulseur sur le pion).
 *
 * Ce que ces tests verrouillent :
 *  - L'EFFET NE PART QU'AUX INSTANTS PUBLIÉS. Une impulsion du document, une seule ; rien
 *    avant elle, rien après sa fenêtre, et RIEN du tout pour une famille que la table de
 *    rendu ne connaît pas (le répulseur n'est pas dans ce canal et n'y sera pas).
 *  - LA DURÉE EST BORNÉE, en temps RÉEL du match : `THRUSTER_DASH_MS`, jamais un nombre
 *    d'images — en lecture accélérée le geste doit rester aussi bref.
 *  - LA DIRECTION SUIT LE DÉPLACEMENT, lu dans la trajectoire autour de l'instant : fenêtre
 *    AVANT d'abord, repli sur l'ARRIÈRE quand la vie se ferme pendant le dash. Sans
 *    déplacement mesurable, AUCUN dash — on n'oriente pas une forme au hasard.
 *  - `prefers-reduced-motion` : la forme se pose, pleine longueur et opacité constante,
 *    identique à tout âge de la fenêtre. L'information reste, l'animation s'éteint.
 *  - L'ENCRE vient de l'appelant (couleur d'équipe résolue depuis les tokens), et se résout à
 *    l'image de L'IMPULSION — jamais à l'image courante, où le slot peut être repris.
 */
import { describe, expect, it } from 'vitest'

import {
  buildThrusterDashFx,
  dashHeading,
  dashProgress,
  drawThrusterDashLayer,
  THRUSTER_DASH_MS,
  type DashStyle,
  type ThrusterDashFx,
} from './thrusterDashFx'
import type { ReplayDocumentReady } from '../replayNormalize'
import { recordingContext } from '../test/recordingContext'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }

/** Une vie qui file vers l'est : la direction du dash doit s'y lire. */
const TRACK = {
  slot: 5,
  team: -1,
  points: [
    { t: 0, x: 1, y: 5 },
    { t: 60, x: 9, y: 5 },
  ],
}

function docWith(over: Partial<ReplayDocumentReady>): ReplayDocumentReady {
  return { abilityImpulses: [], tracks: [], ...over } as ReplayDocumentReady
}

const INK = 'var(--team-ally)'
const STYLE: DashStyle = { colorOfSlot: () => INK }
const TIME = { frame: 30, frameMs: 33.3, k: 1, reducedMotion: false }

describe('buildThrusterDashFx', () => {
  it('joint chaque impulsion aux points de SA vie et lit sa direction', () => {
    const doc = docWith({
      tracks: [TRACK] as ReplayDocumentReady['tracks'],
      abilityImpulses: [{ t: 30, slot: 5, family: 'thruster' }],
    })
    const fx = buildThrusterDashFx(doc, 8)
    expect(fx).toHaveLength(1)
    expect(fx[0].frame).toBe(30)
    expect(fx[0].points).toBe(TRACK.points)
    // Direction lue vers l'AVANT : la position à t=38 est plus à l'est que celle à t=30.
    expect(fx[0].to.x).toBeGreaterThan(fx[0].from.x)
  })

  it('n’ouvre AUCUN dash pour une famille hors de la table — jamais la forme d’une voisine', () => {
    const doc = docWith({
      tracks: [TRACK] as ReplayDocumentReady['tracks'],
      abilityImpulses: [
        { t: 30, slot: 5, family: 'repulsor' },
        { t: 30, slot: 5, family: 'grapple' },
      ],
    })
    expect(buildThrusterDashFx(doc, 8)).toEqual([])
  })

  it('écarte une impulsion dont la vie n’est pas publiée : aucun pion à pousser', () => {
    const doc = docWith({
      tracks: [TRACK] as ReplayDocumentReady['tracks'],
      abilityImpulses: [{ t: 30, slot: 99, family: 'thruster' }],
    })
    expect(buildThrusterDashFx(doc, 8)).toEqual([])
  })

  it('écarte une impulsion sans déplacement mesurable : on n’oriente pas au hasard', () => {
    const immobile = {
      slot: 7,
      team: -1,
      points: [
        { t: 0, x: 3, y: 3 },
        { t: 60, x: 3, y: 3 },
      ],
    }
    const doc = docWith({
      tracks: [immobile] as unknown as ReplayDocumentReady['tracks'],
      abilityImpulses: [{ t: 30, slot: 7, family: 'thruster' }],
    })
    expect(buildThrusterDashFx(doc, 8)).toEqual([])
  })

  it('sans impulsion, rien : le document normalisé garantit un tableau, jamais null', () => {
    expect(buildThrusterDashFx(docWith({}), 8)).toEqual([])
  })

  /**
   * DEUX VIES DU MÊME SLOT — la fixture canonique du dossier (`fireMark.test.ts`), et le seul
   * cas qui distingue la jointure JUSTE de la jointure NAÏVE.
   *
   * Un index `slot -> points` ne garde que la DERNIÈRE piste du slot. L'impulsion de la
   * PREMIÈRE vie irait alors chercher les points de la seconde : ou bien l'instant précède ces
   * points et `positionAt` rend `null` (la poussée disparaît de la carte alors que le son part
   * quand même — on l'entend sans la voir), ou bien il tombe dans la fenêtre de l'usurpatrice
   * et le sillage se peint à la position et dans la direction d'UN AUTRE JOUEUR.
   */
  it('DEUX VIES du même slot : chaque impulsion prend la vie qui COUVRE son instant', () => {
    const vie1 = {
      slot: 5,
      team: -1,
      startFrame: 10,
      endFrame: 20,
      points: [
        { t: 10, x: 2, y: 2 },
        { t: 20, x: 4, y: 2 },
      ],
    }
    const vie2 = {
      slot: 5,
      team: -1,
      startFrame: 30,
      endFrame: 40,
      points: [
        { t: 30, x: 8, y: 8 },
        { t: 40, x: 8, y: 9 },
      ],
    }
    const doc = docWith({
      tracks: [vie1, vie2] as unknown as ReplayDocumentReady['tracks'],
      abilityImpulses: [
        { t: 14, slot: 5, family: 'thruster' },
        { t: 34, slot: 5, family: 'thruster' },
      ],
    })
    const fx = buildThrusterDashFx(doc, 4)
    expect(fx).toHaveLength(2)
    // La première impulsion tient les points de la PREMIÈRE vie — pas ceux de la seconde.
    expect(fx[0].points).toBe(vie1.points)
    expect(fx[0].from.x).toBeCloseTo(2.8, 5)
    expect(fx[0].from.y).toBeCloseTo(2, 5)
    // Et sa direction est celle de CETTE vie (plein est), pas celle de l'autre (plein nord).
    expect(fx[0].to.x).toBeGreaterThan(fx[0].from.x)
    expect(fx[0].to.y).toBeCloseTo(fx[0].from.y, 5)
    // La seconde impulsion, elle, tient bien la seconde vie.
    expect(fx[1].points).toBe(vie2.points)
    expect(fx[1].from.x).toBeCloseTo(8, 5)
  })

  it('une impulsion qu’AUCUNE vie du slot ne couvre ne dessine rien : entre deux vies', () => {
    const vie1 = {
      slot: 5,
      team: -1,
      startFrame: 10,
      endFrame: 20,
      points: [
        { t: 10, x: 2, y: 2 },
        { t: 20, x: 4, y: 2 },
      ],
    }
    const vie2 = {
      slot: 5,
      team: -1,
      startFrame: 30,
      endFrame: 40,
      points: [
        { t: 30, x: 8, y: 8 },
        { t: 40, x: 8, y: 9 },
      ],
    }
    const doc = docWith({
      tracks: [vie1, vie2] as unknown as ReplayDocumentReady['tracks'],
      abilityImpulses: [{ t: 25, slot: 5, family: 'thruster' }],
    })
    expect(buildThrusterDashFx(doc, 4)).toEqual([])
  })
})

describe('dashHeading', () => {
  it('lit la fenêtre AVANT en priorité : la poussée suit le déclenchement', () => {
    const h = dashHeading(TRACK.points, 30, 8)
    expect(h).not.toBeNull()
    expect(h?.from.x).toBeCloseTo(5, 5)
    expect(h?.to.x).toBeGreaterThan(5)
  })

  it('retombe sur la fenêtre ARRIÈRE quand la vie se ferme pendant le dash', () => {
    // `positionAt` fige la dernière position au-delà de la fin de vie : la fenêtre avant est
    // alors NULLE, et seule l'arrière porte encore le déplacement.
    const h = dashHeading(TRACK.points, 60, 8)
    expect(h).not.toBeNull()
    expect(h?.to.x).toBeCloseTo(9, 5)
    expect(h?.from.x).toBeLessThan(9)
  })

  it('rend null quand aucune des deux fenêtres ne porte de déplacement', () => {
    const fige = [
      { t: 0, x: 2, y: 2 },
      { t: 60, x: 2, y: 2 },
    ]
    expect(dashHeading(fige, 30, 8)).toBeNull()
  })
})

describe('dashProgress', () => {
  it('la fenêtre est bornée : rien avant l’instant, rien après la durée déclarée', () => {
    expect(dashProgress(-1, false)).toBeNull()
    expect(dashProgress(0, false)).not.toBeNull()
    expect(dashProgress(THRUSTER_DASH_MS, false)).not.toBeNull()
    expect(dashProgress(THRUSTER_DASH_MS + 1, false)).toBeNull()
  })

  it('s’étire vite puis se calme, et s’efface : easeOutCubic sur la longueur', () => {
    const debut = dashProgress(0, false)
    const milieu = dashProgress(THRUSTER_DASH_MS / 2, false)
    const fin = dashProgress(THRUSTER_DASH_MS, false)
    expect(debut?.reach).toBeCloseTo(0, 5)
    expect(milieu?.reach).toBeGreaterThan(0.5)
    expect(fin?.reach).toBeCloseTo(1, 5)
    expect(fin?.alpha).toBeCloseTo(0, 5)
    expect(debut?.alpha).toBeGreaterThan(0)
  })

  it('mouvement réduit : forme POSÉE — identique à tout âge de la fenêtre', () => {
    const a = dashProgress(0, true)
    const b = dashProgress(THRUSTER_DASH_MS / 2, true)
    const c = dashProgress(THRUSTER_DASH_MS, true)
    expect(a).toEqual(b)
    expect(b).toEqual(c)
    expect(a?.reach).toBe(1)
    // La borne de la fenêtre tient AUSSI sous mouvement réduit : l'effet n'est pas permanent.
    expect(dashProgress(THRUSTER_DASH_MS + 1, true)).toBeNull()
  })
})

/** Un dash prêt à peindre, direction plein est. */
const FX: ThrusterDashFx = {
  frame: 30,
  slot: 5,
  points: TRACK.points,
  from: { x: 4, y: 5 },
  to: { x: 6, y: 5 },
}

function trace(fx: ThrusterDashFx[], time: Partial<typeof TIME> = {}, style: DashStyle = STYLE) {
  const { ops, ctx } = recordingContext()
  drawThrusterDashLayer(ctx, fx, VIEW, { ...TIME, ...time }, style)
  return ops
}

describe('drawThrusterDashLayer', () => {
  it('ne peint qu’à l’intérieur de la fenêtre, en temps RÉEL du match', () => {
    expect(trace([FX], { frame: 29 })).toEqual([])
    expect(trace([FX], { frame: 30 }).length).toBeGreaterThan(0)
    // 20 images à 33,3 ms = 666 ms : au-delà de la fenêtre de 460 ms, plus rien.
    expect(trace([FX], { frame: 50 })).toEqual([])
    // La MÊME image, sur une grille deux fois plus lente, est encore DANS la fenêtre : c'est
    // bien le temps réel qui borne, pas le nombre d'images.
    expect(trace([FX], { frame: 50, frameMs: 10 }).length).toBeGreaterThan(0)
  })

  it('la direction du tracé suit le déplacement : sillage derrière, chevrons devant', () => {
    const ops = trace([FX], { frame: 40 })
    const lines = ops.filter((o) => o.op === 'lineTo').map((o) => o.args as number[])
    // Le pion est à l'est de la scène à l'image 40 ; tout le sillage et les chevrons sont
    // posés EN ARRIÈRE de lui, donc à des abscisses inférieures à la sienne.
    const moves = ops.filter((o) => o.op === 'moveTo').map((o) => o.args as number[])
    expect(moves.length).toBeGreaterThan(0)
    expect(lines.length).toBeGreaterThan(0)
    // Direction inversée : le tracé bascule de l'autre côté du pion.
    const inverse = trace([{ ...FX, from: { x: 6, y: 5 }, to: { x: 4, y: 5 } }], { frame: 40 })
    const xInverse = inverse.filter((o) => o.op === 'lineTo').map((o) => (o.args as number[])[0])
    const xDirect = lines.map((l) => l[0])
    expect(Math.max(...xInverse)).toBeGreaterThan(Math.max(...xDirect))
  })

  it('la couleur se résout à l’image de L’IMPULSION, jamais à l’image courante', () => {
    const vus: number[] = []
    trace([FX], { frame: 40 }, {
      colorOfSlot: (_slot: number, frame: number) => {
        vus.push(frame)
        return INK
      },
    })
    expect(vus).toEqual([FX.frame])
  })

  it('une vie sans propriétaire à cette image ne peint RIEN', () => {
    expect(trace([FX], { frame: 40 }, { colorOfSlot: () => null })).toEqual([])
  })

  it('l’encre est celle de l’appelant — aucune couleur d’ici', () => {
    const ops = trace([FX], { frame: 40 })
    const encres = ops.filter((o) => o.op === 'set fillStyle' || o.op === 'set strokeStyle')
    expect(encres.length).toBeGreaterThan(0)
    for (const e of encres) expect(e.args[0]).toBe(INK)
  })

  it('mouvement réduit : le tracé ne bouge plus d’une image à l’autre', () => {
    // Pion IMMOBILE, pour n'observer que ce que la préférence gouverne : la forme. L'ancrage,
    // lui, suit le marqueur dans les deux régimes — c'est un effet POSÉ SUR le pion.
    const pose: ThrusterDashFx = {
      ...FX,
      points: [
        { t: 0, x: 5, y: 5 },
        { t: 60, x: 5, y: 5 },
      ],
    }
    const a = trace([pose], { frame: 32, reducedMotion: true })
    const b = trace([pose], { frame: 42, reducedMotion: true })
    expect(JSON.stringify(a)).toBe(JSON.stringify(b))
    // Sans la préférence, le même couple d'images DIFFÈRE : le sillage s'étire et s'efface.
    const c = trace([pose], { frame: 32 })
    const d = trace([pose], { frame: 42 })
    expect(JSON.stringify(c)).not.toBe(JSON.stringify(d))
  })
})
