/**
 * Tests — largeur du volet ancré : bornes, borne haute selon la fenêtre, mémorisation tolérante.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  PANEL_DEFAULT_WIDTH,
  PANEL_MAX_WIDTH,
  PANEL_MIN_WIDTH,
  PANEL_WIDTH_STORAGE_KEY,
  clampPanelWidth,
  loadPanelWidth,
  maxPanelWidth,
  savePanelWidth,
} from './panelWidth'

afterEach(() => {
  window.localStorage.clear()
  vi.restoreAllMocks()
})

describe('clampPanelWidth', () => {
  it('garde la largeur entre 280 et 640 px', () => {
    expect(clampPanelWidth(100, 1920)).toBe(PANEL_MIN_WIDTH)
    expect(clampPanelWidth(400, 1920)).toBe(400)
    expect(clampPanelWidth(5000, 1920)).toBe(PANEL_MAX_WIDTH)
  })

  it('arrondit au pixel et traite une valeur non finie comme la largeur par défaut', () => {
    expect(clampPanelWidth(400.6, 1920)).toBe(401)
    expect(clampPanelWidth(Number.NaN, 1920)).toBe(PANEL_DEFAULT_WIDTH)
  })

  it('la borne haute recule quand la fenêtre est étroite, jamais sous le minimum', () => {
    // 1024 px de fenêtre : 240 de barre latérale et 480 de contenu à préserver, il reste 304 px.
    expect(maxPanelWidth(1024)).toBe(304)
    expect(clampPanelWidth(600, 1024)).toBe(304)
    expect(maxPanelWidth(600)).toBe(PANEL_MIN_WIDTH)
  })
})

describe('mémorisation de la largeur', () => {
  it('relit ce qui a été écrit, sous la clé nommée', () => {
    savePanelWidth(450)
    expect(window.localStorage.getItem(PANEL_WIDTH_STORAGE_KEY)).toBe('450')
    expect(loadPanelWidth()).toBe(450)
  })

  it('rien de mémorisé ou valeur illisible : la largeur par défaut', () => {
    expect(loadPanelWidth()).toBe(PANEL_DEFAULT_WIDTH)
    window.localStorage.setItem(PANEL_WIDTH_STORAGE_KEY, 'abc')
    expect(loadPanelWidth()).toBe(PANEL_DEFAULT_WIDTH)
  })

  it('un stockage qui refuse la lecture ou l’écriture ne casse rien', () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('accès refusé')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('quota')
    })
    expect(loadPanelWidth()).toBe(PANEL_DEFAULT_WIDTH)
    expect(() => savePanelWidth(400)).not.toThrow()
  })
})
