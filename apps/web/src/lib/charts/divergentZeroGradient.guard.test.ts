/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le dégradé divergent à bascule EXACTE sur 0 vit
 * UNIQUEMENT dans `lib/charts/divergentZeroGradient.ts`. L'identifiant distinctif
 * `zeroRatio` (fraction verticale où tombe 0 dans la boîte de l'aire) ne doit
 * réapparaître dans AUCUN autre fichier source — sinon la factorisation des 3
 * charts « aire signée » (SessionNetScoreArea, TimeseriesFdaGapTrend,
 * SessionFdaGapCumulative) aurait re-divergé.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const ZERO_RATIO = /\bzeroRatio\b/

const ALLOWED = new Set(['divergentZeroGradient.ts'])

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

describe('garde-rail divergentZeroGradient (source unique du dégradé à bascule sur 0)', () => {
  it('aucune réinjection de `zeroRatio` hors du helper canonique', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      if (ZERO_RATIO.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `dégradé divergent à importer depuis lib/charts/divergentZeroGradient : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
