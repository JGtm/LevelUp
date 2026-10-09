/**
 * Garde-rail (CLAUDE.md n°6) : le chrome d'une carte de section ne se réécrit pas à la main.
 *
 * Contexte : quatre sections de la page match (`MatchKillDistanceSection`,
 * `MatchObjectivesSection`, `MatchEquipmentUsageSection`, `MatchPadControlSection`) posaient
 * chacune `<section className="rounded-lg border-2 border-border">` avec son propre bandeau
 * de titre, là où le reste de l'app pose le gabarit `ChartCard` / `PaneCard`. Elles ont été
 * migrées sur `components/ui/section-card.tsx` le 2026-09-03. Une factorisation sans
 * garde-rail re-diverge (leçon du prédicat bot, 8 → 36 copies) : ce test interdit le retour
 * du littéral qui SIGNAIT l'ancien gabarit.
 *
 * CE QUE CE TEST SURVEILLE, ET SEULEMENT ÇA : le trait double sur une balise `<section>`,
 * dans les deux features de la page match. `border-2 border-border` reste parfaitement
 * légitime ailleurs — vignette d'identité de l'Explorateur, bloc d'état neutre des
 * superpositions du rejeu (`replayOverlayStyles.ts`), tableau des rencontres : interdire le
 * littéral partout ferait échouer des surfaces qui n'ont rien à voir avec une carte de
 * section.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — même mécanique que
// sortable-th.guard.test.ts, pas de dépendance à node:fs.
const sources = import.meta.glob('/src/features/{match-view,match-replay}/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

// Une ouverture de <section ...> dont les attributs portent le trait double.
const SECTION_BORDER_2 = /<section\b[^>]*\bborder-2\s+border-border\b/

describe('garde-rail gabarit de carte de section (SectionCard source unique)', () => {
  it('la page match ne contient aucun fichier source à scanner de moins', () => {
    // Un glob qui ne matche plus rien rendrait le test vert pour de mauvaises raisons.
    expect(Object.keys(sources).length).toBeGreaterThan(50)
  })

  it('aucune <section> ne réécrit le chrome à trait double dans match-view / match-replay', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([, code]) => SECTION_BORDER_2.test(code))
      .map(([path]) => path)
    expect(
      offenders,
      `Chrome de section à migrer vers SectionCard (components/ui/section-card.tsx) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
