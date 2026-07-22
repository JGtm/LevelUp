/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail ratchet (D-10, CLAUDE.md n°6) — le module `lib/title-routing/` est la
 * source UNIQUE de l'interprétation/construction des segments de titre `/t/`.
 *
 * Ce test échoue si un littéral de PARSING du namespace titre `/t/` (chaîne entre
 * quotes/backticks, template `/t/${…}`, ou échappement regex `\/t\/`) apparaît HORS
 * du module `lib/title-routing/`. Toute lecture/construction d'un segment
 * `/t/{slug}` doit passer par ce module (parseRouteSegments / buildLegacyRedirect),
 * jamais par un littéral recopié — sinon la logique de titre re-diverge (leçon
 * CLAUDE.md n°6 : factorisation sans garde-rail).
 *
 * [2026-07-22] PORTÉE PHASE 1 : seul `/t/` est verrouillé. La règle interdisant le
 * littéral `/players/` (routes physiquement déplacées) s'ARMERA en Phase 2e du plan
 * PLAN_TITLE_SLUG_URL, une fois les ~70 fichiers de littéraux migrés — le plan
 * échelonne volontairement (§5 Phase 2e). Ne PAS l'ajouter ici.
 *
 * Allowlist datée [2026-07-22] = fichiers du module `lib/title-routing/`
 * UNIQUEMENT. `routeTree.gen.ts` (généré, jamais édité) est hors périmètre.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Littéral de parsing du namespace `/t/` : `'/t/`, `"/t/`, `` `/t/ ``, `/t/${`, ou
// l'échappement regex `\/t\/`. Un `/t/` NU dans un commentaire ou une prose
// non-quotée (ex. « handleChange/t/frozen ») n'est PAS matché (pas de marqueur).
const TITLE_NS_LITERAL = /(['"`]\/t\/)|(\/t\/\$\{)|(\\\/t\\\/)/

const srcRoot = resolve(process.cwd(), 'src')
// Allowlist datée [2026-07-22] : le module title-routing EST la source unique.
const ALLOWED_DIR = join(srcRoot, 'lib', 'title-routing')

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      // routeTree.gen.ts : généré, jamais édité à la main → hors périmètre.
      if (entry.name === 'routeTree.gen.ts') continue
      out.push(full)
    }
  }
  return out
}

describe('garde-rail title-routing (source unique du parsing /t/)', () => {
  it('aucun littéral /t/ de parsing hors lib/title-routing/', () => {
    const offenders = walk(srcRoot)
      .filter((f) => !f.startsWith(ALLOWED_DIR))
      .filter((f) => TITLE_NS_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(srcRoot, 'src'))
    expect(
      offenders,
      `Littéral /t/ à router via @/lib/title-routing (parseRouteSegments/buildLegacyRedirect) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
