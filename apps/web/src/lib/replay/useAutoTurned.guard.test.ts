/**
 * Garde-rail — LA RÈGLE AUTO DE L'ORIENTATION S'APPLIQUE EN UN SEUL ENDROIT (règle ≤ 2 copies, CLAUDE.md n° 6).
 *
 * Trois vues la lisent (le rejeu, la Tactique, « Occupation du terrain ») : chacune avec SA boîte, par
 * `useAutoTurned`. Une vue qui rappellerait `autoTurns(` serait une seconde définition de l'état d'hystérésis,
 * qui diverge. Le test interdit l'appel dans tout fichier de production hors `useAutoTurned.ts` (et la
 * définition, `orientation.ts`).
 */
import { describe, expect, it } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à node:fs.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const HOTE = '/src/lib/replay/useAutoTurned.ts'
const DEFINITION = '/src/lib/replay/orientation.ts'
const APPEL = /\bautoTurns\(/

const production = Object.entries(sources).filter(([path]) => !/\.test\.tsx?$/.test(path))

describe('garde-rail : autoTurns n’est appelé que par useAutoTurned', () => {
  it('aucun autre fichier de production ne l’appelle', () => {
    const fautifs = production
      .filter(([path, code]) => path !== HOTE && path !== DEFINITION && APPEL.test(code))
      .map(([path]) => path)
    expect(fautifs, 'la règle Auto passe par useAutoTurned (une boîte par vue)').toEqual([])
  })

  it('le fichier hôte l’appelle, sans quoi ce garde ne garderait rien', () => {
    expect(APPEL.test(sources[HOTE] ?? '')).toBe(true)
  })

  it('les trois vues passent par lui', () => {
    const lit = (chemin: string) => sources[chemin] ?? ''
    expect(lit('/src/features/match-replay/hooks/useResolvedTurns.ts')).toContain('useAutoTurned(')
    expect(lit('/src/features/match-replay/settings/useMapViewPreferences.ts')).toContain('useAutoTurned(')
  })
})
