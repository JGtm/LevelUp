/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le format d'un delta SIGNÉ à décimales fixes (zéro
 * → '±0' / '±0.00', préfixe +/−, glyphe '−' U+2212) ne doit exister QUE dans
 * `lib/formatters/number.ts` (`formatSignedFixed`). Ce test échoue si le sentinel
 * de zéro '±0' / '±0.00' (littéral) ou un template `±${…toFixed…}` réapparaît
 * ailleurs — anti-divergence après la centralisation F3 (revue 2026-07-17).
 *
 * Contexte : 3 copies strictes à glyphes divergents ('−' vs '-') —
 * ExplorerBriefing.logic (formatSignedFixed), rating.ts (formatRankDelta),
 * KpiGrid.tsx (formatRankDeltaValue). Toute nouvelle surface consomme
 * `formatSignedFixed` (@/lib/formatters).
 *
 * NON matchés (formatteurs distincts, volontairement locaux) : `formatSignedPoints`
 * ('±0 pts', points entiers sans toFixed) et le `formatDelta` de delta-card
 * (précision dynamique selon magnitude, aucun sentinel '±0').
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Sentinel de zéro d'un delta signé à décimales fixes : '±0' / '±0.00' (littéral
// quoté, PAS '±0 pts'), OU un template `±${ … toFixed … }`.
const SIGNED_ZERO_SENTINEL = /['"]±0(?:\.0+)?['"]|`±\$\{[^}]*toFixed/

const ALLOWED = new Set(['number.ts'])

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

describe('garde-rail delta signé (formatSignedFixed source unique)', () => {
  it("aucun sentinel '±0'/'±0.00' ou template ±${…toFixed} hors number.ts", () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      const content = readFileSync(file, 'utf8')
      if (SIGNED_ZERO_SENTINEL.test(content)) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Format de delta signé à migrer vers @/lib/formatters (formatSignedFixed) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
