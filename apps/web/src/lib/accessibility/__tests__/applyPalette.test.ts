/**
 * applyPalette.test.ts — Tests du resolver CSS vars.
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { applyPalette, _resetActivePalette } from '../applyPalette'
import { defaultPalette } from '../palettes/default'
import { okabePalette } from '../palettes/okabe-ito'
import { ALL_TOKENS, tokenVar } from '../semantic-tokens'
import { log } from '../_logger'

// jsdom ne peuple pas getComputedStyle depuis style.setProperty —
// on inspecte directement root.style
function getVar(token: string): string {
  return document.documentElement.style.getPropertyValue(tokenVar(token as never))
}

beforeEach(() => {
  _resetActivePalette()
  log._resetForTests()
  document.documentElement.removeAttribute('style')
})

afterEach(() => {
  document.documentElement.removeAttribute('style')
})

describe('applyPalette', () => {
  it('écrit toutes les CSS vars sur :root', () => {
    applyPalette(defaultPalette, 'default')
    for (const token of ALL_TOKENS) {
      expect(getVar(token), `var manquante pour ${token}`).not.toBe('')
    }
  })

  it('est idempotent — 2e appel avec la même clé est ignoré', () => {
    const spy = vi.spyOn(document.documentElement.style, 'setProperty')
    applyPalette(defaultPalette, 'default')
    const callsAfterFirst = spy.mock.calls.length
    applyPalette(defaultPalette, 'default')
    expect(spy.mock.calls.length).toBe(callsAfterFirst)
    spy.mockRestore()
  })

  it('remplace correctement la palette quand la clé change', () => {
    applyPalette(defaultPalette, 'default')
    applyPalette(okabePalette, 'okabe-ito')
    expect(getVar('outcome-win')).toBe(okabePalette['outcome-win'])
  })

  it('cleanup — revenir à default restaure les valeurs initiales', () => {
    applyPalette(defaultPalette, 'default')
    const winBefore = getVar('outcome-win')
    applyPalette(okabePalette, 'okabe-ito')
    _resetActivePalette()
    applyPalette(defaultPalette, 'default')
    expect(getVar('outcome-win')).toBe(winBefore)
  })

  it('ne modifie pas les CSS vars hors namespace --ac-*', () => {
    applyPalette(defaultPalette, 'default')
    // Variables structurelles de globals.css ne doivent pas être touchées
    const background = document.documentElement.style.getPropertyValue('--background')
    expect(background).toBe('')
  })

  it('loggue 1 info par changement de palette', () => {
    const spy = vi.spyOn(console, 'info').mockImplementation(() => {})
    applyPalette(defaultPalette, 'default')
    expect(spy).toHaveBeenCalledTimes(1)
    applyPalette(defaultPalette, 'default') // idempotent — pas de 2e log
    expect(spy).toHaveBeenCalledTimes(1)
    spy.mockRestore()
  })
})
