/**
 * Garde-rail (CLAUDE.md n°6) : la règle « une carte seule prend toute la rangée »
 * (`lg:[&>*:only-child]:col-span-2`) ne s'écrit qu'UNE fois, dans `pair-row.ts`. Cinq copies
 * avaient divergé, et la rangée « Frags par ressource | Rendement » de la Vue match l'avait oubliée.
 */
import { describe, expect, it } from 'vitest'

describe('rangée de deux cartes', () => {
  it('le littéral de la rangée n’existe que dans components/ui/pair-row.ts', () => {
    const files = import.meta.glob('/src/**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<
      string,
      string
    >
    const offenders = Object.entries(files)
      .filter(([path, src]) => !path.endsWith('/components/ui/pair-row.ts') && !/\.test\.tsx?$/.test(path) && src.includes('lg:[&>*:only-child]:col-span-2'))
      .map(([path]) => path)
    expect(offenders, `utiliser PAIR_ROW_CLASS (components/ui/pair-row) :\n${offenders.join('\n')}`).toEqual([])
  })
})
