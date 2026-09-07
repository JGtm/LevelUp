import { describe, it, expect } from 'vitest'
import { formatSignedPoints, isFullHistoryScope } from './baseline'

// Ces cas viennent de `features/explorer/ExplorerBriefing.logic.test.ts` : ils ont
// suivi les helpers dans `lib/` le 2026-09-06 (2e consommateur : le KPI d'échange
// de l'Escouade). Ils ne sont pas réécrits — un déplacement qui perd ses tests
// perd son filet.

describe('formatSignedPoints', () => {
  it('convertit un ratio en points de pourcentage signés', () => {
    expect(formatSignedPoints(0.3)).toBe('+30 pts')
    expect(formatSignedPoints(-0.12)).toBe('−12 pts')
    expect(formatSignedPoints(0)).toBe('±0 pts')
  })
  it('rend une chaîne vide quand il n’y a pas d’écart à afficher', () => {
    expect(formatSignedPoints(null)).toBe('')
    expect(formatSignedPoints(undefined)).toBe('')
    expect(formatSignedPoints(Number.NaN)).toBe('')
  })
  it('utilise le glyphe « − » (U+2212), jamais le tiret ASCII', () => {
    expect(formatSignedPoints(-0.05).startsWith('\u2212')).toBe(true)
  })
})

describe('isFullHistoryScope', () => {
  it('vrai quand scope == baseline (aucun filtre)', () => {
    expect(isFullHistoryScope(120, 120)).toBe(true)
  })
  it('faux quand le scope est un sous-ensemble filtré', () => {
    expect(isFullHistoryScope(30, 120)).toBe(false)
  })
  it('faux sans baseline (aucun delta à masquer de toute façon)', () => {
    expect(isFullHistoryScope(120, null)).toBe(false)
    expect(isFullHistoryScope(120, undefined)).toBe(false)
  })
  it('faux quand le scope est absent', () => {
    expect(isFullHistoryScope(null, 120)).toBe(false)
    expect(isFullHistoryScope(undefined, undefined)).toBe(false)
  })
})
