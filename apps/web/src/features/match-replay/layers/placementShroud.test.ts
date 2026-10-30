/**
 * La courbe des éclairs de la bulle : celle de la fiche (`@keyframes replay-zone-bolt`), lue
 * sur l'horloge de la pose. Si l'une change sans l'autre, la carte et la fiche d'un joueur dans
 * l'écran ne battent plus ensemble.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import { SHROUD_BOLT_OFFSETS_S, shroudBoltAlpha } from './placementShroud'

describe('shroudBoltAlpha — le double battement des éclairs', () => {
  it('éteint au début du cycle, au sommet à 8 %, rallumé à 14 %, éteint dès 22 %', () => {
    expect(shroudBoltAlpha(0, 0)).toBe(0)
    expect(shroudBoltAlpha(0.06 * 2600, 0)).toBe(0)
    expect(shroudBoltAlpha(0.08 * 2600, 0)).toBeCloseTo(0.95)
    expect(shroudBoltAlpha(0.11 * 2600, 0)).toBeCloseTo(0.15)
    expect(shroudBoltAlpha(0.14 * 2600, 0)).toBeCloseTo(0.8)
    expect(shroudBoltAlpha(0.22 * 2600, 0)).toBeCloseTo(0)
    expect(shroudBoltAlpha(0.6 * 2600, 0)).toBe(0)
  })

  it('le cycle se répète : 2,6 s plus tard, le même éclair au même point', () => {
    expect(shroudBoltAlpha(208 + 2600 * 3, 0)).toBeCloseTo(shroudBoltAlpha(208, 0))
  })

  it('le décalage avance l’éclair dans son cycle, comme le délai négatif de la fiche', () => {
    const [, second] = SHROUD_BOLT_OFFSETS_S
    expect(shroudBoltAlpha(208 - second * 1000 + 2600, second)).toBeCloseTo(0.95)
  })

  it('la courbe est celle de la fiche : les clés de `replay-zone-bolt` dans globals.css', () => {
    const css = readFileSync(resolve(__dirname, '../../../styles/globals.css'), 'utf8')
    const bloc = /@keyframes replay-zone-bolt \{([^}]*\}[^}]*\}[^}]*\}[^}]*\}[^}]*\})/.exec(css)?.[1] ?? ''
    expect(bloc).toContain('0%, 6%, 100% { opacity: 0; }')
    expect(bloc).toContain('8%  { opacity: 0.95; }')
    expect(bloc).toContain('11% { opacity: 0.15; }')
    expect(bloc).toContain('14% { opacity: 0.8; }')
    expect(bloc).toContain('22% { opacity: 0; }')
    expect(css).toContain('animation: replay-zone-bolt 2.6s linear infinite')
  })
})
