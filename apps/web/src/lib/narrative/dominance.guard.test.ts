/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : les jetons de couleur de la dominance
 * (`dominance_flag` → SemanticToken) ne doivent exister QUE dans
 * `lib/narrative/dominance.ts` (`DOMINANCE_COLOR_TOKENS`). Ce test échoue si
 * une table locale `DOMINANCE_TOKENS` (l'ancien nom, copié dans
 * `MatchHeader.card.tsx` avant migration — D-10 V721-14a) réapparaît ailleurs
 * dans le code source. Anti-divergence après centralisation : une
 * factorisation sans garde-rail re-diverge (ExplorerMatchesTable avait déjà
 * migré ; MatchHeader.card.tsx gardait sa propre copie jusqu'à ce correctif).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const LOCAL_DOMINANCE_TOKENS = /\b(?:const|let)\s+DOMINANCE_TOKENS\s*[:=]/

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

describe('garde-rail DOMINANCE_TOKENS (lib/narrative/dominance source unique)', () => {
  it('aucune redéfinition locale de DOMINANCE_TOKENS hors du module canonique', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      if (LOCAL_DOMINANCE_TOKENS.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `DOMINANCE_COLOR_TOKENS à importer depuis lib/narrative/dominance : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
