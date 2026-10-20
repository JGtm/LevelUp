/**
 * Noms des séries, des axes et des infobulles des graphes de la Synthèse : à la langue de l'application
 * (manifestes `synthesis.scope.*` et `common.charts.axis_matches`), plus de « Escouade » ni de « Matchs »
 * écrits en dur dans la version anglaise.
 */
import { screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ComparisonMetricItem } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { SynthesisBipolaireChart } from './SynthesisBipolaireChart'
import { SynthesisOutcomesByGroupChart } from './SynthesisOutcomesByGroupChart'

const received = vi.hoisted(() => ({ option: null as null | Record<string, unknown> }))
vi.mock('echarts-for-react', () => ({
  default: (p: { option: Record<string, unknown> }) => {
    received.option = p.option
    return <div data-testid="echarts-mock" />
  },
}))

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const METRICS = [{ label: 'kd_ratio', solo_value: 1.2, squad_value: 1.5 }] as unknown as ComparisonMetricItem[]

describe('SynthesisBipolaireChart — noms des séries', () => {
  for (const [locale, names] of [
    ['fr', ['Solo', 'Escouade']],
    ['en', ['Solo', 'Squad']],
  ] as const) {
    it(`${locale} : séries et lignes d’infobulle`, async () => {
      useAppShellStore.setState({ locale })
      renderWithProviders(<SynthesisBipolaireChart metrics={METRICS} fieldLabels={['FDA']} />)
      await waitFor(() => expect(screen.getByTestId('echarts-mock')).toBeInTheDocument())
      const opt = received.option as unknown as {
        series: { name: string }[]
        tooltip: { formatter: (p: unknown) => string }
      }
      expect(opt.series.map((s) => s.name)).toEqual([...names])
      const html = opt.tooltip.formatter([{ dataIndex: 0, name: 'FDA' }])
      expect(html).toContain(`${names[0]} :`)
      expect(html).toContain(`${names[1]} :`)
    })
  }
})

describe('SynthesisOutcomesByGroupChart — axe des matchs', () => {
  for (const [locale, axis] of [
    ['fr', 'Matchs'],
    ['en', 'Matches'],
  ] as const) {
    it(`${locale} : l’axe se nomme à la langue quand la table de champs ne le fournit pas`, async () => {
      useAppShellStore.setState({ locale })
      renderWithProviders(
        <SynthesisOutcomesByGroupChart title="Par carte" entries={[{ name: 'Recharge', wins: 2, losses: 1, ties: 0, unfinished: 0 }]} />,
      )
      await waitFor(() => expect(screen.getByTestId('echarts-mock')).toBeInTheDocument())
      const opt = received.option as unknown as { yAxis: { name?: string } | { name?: string }[] }
      const names = JSON.stringify(opt.yAxis)
      expect(names).toContain(axis)
    })
  }
})
