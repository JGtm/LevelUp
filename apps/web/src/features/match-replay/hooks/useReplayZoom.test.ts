/**
 * Tests — useReplayZoom.
 *
 * CE QU'ILS PROTÈGENT : l'état ne contient JAMAIS un centre qu'on ne pourrait pas montrer.
 * `visibleBounds` reborne déjà à l'affichage, donc rien ne casserait visiblement si l'état
 * dérivait — c'est précisément pour ça qu'il faut un test : le défaut serait silencieux, et le
 * calcul suivant repartirait de la valeur fausse.
 */
import { describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'

import { ZOOM_LEVELS, sceneCenter, visibleBounds } from '../model/replayLogic'
import { useReplayZoom } from './useReplayZoom'

const SCENE = { minX: 0, minY: 0, maxX: 100, maxY: 60 }

function render() {
  return renderHook(() => useReplayZoom(SCENE))
}

/** La fenêtre tient-elle dans la scène ? L'invariant que rien ne doit pouvoir violer. */
function dansLaScene(level: number, c: { x: number; y: number }) {
  const v = visibleBounds(SCENE, level, c.x, c.y)
  return (
    v.minX >= SCENE.minX - 1e-9 &&
    v.maxX <= SCENE.maxX + 1e-9 &&
    v.minY >= SCENE.minY - 1e-9 &&
    v.maxY <= SCENE.maxY + 1e-9
  )
}

describe('useReplayZoom', () => {
  it('démarre à 1x, centré sur la scène, sans déplacement possible', () => {
    const { result } = render()
    expect(result.current.level).toBe(1)
    expect(result.current.center).toEqual(sceneCenter(SCENE))
    expect(result.current.canPan).toBe(false)
    expect(result.current.canZoomOut).toBe(false)
    expect(result.current.canZoomIn).toBe(true)
  })

  it('monte et descend les paliers sans jamais sortir de la liste', () => {
    const { result } = render()
    for (let i = 0; i < 10; i += 1) act(() => result.current.zoomIn())
    expect(result.current.level).toBe(ZOOM_LEVELS[ZOOM_LEVELS.length - 1])
    expect(result.current.canZoomIn).toBe(false)

    for (let i = 0; i < 10; i += 1) act(() => result.current.zoomOut())
    expect(result.current.level).toBe(1)
    expect(result.current.canZoomOut).toBe(false)
  })

  it('à 1x, la croix ne déplace rien — il n existe qu une position légale', () => {
    const { result } = render()
    const avant = result.current.center
    act(() => result.current.panStep(1, 1))
    expect(result.current.center).toEqual(avant)
  })

  it('zoomé, la croix déplace — et la fenêtre reste dans la scène', () => {
    const { result } = render()
    act(() => result.current.zoomIn())
    act(() => result.current.zoomIn())
    const avant = { ...result.current.center }
    act(() => result.current.panStep(1, 0))
    expect(result.current.center.x).toBeGreaterThan(avant.x)
    expect(dansLaScene(result.current.level, result.current.center)).toBe(true)
  })

  // LE PAS SE COMPTE DANS LA FENÊTRE, PAS DANS LA SCÈNE : le même clic doit parcourir la même
  // FRACTION de ce qu'on voit, sinon il paraît deux fois plus rapide à fort grossissement.
  it('le pas rétrécit avec la fenêtre', () => {
    const a = render()
    act(() => a.result.current.zoomIn())
    const avantA = a.result.current.center.x
    act(() => a.result.current.panStep(1, 0))
    const pasA = a.result.current.center.x - avantA

    const b = render()
    act(() => b.result.current.zoomIn())
    act(() => b.result.current.zoomIn())
    const avantB = b.result.current.center.x
    act(() => b.result.current.panStep(1, 0))
    const pasB = b.result.current.center.x - avantB

    expect(pasB).toBeLessThan(pasA)
    expect(pasB).toBeCloseTo(pasA * (ZOOM_LEVELS[1] / ZOOM_LEVELS[2]), 6)
  })

  // LE CAS QUI JUSTIFIE LE REBORNAGE À L'ÉCRITURE : on pousse dans un coin au grossissement
  // maximal, puis on dézoome. La fenêtre s'élargit et ne tient plus aussi près du bord.
  it('en dézoomant depuis un coin, le centre recule au lieu de rester coincé', () => {
    const { result } = render()
    for (let i = 0; i < 3; i += 1) act(() => result.current.zoomIn())
    for (let i = 0; i < 12; i += 1) act(() => result.current.panStep(1, 1))
    const coin = { ...result.current.center }
    expect(dansLaScene(result.current.level, coin)).toBe(true)

    act(() => result.current.zoomOut())
    expect(result.current.center.x).toBeLessThan(coin.x)
    expect(dansLaScene(result.current.level, result.current.center)).toBe(true)
  })

  it('quoi qu on fasse, la fenêtre ne sort jamais de la scène', () => {
    const { result } = render()
    const gestes = [[1, 0], [0, 1], [-1, 0], [0, -1], [1, 1], [-1, -1]]
    for (let i = 0; i < 3; i += 1) {
      act(() => result.current.zoomIn())
      for (const [dx, dy] of gestes) {
        act(() => result.current.panStep(dx * 5, dy * 5))
        expect(dansLaScene(result.current.level, result.current.center)).toBe(true)
      }
    }
  })

  it('le retour ramène au palier 1 ET au centre de la scène', () => {
    const { result } = render()
    act(() => result.current.zoomIn())
    act(() => result.current.panStep(1, 1))
    act(() => result.current.reset())
    expect(result.current.level).toBe(1)
    expect(result.current.center).toEqual(sceneCenter(SCENE))
  })
})

// LA MOLETTE PASSE PAR LE MÊME CHEMIN QUE LES BOUTONS. Ces tests le tiennent : si `zoomAt`
// prenait un jour un raccourci (sauter le rebornage, changer de palier autrement), le zoom au
// bouton et le zoom à la molette cesseraient de donner le même résultat — et personne ne le
// verrait avant de comparer les deux gestes sur la même carte.
describe('useReplayZoom — la molette', () => {
  it('monte et descend les MÊMES paliers que les boutons', () => {
    const { result } = render()
    const cible = { x: 80, y: 50 }
    act(() => result.current.zoomAt(1, cible))
    expect(result.current.level).toBe(ZOOM_LEVELS[1])
    act(() => result.current.zoomAt(-1, cible))
    expect(result.current.level).toBe(1)
  })

  it('ne dépasse jamais les bornes de l échelle', () => {
    const { result } = render()
    const cible = { x: 50, y: 30 }
    for (let i = 0; i < 10; i += 1) act(() => result.current.zoomAt(1, cible))
    expect(result.current.level).toBe(ZOOM_LEVELS[ZOOM_LEVELS.length - 1])
    for (let i = 0; i < 10; i += 1) act(() => result.current.zoomAt(-1, cible))
    expect(result.current.level).toBe(1)
  })

  it('grossir vers un coin y déplace le centre, sans sortir de la scène', () => {
    const { result } = render()
    const avant = { ...result.current.center }
    act(() => result.current.zoomAt(1, { x: SCENE.maxX, y: SCENE.maxY }))
    expect(result.current.center.x).toBeGreaterThan(avant.x)
    expect(result.current.center.y).toBeGreaterThan(avant.y)
    expect(dansLaScene(result.current.level, result.current.center)).toBe(true)
  })

  it('grossir vers le centre ne déplace pas le centre', () => {
    const { result } = render()
    const avant = { ...result.current.center }
    act(() => result.current.zoomAt(1, avant))
    expect(result.current.center.x).toBeCloseTo(avant.x, 9)
    expect(result.current.center.y).toBeCloseTo(avant.y, 9)
  })
})
