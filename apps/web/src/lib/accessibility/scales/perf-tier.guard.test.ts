/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le mapping score → `perf-tier-N` ne doit exister
 * QUE dans les échelles centralisées de `instances.ts` (`perfScale` 80/65/50/35,
 * `perfSessionScale` 75/60/45/30). Ce test échoue si une fonction locale
 * `perfTierToken` (ou une échelle de seuils perf-tier recopiée à la main)
 * réapparaît ailleurs — anti-divergence après la centralisation V7c.
 *
 * Contexte : le mapping était dupliqué en 4 exemplaires (SessionSummaryCard,
 * PlayerDetailPanel, mapPerfVsHistoryChart, squadSessionTimelineChart) avec DEUX
 * jeux de seuils divergents (leçon H7 : centraliser par variantes nommées, pas
 * écraser). Toute nouvelle surface consomme `perfScale` / `perfSessionScale`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Signature d'une redéfinition locale : une fonction/const nommée perfTierToken.
const LOCAL_PERF_TIER_FN = /\b(?:function\s+perfTierToken|(?:const|let)\s+perfTierToken\s*=)/
// Signature d'une ÉCHELLE perf-tier recopiée à la main : un seuil numérique `>= NN`
// interleavé avec un retour de tier `perf-tier-N`. Une simple LISTE de tokens
// (union de type, map de labels) n'a PAS de comparaison numérique → non-matchée.
const PERF_TIER_LADDER = />=\s*\d+\s*\?[\s\S]{0,40}?['"]perf-tier-\d['"]|['"]perf-tier-\d['"][\s\S]{0,40}?>=\s*\d+/

// Seul instances.ts déclare légitimement les échelles perf-tier. Les registres de
// tokens (types, labels) ne contiennent pas de seuil → non-matchés par le ladder.
const ALLOWED = new Set(['instances.ts'])

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

describe('garde-rail perf-tier (perfScale/perfSessionScale source unique)', () => {
  it('aucune redéfinition locale de perfTierToken ni échelle perf-tier recopiée hors instances.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      const content = readFileSync(file, 'utf8')
      if (LOCAL_PERF_TIER_FN.test(content) || PERF_TIER_LADDER.test(content)) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Échelle perf-tier à migrer vers perfScale/perfSessionScale (@/lib/accessibility/scales) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
