/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : le délai des garde-rails de `lib/i18n/` qui lisent tout `src/` vient de
 * `TREE_SCAN_TIMEOUT_MS` (`localeScan.testutil.ts`), jamais d'un nombre écrit dans le test. Un délai
 * recopié en chiffres diverge du suivant ; un délai absent laisse le défaut de vitest (5 s), dépassé
 * sous la charge d'un lot de la CI.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const I18N_DIR = resolve(process.cwd(), 'src/lib/i18n')

/** Un délai écrit en chiffres en dernier argument d'un `it(...)` : `}, 30_000)` ou une ligne `120_000,`. */
const NUMERIC_TIMEOUT = /\},\s*\d[\d_]{3,}\s*\)|^\s*\d[\d_]{3,},?\s*$/m

function numericTimeoutFiles(): string[] {
  return readdirSync(I18N_DIR)
    .filter((name) => name.endsWith('.guard.test.ts') && name !== 'guardTimeout.guard.test.ts')
    .filter((name) => NUMERIC_TIMEOUT.test(readFileSync(join(I18N_DIR, name), 'utf8')))
}

describe('garde-rail : délai des garde-rails de langue', () => {
  it('aucun délai écrit en chiffres : TREE_SCAN_TIMEOUT_MS', () => {
    expect(numericTimeoutFiles()).toEqual([])
  })

  it('auto-preuve : la regex voit les deux formes et ignore la constante', () => {
    expect(NUMERIC_TIMEOUT.test('  }, 30_000)')).toBe(true)
    expect(NUMERIC_TIMEOUT.test('    120_000,\n  )')).toBe(true)
    expect(NUMERIC_TIMEOUT.test('  }, TREE_SCAN_TIMEOUT_MS)')).toBe(false)
    expect(NUMERIC_TIMEOUT.test('expect(files.length).toBeGreaterThan(30)')).toBe(false)
  })
})
