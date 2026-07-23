/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le cumul signé avec report D5 (accumulateur
 * `carryForward`) vit UNIQUEMENT dans le helper générique
 * `lib/charts/cumulativeSeries.ts` — `cumulativeFdaGap.ts` y DÉLÈGUE (et le
 * documente, d'où son maintien dans l'allowlist). L'identifiant distinctif
 * `carryForward` ne doit réapparaître dans AUCUN autre fichier source — sinon la
 * factorisation des charts cumulés (« Écart cumulé au FDA attendu » Sessions /
 * Escouade / Timeseries, « Balance des dégâts » Session / Escouade) aurait
 * re-divergé. Calqué sur `divergentZeroGradient.guard.test.ts`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const CARRY_FORWARD = /\bcarryForward\b/

const ALLOWED = new Set(['cumulativeSeries.ts', 'cumulativeFdaGap.ts'])

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

describe('garde-rail cumulativeFdaGap (source unique du cumul écart FDA réel/attendu)', () => {
  it('aucune réinjection de `carryForward` hors du helper canonique', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      if (CARRY_FORWARD.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `cumul FDA à importer depuis lib/charts/cumulativeFdaGap : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
