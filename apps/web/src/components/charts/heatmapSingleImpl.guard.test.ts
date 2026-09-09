/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6, règle « ≤ 2 copies ») : le lot C1 du plan vague C
 * (`.ai/PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md`, décision D1) a désigné
 * `components/charts/Heatmap2DChart.tsx` comme l'UNIQUE wrapper canonique de grille
 * ECharts. La migration des implémentations existantes est le lot C2 — REPORTÉE par
 * le superviseur (décision S6, hors périmètre de la vague C) : ce test n'en fait
 * QUE le garde-rail, pas la migration.
 *
 * Sans lui, une 6e implémentation du couple `type` / `heatmap` ECharts pourrait
 * apparaître sans que personne ne le remarque (leçon du dépôt : une factorisation
 * sans garde-rail re-diverge — cf. hex-alpha.guard.test.ts, prédicat bot 8 → 36
 * copies). Ce test échoue si une série ECharts déclare ce type hors de
 * Heatmap2DChart.tsx ET hors de l'allowlist ci-dessous.
 *
 * Note grep : ce fichier évite délibérément de citer le littéral avec apostrophes
 * simples (`type:` + apostrophe + `heatmap` + apostrophe) dans ses commentaires —
 * le grep de sanity-check du gate de vague le prendrait pour un 6e site et
 * polluerait sa propre vérification.
 *
 * ALLOWLIST datée 2026-09-09, mise à jour le 2026-09-09 (lot 1.5) après la
 * migration de `SynthesisHeatmapChart.tsx` (lot C2 — routée par le wrapper
 * canonique en mode `divergent`, rampe autour de 50 % préservée via
 * `valueRange={[0, 1]}`) — 4 sites restants relevés par grep (recherche du
 * motif `type:` suivi de `heatmap` entre apostrophes, hors Heatmap2DChart.tsx) :
 *   - features/ascension/ActivityCalendarChart.tsx      : décision S6 du
 *     superviseur — migration à porter un par un, hors périmètre de la vague C.
 *   - features/explorer/ExplorerActivityHeatmapChart.tsx : idem (décision S6).
 *   - features/palmares/RelationsMomentsHeatmap.tsx      : idem (décision S6).
 *   - features/squad/charts/squadMapHeatmapChart.ts      : idem (décision S6).
 *
 * Retirer une entrée de cette liste EXACTEMENT quand son fichier est migré vers
 * Heatmap2DChart (plus de littéral `type: 'heatmap'` dedans) — ne jamais agrandir
 * la liste sans une justification datée du même type que celles ci-dessus.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve, sep } from 'node:path'

const HEATMAP_TYPE_LITERAL = /type:\s*['"]heatmap['"]/

/** Fichier canonique — seul autorisé à porter le littéral sans figurer dans l'allowlist. */
const CANONICAL_FILE = 'components/charts/Heatmap2DChart.tsx'

/** Allowlist datée 2026-09-09 — voir le en-tête du fichier pour la justification de chaque entrée. */
const ALLOWLIST_2026_09_09 = [
  'features/ascension/ActivityCalendarChart.tsx',
  'features/explorer/ExplorerActivityHeatmapChart.tsx',
  'features/palmares/RelationsMomentsHeatmap.tsx',
  'features/squad/charts/squadMapHeatmapChart.ts',
]

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !entry.name.endsWith('.test.ts') && !entry.name.endsWith('.test.tsx')) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail heatmapSingleImpl (source unique components/charts/Heatmap2DChart.tsx)', () => {
  const srcRoot = resolve(process.cwd(), 'src')
  const files = walk(srcRoot)

  it("aucune série heatmap ECharts hors du wrapper canonique et de l'allowlist datée", () => {
    const porteurs = files
      .filter((f) => HEATMAP_TYPE_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.split(sep).join('/').replace(`${srcRoot.split(sep).join('/')}/`, ''))
      .filter((relPath) => relPath !== CANONICAL_FILE)

    const horsAllowlist = porteurs.filter((relPath) => !ALLOWLIST_2026_09_09.includes(relPath))

    expect(
      horsAllowlist,
      `Nouvelle implémentation heatmap hors du wrapper canonique : ${horsAllowlist.join(', ')}. ` +
        `Passer par components/charts/Heatmap2DChart.tsx (décision D1) ou, si le report est ` +
        `délibéré, ajouter une entrée DATÉE et justifiée à ALLOWLIST_2026_09_09.`,
    ).toEqual([])
  })

  it("l'allowlist ne contient que des fichiers qui portent RÉELLEMENT le littéral (pas de dette fantôme)", () => {
    const manquants = ALLOWLIST_2026_09_09.filter((relPath) => {
      const full = join(srcRoot, ...relPath.split('/'))
      try {
        return !HEATMAP_TYPE_LITERAL.test(readFileSync(full, 'utf8'))
      } catch {
        return true // fichier absent : entrée fantôme, à retirer.
      }
    })
    expect(
      manquants,
      `Entrées de l'allowlist qui ne portent plus (ou n'ont jamais porté) le littéral — à retirer : ${manquants.join(', ')}`,
    ).toEqual([])
  })
})
