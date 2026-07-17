/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le helper `deltaToken` (token couleur d'un delta
 * signé : outcome-win / outcome-loss / outcome-draw) ne doit exister QUE dans
 * `ExplorerBriefing.logic.ts`. Ce test échoue si une définition locale
 * `function deltaToken(` réapparaît ailleurs — anti-divergence après la
 * centralisation F5 (revue 2026-07-17 ; 2 copies identiques dans
 * ExplorerBriefingModules.tsx et ExplorerBriefingStrip.tsx).
 *
 * Cible une DÉFINITION de helper (`function deltaToken(` ou arrow
 * `const deltaToken = (`). Ne matche PAS une variable locale homonyme
 * (`const deltaToken = skillDeltaScale(...)` de MatchStatCards, sémantique
 * distincte : échelle de delta de skill, pas le token outcome-win/loss/draw).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const LOCAL_DELTA_TOKEN = /function\s+deltaToken\s*\(|(?:const|let)\s+deltaToken\s*=\s*\(/

const ALLOWED = new Set(['ExplorerBriefing.logic.ts'])

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

describe('garde-rail deltaToken (ExplorerBriefing.logic source unique)', () => {
  it('aucune redéfinition locale de deltaToken hors ExplorerBriefing.logic.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      if (LOCAL_DELTA_TOKEN.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `deltaToken à importer depuis ExplorerBriefing.logic : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
