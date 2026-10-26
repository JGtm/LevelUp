/**
 * Formats des infobulles de la Synthèse (retours du 2026-10-17) : nombres à la langue, une ligne par
 * série qui a une valeur, pile lue de haut en bas.
 */
import { afterEach, describe, expect, it } from 'vitest'

import type { TopWeekItem } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildOutcomesOption } from './SynthesisOutcomesByGroupChart'
import { buildTopWeeksOption, topWeeksNames } from './SynthesisTopWeeksChart'

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

type Formatter = (params: unknown) => string
const tip = (opt: unknown): Formatter => (opt as { tooltip: { formatter: Formatter } }).tooltip.formatter
const flat = (html: string) => html.normalize('NFKC')

describe('infobulles de la Synthèse', () => {
  it('résultats par groupe : issues présentes seulement, de la plus haute à la plus basse de la pile', () => {
    const opt = buildOutcomesOption(
      [{ name: 'Recharge', wins: 12, losses: 8, ties: 0, unfinished: 0 }],
      { win: 'Victoires', loss: 'Défaites', tie: 'Égalités', dnf: 'Abandons' },
      'Matchs',
      'Score de performance',
    )
    const html = tip(opt)([
      { seriesName: 'Victoires', seriesIndex: 0, value: 12, color: '#111', axisValueLabel: 'Recharge' },
      { seriesName: 'Défaites', seriesIndex: 1, value: 8, color: '#222', axisValueLabel: 'Recharge' },
      { seriesName: 'Égalités', seriesIndex: 2, value: 0, color: '#333', axisValueLabel: 'Recharge' },
      { seriesName: 'Abandons', seriesIndex: 3, value: 0, color: '#444', axisValueLabel: 'Recharge' },
    ])
    expect(html).toContain('Recharge')
    expect(html).not.toContain('Égalités')
    expect(html).not.toContain('Abandons')
    expect(html.indexOf('Défaites')).toBeLessThan(html.indexOf('Victoires'))
  })

  it('meilleures semaines : décompte entier, taux en pourcentage à une décimale, à la langue', () => {
    const weeks = [
      { week_label: '12/10', week_start: '2026-10-12', match_count: 1500, wins: 700, win_rate: 46.666666, kd_ratio: null },
    ] as TopWeekItem[]
    const opt = buildTopWeeksOption(weeks, topWeeksNames('fr'))
    const html = flat(
      tip(opt)([
        { seriesName: 'Victoires', seriesIndex: 0, value: 700, color: '#111', axisValueLabel: '12/10' },
        { seriesName: 'Autres matchs', seriesIndex: 1, value: 800, color: '#222', axisValueLabel: '12/10' },
        { seriesName: 'Taux de victoire', seriesIndex: 2, value: 46.7, color: '#333', axisValueLabel: '12/10' },
      ]),
    )
    expect(html).toContain('46,7 %')
    expect(html).toContain('800')
    expect(html.indexOf('Autres matchs')).toBeLessThan(html.indexOf('Victoires'))
  })

  it('meilleures semaines : noms des séries et de l’axe traduits, sans « Top 1 » ni « Win % »', () => {
    const weeks = [{ week_label: '12/10', week_start: '2026-10-12', match_count: 10, wins: 6, win_rate: 60, kd_ratio: null }] as TopWeekItem[]
    for (const [locale, names] of [
      ['fr', ['Victoires', 'Autres matchs', 'Taux de victoire', 'Taux de victoire (%)']],
      ['en', ['Wins', 'Other matches', 'Win rate', 'Win rate (%)']],
    ] as const) {
      const opt = buildTopWeeksOption(weeks, topWeeksNames(locale)) as unknown as {
        series: { name: string }[]
        yAxis: { name?: string }[]
      }
      expect(opt.series.map((x) => x.name)).toEqual([names[0], names[1], names[2]])
      expect(opt.yAxis[1].name).toBe(names[3])
      expect(JSON.stringify(opt)).not.toMatch(/Top 1|Taux Top|Win %/)
    }
  })
})
