/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : la forme « texte + séparateur + réserve d'échantillon
 * faible » ne doit exister QUE dans `lib/formatters/lowSampleNote.ts`
 * (`withLowSampleNote`). Ce test échoue si un template `${…lowSample}` (interpolation
 * directe de la chaîne localisée de réserve) réapparaît ailleurs — anti-divergence après
 * la centralisation de la revue de la vague 1 (2026-09-07 : trois copies, deux
 * séparateurs différents, dans SquadEchangeKpi, TacticalAnalysisView et
 * SquadIsolementNuageCard).
 *
 * Contexte : le drapeau `echantillon_faible` interdit de comparer une valeur, il ne la
 * cache pas ; la réserve s'accole donc au texte, et cette forme a une seule source.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Interpolation directe de la chaîne de réserve dans un template : `${t.lowSample}`,
// `${strings.lowSample}`, `${lowSample}`…
const LOW_SAMPLE_TEMPLATE = /\$\{[^}]*\blowSample\}/

const ALLOWED = new Set(['lowSampleNote.ts'])

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

describe('garde-rail réserve échantillon faible (withLowSampleNote source unique)', () => {
  it('aucun template ${…lowSample} hors lowSampleNote.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      const content = readFileSync(file, 'utf8')
      if (LOW_SAMPLE_TEMPLATE.test(content)) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Réserve d'échantillon faible à accoler via @/lib/formatters/lowSampleNote (withLowSampleNote) : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  it('self-check : le noyau existe bien dans lib/formatters', () => {
    const src = readFileSync(
      resolve(process.cwd(), 'src', 'lib', 'formatters', 'lowSampleNote.ts'),
      'utf8',
    )
    expect(src).toMatch(/export function withLowSampleNote\(/)
  })
})
