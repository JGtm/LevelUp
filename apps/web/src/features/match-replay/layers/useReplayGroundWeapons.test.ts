/**
 * Tests — useReplayGroundWeapons : LE SURVOL (item A) du lot 6.5 (2026-09-10,
 * `.ai/V7.5/RAPPORT_ARMES_AU_SOL_2026-09-10.md`).
 *
 * CE QUE CE FICHIER VERROUILLE :
 *  - l'infobulle tient en UNE ligne composée de trois fragments (arme, origine, reprise) ;
 *  - une arme `spawned` n'a JAMAIS de lâcheur affiché, même si `dropper` était renseigné ;
 *  - un lâcheur/ramasseur NON résolu en joueur se dit, il ne se tait jamais.
 *
 * La résolution de nom (`padNameFor`) et la normalisation de clé (`weaponLabelKeyOf`) sont
 * déjà verrouillées dans `useReplayWeaponPads.test.ts` : ce fichier ne les retraverse pas.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { createRef, type PointerEvent, type RefObject } from 'react'

import type { ReplayGroundWeapon } from '@/lib/api/types'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { testReplayDoc } from '../test/testDoc'
import { useReplayGroundWeapons } from './useReplayGroundWeapons'

const VUE = { bounds: { minX: -20, minY: -20, maxX: 20, maxY: 20 }, width: 200, height: 200, pad: 0 }

// t0 = 0 : le survol est testé à l'image 0 (`frameRef.current = 0` dans `monter`) — l'objet
// doit donc déjà être VISIBLE à cette image (`groundWeaponsAt` exige `frame >= item.t0`).
function item(over: Partial<ReplayGroundWeapon> = {}): ReplayGroundWeapon {
  return {
    t0: 0, t1: 0, t1max: 40, x: 0, y: 0,
    w: '2b1824d5', origin: 'dropped', dropper: 7, end: 'seen', picker: -1,
    ...over,
  }
}

const LABELS: ReplayDocumentReady['weaponLabels'] = {
  '0x2B1824D5': { en: 'BR75', fr: 'BR75', img: '/x.png', tinted: true },
}

function monter(
  items: ReplayGroundWeapon[],
  opts: {
    nameOfSlot?: (slot: number, frame: number) => string | null
    labels?: ReplayDocumentReady['weaponLabels']
  } = {},
) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = 0
  const doc = testReplayDoc({ groundWeapons: items, weaponLabels: opts.labels ?? LABELS })
  return renderHook(() =>
    useReplayGroundWeapons({
      doc,
      view: VUE,
      enabled: true,
      ink: { fill: 'fill', outline: 'outline' },
      redraw: vi.fn(),
      frameRef,
      nameOfSlot: opts.nameOfSlot ?? (() => null),
      locale: 'fr',
    }),
  )
}

/** Un pointeur factice, à l'échelle 1 (rectangle CSS == cadrage du canvas). */
function pointeurSur(at: { x: number; y: number }): PointerEvent<HTMLCanvasElement> {
  return {
    clientX: at.x,
    clientY: at.y,
    currentTarget: {
      getBoundingClientRect: () => ({ left: 0, top: 0, width: VUE.width, height: VUE.height }),
    },
  } as unknown as PointerEvent<HTMLCanvasElement>
}

/**
 * Simule un survol exactement sur l'objet projeté (au centre de VUE, x=0,y=0 -> (100,100)).
 * SOUS `act` : `onPointerMove` écrit un état React (`setHover`), et `result.current` ne le
 * reflète qu'une fois le rendu qu'il déclenche traité (même piège documenté dans
 * `useReplayWeaponPadsHover.test.ts`).
 */
function survoler(hook: ReturnType<typeof monter>['result'], at: { x: number; y: number } = { x: 100, y: 100 }) {
  act(() => hook.current.onPointerMove(pointeurSur(at)))
}

describe('useReplayGroundWeapons — le survol (item A)', () => {
  it('lâcheur RÉSOLU : une ligne « <arme> · lâchée par X »', () => {
    const { result } = monter([item({ dropper: 42 })], {
      nameOfSlot: (slot) => (slot === 42 ? 'DinoR00' : null),
    })
    survoler(result)
    expect(result.current.hover?.weaponName).toBe('BR75')
    expect(result.current.hover?.originLine).toBe('lâchée par DinoR00')
    expect(result.current.hover?.pickupLine).toBeNull()
  })

  it('lâcheur NON résolu : « lâchée par un joueur non nommé », jamais le silence', () => {
    const { result } = monter([item({ dropper: 42 })], { nameOfSlot: () => null })
    survoler(result)
    expect(result.current.hover?.originLine).toBe('lâchée par un joueur non nommé')
  })

  it('origine `spawned` : « apparue », MÊME SI dropper était renseigné par erreur', () => {
    const { result } = monter([item({ origin: 'spawned', dropper: 42 })], {
      nameOfSlot: () => 'DinoR00',
    })
    survoler(result)
    expect(result.current.hover?.originLine).toBe('apparue')
  })

  it('ramasseur CONNU : une seconde clause « · reprise par Y »', () => {
    const { result } = monter([item({ picker: 9, end: 'pickup' })], {
      nameOfSlot: (slot) => (slot === 9 ? 'SHROOM' : slot === 7 ? 'DinoR00' : null),
    })
    survoler(result)
    expect(result.current.hover?.pickupLine).toBe('reprise par SHROOM')
  })

  it('ramasseur NON résolu : « reprise par un joueur non nommé »', () => {
    const { result } = monter([item({ picker: 9, end: 'pickup' })], { nameOfSlot: () => null })
    survoler(result)
    expect(result.current.hover?.pickupLine).toBe('reprise par un joueur non nommé')
  })

  it('sans ramasseur (picker = -1) : pas de seconde clause DU TOUT', () => {
    const { result } = monter([item({ picker: -1 })])
    survoler(result)
    expect(result.current.hover?.pickupLine).toBeNull()
  })

  it('hors de la vignette : rien à survoler', () => {
    const { result } = monter([item({ x: 15, y: 15 })])
    survoler(result)
    expect(result.current.hover).toBeNull()
  })

  it('calque ÉTEINT : le survol reste muet', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const doc = testReplayDoc({ groundWeapons: [item()], weaponLabels: LABELS })
    const { result } = renderHook(() =>
      useReplayGroundWeapons({
        doc, view: VUE, enabled: false,
        ink: { fill: 'fill', outline: 'outline' }, redraw: vi.fn(),
        frameRef, nameOfSlot: () => null, locale: 'fr',
      }),
    )
    survoler(result)
    expect(result.current.hover).toBeNull()
  })
})
