/// <reference types="node" />
// @vitest-environment node
/**
 * encountersFormat.guard.test.ts — ratchet anti-recopie des formats de rencontre.
 *
 * Le 2026-09-17 (revue adversariale des lots feat/v75, item 6), `formatKDCross`,
 * `formatKDRatio`, `formatRelativeFR` et `formatRelativeEN` vivaient EN DOUBLE, au caractère
 * près, dans `features/match-view/MatchEncountersTable.tsx` et
 * `features/explorer/ExplorerEncounterBriefing.tsx` — l'en-tête du second l'écrivait
 * lui-même (« copiés à l'identique »). Règle CLAUDE.md n° 6 : le helper canonique
 * (`features/_shared/encounters/format.ts`) ET le garde-rail qui interdit l'ancien littéral,
 * sans quoi la factorisation re-diverge.
 *
 * INTERDIT hors de ce dossier : la DÉCLARATION de l'une de ces quatre fonctions. Les
 * IMPORTER et les appeler reste évidemment libre — c'est le but.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

const SRC = join(process.cwd(), 'src')
/** Le seul dossier qui a le droit de DÉCLARER ces formats (chemin relatif à src/). */
const MODULE_DIR = 'features/_shared/encounters'

const FORBIDDEN: Array<{ label: string; re: RegExp }> = [
  { label: 'function formatKDCross(', re: /function\s+formatKDCross\s*\(/ },
  { label: 'function formatKDRatio(', re: /function\s+formatKDRatio\s*\(/ },
  { label: 'function formatRelativeFR(', re: /function\s+formatRelativeFR\s*\(/ },
  { label: 'function formatRelativeEN(', re: /function\s+formatRelativeEN\s*\(/ },
]

function walk(dir: string, out: string[]): string[] {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) walk(p, out)
    else if (/\.(ts|tsx)$/.test(name)) out.push(p)
  }
  return out
}

/** Déclarations trouvées, dans le module (`scanModule`) ou hors de lui : `fichier|motif`. */
function findDeclarations(scanModule: boolean): string[] {
  const found = new Set<string>()
  for (const file of walk(SRC, [])) {
    const rel = relative(SRC, file).replace(/\\/g, '/')
    const inModule = rel.startsWith(`${MODULE_DIR}/`)
    if (inModule !== scanModule) continue
    const body = readFileSync(file, 'utf8')
    for (const { label, re } of FORBIDDEN) {
      if (re.test(body)) found.add(`${rel}|${label}`)
    }
  }
  return [...found].sort()
}

describe('formats de rencontre — une seule déclaration (features/_shared/encounters)', () => {
  it('aucune de ces quatre fonctions n’est redéclarée ailleurs dans src/', () => {
    expect(
      findDeclarations(false),
      'importer depuis @/features/_shared/encounters/format au lieu de redéclarer',
    ).toEqual([])
  })

  it('témoin positif : le balayage voit bien les quatre déclarations DANS le module', () => {
    // Sans ce témoin, un renommage ou une regex cassée rendrait le test vert pour de
    // mauvaises raisons — le garde-rail ne garderait plus rien.
    const inModule = findDeclarations(true)
    for (const { label } of FORBIDDEN) {
      expect(inModule).toContain(`${MODULE_DIR}/format.ts|${label}`)
    }
  })
})
