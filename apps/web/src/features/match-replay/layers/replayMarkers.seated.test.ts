/**
 * replayMarkers.seated.test.ts — LA RÈGLE DU PION : un pion n'est dessiné que pour un joueur qui
 * occupe une fiche À CET INSTANT (utilisateur, 2026-10-21 : « on a autant de joueurs sur le terrain
 * que sur les fiches »).
 *
 * Un bot relevé par un humain n'a plus de fiche à partir de la relève, alors que le film lui donne
 * encore des vies : sa trajectoire, son cône, sa traînée, son étiquette et sa croix de mort
 * disparaissent de la carte, sans toucher à la vie qu'il avait avant.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { recordingContext } from '../test/recordingContext'
import { drawTracksLayer, type MarkerStyle } from './replayMarkers'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 200, height: 100, pad: 4 }

/** Une vie d'un seul point : aucun segment de traînée ne pollue les comptages. */
function vie(over: Partial<ReplayTrackReady> = {}): ReplayTrackReady {
  return {
    slot: 512,
    team: -1,
    xuid: 'bot:B',
    startFrame: 0,
    endFrame: 100,
    points: [{ t: 50, x: 5, y: 5, z: 0, h: 90 }],
    ...over,
  } as ReplayTrackReady
}

function style(over: Partial<MarkerStyle> = {}): MarkerStyle {
  return {
    colorOfSlot: () => 'rgb(1 2 3)',
    ink: 'rgb(9 9 9)',
    frame: 50,
    timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 },
    z: { min: 0, max: 10 },
    k: 1,
    showAim: true,
    markOfSlot: () => undefined,
    nameOfSlot: () => 'Bot',
    showTrail: true,
    selfInk: 'rgb(4 4 4)',
    deathInk: 'rgb(5 5 5)',
    labelStroke: 'rgb(8 12 18)',
    offscreenLabelOf: (name, meters) => `${name} ${Math.round(meters)}`,
    ...over,
  }
}

function trace(track: ReplayTrackReady, over: Partial<MarkerStyle>): number {
  const { ops, ctx } = recordingContext()
  drawTracksLayer(ctx, [track], VIEW, style(over))
  // Les réglages d'ouverture/fermeture du calque (`set lineCap`, `set globalAlpha`) ne sont pas du tracé.
  return ops.filter((o) => !o.op.startsWith('set ')).length
}

describe('drawTracksLayer — le pion suit la fiche', () => {
  it('une vie dont le joueur occupe une fiche à l’image se dessine', () => {
    expect(trace(vie(), { seatedAtSlot: () => true })).toBeGreaterThan(0)
  })

  it('une vie dont le joueur n’occupe AUCUNE fiche à l’image ne dessine rien : ni pion, ni cône, ni nom', () => {
    expect(trace(vie(), { seatedAtSlot: () => false })).toBe(0)
  })

  it('sans prédicat de fiche, rien ne change (les appelants sans roster)', () => {
    expect(trace(vie(), {})).toBeGreaterThan(0)
  })

  it('la vie d’avant la relève reste dessinée, celle d’après disparaît', () => {
    // Le bot tient sa fiche jusqu'à l'image 319 : la même vie, lue à 300 puis à 400.
    const seated = (_slot: number, _owner: number, frame: number) => frame <= 319
    const enCours = vie({ startFrame: 223, endFrame: 652, points: [{ t: 223, x: 5, y: 5, z: 0, h: 90 }] })
    expect(trace(enCours, { frame: 300, seatedAtSlot: seated })).toBeGreaterThan(0)
    expect(trace(enCours, { frame: 400, seatedAtSlot: seated })).toBe(0)
  })

  it('la croix de mort d’une vie finie AVANT la relève ne survit pas à la relève', () => {
    // Vie finie à 310, croix visible 20 images (`timing.death`) : à 325 le bot n'a plus de fiche.
    const morte = vie({ startFrame: 223, endFrame: 310, points: [{ t: 310, x: 5, y: 5, z: 0 }] })
    const seated = (_slot: number, _owner: number, frame: number) => frame <= 319
    expect(trace(morte, { frame: 315, seatedAtSlot: seated })).toBeGreaterThan(0)
    expect(trace(morte, { frame: 325, seatedAtSlot: seated })).toBe(0)
  })

  it('le prédicat reçoit l’image du PROPRIÉTAIRE de la vie, pas l’image lue, pour une croix', () => {
    const vues: Array<[number, number, number]> = []
    const morte = vie({ startFrame: 223, endFrame: 310, points: [{ t: 310, x: 5, y: 5, z: 0 }] })
    trace(morte, {
      frame: 315,
      seatedAtSlot: (slot, owner, frame) => {
        vues.push([slot, owner, frame])
        return true
      },
    })
    expect(vues).toEqual([[512, 310, 315]])
  })
})
