/**
 * Garde-rail — UN SEUL PEINTRE DE CHALEUR (règle ≤ 2 copies, CLAUDE.md n° 6 ; Q7, 2026-09-07).
 *
 * POURQUOI. Le rejeu et le tactique peignaient chacun leur propre `buildHeatmap` /
 * `drawHeatmapLayer` / `buildTacticalGrid` / `drawTacticalHeatmap` / `heatRamp` : deux
 * copies d'un même patron, la règle du dépôt les tolérait tout juste. Le lot Q7 les a
 * fusionnées dans `lib/replay/heatPaint.ts` ; ce test interdit qu'une troisième réapparaisse
 * (ou qu'une des deux réapparaisse hors du noyau).
 *
 * Une factorisation sans garde-rail re-diverge — leçon du prédicat bot, passé de 8 à 36
 * copies après centralisation.
 */
import { describe, expect, it } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à
// node:fs ni aux types node dans le tsconfig applicatif.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/lib/replay/heatPaint.ts'

/** Les cinq signatures du noyau — la copie du rejeu et celle du tactique les portaient
 *  toutes les cinq avant fusion (voir le thought_log du 2026-09-07). */
const NOMS = ['buildHeatmap', 'drawHeatmapLayer', 'drawTacticalHeatmap', 'buildTacticalGrid', 'heatRamp']

/** La signature d'une DÉFINITION (jamais un import ni un appel) : `function <nom>(`. */
function definitionRe(nom: string): RegExp {
  return new RegExp(`\\bfunction\\s+${nom}\\s*\\(`)
}

describe('garde-rail : un seul peintre de chaleur (lib/replay/heatPaint.ts)', () => {
  it.each(NOMS)('%s n’est défini que dans heatPaint.ts', (nom) => {
    const re = definitionRe(nom)
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL)
      .filter(([, code]) => re.test(code))
      .map(([path]) => path)
    expect(
      offenders,
      `${nom} redéfini hors du noyau canonique : ${offenders.join(', ')} — ` +
        'importer depuis lib/replay/heatPaint plutôt que de re-peindre.',
    ).toEqual([])
  })

  it('et le noyau, lui, définit bien les cinq — sans quoi ce garde ne garderait rien', () => {
    const code = sources[CANONICAL] ?? ''
    for (const nom of NOMS) {
      expect(definitionRe(nom).test(code), `${nom} absent de heatPaint.ts`).toBe(true)
    }
  })
})
