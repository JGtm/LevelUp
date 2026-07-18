/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le mapping code outcome numérique → valeur
 * 'win'/'loss'/'tie'/'dnf' ne doit exister QUE dans `lib/outcome.ts`
 * (`outcomeCodeToValue` / `outcomeCodeToTapeValue`). Ce test échoue si un
 * mapping local par comparaison (`code === N` / `case N: return 'win'`)
 * réapparaît ailleurs — anti-divergence après la centralisation F1 (revue
 * 2026-07-17).
 *
 * Contexte : le mapping était dupliqué en 5 exemplaires à DÉFAUTS DIVERGENTS
 * ('tie' / null / 'dnf') — ExplorerBriefing.logic, session-detail/_shared
 * (outcomeIntToKey), TimeseriesPage.summary, SquadSynergiesPage,
 * MediaMatchPicker. Toute nouvelle surface consomme `lib/outcome.ts`.
 *
 * NB : la variante 'draw' des échelles d'accessibilité (`outcomeKey` de
 * `outcome-color.ts`, littéraux objet `{ 2: 'win', ... }`) est un mapping
 * DISTINCT (clés draw/dnf) — non matché ici car sans comparaison numérique.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Signature d'un mapping code→outcome recopié à la main : une comparaison
// numérique `case N` / `=== N` (N ∈ 1..4) suivie de près d'un retour de valeur
// d'outcome. Un simple registre de tokens (union de type, littéral objet, ordre
// d'affichage) n'a PAS de comparaison numérique → non matché.
const OUTCOME_CODE_LADDER =
  /(?:case\s+[1-4]\b|===\s*[1-4]\b)[\s\S]{0,40}?['"](?:win|loss|tie|dnf)['"]/

// Seul lib/outcome.ts déclare légitimement le mapping code→valeur d'outcome.
const ALLOWED = new Set(['outcome.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail outcome (lib/outcome.ts source unique)', () => {
  it('aucun mapping code→outcome recopié hors lib/outcome.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      const content = readFileSync(file, 'utf8')
      if (OUTCOME_CODE_LADDER.test(content)) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Mapping code→outcome à migrer vers @/lib/outcome (outcomeCodeToValue / outcomeCodeToTapeValue) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
