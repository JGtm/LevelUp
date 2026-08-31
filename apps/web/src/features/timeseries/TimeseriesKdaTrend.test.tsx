/**
 * TimeseriesKdaTrend.test.tsx — PREMIER test de ce composant (retours
 * utilisateur 2026-08-29, item 5 / DEC-5).
 *
 * Verrouille la structure de base (3 séries Frags/Bonus/Morts, Frags+Bonus
 * empilés, Bonus masqué par défaut via la légende ECharts NATIVE) et surtout
 * la STABILITÉ de l'axe Y : togglé via `legendselectchanged` (simulation de
 * la légende canvas, non-DOM), l'axe ne doit JAMAIS bouger — l'extent est
 * calculé sur le jeu complet (bonus inclus, qu'il soit affiché ou masqué), cf.
 * `stackedAxisExtent` (`components/charts/_utils.ts`).
 */
import { describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'

import type { TimeseriesMatchRow } from '@/lib/api/types'
import { TimeseriesKdaTrend } from './TimeseriesKdaTrend'

let capturedOption: Record<string, unknown> | null = null
let capturedOnEvents: Record<string, (params: unknown) => void> | undefined

vi.mock('echarts-for-react', () => ({
  default: ({
    option,
    onEvents,
  }: {
    option: Record<string, unknown>
    onEvents?: Record<string, (params: unknown) => void>
  }) => {
    capturedOption = option
    capturedOnEvents = onEvents
    return <div data-testid="echarts-mock" />
  },
}))

function row(i: number, kills: number, deaths: number, assists: number): TimeseriesMatchRow {
  return {
    match_id: `m${i}`,
    kills,
    deaths,
    assists,
    damage_dealt: null,
    damage_taken: null,
  } as TimeseriesMatchRow
}

const LABELS = {
  kills: 'Frags',
  deaths: 'Morts',
  yAxis: 'Frags / Morts',
  bonus: 'Bonus',
  bonusInfo: 'Une assistance vaut 1/3 de frag (ADR 0006).',
}

interface CapturedOption {
  series: Array<{ name: string; data: number[]; stack?: string }>
  yAxis: { min: number; max: number; name: string }
  legend: { data: string[]; selected: Record<string, boolean> }
  xAxis: { data: string[] }
}

async function renderChart(rows: TimeseriesMatchRow[]) {
  capturedOption = null
  capturedOnEvents = undefined
  render(<TimeseriesKdaTrend rows={rows} labels={LABELS} />)
  await screen.findByTestId('echarts-mock')
  return capturedOption as unknown as CapturedOption
}

describe('TimeseriesKdaTrend — rendu de base', () => {
  it('3 séries (Frags, Bonus, Morts), Frags+Bonus empilés, Bonus masqué par défaut', async () => {
    const opt = await renderChart([row(0, 12, 4, 9)]) // bonus = 9/3 = 3
    expect(opt.series).toHaveLength(3)
    expect(opt.series[0]).toMatchObject({ name: 'Frags', data: [12], stack: 'kills' })
    expect(opt.series[1]).toMatchObject({ name: 'Bonus', data: [3], stack: 'kills' })
    expect(opt.series[2]).toMatchObject({ name: 'Morts', data: [4] })
    expect(opt.series[2].stack).toBeUndefined() // morts jamais empilées avec kills/bonus
    expect(opt.legend.selected[LABELS.bonus]).toBe(false)
  })

  it('aucune ligne → rien à tracer, pas de rendu ECharts', () => {
    render(<TimeseriesKdaTrend rows={[]} labels={LABELS} />)
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})

describe('TimeseriesKdaTrend — échelle Y stable au toggle Bonus (item 5, DEC-5)', () => {
  it('yAxis.min/max IDENTIQUES avant et après avoir affiché Bonus via la légende native', async () => {
    // kills+bonus = 12+3 = 15 → dizaine sup. 20 ; morts (4) jamais négatives → min 0.
    const opt = await renderChart([row(0, 12, 4, 9)])
    const before = { min: opt.yAxis.min, max: opt.yAxis.max }
    expect(before).toEqual({ min: 0, max: 20 })

    // Simule ECharts qui notifie le clic sur l'item de légende « Bonus » (canvas,
    // pas de nœud DOM cliquable — c'est le seul point d'entrée testable).
    await act(() => {
      capturedOnEvents?.legendselectchanged?.({ selected: { [LABELS.bonus]: true } })
    })

    const after = capturedOption as unknown as CapturedOption
    expect({ min: after.yAxis.min, max: after.yAxis.max }).toEqual(before)
    // Le toggle a bien été pris en compte ailleurs (légende) : ce n'est pas un
    // test qui ne teste rien — seule l'échelle reste stable, pas tout l'état.
    expect(after.legend.selected[LABELS.bonus]).toBe(true)
  })

  it('min reste à 0 même quand les morts dépassent kills+bonus (jamais négatif sur ce chart)', async () => {
    const opt = await renderChart([row(0, 2, 20, 0)]) // deaths=20 largement > kills+bonus=2
    expect(opt.yAxis.min).toBe(0)
    expect(opt.yAxis.max).toBe(20) // max(kills+bonus=2, morts=20) = 20
  })

  it('extent calculé sur PLUSIEURS matchs : le pire cas (pas la somme) fixe le plafond', async () => {
    const opt = await renderChart([row(0, 5, 2, 0), row(1, 18, 3, 3)]) // #2 : 18+1=19 → 20
    expect(opt.yAxis.max).toBe(20)
  })
})
