/**
 * useTabStripScroll — la barre d'onglets d'Ascension défile et garde l'onglet actif visible.
 */
import { describe, expect, it } from 'vitest'

import { edgeFadeMask, hiddenEdges, scrollLeftToReveal } from './useTabStripScroll'

describe('scrollLeftToReveal', () => {
  it('onglet déjà visible : la barre ne bouge pas', () => {
    expect(scrollLeftToReveal(100, 180, 50, 300)).toBe(50)
  })
  it('onglet coupé à droite (« Tactique » sous 620 px) : son bord droit vient au bord droit', () => {
    expect(scrollLeftToReveal(520, 610, 0, 400)).toBe(210)
  })
  it('onglet caché à gauche : son bord gauche vient au bord gauche', () => {
    expect(scrollLeftToReveal(0, 80, 210, 400)).toBe(0)
  })
})

describe('hiddenEdges / edgeFadeMask', () => {
  it('tout tient : aucun fondu', () => {
    expect(hiddenEdges(0, 600, 600)).toEqual({ before: false, after: false })
    expect(edgeFadeMask({ before: false, after: false })).toBeUndefined()
  })
  it('suite à droite : fondu à droite seulement', () => {
    const edges = hiddenEdges(0, 400, 610)
    expect(edges).toEqual({ before: false, after: true })
    const mask = edgeFadeMask(edges)?.maskImage as string
    expect(mask).toMatch(/transparent\)$/)
    expect(mask).not.toMatch(/\(to right, transparent/)
  })
  it('défilée au bout : fondu à gauche seulement', () => {
    const edges = hiddenEdges(210, 400, 610)
    expect(edges).toEqual({ before: true, after: false })
    expect(edgeFadeMask(edges)?.maskImage).toMatch(/^linear-gradient\(to right, transparent/)
  })
})
