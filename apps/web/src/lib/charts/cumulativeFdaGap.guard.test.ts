/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le cumul de l'écart FDA réel vs attendu (somme
 * cumulée signée avec report D5) vit UNIQUEMENT dans
 * `lib/charts/cumulativeFdaGap.ts`. L'identifiant distinctif `carryForward`
 * (accumulateur reporté quand un match n'a pas d'attendu) ne doit réapparaître
 * dans AUCUN autre fichier source — sinon la factorisation des 3 charts
 * « Écart cumulé au FDA attendu » (Sessions, Escouade, Timeseries) aurait
 * re-divergé. Calqué sur `divergentZeroGradient.guard.test.ts`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const CARRY_FORWARD = /\bcarryForward\b/

const ALLOWED = new Set(['cumulativeFdaGap.ts'])

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
