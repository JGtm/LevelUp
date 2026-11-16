/**
 * Garde-rail — LA ROTATION DE LA CARTE S'APPLIQUE EN UN SEUL ENDROIT (règle ≤ 2 copies, CLAUDE.md n° 6).
 *
 * POURQUOI. Le fond tourné était posé par trois vues avec trois écritures de la même rotation : le
 * rejeu (toile), « Occupation du terrain » (toile) et l'onglet Tactique (CSS). Une rotation écrite à la
 * main dans une vue est une quatrième définition du sens des quarts de tour, que `orientation.ts` fixe
 * seul (horaire à l'écran). Les vues appellent `rotateCanvasAbout` (toile) et `cssRotation` (CSS).
 *
 * Ce que le test interdit, dans tout fichier de production hors `lib/replay/orientation.ts` : un
 * `rotate(` dont l'argument porte `turns` (`ctx.rotate((turns * Math.PI) / 2)`, `rotate(${turns * 90}deg)`).
 * Les rotations propres à un dessin (cap d'un projectile, d'une flèche) ne portent pas `turns` et ne sont
 * pas visées.
 */
import { describe, expect, it } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à node:fs.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/lib/replay/orientation.ts'
const ROTATION_DE_CARTE = /rotate\([^)\n]*turns/

const estUnTest = (path: string) => /\.test\.tsx?$/.test(path)
const production = Object.entries(sources).filter(([path]) => !estUnTest(path))

describe('garde-rail : la rotation de la carte ne s’écrit que dans lib/replay/orientation.ts', () => {
  it('aucun rotate( ne porte les quarts de tour de la carte hors du fichier canonique', () => {
    const fautifs = production
      .filter(([path, code]) => path !== CANONICAL && ROTATION_DE_CARTE.test(code))
      .map(([path]) => path)
    expect(fautifs, 'rotation de carte réécrite : appeler rotateCanvasAbout ou cssRotation').toEqual([])
  })

  it('le fichier canonique, lui, la porte — sans quoi ce garde ne garderait rien', () => {
    expect(ROTATION_DE_CARTE.test(sources[CANONICAL] ?? '')).toBe(true)
  })

  it('les trois vues passent par les deux fonctions', () => {
    const lit = (chemin: string) => sources[chemin] ?? ''
    expect(lit('/src/features/match-replay/layers/mapBackgroundDraw.ts')).toContain('rotateCanvasAbout(')
    expect(lit('/src/features/match-view/_positionsPaint.ts')).toContain('rotateCanvasAbout(')
    expect(lit('/src/features/tactical/plan.logic.ts')).toContain('cssRotation(')
  })
})
