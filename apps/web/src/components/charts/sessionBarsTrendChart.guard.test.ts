/**
 * Garde-rail : UNE SEULE implémentation de la frise « une soirée, un bâton ».
 *
 * La grammaire a été hissée dans `components/charts/sessionBarsTrendChart.ts` le
 * 2026-09-22 (lot Q) parce que les Séries temporelles en montent deux instances de plus.
 * CLAUDE.md n°6 : une factorisation sans garde-rail re-diverge. Ce test échoue si un
 * appelant reconstruit la frise chez lui — c'est-à-dire s'il écrit les clés d'axe d'un
 * graphe de soirées au lieu d'appeler le constructeur partagé.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const SRC = join(process.cwd(), 'src')

/** Les appelants connus de la frise, et ce qu'ils ont le droit d'écrire. */
const APPELANTS = [
  'features/squad/charts/squadRiposteSessionsChart.ts',
  'features/timeseries/TimeseriesCoordinationSection.tsx',
]

describe('frise des soirées — une seule grammaire', () => {
  it.each(APPELANTS)('%s délègue au constructeur partagé', (rel) => {
    const source = readFileSync(join(SRC, rel), 'utf8')
    expect(source).toMatch(/sessionBarsTrendChart|SessionBarsTrendCard/)
  })

  it.each(APPELANTS)('%s n’écrit AUCUNE clé d’option d’axe de son côté', (rel) => {
    const source = readFileSync(join(SRC, rel), 'utf8')
    const fautes = ['xAxis:', 'yAxis:', 'markLine:', 'barMaxWidth'].filter((k) =>
      source.includes(k),
    )
    expect(fautes, 'appeler buildSessionBarsTrendOption plutôt que recopier la frise').toEqual([])
  })
})
