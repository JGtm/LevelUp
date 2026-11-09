/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — le motif de substitution `{nom}` a UNE définition (`lib/i18n/template.ts`). Trois copies avaient
 * poussé (notifications, Ascension, Réglages) ; une copie de plus re-diverge (règle des deux copies).
 *
 * Le balayage (`scanLines`) ignore `lib/i18n/` et `lib/formatters/`, qui possèdent la langue : le helper y vit.
 */
import { describe, expect, it } from 'vitest'
import { formatOverflow, scanLines, TREE_SCAN_TIMEOUT_MS } from './localeScan.testutil'

/** Interdit : aucune copie du motif hors de `lib/i18n/template.ts`. */
const PLACEHOLDER_COPY_CEILING = 0

/** Le motif tel qu'on l'écrit dans un `replace` : une expression régulière qui contient `\{(\w+)\}`. */
const PLACEHOLDER_COPY = /\\\{\(\\w\+\)\\\}/

describe('garde-rail gabarits — un seul motif {nom}', () => {
  it('aucune copie du motif hors de lib/i18n/template.ts', () => {
    const hits = scanLines(PLACEHOLDER_COPY)
    expect(
      hits.length,
      formatOverflow(hits, PLACEHOLDER_COPY_CEILING, 'Passer par fillTemplate ou substitute (@/lib/i18n/template).'),
    ).toBeLessThanOrEqual(PLACEHOLDER_COPY_CEILING)
  }, TREE_SCAN_TIMEOUT_MS)

  it('auto-preuve : le motif désigne bien la copie à interdire', () => {
    const copy = String.raw`t.replace(/\{(\w+)\}/g, (_, k) => k)`
    expect(PLACEHOLDER_COPY.test(copy)).toBe(true)
    expect(PLACEHOLDER_COPY.test('t.replace(/[a-z]+/g, "")')).toBe(false)
  })
})
