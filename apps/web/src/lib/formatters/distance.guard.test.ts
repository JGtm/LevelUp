/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — LA CONVERSION UNITÉS DU JEU → MÈTRES A UN SEUL FOYER.
 *
 * Le rejeu et l'API portent des positions et des distances en unités du jeu (1 unité = 10 pieds
 * = 3,048 m) ; une valeur affichée « en m » passe par `lib/formatters/distance.ts`. Le facteur
 * n'est écrit nulle part ailleurs : une copie « * 3.048 » (ou « 3 », arrondi à l'œil) diverge.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { scanSources } from '../archScan.testutil'

const FACTEUR = /(?<![\d.])0?\.3048(?!\d)|(?<![\d.])3\.048(?!\d)/

describe('garde-rail : un seul foyer du facteur unités → mètres', () => {
  it('le facteur n’est écrit que dans lib/formatters/distance.ts', () => {
    const fautifs = scanSources({ includeTests: false })
      .filter((s) => s.rel !== 'lib/formatters/distance.ts' && FACTEUR.test(s.text))
      .map((s) => s.rel)
    expect(fautifs).toEqual([])
  })

  it('et distance.ts le porte bien — sans quoi ce test ne garderait rien', () => {
    expect(FACTEUR.test(readFileSync(join(__dirname, 'distance.ts'), 'utf8'))).toBe(true)
  })
})
