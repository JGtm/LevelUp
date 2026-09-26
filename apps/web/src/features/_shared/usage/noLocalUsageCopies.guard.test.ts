/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (E5.3, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md — CLAUDE.md n°6 « ≤ 2 copies ») :
 * SOURCE UNIQUE des formes et projections du bloc « usages d'équipement, armes spéciales et
 * objectifs ».
 *
 * Le bloc a été DÉMÉNAGÉ de `features/session-detail/` vers ici le 2026-09-09 (étape E5.1)
 * précisément pour que `features/synthesis` et `features/squad` puissent le consommer (E5.8+,
 * E6, lots suivants) sans violer `lint-cross-feature-imports`. Une factorisation sans
 * garde-rail re-diverge (leçon citée par CLAUDE.md n°6, déjà observée sur le prédicat bot :
 * 8 -> 36 copies après une première centralisation sans garde-rail) : ce test échoue si une
 * feature réimplémente sa propre copie d'une des fonctions/formes/types canoniques au lieu
 * d'importer celle-ci.
 *
 * Ce que ça protège : les DÉFINITIONS (`function`/`interface`/`type`), pas les imports ni les
 * mentions en commentaire ou en JSDoc — importer `buildGaugeRow` depuis
 * `@/features/_shared/usage/usageGaugeModel` est le comportement voulu et ne doit jamais faire
 * rougir ce test. Les motifs ci-dessous exigent tous le mot-clé de déclaration juste avant le
 * nom (`function buildGaugeRow(`, `interface UsageGaugeModel`) : une simple mention en prose ne
 * matche pas.
 *
 * Preuve de mordant (mutation, 2026-09-09) : une copie de `usageAvailability` insérée dans
 * `features/squad/` fait rougir ce test (voir le journal du plan, section E5.3) ; retirée, il
 * revient au vert. Non committée — seul le garde-rail l'est.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const FEATURES_ROOT = resolve(process.cwd(), 'src', 'features')
const CANONICAL_DIR = join(FEATURES_ROOT, '_shared', 'usage')

// Les identifiants canoniques du bloc — une DÉFINITION locale de l'un d'eux, hors du dossier
// canonique, est la copie interdite. Un échantillon représentatif de chaque fichier du bloc
// (formes, projections, classification, disponibilité) suffit : il n'est pas nécessaire de
// lister les ~40 exports pour que le garde-rail tienne sa frontière.
const FORBIDDEN_DEFINITIONS: RegExp[] = [
  /\bfunction buildGaugeRow\b/,
  /\bfunction buildOutcomeSegments\b/,
  /\bfunction buildLobbyTrack\b/,
  /\bfunction buildRegularityBand\b/,
  /\bfunction usageAvailability\b/,
  /\bfunction equipmentMetrics\b/,
  /\binterface UsageGaugeModel\b/,
  /\binterface UsageGaugeRowModel\b/,
  /\btype UsageOutcomeKind\b/,
]

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (full === CANONICAL_DIR) continue // le module canonique lui-même, exclu du scan
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail source unique du bloc usage (E5.3)', () => {
  const files = walk(FEATURES_ROOT)

  it('aucune feature hors _shared/usage ne redéfinit une forme/projection du bloc usage', () => {
    const offenders: string[] = []
    for (const f of files) {
      const text = readFileSync(f, 'utf8')
      if (FORBIDDEN_DEFINITIONS.some((re) => re.test(text))) {
        offenders.push(f.replace(FEATURES_ROOT, 'src/features'))
      }
    }
    expect(
      offenders,
      'copie locale du bloc usage détectée — importer depuis @/features/_shared/usage/... ' +
        `plutôt que de redéfinir : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
