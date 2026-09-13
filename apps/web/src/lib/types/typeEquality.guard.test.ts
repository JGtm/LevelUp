/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le motif d'égalité stricte de types a UN foyer,
 * `lib/types/typeEquality.ts`.
 *
 * POURQUOI CE GARDE. La centralisation SANS garde-rail re-diverge — c'est la leçon écrite du
 * dépôt (un prédicat passé de 8 à 36 copies après sa centralisation). Le motif est court, on le
 * réécrit de mémoire en dix secondes, et deux copies qui divergent d'un `extends` ne prouvent
 * plus la même chose : l'une attrape un `any` là où l'autre le laisse passer.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

/** La signature du motif, construite pour que CE fichier ne se dénonce pas lui-même. */
const MOTIF = new RegExp(['\\(<T>\\(\\)', '=>', 'T', 'extends', 'A'].join('\\s*') + '\\s*\\?')

/** Le seul fichier qui a le droit de le porter. */
const FOYER = 'typeEquality.ts'

function racineSrc(): string {
  return resolve(__dirname, '..', '..')
}

function fichiers(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...fichiers(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail : une seule égalité stricte de types', () => {
  it('aucun fichier ne réécrit le motif hors du foyer', () => {
    const fautifs = fichiers(racineSrc())
      .filter((f) => !f.endsWith(FOYER) && !f.endsWith('typeEquality.guard.test.ts'))
      .filter((f) => MOTIF.test(readFileSync(f, 'utf8')))
      .map((f) => f.slice(racineSrc().length + 1).split('\\').join('/'))
    expect(
      fautifs,
      `ces fichiers réécrivent l'égalité stricte de types : [${fautifs.join(', ')}]. ` +
        `Importer Equals / Expect depuis @/lib/types/typeEquality.`,
    ).toEqual([])
  })

  it('et le foyer, lui, le porte bien — sans quoi ce garde ne garderait rien', () => {
    expect(MOTIF.test(readFileSync(join(racineSrc(), 'lib', 'types', FOYER), 'utf8'))).toBe(true)
  })
})
