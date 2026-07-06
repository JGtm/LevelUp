/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°13 + n°6) : TOUTES les query keys vivent dans
 * `lib/query/keys.ts` (`queryKeys`). Ce test échoue si :
 *   1. un registre feature-local `export const xxxKeys = { … }` réapparaît
 *      (les 7 ex-registres prestige/arc/challenge/squad/profile/watcher/admin
 *      ont été centralisés — L5) ;
 *   2. un `queryKey: ['…']` littéral bypasse `queryKeys` (les extensions
 *      `queryKey: [...queryKeys.X, …]` restent autorisées).
 *
 * Sans ce garde-rail, les clés re-divergent (leçon : registres dupliqués par
 * feature → collisions de cache silencieuses).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const LOCAL_REGISTRY = /export\s+const\s+\w+Keys\s*=\s*\{/
// queryKey dont le tableau commence par un littéral chaîne → bypass interdit.
// Autorisé : queryKey: queryKeys.X(...) et queryKey: [...queryKeys.X, ...].
const INLINE_LITERAL = /queryKey:\s*\[\s*['"]/
// keys.ts EST la source unique (queryKeys y est défini) → exempté.
const ALLOWED = new Set(['keys.ts'])

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

describe('garde-rail query keys (queryKeys source unique — L5)', () => {
  const srcRoot = resolve(process.cwd(), 'src')
  const files = walk(srcRoot).filter((f) => !ALLOWED.has(f.split(/[\\/]/).pop() ?? ''))

  it('aucun registre *Keys feature-local hors keys.ts', () => {
    const offenders = files
      .filter((f) => LOCAL_REGISTRY.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(srcRoot, 'src'))
    expect(offenders, `Registre à centraliser dans queryKeys : ${offenders.join(', ')}`).toEqual([])
  })

  it('aucun queryKey littéral inline (utiliser queryKeys.*)', () => {
    const offenders = files
      .filter((f) => INLINE_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(srcRoot, 'src'))
    expect(offenders, `queryKey littéral à router via queryKeys : ${offenders.join(', ')}`).toEqual([])
  })
})
