/**
 * Tests — LE RAFFINEMENT DE L'INCERTAIN, BRANCHÉ DANS LE HOOK.
 *
 * POURQUOI UN CAS DE CÂBLAGE EN PLUS DES CAS PURS : `refinePadPresence` peut être parfaitement
 * juste et n'être appelé nulle part. C'est exactement ce qui s'est passé dans ce dossier
 * (survol non testé, revue du 2026-08-27) ; le seul test qui l'attrape est celui qui monte le
 * hook et regarde ce qu'il rend.
 *
 * LE CAS EST CELUI DE L'UTILISATEUR : un socle dont l'absence est prouvée tard, et TOUS les
 * joueurs à l'autre bout de la carte pendant la fenêtre. Avant le raffinement, l'écran disait
 * « incertain » ; il doit dire « disponible » — et basculer « pris » à la preuve d'absence,
 * sans quoi le compte à rebours disparaîtrait avec elle.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { createRef, type PointerEvent, type RefObject } from 'react'

import { worldToCanvas } from '../../../lib/replay/replayLogic'
import type { ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'
import { testReplayDoc } from '../test/testDoc'
import { useReplayWeaponPads } from './useReplayWeaponPads'
import { padStateAt } from '../model/weaponPadTime'

const VUE = {
  bounds: { minX: -50, minY: -50, maxX: 50, maxY: 50 },
  width: 400,
  height: 400,
  pad: 0,
}

/** Le socle du cas : incertain de l'image 100 à 199, vidé (prouvé) à l'image 200. */
const SOCLE: ReplayWeaponPadReady = {
  x: 0,
  y: 0,
  weapon: '0x0A1992BC',
  spawns: [0],
  presence: [{ t0: 0, tLow: 100, tHigh: 200 }],
}

/** Tout le monde à l'autre bout de la carte, du début à la fin. */
const LOIN = [
  { slot: 1, team: -1, points: [{ t: 0, x: 45, y: 45 }, { t: 300, x: 45, y: 45 }] },
  { slot: 2, team: -1, points: [{ t: 0, x: -45, y: 45 }, { t: 300, x: -45, y: 40 }] },
]

/** Un joueur qui traverse le socle à l'image 150. */
const DESSUS = [
  { slot: 1, team: -1, points: [{ t: 100, x: 20, y: 0 }, { t: 150, x: 0, y: 0 }, { t: 200, x: 20, y: 0 }] },
]

function monter(tracks: (typeof LOIN)[number][], frame: number) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  return renderHook(() =>
    useReplayWeaponPads({
      doc: testReplayDoc({ frameCount: 400, bounds: VUE.bounds, weaponPads: [SOCLE], tracks }),
      view: VUE,
      frameRef,
      enabled: true,
      ink: {
        neutral: 'encre',
        fill: 'remplissage',
        outline: 'contour',
        family: { powerup: 'p', power: 'a', classic: 'c' },
      },
      locale: 'fr',
      redraw: vi.fn(),
    }),
  )
}

function pointeurSur(at: { x: number; y: number }): PointerEvent<HTMLCanvasElement> {
  return {
    clientX: at.x,
    clientY: at.y,
    currentTarget: {
      getBoundingClientRect: () => ({ left: 0, top: 0, width: VUE.width, height: VUE.height }),
    },
  } as unknown as PointerEvent<HTMLCanvasElement>
}

const CENTRE = worldToCanvas(SOCLE, VUE.bounds, VUE.width, VUE.height, VUE.pad)

/** L'état LU AU SURVOL : c'est ce que l'infobulle écrit, donc ce que l'utilisateur voit. */
function etatAuSurvol(tracks: (typeof LOIN)[number][], frame: number) {
  const { result } = monter(tracks, frame)
  act(() => result.current.onPointerMove(pointeurSur(CENTRE)))
  return result.current.hover?.state
}

describe('useReplayWeaponPads — l’incertain raffiné arrive bien jusqu’à l’écran', () => {
  it('TOUS LES JOUEURS AU LOIN : le socle est DISPONIBLE là où il était incertain', () => {
    // Le témoin de la régression : sans raffinement, cette même image se lit « incertain ».
    expect(padStateAt(SOCLE, 150)).toBe('uncertain')
    expect(etatAuSurvol(LOIN, 150)).toBe('full')
  })

  it('et il bascule PRIS à la preuve d’absence — le compte à rebours n’est pas perdu', () => {
    expect(etatAuSurvol(LOIN, 199)).toBe('full')
    expect(etatAuSurvol(LOIN, 200)).toBe('empty')
  })

  it('UN JOUEUR PASSE DESSUS : le doute revient, et pas avant son passage', () => {
    expect(etatAuSurvol(DESSUS, 120)).toBe('full')
    expect(etatAuSurvol(DESSUS, 150)).toBe('uncertain')
  })
})
