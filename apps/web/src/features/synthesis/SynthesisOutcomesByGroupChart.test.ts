/**
 * Courbe de performance des graphes « Par carte » / « Par mode » : une seule grille, axe de droite de
 * 0 à 100, points colorés par palier, trou de courbe pour une colonne sans match noté, infobulle commune.
 */
import { beforeEach, describe, expect, it } from 'vitest'

import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildOutcomesOption } from './SynthesisOutcomesByGroupChart'

const LABELS = { win: 'Victoires', loss: 'Défaites', tie: 'Égalités', dnf: 'Abandons' }
const PERF = 'Score de performance'

const ENTRIES = [
  { name: 'Recharge', wins: 5, losses: 3, ties: 0, unfinished: 1, performance_avg: 72.4, perf_tier: 2 },
  { name: 'Streets', wins: 2, losses: 2, ties: 0, unfinished: 0, performance_avg: 83, perf_tier: 1 },
  { name: 'Solitude', wins: 0, losses: 1, ties: 0, unfinished: 2 },
]

interface Series {
  type: string
  name: string
  stack?: string
  yAxisIndex?: number
  connectNulls?: boolean
  data: ({ value: number; itemStyle?: { color: string } } | number | null)[]
  label?: { show?: boolean }
}
interface Option {
  grid: { top: number; right: number; bottom: number }
  xAxis: { data: string[] }
  yAxis: { name: string; min?: number; max?: number; position?: string }[]
  series: Series[]
  legend: { data?: string[] }
  tooltip: { formatter: (p: unknown) => string }
}

const build = (entries = ENTRIES) => buildOutcomesOption(entries, LABELS, 'Matchs', PERF) as unknown as Option
const flat = (html: string) => html.normalize('NFKC')
const curve = (opt: Option) => opt.series.filter((s) => s.type === 'line')

beforeEach(() => {
  _resetActivePalette()
  applyPalette(defaultPalette, 'default')
  useAppShellStore.setState({ locale: 'fr' })
})

describe('courbe de performance', () => {
  it('une seule série en courbe, sur l’axe de droite, rendue après les quatre issues', () => {
    const opt = build()
    expect(curve(opt)).toHaveLength(1)
    const [line] = curve(opt)
    expect(line.name).toBe(PERF)
    expect(line.yAxisIndex).toBe(1)
    expect(opt.series.indexOf(line)).toBe(4)
    expect(opt.series.filter((s) => s.type === 'bar').map((s) => s.name)).toEqual([
      'Victoires',
      'Défaites',
      'Égalités',
      'Abandons',
    ])
  })

  it('colore chaque point du palier de son entrée', () => {
    const [line] = curve(build())
    const colors = line.data.map((d) => (d && typeof d === 'object' ? d.itemStyle?.color : undefined))
    expect(colors[0]).toBe(defaultPalette['perf-tier-2'])
    expect(colors[1]).toBe(defaultPalette['perf-tier-1'])
    expect(colors[2]).toBeUndefined()
    expect(colors[0]).not.toBe(colors[1])
  })

  it('porte la valeur moyenne ; colonne sans score : null et trou de courbe', () => {
    const [line] = curve(build())
    const values = line.data.map((d) => (d && typeof d === 'object' ? d.value : d))
    expect(values).toEqual([72.4, 83, null])
    expect(line.connectNulls).toBe(false)
  })

  it('aucune étiquette de valeur sur les points', () => {
    expect(curve(build())[0].label?.show).toBe(false)
  })

  it('un palier hors 1..5 ou une entrée sans champ de performance : null, aucune erreur', () => {
    const opt = build([
      { name: 'Origin', wins: 1, losses: 0, ties: 0, unfinished: 0, performance_avg: 60, perf_tier: 0 },
      { name: 'Recharge', wins: 1, losses: 0, ties: 0, unfinished: 0 },
    ])
    expect(curve(opt)[0].data).toEqual([null, null])
  })
})

describe('une grille, deux axes verticaux', () => {
  it('grille unique avec une marge droite pour l’axe de performance', () => {
    const opt = build()
    expect(Array.isArray(opt.grid)).toBe(false)
    expect(opt.grid.right).toBeGreaterThan(20)
    expect(Array.isArray(opt.xAxis)).toBe(false)
  })

  it('même ordre des catégories que les entrées', () => {
    expect(build().xAxis.data).toEqual(['Recharge', 'Streets', 'Solitude'])
  })

  it('axe de gauche = matchs ; axe de droite 0..100 nommé par le libellé du champ', () => {
    const opt = build()
    expect(opt.yAxis).toHaveLength(2)
    expect(opt.yAxis[0].name).toBe('Matchs')
    expect(opt.yAxis[1]).toMatchObject({ name: PERF, min: 0, max: 100, position: 'right' })
  })

  it('les issues restent empilées sur l’axe de gauche', () => {
    const bars = build().series.filter((s) => s.type === 'bar')
    expect(bars.every((s) => s.stack === 'total' && (s.yAxisIndex ?? 0) === 0)).toBe(true)
  })

  it('liste vide : option vide, comme avant', () => {
    const opt = buildOutcomesOption([], LABELS, 'Matchs', PERF) as unknown as Record<string, unknown>
    expect(opt.series).toBeUndefined()
    expect(opt.grid).toBeUndefined()
  })
})

describe('infobulle', () => {
  const tip = (opt: Option) => opt.tooltip.formatter
  const row = (seriesName: string, seriesIndex: number, value: number | null) => ({
    seriesName,
    seriesIndex,
    value,
    color: '#111',
    axisValueLabel: 'Recharge',
    dataIndex: 0,
  })

  it('colonne notée : la performance en premier, à l’entier, puis les issues présentes', () => {
    const html = flat(
      tip(build())([
        row('Victoires', 0, 5),
        row('Défaites', 1, 3),
        row('Égalités', 2, 0),
        row('Abandons', 3, 1),
        row(PERF, 4, 72.4),
      ]),
    )
    expect(html).toContain(PERF)
    expect(html).toContain('72')
    expect(html).not.toContain('72,4')
    expect(html).not.toContain('Égalités')
    expect(html.indexOf(PERF)).toBeLessThan(html.indexOf('Abandons'))
    expect(html.indexOf('Abandons')).toBeLessThan(html.indexOf('Victoires'))
  })

  it('colonne sans score (valeur nulle) : « — » en première ligne', () => {
    const html = flat(tip(build())([row('Victoires', 0, 0), row('Défaites', 1, 1), row('Abandons', 3, 2), row(PERF, 4, null)]))
    expect(html).toContain(PERF)
    expect(html).toContain('—')
    expect(html.indexOf(PERF)).toBeLessThan(html.indexOf('Abandons'))
  })

  it('colonne sans score (série omise par le moteur) : « — » quand même', () => {
    const html = flat(tip(build())([row('Victoires', 0, 0), row('Défaites', 1, 1), row('Abandons', 3, 2)]))
    expect(html).toContain(PERF)
    expect(html).toContain('—')
  })
})
