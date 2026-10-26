/**
 * Barre de performance au-dessus des colonnes « Par carte » / « Par mode » : couleur du palier, emplacement
 * en pointillé quand aucun match n'est noté, mêmes catégories dans les deux grilles, infobulle commune.
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
  name: string
  stack?: string
  xAxisIndex?: number
  data: ({ value: number; itemStyle?: { color: string } } | number | null)[]
  itemStyle?: { color: string; borderType?: string; borderColor?: string }
  label?: { show?: boolean; formatter?: (p: { value: number }) => string }
}
interface Option {
  grid: { top: number; height?: number; bottom?: number }[]
  xAxis: { data: string[]; axisLabel: { show?: boolean } }[]
  yAxis: { name: string; min?: number; max?: number }[]
  series: Series[]
  axisPointer: { link: { xAxisIndex: string }[] }
  legend: { data: string[] }
  tooltip: { formatter: (p: unknown) => string }
}

const build = (entries = ENTRIES) => buildOutcomesOption(entries, LABELS, 'Matchs', PERF) as unknown as Option
const flat = (html: string) => html.normalize('NFKC')

beforeEach(() => {
  _resetActivePalette()
  applyPalette(defaultPalette, 'default')
  useAppShellStore.setState({ locale: 'fr' })
})

describe('barre de performance par colonne', () => {
  it('colore chaque barre du palier de son entrée', () => {
    const [scored] = build().series.filter((s) => s.name === PERF)
    const colors = scored.data.map((d) => (d && typeof d === 'object' ? d.itemStyle?.color : undefined))
    expect(colors[0]).toBe(defaultPalette['perf-tier-2'])
    expect(colors[1]).toBe(defaultPalette['perf-tier-1'])
    expect(colors[2]).toBeUndefined()
    expect(colors[0]).not.toBe(colors[1])
  })

  it('porte la valeur moyenne, étiquetée à l’entier', () => {
    const [scored] = build().series.filter((s) => s.name === PERF)
    const values = scored.data.map((d) => (d && typeof d === 'object' ? d.value : null))
    expect(values).toEqual([72.4, 83, null])
    expect(flat(scored.label?.formatter?.({ value: 72.4 }) ?? '')).toBe('72')
  })

  it('colonne sans score : emplacement en pointillé pleine hauteur, rien d’autre', () => {
    const [, placeholder] = build().series.filter((s) => s.name === PERF)
    expect(placeholder.data).toEqual([null, null, 100])
    expect(placeholder.itemStyle?.borderType).toBe('dashed')
    expect(placeholder.itemStyle?.color).toBe('transparent')
    expect(placeholder.label?.show).toBe(false)
  })

  it('ne dessine un palier hors 1..5 que comme emplacement en pointillé', () => {
    const opt = build([{ name: 'Origin', wins: 1, losses: 0, ties: 0, unfinished: 0, performance_avg: 60, perf_tier: 0 }])
    const [scored, placeholder] = opt.series.filter((s) => s.name === PERF)
    expect(scored.data).toEqual([null])
    expect(placeholder.data).toEqual([100])
  })

  it('entrée sans champ de performance (réponse antérieure) : pointillé, aucune erreur', () => {
    const opt = build([{ name: 'Recharge', wins: 1, losses: 0, ties: 0, unfinished: 0 }])
    const [, placeholder] = opt.series.filter((s) => s.name === PERF)
    expect(placeholder.data).toEqual([100])
  })
})

describe('deux grilles sur le même axe des colonnes', () => {
  it('même ordre des catégories, étiquettes masquées en haut, performance au-dessus des issues', () => {
    const opt = build()
    expect(opt.xAxis[0].data).toEqual(['Recharge', 'Streets', 'Solitude'])
    expect(opt.xAxis[1].data).toEqual(opt.xAxis[0].data)
    expect(opt.xAxis[0].axisLabel.show).toBe(false)
    expect(opt.xAxis[1].axisLabel.show).not.toBe(false)
    expect(opt.grid[0].top).toBeLessThan(opt.grid[1].top)
    expect((opt.grid[0].top ?? 0) + (opt.grid[0].height ?? 0)).toBeLessThanOrEqual(opt.grid[1].top)
  })

  it('axe de performance de 0 à 100 nommé par le libellé du champ ; axe des matchs inchangé', () => {
    const opt = build()
    expect(opt.yAxis[0]).toMatchObject({ name: PERF, min: 0, max: 100 })
    expect(opt.yAxis[1].name).toBe('Matchs')
  })

  it('séries de performance sur la grille du haut, issues sur celle du bas, légende réduite aux issues', () => {
    const opt = build()
    const perf = opt.series.filter((s) => s.name === PERF)
    const outcomes = opt.series.filter((s) => s.name !== PERF)
    expect(perf).toHaveLength(2)
    expect(perf.every((s) => s.xAxisIndex === 0)).toBe(true)
    expect(outcomes.map((s) => s.name)).toEqual(['Victoires', 'Défaites', 'Égalités', 'Abandons'])
    expect(outcomes.every((s) => s.xAxisIndex === 1)).toBe(true)
    expect(opt.legend.data).toEqual(['Victoires', 'Défaites', 'Égalités', 'Abandons'])
  })

  it('curseur commun aux deux abscisses', () => {
    expect(build().axisPointer.link).toEqual([{ xAxisIndex: 'all' }])
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
  })

  it('colonne notée : la performance en premier, à l’entier, puis les issues présentes', () => {
    const html = flat(
      tip(build())([
        row('Victoires', 0, 5),
        row('Défaites', 1, 3),
        row('Égalités', 2, 0),
        row('Abandons', 3, 1),
        row(PERF, 4, 72.4),
        row(PERF, 5, null),
      ]),
    )
    expect(html).toContain(`${PERF}`)
    expect(html).toContain('72')
    expect(html).not.toContain('72,4')
    expect(html).not.toContain('Égalités')
    expect(html.indexOf(PERF)).toBeLessThan(html.indexOf('Abandons'))
    expect(html.indexOf('Abandons')).toBeLessThan(html.indexOf('Victoires'))
  })

  it('colonne sans score : « — » à la place de la valeur, jamais le 100 du pointillé', () => {
    const html = flat(
      tip(build())([row('Victoires', 0, 0), row('Défaites', 1, 1), row('Abandons', 3, 2), row(PERF, 4, null), row(PERF, 5, 100)]),
    )
    expect(html).toContain(PERF)
    expect(html).toContain('—')
    expect(html).not.toContain('100')
  })
})
