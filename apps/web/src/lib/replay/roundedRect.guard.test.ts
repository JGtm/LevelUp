/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : UN SEUL TRACÉ DE RECTANGLE ARRONDI (règle <= 2 copies, CLAUDE.md n° 6).
 *
 * Le rectangle arrondi était écrit trois fois (légende et pastille de la probabilité de réapparition,
 * surimpression de l'export), chacun avec son repli. `roundedRectPath` (`lib/replay/roundedRect.ts`)
 * est le seul à appeler `arcTo` ou `roundRect` : un appel ailleurs sous `src/` rouvre la divergence.
 * Plafond 0 hors du helper ; les tests ne comptent pas.
 */
import { describe, expect, it } from 'vitest'
import { callsNamed, formatHits, parseSource, scanSources, SCAN_TIMEOUT_MS, type ScannedSource } from '@/lib/archScan.testutil'

const NOMS = new Set(['arcTo', 'roundRect'])
const HELPER = 'lib/replay/roundedRect.ts'

const horsHelper = (source: ScannedSource) => (source.rel === HELPER ? [] : callsNamed(source, NOMS))

describe('garde-rail : le rectangle arrondi se trace par roundedRectPath', () => {
  it('aucun appel à arcTo ni à roundRect hors du helper (plafond 0)', () => {
    const hits = scanSources().flatMap(horsHelper)
    expect(hits, formatHits(hits, 'Tracer par roundedRectPath (@/lib/replay/roundedRect).')).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('le helper existe et appelle bien les deux (sans quoi ce garde ne garderait rien)', () => {
    const helper = scanSources({ under: 'lib/replay' }).find((s) => s.rel === HELPER)
    expect(helper).toBeDefined()
    expect(callsNamed(helper!, NOMS).map((h) => h.detail).sort()).toContain('arcTo(...)')
    expect(callsNamed(helper!, NOMS).map((h) => h.detail)).toContain('roundRect(...)')
  })

  it('auto-preuve : le détecteur voit un appel de méthode, pas une chaîne ni un autre nom', () => {
    const n = (code: string): number => horsHelper(parseSource('features/x/a.ts', code)).length
    expect(n('ctx.arcTo(1, 2, 3, 4, 5)')).toBe(1)
    expect(n('if (typeof ctx.roundRect === "function") ctx.roundRect(1, 2, 3, 4, 5)')).toBe(1)
    expect(n("const icon = 'roundRect'")).toBe(0)
    expect(n('ctx.rect(1, 2, 3, 4)')).toBe(0)
  })
})
