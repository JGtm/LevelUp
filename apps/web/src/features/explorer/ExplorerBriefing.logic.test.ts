import { describe, it, expect } from 'vitest'
import { signOf } from './ExplorerBriefing.logic'

// formatSignedFixed a migré vers `@/lib/formatters` (number.ts) — testé dans
// `lib/formatters/formatters.test.ts`.
//
// formatSignedPoints et isFullHistoryScope ont migré vers `@/lib/baseline` le
// 2026-09-06 (2e consommateur : le KPI d'échange de l'Escouade) — testés dans
// `lib/baseline.test.ts`.

describe('signOf', () => {
  it('retourne -1 / 0 / 1', () => {
    expect(signOf(2)).toBe(1)
    expect(signOf(-2)).toBe(-1)
    expect(signOf(0)).toBe(0)
    expect(signOf(null)).toBe(0)
  })
})
