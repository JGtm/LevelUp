/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : les libellés de jours `['Lun', 'Mar', ...]`
 * ne doivent exister QUE dans `calendar.ts` (source unique). Ce test échoue si
 * le littéral réapparaît ailleurs — anti-divergence après centralisation I2.
 *
 * Contexte : le tableau était dupliqué en FR-only dans 4 fichiers (heatmaps
 * explorer/synthesis, timeseries adapters, lab showcase), cassant le
 * bilinguisme (règle n°1). Toute nouvelle heatmap/axe jour consomme
 * `dowLabels(locale)`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Signature du littéral prohibé (début du tableau — tolère guillemets simples
// ou doubles et espaces variables). Le fichier source légitime est exclu.
const DOW_LITERAL = /\[\s*['"]Lun['"]\s*,\s*['"]Mar['"]\s*,\s*['"]Mer['"]/
const ALLOWED = new Set(['calendar.ts'])

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

describe('garde-rail DOW labels (calendar.ts source unique)', () => {
  it('aucun littéral de jours FR hors de calendar.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      if (DOW_LITERAL.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(offenders, `Littéral DOW à migrer vers dowLabels(locale) : ${offenders.join(', ')}`).toEqual([])
  })
})
