/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail I19 (règle CLAUDE.md « ≤ 2 copies d'un même pattern ») : la
 * construction de l'URL Halo Waypoint d'un match DOIT passer par
 * `buildWaypointMatchUrl` (waypointUrl.ts). Interdit toute reconstruction ad
 * hoc du littéral "halowaypoint.com" ailleurs dans apps/web/src — la 3e copie du
 * pattern (ExplorerMatchesTable, SquadSynergyHistoryTable, et un tableau carrière
 * depuis retiré) a motivé la centralisation ; ce garde-rail empêche la dette de
 * re-diverger.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const SRC_ROOT = resolve(__dirname, '..', '..')
const ALLOWED_FILE = 'waypointUrl.ts'

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(full, out)
    } else if (
      /\.(ts|tsx)$/.test(entry.name) &&
      entry.name !== ALLOWED_FILE &&
      // Les fichiers de test peuvent asserter l'URL complète en littéral (ça
      // fige le contrat du helper) — le garde-rail ne vise que le code produit.
      !/\.test\.(ts|tsx)$/.test(entry.name)
    ) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail waypointUrl (I19)', () => {
  it('aucune reconstruction ad hoc du domaine halowaypoint.com hors waypointUrl.ts', () => {
    const offenders: string[] = []
    for (const file of walk(SRC_ROOT)) {
      const content = readFileSync(file, 'utf8')
      if (content.includes('halowaypoint.com')) {
        offenders.push(file)
      }
    }
    expect(offenders).toEqual([])
  })
})
