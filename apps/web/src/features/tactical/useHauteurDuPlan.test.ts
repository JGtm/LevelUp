/**
 * useHauteurDuPlan — le plan suit la RÈGLE DE HAUTEUR DE LA TOILE DU REJEU (retour du 2026-10-17) : même
 * plancher (360), même plafond (720), même marge basse (16), même pas (8), et le chrome — légende, zone
 * « séquence » et barre de lecture comprises — déduit de ce que la fenêtre laisse au cadre.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  fitCanvasHeight,
  VIEWPORT_BOTTOM_MARGIN,
  VIEWPORT_HEIGHT_STEP,
} from '@/features/match-replay/hooks/useReplayViewport'
import {
  CANVAS_HEIGHT_CEILING,
  CANVAS_HEIGHT_MIN,
} from '@/features/match-replay/hooks/useReplayView'

import { useHauteurDuPlan } from './useHauteurDuPlan'

const TOP = 120
let peint = 0
let chrome = 0
let carte: HTMLDivElement
let cadre: HTMLDivElement

/** La carte porte le cadre ET le chrome (titre, légende, zone séquence) : sa hauteur est leur somme. */
function monter() {
  carte = document.createElement('div')
  cadre = document.createElement('div')
  Object.defineProperty(carte, 'offsetHeight', { configurable: true, get: () => chrome + peint })
  carte.getBoundingClientRect = () => ({ top: TOP, left: 0, width: 900, height: chrome + peint }) as DOMRect
  cadre.getBoundingClientRect = () => ({ top: TOP, left: 0, width: 900, height: peint }) as DOMRect
  carte.appendChild(cadre)
  document.body.appendChild(carte)
}

function mesurer(hauteurFenetre: number, chromeDuPlan: number) {
  window.innerHeight = hauteurFenetre
  chrome = chromeDuPlan
  peint = 480
  const { result } = renderHook(() => useHauteurDuPlan({ current: carte }, { current: cadre }))
  return result.current
}

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} })
  monter()
})
afterEach(() => {
  carte.remove()
  vi.unstubAllGlobals()
})

describe('useHauteurDuPlan — la règle du rejeu', () => {
  it('sur un grand écran : le plafond du rejeu (720), pas davantage', () => {
    expect(mesurer(2000, 200)).toBe(CANVAS_HEIGHT_CEILING)
  })

  it('sur un écran très court : le plancher du rejeu (360)', () => {
    expect(mesurer(500, 200)).toBe(CANVAS_HEIGHT_MIN)
  })

  it('entre les deux : ce qui reste sous le haut de la carte, chrome et marge basse de 16 px déduits, au pas de 8', () => {
    const hauteur = mesurer(900, 220)
    expect(hauteur).toBe(fitCanvasHeight(900 - TOP - 220 - VIEWPORT_BOTTOM_MARGIN))
    expect(hauteur % VIEWPORT_HEIGHT_STEP).toBe(0)
    expect(VIEWPORT_BOTTOM_MARGIN).toBe(16)
  })

  it('la légende et la barre de lecture comptent : un pied plus haut rétrécit le cadre d’autant', () => {
    const sans = mesurer(900, 150)
    carte.remove()
    monter()
    const avec = mesurer(900, 150 + 96 + 8)
    expect(sans - avec).toBeGreaterThanOrEqual(96)
    expect(sans - avec).toBeLessThan(96 + 8 + VIEWPORT_HEIGHT_STEP)
  })
})
