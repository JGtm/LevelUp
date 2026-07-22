/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6, règle « ≤ 2 copies » → helper + garde-rail) : le « Détails des
 * frags » (armes gun + détail mêlée/grenade/capacités tiré de la FragDistribution) a une
 * SOURCE UNIQUE = buildFragDetailBreakdown (components/charts/fragDetailBreakdown.ts).
 *
 * Match view + Synthesis + Sessions (3 surfaces) passent TOUS par ce helper. Le set des
 * classes-armes `['shoulder','sidearm','heavy']` (GUN_CLASSES) ne doit donc être réinliné
 * dans AUCUNE feature/aucun composant — sinon la logique re-diverge (double-comptage mêlée).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Signature du set GUN_CLASSES (les 3 classes-armes), tolérante aux espaces.
const NEEDLE = /\[\s*['"]shoulder['"]\s*,\s*['"]sidearm['"]\s*,\s*['"]heavy['"]\s*\]/

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\./.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail buildFragDetailBreakdown (source unique du « Détails des frags »)', () => {
  it('le set GUN_CLASSES n\'est réinliné dans aucune feature/aucun composant (helper seul)', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const allowed = join(srcRoot, 'components', 'charts', 'fragDetailBreakdown.ts')
    const roots = [join(srcRoot, 'features'), join(srcRoot, 'components')]
    const offenders: string[] = []
    for (const root of roots) {
      for (const file of walk(root)) {
        if (file === allowed) continue
        if (NEEDLE.test(readFileSync(file, 'utf8'))) offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `« Détails des frags » à construire via buildFragDetailBreakdown (components/charts/fragDetailBreakdown), pas de GUN_CLASSES réinliné : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
