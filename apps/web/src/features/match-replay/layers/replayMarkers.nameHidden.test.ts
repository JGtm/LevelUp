/**
 * replayMarkers.nameHidden.test.ts — LE NOM SOUS LE PION CÈDE LA PLACE À L'ÉTIQUETTE PLEINE
 * pendant la mise en valeur d'une élimination (lot 2 du plan « maquettes du rejeu »).
 *
 * L'étiquette du tueur et de la victime REMPLACE le nom du pion : le même nom écrit deux fois sur
 * le même pion est exactement ce qu'on évite. Seul le nom SOUS le pion cède — la flèche d'un
 * joueur hors cadre garde le sien, et le pion lui-même reste dessiné.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { recordingContext } from '../test/recordingContext'
import { drawTracksLayer, type MarkerStyle } from './replayMarkers'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 200, height: 100, pad: 4 }

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

function peint(track: ReplayTrackReady, over: Partial<MarkerStyle>) {
  const { ops, ctx } = recordingContext()
  drawTracksLayer(ctx, [track], VIEW, style(over))
  return ops
}

const noms = (ops: ReturnType<typeof peint>) => ops.filter((o) => o.op === 'fillText').map((o) => o.args[0])

describe('drawTracksLayer — nameHiddenAtSlot', () => {
  it('sans prédicat, le nom s’écrit sous le pion (comportement inchangé)', () => {
    expect(noms(peint(vie(), {}))).toEqual(['Bot'])
  })

  it('un slot masqué n’écrit plus son nom, mais garde son pion', () => {
    const normal = peint(vie(), {})
    const masque = peint(vie(), { nameHiddenAtSlot: (slot) => slot === 512 })
    expect(noms(masque)).toEqual([])
    // Le pion (remplissages) est toujours là : seul le texte a disparu.
    expect(masque.filter((o) => o.op === 'fill').length).toBe(normal.filter((o) => o.op === 'fill').length)
  })

  it('un autre slot garde son nom', () => {
    expect(noms(peint(vie(), { nameHiddenAtSlot: (slot) => slot === 7 }))).toEqual(['Bot'])
  })

  it('un joueur HORS CADRE garde le nom de sa flèche de bord', () => {
    const loin = vie({ points: [{ t: 50, x: 500, y: 5, z: 0, h: 90 }] })
    const ops = peint(loin, { nameHiddenAtSlot: () => true })
    expect(noms(ops).length).toBe(1)
    expect(String(noms(ops)[0])).toContain('Bot')
  })
})
