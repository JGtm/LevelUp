/**
 * Tests — useReplayPlacements : LE HOOK SERT LES POSES FABRIQUÉES AU COMPTE ET AU SURVOL.
 *
 * C'est le branchement qui a manqué à l'origine : le calque, les zones et le survol lisaient
 * `doc.equipmentPlacements`, où le capteur, le traqueur, l'écran et le champ n'arrivent plus.
 * Ce test les lit sur un extrait RÉEL (`test/spawnedObjectWitnesses.ts`, capteur `2ba491cf`) : un
 * hook qui repasserait au tableau du document seul rougit ici, sur le compte comme sur le survol.
 * Le branchement du canvas et des fiches est tenu par le garde-rail de lecture de source de
 * `layers/placementRenderWitness.test.ts`.
 */
import { act, renderHook } from '@testing-library/react'
import type { PointerEvent } from 'react'
import { describe, expect, it } from 'vitest'

import { SPAWNED_WITNESSES } from '../test/spawnedObjectWitnesses'
import { project } from './placementShapes'
import { useReplayPlacements } from './useReplayPlacements'

const VIEW = { bounds: { minX: 0, minY: -20, maxX: 30, maxY: 0 }, width: 300, height: 200, pad: 0 }
/** Entre le repos (2176) et la fin lue (2252) du premier capteur de l'extrait. */
const FRAME = 2200

function rendre(focusFilter?: Parameters<typeof useReplayPlacements>[0]['focusFilter']) {
  const doc = SPAWNED_WITNESSES.sensor()
  return {
    doc,
    hook: renderHook(() =>
      useReplayPlacements({
        doc,
        view: VIEW,
        frameRef: { current: FRAME },
        enabled: true,
        showUnnamed: false,
        showDropped: true,
        focusFilter,
      }),
    ),
  }
}

describe('useReplayPlacements — les objets engendrés', () => {
  it('compte et sert les poses fabriquées : le document seul n’en porte aucune', () => {
    const { doc, hook } = rendre()
    expect(doc.equipmentPlacements).toHaveLength(0)
    expect(hook.result.current.all.length).toBeGreaterThan(0)
    expect(hook.result.current.counts.drawable).toBe(hook.result.current.all.length)
  })

  it('le survol désigne la pose fabriquée sous le pointeur', () => {
    const { hook } = rendre()
    const c = project({ x: 15.2, y: -12.26 }, VIEW)
    const event = {
      clientX: c.x,
      clientY: c.y,
      currentTarget: { getBoundingClientRect: () => ({ left: 0, top: 0, width: VIEW.width, height: VIEW.height }) },
    } as unknown as PointerEvent<HTMLCanvasElement>
    act(() => hook.result.current.hover.onPointerMove(event))
    expect(hook.result.current.hover.hover?.placement.family).toBe('sensor')
  })
})

describe('useReplayPlacements — Seul et Masquer', () => {
  const tout = { placement: () => true, slotAt: () => true }
  const rien = { placement: () => false, slotAt: () => false }

  it('le survol ne désigne pas une pose retirée, et les comptes ne bougent pas', () => {
    const { hook } = rendre(rien)
    const c = project({ x: 15.2, y: -12.26 }, VIEW)
    const event = {
      clientX: c.x,
      clientY: c.y,
      currentTarget: { getBoundingClientRect: () => ({ left: 0, top: 0, width: VIEW.width, height: VIEW.height }) },
    } as unknown as PointerEvent<HTMLCanvasElement>
    act(() => hook.result.current.hover.onPointerMove(event))
    expect(hook.result.current.hover.hover).toBeNull()
    expect(hook.result.current.counts.drawable).toBe(rendre().hook.result.current.counts.drawable)
  })

  it('un filtre qui garde tout rend la même scène de failles, un filtre qui retire tout la vide', () => {
    const base = rendre().hook.result.current.rift
    expect(rendre(tout).hook.result.current.rift).toEqual(base)
    const vide = rendre(rien).hook.result.current.rift
    expect(vide.rifts).toHaveLength(0)
    expect(vide.teleports).toHaveLength(0)
  })
})
