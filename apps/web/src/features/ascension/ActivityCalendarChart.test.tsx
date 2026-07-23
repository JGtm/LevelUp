/**
 * Tests ActivityCalendarChart (DEC-5/D3) — géométrie pure du calendrier
 * (semaine × jour) + builder d'option ECharts + états rendus.
 *
 * La heatmap monte ECharts via ChartCard (lazy import echarts-for-react) ; on
 * mocke echarts-for-react car jsdom n'a pas de canvas (cf. mémoire
 * reference_echarts_jsdom_test_mock).
 */
import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ActivityDay } from './types'
import {
  ActivityCalendarChart,
  buildActivityCalendarOption,
  buildCalendarGrid,
} from './ActivityCalendarChart'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

describe('buildCalendarGrid — géométrie semaine × jour', () => {
  it('place deux jours à 7 jours d\'écart sur la même ligne, semaines adjacentes', () => {
    const days: ActivityDay[] = [
      { date: '2026-05-04', count: 3 },
      { date: '2026-05-11', count: 1 }, // +7 jours
    ]
    const grid = buildCalendarGrid('2026-05-04', '2026-05-31', days)
    expect(grid.cells).toHaveLength(2)
    expect(grid.cells[0].weekday).toBe(grid.cells[1].weekday) // même jour de semaine
    expect(Math.abs(grid.cells[1].week - grid.cells[0].week)).toBe(1)
    expect(grid.maxCount).toBe(3)
  })

  it('ignore les jours hors fenêtre (avant since) et les counts <= 0', () => {
    const days: ActivityDay[] = [
      { date: '2026-04-20', count: 5 }, // avant since → ignoré
      { date: '2026-05-05', count: 0 }, // count 0 → ignoré
      { date: '2026-05-06', count: 2 },
    ]
    const grid = buildCalendarGrid('2026-05-04', '2026-05-31', days)
    expect(grid.cells).toHaveLength(1)
    expect(grid.cells[0].date).toBe('2026-05-06')
    expect(grid.maxCount).toBe(2)
  })
})

describe('buildActivityCalendarOption', () => {
  it('produit une série heatmap avec un point par jour joué + visualMap', () => {
    const days: ActivityDay[] = [
      { date: '2026-05-04', count: 3 },
      { date: '2026-05-06', count: 1 },
    ]
    const opt = buildActivityCalendarOption({
      since: '2026-05-04',
      until: '2026-05-31',
      days,
      locale: 'fr',
    })
    const series = (opt.series as Array<{ type: string; data: unknown[] }>)[0]
    expect(series.type).toBe('heatmap')
    expect(series.data).toHaveLength(2)
    expect(opt.visualMap).toBeDefined()
  })

  it('sans jour joué : pas de visualMap (état vide)', () => {
    const opt = buildActivityCalendarOption({
      since: '2026-05-04',
      until: '2026-05-31',
      days: [],
      locale: 'fr',
    })
    expect(opt.visualMap).toBeUndefined()
  })
})

describe('ActivityCalendarChart — rendu', () => {
  it('rend la heatmap quand il y a des jours joués', async () => {
    renderWithProviders(
      <ActivityCalendarChart
        since="2026-05-04"
        until="2026-05-31"
        days={[{ date: '2026-05-06', count: 2 }]}
      />,
    )
    expect(await screen.findByTestId('echarts-stub')).toBeInTheDocument()
  })

  it('affiche l\'état vide quand aucun jour joué', () => {
    renderWithProviders(
      <ActivityCalendarChart since="2026-05-04" until="2026-05-31" days={[]} />,
    )
    expect(screen.getByTestId('chart-card-empty')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-stub')).not.toBeInTheDocument()
  })
})
