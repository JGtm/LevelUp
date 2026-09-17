/**
 * themeInk.test.ts — L'EXPORT SE PEINT DANS LE THÈME SOMBRE, la page garde le sien (D8/D9).
 *
 * La feuille de style RÉELLE du dépôt (`styles/globals.css`) est injectée dans le document : ce
 * qui se vérifie est donc la cascade telle que le thème l'écrit, pas une copie de test.
 */
/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { act, renderHook } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { requestExportLayout } from '../export/exportLayoutStore'
import { racineWeb } from '../test/featureFiles'
import { readInk } from './canvasInk'
import { readFxInk } from './fxInk'
import { readThemeVar } from './themeInk'
import { useReplayInks } from './useReplayInks'
import { withLoadedImage } from './loadedImage'

let feuille: HTMLStyleElement

beforeAll(() => {
  feuille = document.createElement('style')
  feuille.textContent = readFileSync(resolve(racineWeb(), 'src/styles/globals.css'), 'utf8')
  document.head.appendChild(feuille)
})

afterAll(() => {
  feuille.remove()
})

afterEach(() => {
  requestExportLayout(null)
  document.documentElement.setAttribute('data-theme', 'dark')
  document.documentElement.style.removeProperty('--ac-team-ally')
})

/** Ce que la page lit, hors export, dans un thème donné. */
function horsExport(theme: 'dark' | 'light', lire: () => unknown): unknown {
  document.documentElement.setAttribute('data-theme', theme)
  return lire()
}

const exporter = () => requestExportLayout(exportLayoutFor(exportFormatOf('1080p')))

describe('themeInk — encres de l’export', () => {
  const ENCRES = ['--foreground', '--background', '--muted-foreground', '--border', '--card'] as const

  it('le theme sombre et le theme clair different bien (sinon ce test ne prouverait rien)', () => {
    expect(horsExport('dark', () => readInk('--foreground'))).not.toBe(horsExport('light', () => readInk('--foreground')))
  })

  it('theme CLAIR actif : les encres de l’export sont celles du theme SOMBRE, et la page reste claire', () => {
    const sombre = horsExport('dark', () => ENCRES.map(readInk))
    const fxSombre = horsExport('dark', () => readFxInk())
    const clair = horsExport('light', () => ENCRES.map(readInk))
    exporter()
    expect(ENCRES.map(readInk)).toEqual(sombre)
    expect(readFxInk()).toEqual(fxSombre)
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    requestExportLayout(null)
    expect(ENCRES.map(readInk)).toEqual(clair)
  })

  it('le fond de la video est NOIR PUR dans les deux themes, export ou pas', () => {
    expect(horsExport('dark', () => readInk('--replay-export-backdrop'))).toBe('rgb(0 0 0)')
    expect(horsExport('light', () => readInk('--replay-export-backdrop'))).toBe('rgb(0 0 0)')
    exporter()
    expect(readInk('--replay-export-backdrop')).toBe('rgb(0 0 0)')
  })

  it('le style en ligne de la racine (palettes d’accessibilite) l’emporte, comme dans le navigateur', () => {
    document.documentElement.setAttribute('data-theme', 'light')
    document.documentElement.style.setProperty('--ac-team-ally', 'rgb(1 2 3)')
    exporter()
    expect(readThemeVar('--ac-team-ally')).toBe('rgb(1 2 3)')
  })

  it('une reference var() se resout dans le theme de l’export', () => {
    const sombre = horsExport('dark', () => readInk('--muted-foreground'))
    document.documentElement.setAttribute('data-theme', 'light')
    exporter()
    // `--narrative-trend-neutral: var(--muted-foreground)` dans le bloc sombre.
    expect(readThemeVar('--narrative-trend-neutral')).toBe(sombre)
  })
})

describe('useReplayInks — le memo suit l’entree et la sortie de l’export', () => {
  it('theme clair : encres sombres pendant l’export, encres claires apres, sans toucher au theme', () => {
    const sombre = JSON.stringify(horsExport('dark', () => renderHook(() => useReplayInks(0)).result.current))
    document.documentElement.setAttribute('data-theme', 'light')
    const { result } = renderHook(() => useReplayInks(0))
    const clair = JSON.stringify(result.current)
    expect(clair).not.toBe(sombre)
    act(() => exporter())
    expect(JSON.stringify(result.current)).toBe(sombre)
    act(() => requestExportLayout(null))
    expect(JSON.stringify(result.current)).toBe(clair)
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })
})

describe('withLoadedImage — une vignette deja chargee se reteint SANS attendre', () => {
  it('le premier appel attend le chargement, les suivants rendent l’image tout de suite', () => {
    const creees: { onload: (() => void) | null; src: string }[] = []
    class FausseImage {
      onload: (() => void) | null = null
      src = ''
      constructor() {
        creees.push(this)
      }
    }
    vi.stubGlobal('Image', FausseImage)
    try {
      const url = '/vignette-test-' + Math.random()
      const recues: unknown[] = []
      withLoadedImage(url, (im) => recues.push(im))
      expect(recues).toHaveLength(0)
      creees[0].onload?.()
      expect(recues).toEqual([creees[0]])
      withLoadedImage(url, (im) => recues.push(im))
      expect(recues).toEqual([creees[0], creees[0]])
      expect(creees).toHaveLength(1)
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
