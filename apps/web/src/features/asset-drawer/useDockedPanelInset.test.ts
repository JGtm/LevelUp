/**
 * useDockedPanelInset : la largeur que le volet ancré prend au contenu — 0 fermé, sa largeur bornée
 * (mêmes bornes que le volet) ouvert, recalculée au redimensionnement de la fenêtre.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { useAssetDrawerStore } from './assetDrawer.store'
import { clampPanelWidth, PANEL_DEFAULT_WIDTH, PANEL_MAX_WIDTH } from './panelWidth'
import { useDockedPanelInset } from './useDockedPanelInset'

const initialInnerWidth = window.innerWidth

function setViewport(width: number) {
  Object.defineProperty(window, 'innerWidth', { configurable: true, writable: true, value: width })
}

beforeEach(() => {
  setViewport(1600)
  useAssetDrawerStore.setState({ isOpen: false, panelWidth: PANEL_DEFAULT_WIDTH })
})
afterEach(() => {
  setViewport(initialInnerWidth)
  useAssetDrawerStore.setState({ isOpen: false })
})

describe('useDockedPanelInset', () => {
  it('fermé : 0, quelle que soit la largeur mémorisée', () => {
    useAssetDrawerStore.setState({ isOpen: false, panelWidth: 500 })
    const { result } = renderHook(() => useDockedPanelInset())
    expect(result.current).toBe(0)
  })

  it('ouvert : la largeur du volet, bornée comme le volet lui-même', () => {
    useAssetDrawerStore.setState({ isOpen: true, panelWidth: 500 })
    const { result } = renderHook(() => useDockedPanelInset())
    expect(result.current).toBe(clampPanelWidth(500, 1600))
    expect(result.current).toBe(500)
  })

  it('ouvert avec une largeur démesurée : ramenée à la borne haute', () => {
    useAssetDrawerStore.setState({ isOpen: true, panelWidth: 5000 })
    const { result } = renderHook(() => useDockedPanelInset())
    expect(result.current).toBe(PANEL_MAX_WIDTH)
  })

  it('suit l’ouverture, la fermeture et le redimensionnement de la fenêtre', () => {
    useAssetDrawerStore.setState({ isOpen: false, panelWidth: 600 })
    const { result } = renderHook(() => useDockedPanelInset())
    expect(result.current).toBe(0)
    act(() => useAssetDrawerStore.setState({ isOpen: true }))
    expect(result.current).toBe(600)
    // Fenêtre étroite : la borne haute recule pour laisser sa place au contenu.
    act(() => {
      setViewport(900)
      window.dispatchEvent(new Event('resize'))
    })
    expect(result.current).toBe(clampPanelWidth(600, 900))
    expect(result.current).toBeLessThan(600)
    act(() => useAssetDrawerStore.setState({ isOpen: false }))
    expect(result.current).toBe(0)
  })
})
