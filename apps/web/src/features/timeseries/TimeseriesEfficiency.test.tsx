/**
 * TimeseriesEfficiency.test.tsx — « Rendement & Résistance » solo (dégâts BRUTS).
 *
 * Verrouille le cadre de lecture aligné sur les cartes Escouade : fenêtre d'axe
 * FIXE autour du repère « une vie » (jamais dérivée de la session) et surtout
 * l'ORIENTATION des zones. Cette courbe est en dégâts dépensés par frag :
 * dépenser MOINS d'une vie par frag est efficace, donc le vert va SOUS le repère
 * — l'inverse des cartes Escouade (en taux normalisés, où le vert va au-dessus).
 * Ce test échoue si quelqu'un ré-uniformise l'orientation des deux surfaces.
 */
import { describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'

import type { TimeseriesMatchRow } from '@/lib/api/types'
import {
  ONE_LIFE_DAMAGE,
  ONE_LIFE_ZONE_OPACITY,
  oneLifeWindowBounds,
} from '@/lib/charts/oneLifeDamageGradient'
import { TimeseriesEfficiency } from './TimeseriesSquadAdapted'

let captured: Record<string, unknown> | null = null
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: Record<string, unknown> }) => {
    captured = option
    return <div data-testid="echarts-mock" />
  },
}))

interface MarkAreaItem {
  yAxis: number
  itemStyle?: { color: string; opacity: number }
}
interface EfficiencySeries {
  name?: string
  markArea?: { data: MarkAreaItem[][] }
  markLine?: { data: Array<{ yAxis: number }> }
  areaStyle?: unknown
}

function row(i: number, dealt: number, kills: number): TimeseriesMatchRow {
  return {
    match_id: `m${i}`,
    damage_dealt: dealt,
    damage_taken: 900,
    kills,
    deaths: 4,
  } as TimeseriesMatchRow
}

const LABELS = {
  rendementLabel: 'Dégâts / frag',
  resistanceLabel: 'Dégâts / mort',
  refLabel: '1 vie ({{HP}})',
}

async function optionFor(rows: TimeseriesMatchRow[]) {
  // Deux options comparées DANS le même test : on démonte le rendu précédent
  // (l'auto-cleanup de testing-library n'intervient qu'entre les tests).
  cleanup()
  captured = null
  render(<TimeseriesEfficiency rows={rows} title="Rendement" emptyMessage="vide" {...LABELS} />)
  await screen.findByTestId('echarts-mock')
  return captured as unknown as {
    series: EfficiencySeries[]
    yAxis: { min: number; max: number }
  }
}

describe('TimeseriesEfficiency — cadre « une vie » en dégâts bruts', () => {
  it('zones orientées below-is-good : ROUGE au-dessus du repère, VERT en dessous', async () => {
    const opt = await optionFor([row(0, 1800, 8), row(1, 1000, 3)])
    const zones = opt.series[0].markArea?.data
    expect(zones).toHaveLength(2)
    // Bande haute (repère → haut de fenêtre) = gaspillage : token défavorable.
    expect(zones![0][0].yAxis).toBe(ONE_LIFE_DAMAGE)
    expect(zones![0][0].itemStyle?.opacity).toBe(ONE_LIFE_ZONE_OPACITY.neg)
    // Bande basse (bas de fenêtre → repère) = efficace : token favorable.
    expect(zones![1][1].yAxis).toBe(ONE_LIFE_DAMAGE)
    expect(zones![1][0].itemStyle?.opacity).toBe(ONE_LIFE_ZONE_OPACITY.pos)
    // Garde explicite contre une ré-uniformisation sur les cartes Escouade
    // (qui, elles, sont above-is-good).
    expect(zones![0][0].itemStyle?.opacity).not.toBe(zones![1][0].itemStyle?.opacity)
  })

  it('plus d\'aire PAR COURBE : la surface est portée par les zones de grille', async () => {
    const opt = await optionFor([row(0, 1800, 8), row(1, 1000, 3)])
    expect(opt.series[0].areaStyle).toBeUndefined()
    // La courbe « Dégâts / mort » ne peint pas un second jeu de bandes (sens
    // inverse sur le même axe → bandes contradictoires).
    expect(opt.series[1]?.markArea).toBeUndefined()
  })

  it('fenêtre d\'axe FIXE : deux sessions d\'amplitudes opposées, mêmes bornes', async () => {
    const bounds = oneLifeWindowBounds(ONE_LIFE_DAMAGE)
    const tight = await optionFor([row(0, 900, 4), row(1, 920, 4)])
    const wide = await optionFor([row(0, 4000, 1), row(1, 900, 12)])
    expect(tight.yAxis.min).toBe(bounds.min)
    expect(tight.yAxis.max).toBe(bounds.max)
    expect(wide.yAxis.min).toBe(tight.yAxis.min)
    expect(wide.yAxis.max).toBe(tight.yAxis.max)
  })

  it('repère « une vie » tracé au barème du titre', async () => {
    const opt = await optionFor([row(0, 1800, 8)])
    expect(opt.series[0].markLine?.data).toEqual([{ yAxis: ONE_LIFE_DAMAGE }])
  })
})
