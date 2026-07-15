/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6, règle « ≤ 2 copies ») : le helper `hexToRgba(hex, alpha)`
 * (alpha-mix structurel sur un hex résolu via token, contexte canvas/ECharts) a une
 * SOURCE UNIQUE : `components/charts/_utils.ts`. Ce test échoue si une déclaration
 * locale `function hexToRgba(` / `const hexToRgba` réapparaît sous `src/features/**`.
 *
 * Contexte : 2 copies identiques (MatchTugOfWarChart, MatchImpactBadgesBar) avaient
 * divergé ; l'histogramme momentum en ajoutait un 3e usage → centralisation + ce
 * garde-rail. Sans lui, le pattern re-diverge (leçon récurrente du repo).
 *
 * NB : la variante `color-mix(...)` de `components/ui/match-card-presentation.ts` est
 * un AUTRE pattern (CSS var en contexte DOM, pas un hex résolu) — hors périmètre, et
 * hors du champ de scan (features/** uniquement).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Déclaration locale interdite : `function hexToRgba(` ou `const hexToRgba`.
const LOCAL_DECL = /(?:function\s+hexToRgba\s*\(|const\s+hexToRgba\s*[=:])/

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail hexToRgba (source unique components/charts/_utils.ts)', () => {
  const featuresRoot = resolve(process.cwd(), 'src', 'features')
  const files = walk(featuresRoot)

  it('aucune déclaration locale hexToRgba sous src/features/**', () => {
    const offenders = files
      .filter((f) => LOCAL_DECL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(featuresRoot, 'src/features'))
    expect(
      offenders,
      `hexToRgba à importer depuis components/charts/_utils.ts : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
