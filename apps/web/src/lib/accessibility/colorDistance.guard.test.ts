/**
 * colorDistance.guard.test.ts — ratchet anti-copie de la conversion OKLab.
 *
 * La conversion sRGB → OKLab (et donc ΔE) vit dans `colorDistance.ts` seulement
 * (centralisée le 2026-09-17 après deux copies dans les garde-fous de couleurs). Ce test
 * échoue dès qu'un autre fichier de `src/` réécrit la matrice : importer
 * `deltaE` / `simulateCvd` / `oklab` depuis `lib/accessibility/colorDistance`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

const SRC = join(process.cwd(), 'src')
// Premier coefficient de la matrice LMS d'Ottosson : signature d'une copie.
const OKLAB_SIGNATURE = /0\.4122214708/
const OWNER = 'lib/accessibility/colorDistance.ts'
const SELF = 'lib/accessibility/colorDistance.guard.test.ts'

function walk(dir: string, out: string[]): string[] {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) walk(p, out)
    else if (/\.(ts|tsx)$/.test(name)) out.push(p)
  }
  return out
}

describe('colorDistance — source unique de la conversion OKLab', () => {
  it("aucune copie de la matrice OKLab hors de colorDistance.ts", () => {
    const offenders = walk(SRC, [])
      .map((p) => relative(SRC, p).replace(/\\/g, '/'))
      .filter((rel) => rel !== OWNER && rel !== SELF)
      .filter((rel) => OKLAB_SIGNATURE.test(readFileSync(join(SRC, rel), 'utf8')))
    expect(offenders, 'importer deltaE/simulateCvd depuis lib/accessibility/colorDistance').toEqual([])
  })
})
