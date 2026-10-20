/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : le choix « thème clair -> icône noire, thème sombre -> icône
 * blanche » n'existe QUE dans `lib/themedIcon.ts`. Il était recopié dans
 * `waypointUrl.ts` et `MatchNemesisCards.tsx` ; le logo du rejeu 2D en aurait fait une
 * troisième copie (2026-08-16). Ce test échoue si un ternaire de suffixe ou un chemin
 * `-black.png` / `-white.png` littéral réapparaît ailleurs dans `src/`.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { themedIconSrc } from './themedIcon'

/** Le ternaire de suffixe, dans les deux sens, et les chemins littéraux. */
const SUFFIX_TERNARY = /['"]black['"]\s*:\s*['"]white['"]|['"]white['"]\s*:\s*['"]black['"]/
const LITERAL_PATH = /-(?:black|white)\.png/

const ALLOWED = new Set(['themedIcon.ts'])

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

describe('themedIconSrc', () => {
  it('noir sur thème clair, blanc sur thème sombre', () => {
    expect(themedIconSrc('replay', 'light')).toBe('/icons/replay-black.png')
    expect(themedIconSrc('replay', 'dark')).toBe('/icons/replay-white.png')
  })
})

describe('garde-rail themedIcon (lib/themedIcon.ts source unique)', () => {
  it('aucun suffixe de thème ni chemin -black/-white.png recopié hors lib/themedIcon.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      const content = readFileSync(file, 'utf8')
      if (SUFFIX_TERNARY.test(content) || LITERAL_PATH.test(content)) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Suffixe d'icône par thème à migrer vers @/lib/themedIcon (themedIconSrc) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
