/**
 * useLectureDeSequence — la lecture animée : départ automatique à −4 s, avance en temps réel, arrêt sur
 * +3 s, relance, pause et saut à une étape ; sous mouvement réduit, aucune lecture automatique.
 * Les images d'animation sont pilotées à la main (`image(ms)`).
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TacticalSequence } from '@/lib/api/types'

import { SEQUENCE } from './mort.fixtures'
import { useLectureDeSequence } from './useLectureDeSequence'

let file: Map<number, FrameRequestCallback>
let prochain: number
let horloge: number

/** Rend `ms` de temps réel, en images de 16 ms. */
function image(ms: number) {
  const pas = 16
  for (let fait = 0; fait < ms; fait += pas) {
    horloge += pas
    const callbacks = [...file.values()]
    file.clear()
    act(() => callbacks.forEach((cb) => cb(horloge)))
  }
}

beforeEach(() => {
  file = new Map()
  prochain = 1
  horloge = 1000
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    file.set(prochain, cb)
    return prochain++
  })
  vi.stubGlobal('cancelAnimationFrame', (id: number) => file.delete(id))
})
afterEach(() => vi.unstubAllGlobals())

describe('useLectureDeSequence', () => {
  it('sans séquence : pas de lecture', () => {
    const { result } = renderHook(() => useLectureDeSequence(null, false))
    expect(result.current.lecture).toBeNull()
    expect(result.current.etape).toBeNull()
    expect(file.size).toBe(0)
  })

  it('part tout seul à −4 s, avance en temps réel et suit l’étape en vigueur', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    expect(result.current.lecture?.etat).toBe('lecture')
    expect(result.current.lecture?.horloge.lire()).toBe(-4000)
    expect(result.current.etape).toBe(0)
    image(2016)
    // 2 s de temps réel : environ −2 s, étape −2 s (index 2).
    expect(result.current.lecture?.horloge.lire()).toBeGreaterThan(-2100)
    expect(result.current.lecture?.horloge.lire()).toBeLessThan(-1900)
    expect(result.current.etape).toBe(2)
  })

  it('s’arrête sur +3 s et le dit ; l’horloge ne dépasse pas', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    image(7200)
    expect(result.current.lecture?.etat).toBe('fin')
    expect(result.current.lecture?.horloge.lire()).toBe(3000)
    expect(result.current.etape).toBe(7)
    expect(file.size).toBe(0)
  })

  it('« Rejouer » repart du début depuis la fin', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    image(7200)
    act(() => result.current.lecture?.basculer())
    expect(result.current.lecture?.etat).toBe('lecture')
    expect(result.current.lecture?.horloge.lire()).toBe(-4000)
    expect(result.current.etape).toBe(0)
  })

  it('pause en cours de route, puis reprise de là où la tête s’est arrêtée', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    image(1000)
    act(() => result.current.lecture?.basculer())
    expect(result.current.lecture?.etat).toBe('pause')
    const arret = result.current.lecture?.horloge.lire() ?? 0
    image(500)
    expect(result.current.lecture?.horloge.lire()).toBe(arret)
    act(() => result.current.lecture?.basculer())
    expect(result.current.lecture?.etat).toBe('lecture')
    image(160)
    expect(result.current.lecture?.horloge.lire()).toBeGreaterThan(arret)
  })

  it('un clic sur une étape met en pause et saute à son décalage', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    image(500)
    act(() => result.current.choisirEtape(4))
    expect(result.current.lecture?.etat).toBe('pause')
    expect(result.current.lecture?.horloge.lire()).toBe(0)
    expect(result.current.etape).toBe(4)
    image(500)
    expect(result.current.lecture?.horloge.lire()).toBe(0)
  })

  it('« Instantané » met en pause et rend la vue de la mort (étape nulle)', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    image(300)
    act(() => result.current.choisirEtape(null))
    expect(result.current.etape).toBeNull()
    expect(result.current.lecture?.etat).toBe('pause')
  })

  it('reprendre une pause posée sur la dernière étape rejoue au lieu de s’arrêter aussitôt', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    act(() => result.current.choisirEtape(7))
    act(() => result.current.lecture?.basculer())
    expect(result.current.lecture?.etat).toBe('lecture')
    expect(result.current.lecture?.horloge.lire()).toBe(-4000)
  })

  it('mouvement réduit : aucune lecture automatique, l’étape de la mort est affichée', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, true))
    expect(result.current.lecture?.etat).toBe('pause')
    expect(result.current.lecture?.horloge.lire()).toBe(0)
    expect(result.current.etape).toBe(4)
    image(500)
    expect(result.current.lecture?.horloge.lire()).toBe(0)
    // La lecture reste possible sur demande.
    act(() => result.current.lecture?.basculer())
    expect(result.current.lecture?.etat).toBe('lecture')
  })

  it('une autre mort repart de son début, sans une image de l’ancienne tête', () => {
    const autre: TacticalSequence = { ...SEQUENCE, mort: { x: 1, y: 1 } }
    const { result, rerender } = renderHook(({ s }) => useLectureDeSequence(s, false), { initialProps: { s: SEQUENCE } })
    image(3000)
    rerender({ s: autre })
    expect(result.current.lecture?.horloge.lire()).toBe(-4000)
    expect(result.current.lecture?.etat).toBe('lecture')
    expect(result.current.etape).toBe(0)
  })

  it('abonne l’horloge : chaque image prévient ses abonnés', () => {
    const { result } = renderHook(() => useLectureDeSequence(SEQUENCE, false))
    const rappel = vi.fn()
    const quitter = result.current.lecture?.horloge.abonner(rappel)
    image(48)
    expect(rappel.mock.calls.length).toBeGreaterThanOrEqual(2)
    quitter?.()
    const n = rappel.mock.calls.length
    image(48)
    expect(rappel.mock.calls.length).toBe(n)
  })
})
