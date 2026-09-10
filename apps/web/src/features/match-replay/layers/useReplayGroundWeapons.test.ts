/**
 * Tests — useReplayGroundWeapons : LE SURVOL (item A) et LE FILTRE « ARMES SPÉCIALES » (item B)
 * du lot 6.5 (2026-09-10, `.ai/V7.5/RAPPORT_ARMES_AU_SOL_2026-09-10.md`).
 *
 * CE QUE CE FICHIER VERROUILLE :
 *  - l'infobulle tient en UNE ligne composée de trois fragments (arme, origine, reprise) ;
 *  - une arme `spawned` n'a JAMAIS de lâcheur affiché, même si `dropper` était renseigné ;
 *  - un lâcheur/ramasseur NON résolu en joueur se dit, il ne se tait jamais ;
 *  - le filtre retire les objets AVANT le tracé et le survol — une arme filtrée ne se survole
 *    plus, exactement comme si le film ne la publiait pas ;
 *  - un rôle ABSENT reste visible bascule éteinte et disparaît bascule allumée.
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
  '0x0A1992BC': { en: 'S7 Sniper', fr: 'S7 Sniper', img: '/y.png', tinted: true, role: 'sniper' },
  '0x230447B1': { en: 'M41 SPNKr', fr: 'M41 SPNKr', img: '/z.png', tinted: true, role: 'power' },
}

function monter(
  items: ReplayGroundWeapon[],
  opts: {
    specialOnly?: boolean
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
      specialOnly: opts.specialOnly ?? false,
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
        doc, view: VUE, enabled: false, specialOnly: false,
        ink: { fill: 'fill', outline: 'outline' }, redraw: vi.fn(),
        frameRef, nameOfSlot: () => null, locale: 'fr',
      }),
    )
    survoler(result)
    expect(result.current.hover).toBeNull()
  })
})

describe('useReplayGroundWeapons — le filtre « armes spéciales seulement » (item B)', () => {
  const SNIPER = item({ w: '0a1992bc', origin: 'spawned', dropper: -1, x: 15, y: 15 })
  const POWER = item({ w: '230447b1', origin: 'spawned', dropper: -1, x: -15, y: 15 })
  // BR75, rôle CONNU mais NEUTRE — au CENTRE de la vue, seul objet que le test survole.
  const NEUTRE = item({ w: '2b1824d5', origin: 'spawned', dropper: -1, x: 0, y: 0 })
  const HORS_REGISTRE = item({ w: 'deadbeef', origin: 'spawned', dropper: -1, x: 15, y: -15 }) // rôle ABSENT

  it('bascule ÉTEINTE : les QUATRE objets restent survolables, rôle absent compris', () => {
    const { result } = monter([SNIPER, POWER, NEUTRE, HORS_REGISTRE], { specialOnly: false })
    expect(result.current.available).toBe(true)
    // Projection (worldToCanvas, bornes [-20, 20] sur 200 px, Y inversé) : screen =
    // ((x + 20) * 5, (20 - y) * 5).
    for (const [pos, attendu] of [
      [{ x: 175, y: 25 }, 'S7 Sniper'], // SNIPER (15, 15)
      [{ x: 25, y: 25 }, 'M41 SPNKr'], // POWER (-15, 15)
      [{ x: 100, y: 100 }, 'BR75'], // NEUTRE (0, 0)
      [{ x: 175, y: 175 }, 'deadbeef'], // HORS_REGISTRE (15, -15), sans libellé
    ] as const) {
      survoler(result, pos)
      expect(result.current.hover?.weaponName, `arme attendue à (${pos.x}, ${pos.y})`).toBe(attendu)
    }
  })

  it('bascule ALLUMÉE : seuls sniper/power/special restent — le rôle ABSENT est exclu', () => {
    const { result: eteint } = monter([SNIPER, POWER, NEUTRE, HORS_REGISTRE], { specialOnly: false })
    const { result: allume } = monter([SNIPER, POWER, NEUTRE, HORS_REGISTRE], { specialOnly: true })
    // La disponibilité elle-même distingue déjà les deux mondes : allumée, seules deux
    // armes (sniper + power) restent, jamais zéro (il y a bien des spéciales dans le lot).
    expect(eteint.current.available).toBe(true)
    expect(allume.current.available).toBe(true)
    // Survoler la position du NEUTRE (BR75, rôle connu mais pas spécial) : visible bascule
    // éteinte, invisible bascule allumée.
    survoler(eteint)
    expect(eteint.current.hover?.weaponName).toBe('BR75')
    survoler(allume)
    expect(allume.current.hover).toBeNull()
  })

  it('bascule ALLUMÉE, liste vide de spéciales : le calque n’est plus DISPONIBLE', () => {
    const { result } = monter([NEUTRE, HORS_REGISTRE], { specialOnly: true })
    expect(result.current.available).toBe(false)
  })
})
