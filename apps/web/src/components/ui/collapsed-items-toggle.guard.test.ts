/**
 * Garde-rail — foyer canonique de CollapsedItemsToggle (règle <= 2 copies, CLAUDE.md n°6).
 *
 * Le bouton « Voir plus (N) / Replier » du repli « game changers » vivait en deux copies
 * locales identiques (`MatchEquipmentUsageSection.CollapsedColumnsToggle`,
 * `MatchPadControlSection.CollapsedWeaponsToggle`) — patron `MedalDigest.tsx:281-294`,
 * accepté jusqu'à 2 copies par le plan (D3). Le lot H (`.ai/PLAN_REPLI_GAME_CHANGERS_2026-09-05.md`,
 * H-D5, H2) les a migrées vers `components/ui/collapsed-items-toggle.tsx`. Ce test interdit
 * toute RÉ-INLINE du bouton — même patron que `sortable-th.guard.test.ts` : une factorisation
 * sans garde-rail re-diverge (leçon du prédicat bot, 8 -> 36 copies).
 *
 * CE QUE CE TEST SURVEILLE : une déclaration de fonction/composant nommée comme le bouton de
 * repli (`CollapsedColumnsToggle`, `CollapsedWeaponsToggle`, ou `CollapsedItemsToggle` lui-même)
 * hors du foyer canonique — pas le nom générique « toggle », trop répandu ailleurs dans l'app
 * pour servir de signal sans faux positifs.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — même mécanique que
// sortable-th.guard.test.ts, pas de dépendance à node:fs.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/components/ui/collapsed-items-toggle.tsx'

// Les DEUX anciens noms locaux (lots G1/G2, retirés par la migration) + le nom canonique
// lui-même : aucun des trois ne doit être (re)déclaré ailleurs que dans le foyer.
const OFFENDING_DECLARATIONS = [
  /function\s+CollapsedColumnsToggle\b/,
  /function\s+CollapsedWeaponsToggle\b/,
  /function\s+CollapsedItemsToggle\b/,
  /const\s+CollapsedColumnsToggle\s*=/,
  /const\s+CollapsedWeaponsToggle\s*=/,
  /const\s+CollapsedItemsToggle\s*=/,
]

describe('garde-rail CollapsedItemsToggle canonique', () => {
  it('le dépôt garde assez de fichiers source à scanner (glob non cassé)', () => {
    // Un glob qui ne matche plus rien rendrait ce test vert pour de mauvaises raisons.
    expect(Object.keys(sources).length).toBeGreaterThan(200)
  })

  it('le foyer canonique existe et déclare bien le composant', () => {
    const canonicalSource = sources[CANONICAL]
    expect(canonicalSource, `${CANONICAL} devrait être chargé par le glob`).toBeTruthy()
    expect(/function\s+CollapsedItemsToggle\b/.test(canonicalSource ?? '')).toBe(true)
  })

  it("le bouton de repli n'est déclaré nulle part ailleurs que dans son foyer canonique", () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL)
      .filter(([path]) => !/\.(test|guard\.test)\.tsx?$/.test(path))
      .filter(([, code]) => OFFENDING_DECLARATIONS.some((re) => re.test(code)))
      .map(([path]) => path)
    expect(
      offenders,
      `Bouton de repli ré-inliné hors du foyer canonique (${CANONICAL}) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
