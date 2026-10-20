/**
 * Garde-rail — UN SEUL PEINTRE DE ZONES NOMMÉES (règle ≤ 2 copies, CLAUDE.md n° 6 ; plan Tactique
 * v2, lot L13 F7).
 *
 * POURQUOI. Le rejeu 2D dessinait les zones nommées (callouts) dans son propre calque ; le plan de
 * l'onglet Tactique les dessine à son tour. Le calque a été DÉPLACÉ dans `lib/replay/calloutsPaint.ts`
 * (comme le peintre de chaleur, `heatPaint.ts`) et les deux vues l'importent : un second peintre
 * donnerait aux mêmes zones deux aspects, et la règle de nommage du rejeu divergerait.
 *
 * Ce que le test interdit :
 *   1. une DÉFINITION d'une fonction du peintre hors de `calloutsPaint.ts` ;
 *   2. une lecture des zones BRUTES du contrat (`ReplayCalloutZone`, `CalloutZone`) hors du peintre
 *      et des types d'API — c'est par là que commencerait une seconde normalisation ;
 *   3. le retour de l'ancien fichier du rejeu (`calloutsLayer`) ;
 *   4. des zones sur les vignettes de la colonne des cartes (elles n'en portent aucune).
 */
import { describe, expect, it } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à node:fs.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/lib/replay/calloutsPaint.ts'
const TYPES_API = ['/src/lib/api/types.ts', '/src/lib/api/generated.ts']
const VIGNETTE = '/src/features/tactical/TacticalMapTile.tsx'

const NOMS = [
  'normalizeCallouts',
  'normalizeCalloutZones',
  'calloutLabel',
  'drawCalloutsLayer',
  'drawCalloutsShapes',
  'drawCalloutsLabels',
]

const estUnTest = (path: string) => /\.test\.tsx?$/.test(path)
const production = Object.entries(sources).filter(([path]) => !estUnTest(path))

/** La signature d'une DÉFINITION (jamais un import ni un appel). */
function definitionRe(nom: string): RegExp {
  return new RegExp(`\\bfunction\\s+${nom}\\s*[(<]|\\b(?:const|let)\\s+${nom}\\s*[=:]`)
}

const ZONES_BRUTES = /\bReplayCalloutZone\b|\['CalloutZone'\]/

describe('garde-rail : un seul peintre de zones nommées (lib/replay/calloutsPaint.ts)', () => {
  it.each(NOMS)('%s n’est défini que dans calloutsPaint.ts', (nom) => {
    const re = definitionRe(nom)
    const offenders = production.filter(([path, code]) => path !== CANONICAL && re.test(code)).map(([path]) => path)
    expect(offenders, `${nom} redéfini hors du peintre partagé — importer lib/replay/calloutsPaint`).toEqual([])
  })

  it('et le peintre, lui, définit bien les six — sans quoi ce garde ne garderait rien', () => {
    const code = sources[CANONICAL] ?? ''
    for (const nom of NOMS) expect(definitionRe(nom).test(code), `${nom} absent de calloutsPaint.ts`).toBe(true)
  })

  it('les zones brutes du contrat ne se lisent que dans le peintre', () => {
    const offenders = production
      .filter(([path, code]) => path !== CANONICAL && !TYPES_API.includes(path) && ZONES_BRUTES.test(code))
      .map(([path]) => path)
    expect(offenders, 'zones brutes lues hors du peintre — passer par normalizeCalloutZones').toEqual([])
    expect(ZONES_BRUTES.test(sources[CANONICAL] ?? '')).toBe(true)
  })

  it('l’ancien calque du rejeu n’existe plus (déplacement, pas copie)', () => {
    expect(Object.keys(sources).filter((path) => /calloutsLayer/.test(path))).toEqual([])
  })

  it('les vignettes de la colonne des cartes ne dessinent aucune zone', () => {
    const code = sources[VIGNETTE]
    expect(code, `${VIGNETTE} introuvable — mettre le garde-rail à jour`).toBeDefined()
    expect(code).not.toMatch(/calloutsPaint|planPaint/)
  })
})
