/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : le pied d'axe des graphes à bande (`grid.bottom`, position de la bande) se CALCULE
 * par `layoutAxisFoot`, jamais en nombre écrit à la main. Écrits à la main, ces nombres
 * dérivaient dès qu'une étiquette changeait de taille et la bande recouvrait les dates.
 *
 * Interdit, dans les cinq fichiers de graphes concernés : une grille dont `bottom` est un
 * littéral numérique, et une constante `FOOT` / `BAND_BOTTOM` numérique.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const SQUAD = resolve(process.cwd(), 'src', 'features', 'squad')
const FILES = [
  'objectif/eveningsChart.ts',
  'objectif/objectifCharts.ts',
  'emprise/empriseCharts.ts',
  'emprise/habitChart.ts',
]

describe('pied d’axe calculé, pas écrit à la main', () => {
  for (const file of FILES) {
    const source = readFileSync(resolve(SQUAD, file), 'utf8')

    it(`${file} : aucune grille à ${'`bottom`'} littéral`, () => {
      expect(source).not.toMatch(/\bbottom:\s*-?\d/)
    })

    it(`${file} : aucune constante FOOT ou BAND_BOTTOM numérique`, () => {
      expect(source).not.toMatch(/\b(?:[A-Z_]*FOOT|BAND_BOTTOM)\s*=\s*\d/)
    })

    it(`${file} : passe par layoutAxisFoot`, () => {
      expect(source).toContain('layoutAxisFoot(')
    })
  }
})
