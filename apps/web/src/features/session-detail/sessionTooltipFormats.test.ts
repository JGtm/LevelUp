/**
 * Formats des infobulles de la page Sessions (retours du 2026-10-17) : un nombre ne s'écrit jamais
 * brut (« 1.3333333333333333 », point décimal), il suit la langue de l'application. Un test par
 * graphe dont la bulle écrivait la valeur telle quelle.
 */
import { afterEach, describe, expect, it } from 'vitest'

import type { ChartSeries } from '@/components/charts/ChartCard'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildSessionEngagementOption } from './SessionEngagementChart'
import { buildSessionFdaBarsOption } from './SessionFdaBars'
import { buildSessionFdaGapOption, type FdaGapPoint } from './SessionFdaGapCumulative'
import { buildSessionModeBreakdownOption } from './SessionModeBreakdown'
import { buildSessionNetLivesOption, type NetLivesPoint } from './SessionNetLivesCumulative'
import { buildSessionParticipationBarsOption } from './SessionParticipationBars'
import { buildSessionPerfOption } from './SessionPerfTrend'

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

type Formatter = (params: unknown) => string
const tip = (opt: unknown): Formatter => (opt as { tooltip: { formatter: Formatter } }).tooltip.formatter
const strip = (html: string) => html.normalize('NFKC')

describe('infobulles de Sessions : nombres à la langue', () => {
  it('engagement par match : valeur signée à deux décimales au plus, jamais la queue du flottant', () => {
    const opt = buildSessionEngagementOption(
      [{ key: 'e', datapoints: [{ label: '#1', value: 4 / 3 }, { label: '#2', value: -2 }] }],
      { meanLabel: 'Moyenne', axisName: 'évén./min' },
    )
    const html = tip(opt)([{ name: '#1', value: 4 / 3, marker: '' }])
    expect(html).toContain('<b>+1,33</b>')
    expect(html).not.toContain('1.3333')
    const mark = (opt as { series: Array<{ markLine: { label: { formatter: string } } }> }).series[0].markLine.label.formatter
    expect(mark).toBe('Moyenne −0,33')
  })

  it('score de performance : moyenne et valeur à la langue de l’application', () => {
    useAppShellStore.setState({ locale: 'en' })
    const opt = buildSessionPerfOption(
      [{ key: 'p', datapoints: [{ label: '#1', score: 72.5, tier: 3 }, { label: '#2', score: 60, tier: 3 }] }],
      { scoreLabel: 'Score', meanLabel: 'Mean' },
    )
    expect(tip(opt)([{ name: '#1', value: 72.5, marker: '' }])).toContain('<b>72.5</b>')
    const mark = (opt as { series: Array<{ markLine: { label: { formatter: string } } }> }).series[0].markLine.label.formatter
    expect(mark).toBe('Mean 66.3')
  })

  it('écart de FDA cumulé : une décimale au plus, signe typographique', () => {
    const points: FdaGapPoint[] = [
      { label: '#1', real: 2, expected: 1.5, gap: 0.5, cumulative: 0.5 },
      { label: '#2', real: 1, expected: 2.3333333333333335, gap: -4 / 3, cumulative: -5 / 6 },
    ] as FdaGapPoint[]
    const opt = buildSessionFdaGapOption([{ key: 'g', datapoints: points }] as ChartSeries<FdaGapPoint>[], {
      seriesLabel: 'Cumul',
      realLabel: 'Réel',
      expectedLabel: 'Attendu',
      gapLabel: 'Écart',
    })
    const html = strip(tip(opt)([{ dataIndex: 1 }]))
    expect(html).toContain('Cumul: <b>−0,8</b>')
    expect(html).toContain('Attendu: 2,3')
    expect(html).toContain('Écart: −1,3')
    expect(html).not.toMatch(/\d\.\d{3,}/)
  })

  it('balance de vies cumulée : une décimale au plus', () => {
    const points: NetLivesPoint[] = [{ label: '#1', value: 4 / 3, cumulative: 4 / 3 }] as NetLivesPoint[]
    const opt = buildSessionNetLivesOption([{ key: 'n', datapoints: points }] as ChartSeries<NetLivesPoint>[], {
      seriesLabel: 'Cumul',
      matchLabel: 'Match',
    })
    const html = strip(tip(opt)([{ dataIndex: 0 }]))
    expect(html).toContain('Cumul: <b>+1,3</b>')
    expect(html).toContain('Match: +1,3')
  })

  it('FDA par stat : valeur absolue à la langue, morts sans signe', () => {
    const opt = buildSessionFdaBarsOption(
      [{ key: 'f', datapoints: [{ key: 'deaths', label: 'Morts', value: 7.5 }] }] as never,
      { decimals: 1 },
    )
    expect(tip(opt)([{ name: 'Morts', value: -7.5 }])).toBe('Morts: <b>7,5</b>')
  })

  it('répartition par mode et participation : comptes à la langue, libellés échappés', () => {
    const mode = buildSessionModeBreakdownOption([{ key: 'm', datapoints: [{ mode: 'Slayer', count: 1200 }] }], {
      countLabel: 'matchs',
    })
    expect(strip(tip(mode)([{ name: 'Slayer', value: 1200 }]))).toBe('Slayer: <b>1 200</b> matchs')
    const part = buildSessionParticipationBarsOption(
      [{ key: 'p', datapoints: [{ label: 'Combat', value: 72 }] }],
      { axisSide: 'right', colorToken: 'compare-a' },
    )
    expect(tip(part)([{ name: 'Combat', value: 72 }])).toBe('Combat: <b>72</b>')
  })
})
