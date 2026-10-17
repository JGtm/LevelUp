/**
 * Garde-rail (CLAUDE.md n°6) : le constat « aucun match concerné » d'un bloc qui reste affiché ne
 * s'écrit qu'UNE fois (`scope-empty-text.ts`), et son bloc parent qu'en un composant (`scope-empty.tsx`).
 * Plus de trois blocs (Escouade, Sessions, Séries temporelles, anneaux de frags) portent le même
 * constat : sans garde, chaque page le reformule à sa façon.
 */
import { describe, expect, it } from 'vitest'

const PHRASES = /Aucun match à objectif sur ce périmètre|No objective match in this scope|Aucune action d’objectif sur ce périmètre/
const OWNER = /\/components\/ui\/scope-empty-text\.ts$/

const withoutComments = (src: string) => src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '')

describe('garde « aucun match concerné »', () => {
  it('les constats ne s’écrivent que dans scope-empty-text.ts', () => {
    const files = import.meta.glob('/src/**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<
      string,
      string
    >
    const offenders = Object.entries(files)
      .filter(([path]) => !OWNER.test(path) && !/\.test\.tsx?$/.test(path) && !/\/lib\/i18n\/generated\//.test(path))
      .filter(([, src]) => PHRASES.test(withoutComments(src)))
      .map(([path]) => path)
    expect(offenders, `utiliser scopeEmptyText / ScopeEmptyCard :\n${offenders.join('\n')}`).toEqual([])
  })
})
