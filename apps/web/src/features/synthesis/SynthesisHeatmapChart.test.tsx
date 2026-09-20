/**
 * SynthesisHeatmapChart — migration vers le wrapper canonique (lot C2, plan
 * vague C formes). Ce que ces tests verrouillent, spécifiquement à risque
 * dans une migration de ce type :
 *
 *  - la rampe DIVERGENTE reste centrée sur 50 % (`valueRange={[0, 1]}`), quelle
 *    que soit la plage réelle des taux de victoire mesurés — sinon le wrapper
 *    auto-ajusterait min/max et décentrerait le neutre (régression silencieuse) ;
 *  - Lundi reste en haut de l'axe Y, Dimanche en bas (l'ancienne implémentation
 *    utilisait `yAxis.inverse: true`, absent du wrapper canonique — l'ordre
 *    d'émission des points compense) ;
 *  - le tooltip conserve son contenu (jour, heure, taux, nombre de matchs) ;
 *  - une case sans match reste identifiable (count 0 → value null).
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { vi } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HeatmapCell } from '@/lib/api/types'

import { SynthesisHeatmapChart } from './SynthesisHeatmapChart'

// jsdom n'a pas de canvas : on mocke echarts-for-react (comme
// SquadIsolementNuageCard.test.tsx) pour capturer l'option ECharts construite.
const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="synthesis-heatmap-stub" />
  },
}))

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  captured.length = 0
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

function cell(over: Partial<HeatmapCell>): HeatmapCell {
  return { dow: 0, hour: 0, count: 0, ...over }
}

type RawTuple = [number, number, number | string, Record<string, unknown> | undefined]
type CellDatum = RawTuple | { value: RawTuple; itemStyle: unknown }

/** Lit `[x, y, value, detail]` d'une case, tuple brut OU objet `{ value, itemStyle }`
 *  (cf. `HeatCellDatum` de Heatmap2DChart — les deux formes coexistent dans `data`). */
function tupleOf(d: CellDatum): RawTuple {
  return Array.isArray(d) ? d : d.value
}

interface CapturedOption {
  visualMap: {
    min: number
    max: number
    inRange: { color: string[] }
    orient?: string
    show?: boolean
  }
  xAxis: { data: string[]; name?: string }
  yAxis: { data: string[]; name?: string; inverse?: boolean }
  tooltip: { formatter: (p: unknown) => string }
  series: Array<{ data: CellDatum[] }>
}

describe('SynthesisHeatmapChart — migration Heatmap2DChart (lot C2)', () => {
  it('rampe DIVERGENTE figée sur [0, 1] : le neutre reste à 50 % même si aucun match ne dépasse 60 %', async () => {
    renderWithProviders(
      <SynthesisHeatmapChart
        cells={[cell({ dow: 0, hour: 9, count: 10, win_rate: 0.6, wins: 6 })]}
      />,
    )
    await screen.findByTestId('synthesis-heatmap-stub')
    const option = captured[captured.length - 1].option as CapturedOption
    expect(option.visualMap.min).toBe(0)
    expect(option.visualMap.max).toBe(1)
    // Rampe divergente : au moins 3 teintes (perdant / neutre / gagnant).
    expect(option.visualMap.inRange.color.length).toBeGreaterThanOrEqual(3)
  })

  // FORME RESTAURÉE le 2026-09-13 : les jours sont émis DANS LE SENS DE LA SEMAINE et
  // c'est `yAxis.inverse` qui met Lundi en haut — l'ordre des données ne porte plus une
  // décision d'affichage (cf. doc de tête du composant).
  it('Lundi en haut, Dimanche en bas, par axe INVERSÉ', async () => {
    renderWithProviders(<SynthesisHeatmapChart cells={[cell({ dow: 0, hour: 0, count: 1, win_rate: 1 })]} />)
    await screen.findByTestId('synthesis-heatmap-stub')
    const option = captured[captured.length - 1].option as CapturedOption
    expect(option.yAxis.data[0]).toBe('Lun')
    expect(option.yAxis.data[option.yAxis.data.length - 1]).toBe('Dim')
    expect(option.yAxis.inverse).toBe(true)
  })

  // Les deux axes portent leur titre. La BARRE DE DÉGRADÉ, elle, est partie le 2026-09-20
  // (retour utilisateur, lot 3 des ajustements pré-v7.5) : elle redisait ce que chaque case
  // montre, et l'infobulle donne le taux exact. L'orientation verticale RESTE déclarée —
  // c'est elle qui décide des marges du tracé, celles qui logent les titres d'axes.
  it('les deux axes sont titrés, et aucune barre de dégradé n’est rendue', async () => {
    renderWithProviders(<SynthesisHeatmapChart cells={[cell({ dow: 0, hour: 0, count: 1, win_rate: 1 })]} />)
    await screen.findByTestId('synthesis-heatmap-stub')
    const option = captured[captured.length - 1].option as CapturedOption
    expect(option.xAxis.name).toBe('Heure')
    expect(option.yAxis.name).toBe('Jour')
    expect(option.visualMap.orient).toBe('vertical')
    expect(option.visualMap.show).toBe(false)
  })

  it('l’axe des heures couvre 00h à 23h, dans l’ordre', async () => {
    renderWithProviders(<SynthesisHeatmapChart cells={[cell({ dow: 0, hour: 0, count: 1, win_rate: 1 })]} />)
    await screen.findByTestId('synthesis-heatmap-stub')
    const option = captured[captured.length - 1].option as CapturedOption
    expect(option.xAxis.data[0]).toBe('00h')
    expect(option.xAxis.data[23]).toBe('23h')
  })

  it('le tooltip nomme le jour, l’heure, le taux de victoire et le nombre de matchs', async () => {
    renderWithProviders(
      <SynthesisHeatmapChart cells={[cell({ dow: 2, hour: 14, count: 5, win_rate: 0.4, wins: 2 })]} />,
    )
    await screen.findByTestId('synthesis-heatmap-stub')
    const option = captured[captured.length - 1].option as CapturedOption
    const datum = option.series[0].data.find((d) => tupleOf(d)[2] === 0.4)
    const html = option.tooltip.formatter({ data: datum })
    expect(html).toContain('Mer')
    expect(html).toContain('14h')
    expect(html).toContain('40.0%')
    expect(html).toContain('Taux de victoire')
    expect(html).toContain('Matchs')
    expect(html).toContain('5')
  })

  it('une case sans match (count 0) est une case VIDE (value null), pas un taux à 0 %', async () => {
    renderWithProviders(<SynthesisHeatmapChart cells={[cell({ dow: 0, hour: 0, count: 3, win_rate: 0.5 })]} />)
    await screen.findByTestId('synthesis-heatmap-stub')
    const option = captured[captured.length - 1].option as CapturedOption
    // 168 cases émises (24h x 7j), une seule mesurée : les 167 autres sont
    // des tuples `[x, y, '-']` ou des objets `{ value: [..., '-'], itemStyle }`.
    const vides = option.series[0].data.map((d) => tupleOf(d)[2]).filter((v) => v === '-')
    expect(vides.length).toBe(167)
  })

  it('aucune session mesurée : pas de nuage (état vide du wrapper, series vide)', async () => {
    renderWithProviders(<SynthesisHeatmapChart cells={[]} />)
    // ChartCard doit gérer l'état vide sans lever — le stub echarts n'apparaît
    // jamais puisque `series` est vide (Heatmap2DChart ne monte pas de graphe).
    expect(screen.queryByTestId('synthesis-heatmap-stub')).toBeNull()
  })
})
