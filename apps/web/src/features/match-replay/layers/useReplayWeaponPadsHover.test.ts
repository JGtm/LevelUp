/**
 * Tests — LE SURVOL D'UN SOCLE, du pointeur à l'infobulle.
 *
 * CE QUI MANQUAIT, ET CE QUE ÇA LAISSAIT PASSER (revue adversariale du 2026-08-27) : aucun cas ne
 * traversait `onPointerMove`. Le survol construit pourtant l'objet que l'infobulle affiche — nom,
 * état, compte à rebours — et remplacer `respawn: padRespawnAt(...)` par `respawn: null` laissait
 * TOUTE la suite verte. Les fonctions pures étaient testées ; le fil qui les relie à l'écran ne
 * l'était pas.
 *
 * L'ÉVÉNEMENT EST FACTICE MAIS LA GÉOMÉTRIE EST VRAIE : le hook lit `getBoundingClientRect` pour
 * ramener les pixels CSS au cadrage du canvas. Un rectangle à l'échelle 1 rend donc `clientX/Y`
 * directement comparables aux coordonnées projetées, et le survol se rejoue sur la DONNÉE.
 *
 * UN FICHIER À PART, et c'est une conséquence de taille : `useReplayWeaponPads.test.ts` porte
 * déjà 550 lignes de résolution (taille, nom, vignette, croisement) — y ajouter le survol aurait
 * accru une dette existante au lieu de la contenir.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { createRef, type PointerEvent, type RefObject } from 'react'

import { worldToCanvas } from '../model/replayLogic'
import type { ReplayWeaponPadReady } from '../model/replayNormalize'
import { testReplayDoc } from '../test/testDoc'
import { useReplayWeaponPads } from './useReplayWeaponPads'

/** Le cadrage réel de la carte du rejeu : 700 x 480 avec sa marge de 24 px. */
const VUE = {
  bounds: { minX: -18.68, minY: -25.27, maxX: 19.28, maxY: 25.37 },
  width: 700,
  height: 480,
  pad: 24,
}

/**
 * UN SOCLE QUI SE VIDE PUIS SE REMPLIT, copié de l'artefact `530820e5` (le BR75 du centre).
 *
 * POURQUOI CELUI-LÀ : il se vide à l'image 346 et une arme y REVIENT à l'image 606. À l'image
 * 500, le compte à rebours est donc MESURÉ (106 images x 100 ms = 10,6 s) — et ce socle porte
 * AUSSI un cycle de 30,3 s, qui donnerait un tout autre chiffre s'il était consulté. Un seul
 * socle suffit : ce qu'on éprouve ici est le câblage, pas la résolution.
 */
const SOCLE: ReplayWeaponPadReady = {
  x: 6.2765,
  y: 6.9393,
  z: 27.0176,
  weapon: '0x2B1824D5',
  spawns: [0, 606, 3353],
  presence: [
    { t0: 0, tLow: 146, tHigh: 346 },
    { t0: 606, tLow: 2946, tHigh: 3146 },
  ],
  cycle: { medianS: 30.3, p10S: 30.3, p90S: 30.3, gaps: 2, missing: 0 },
}

const CENTRE = worldToCanvas(SOCLE, VUE.bounds, VUE.width, VUE.height, VUE.pad)

function monter(frame: number) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  return renderHook(() =>
    useReplayWeaponPads({
      doc: testReplayDoc({ frameCount: 4751, frameIntervalMs: 100, bounds: { ...VUE.bounds, minZ: 0.68, maxZ: 29.5 }, weaponPads: [SOCLE] }),
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

describe('useReplayWeaponPads — le SURVOL, du pointeur à l’infobulle', () => {
  it('le pointeur sur un socle rend son NOM, son ÉTAT et son compte à rebours', () => {
    const { result } = monter(500)
    act(() => result.current.onPointerMove(pointeurSur(CENTRE)))
    const hover = result.current.hover
    expect(hover, 'aucun socle attrapé sous le pointeur').not.toBeNull()
    // `toEqual` et non `toBe` : la frontière normalise le document, donc le socle survolé est une
    // COPIE de la fixture — c'est sa donnée qui compte, pas son identité d'objet.
    expect(hover?.pad).toEqual(SOCLE)
    expect(hover?.name).toBeTruthy()
    expect(hover?.state).toBe('empty')
    // LA VALEUR EXACTE, pas seulement « non nul » : c'est elle qui tue le mutant `respawn: null`
    // comme celui qui rebrancherait le cycle (30,3 s) à la place de la mesure.
    expect(hover?.respawn).toEqual({ seconds: 10.6, measured: true })
  })

  it('le pointeur LOIN de tout socle n’invente aucun survol', () => {
    const { result } = monter(500)
    act(() => result.current.onPointerMove(pointeurSur({ x: 4, y: 4 })))
    expect(result.current.hover).toBeNull()
  })

  it('quitter le canvas referme l’infobulle', () => {
    const { result } = monter(500)
    act(() => result.current.onPointerMove(pointeurSur(CENTRE)))
    expect(result.current.hover).not.toBeNull()
    act(() => result.current.onPointerLeave())
    expect(result.current.hover).toBeNull()
  })

  it('sur un socle PLEIN, aucun compte à rebours ne remonte au survol', () => {
    const { result } = monter(100)
    act(() => result.current.onPointerMove(pointeurSur(CENTRE)))
    expect(result.current.hover?.state).toBe('full')
    expect(result.current.hover?.respawn).toBeNull()
  })
})
