/**
 * Tests B1 (plan PLAN_EXPECTED_FDA) + retouche UX 2026-07-24 — « Écart cumulé au
 * FDA attendu » (Timeseries), forme alignée sur Sessions.
 *
 * - `buildFdaGapCumulativeOption` (pur) : cumul signé ancré à 0 (1 aire
 *   divergente + markLine 0) PLUS 1 courbe fine « FDA attendu » PAR MATCH sur
 *   le MÊME axe Y (pas de double axe — retour utilisateur 2026-07-24), report D5
 *   (report du cumul, jamais 0, point conservé), ORDRE DU SERVICE respecté
 *   (pas de re-tri). `resolveToken`
 *   renvoie '' hors runtime CSS → on teste la structure/les données, pas les
 *   couleurs.
 * - `TimeseriesFdaGapTrend` (composant) : masquage par capability `expected_stats`
 *   (absente → non rendu). echarts-for-react mocké (canvas jsdom instable).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { TimeseriesMatchRow } from '@/lib/api/types'

import {
  TimeseriesFdaGapTrend,
  buildFdaGapCumulativeOption,
  type FdaGapCumulativeLabels,
} from './TimeseriesFdaGapTrend'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const LABELS: FdaGapCumulativeLabels = {
  series: 'Écart cumulé',
  real: 'Réel',
  expected: 'Attendu',
  gap: 'Écart',
  avgCaption: 'Écart moyen par match',
  perMatch: '/match',
}

function row(
  kda: number | undefined,
  kdaExpected: number | undefined,
  map = 'Bazaar',
  start = '2025-01-01T00:00:00Z',
): TimeseriesMatchRow {
  return {
    accuracy: null,
    assists: 0,
    damage_dealt: null,
    damage_taken: null,
    deaths: 0,
    index: 0,
    kills: 0,
    match_id: 'm',
    outcome: null,
    perf_score: null,
    personal_score: null,
    playlist_name: 'pl',
    rank: null,
    start_time: start,
    time_played_seconds: null,
    map_name: map,
    kda: kda,
    kda_expected: kdaExpected,
  }
}

interface OptShape {
  series: Array<{
    type: string
    name: string
    data: (number | null)[]
    yAxisIndex?: number
    areaStyle?: { origin?: number; color?: { type?: string } }
    lineStyle?: { color?: { type?: string } | string }
    markLine?: { data: Array<{ yAxis: number }> }
  }>
  legend?: { data: string[] }
  xAxis: { data: string[]; boundaryGap?: boolean }
  yAxis: unknown
}

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true, effective_hp_to_kill: 225 },
    ],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('buildFdaGapCumulativeOption', () => {
  it('série vide → null', () => {
    expect(buildFdaGapCumulativeOption([], LABELS)).toBeNull()
  })

  it('cumul signé ancré à 0 : 1 aire divergente + markLine 0 + 1 courbe FDA attendu par match sur le même axe', () => {
    const opt = buildFdaGapCumulativeOption(
      [row(1.5, 1.0), row(0.8, 1.2), row(2.0, 1.0)],
      LABELS,
    ) as unknown as OptShape
    // 2 séries : le cumul signé (aire) + FDA attendu par match.
    expect(opt.series).toHaveLength(2)
    expect(opt.series[0].name).toBe('Écart cumulé')
    // Cumul du différentiel réel − attendu, arrondi 2 décimales.
    expect(opt.series[0].data).toEqual([0.5, 0.1, 1.1])
    // Aire ancrée à 0 + markLine 0 + dégradé divergent (linéaire).
    expect(opt.series[0].areaStyle?.origin).toBe(0)
    expect(opt.series[0].markLine?.data[0].yAxis).toBe(0)
    expect((opt.series[0].areaStyle?.color as { type?: string })?.type).toBe('linear')
    // Courbe FDA attendu PAR MATCH (pas cumulée) — MÊME axe Y (pas de yAxisIndex).
    expect(opt.series[1].name).toBe('Attendu')
    expect(opt.series[1].yAxisIndex).toBeUndefined()
    expect(opt.series[1].data).toEqual([1, 1.2, 1])
    // Axe Y UNIQUE (objet, pas tableau) + légende sur les 2 séries.
    expect(Array.isArray(opt.yAxis)).toBe(false)
    expect(opt.legend?.data).toEqual(['Écart cumulé', 'Attendu'])
    expect(opt.xAxis.boundaryGap).toBe(false)
    expect(opt.xAxis.data).toHaveLength(3)
  })

  it('report D5 : un match sans attendu reporte le cumul (jamais 0, point conservé) et troue la courbe par-match', () => {
    const opt = buildFdaGapCumulativeOption(
      [row(1.5, 1.0), row(0.8, undefined), row(2.0, 1.0)],
      LABELS,
    ) as unknown as OptShape
    // Cumul : 0.5, report 0.5 (pas 0, pas de trou), puis reprise +1.0 = 1.5.
    expect(opt.series[0].data).toEqual([0.5, 0.5, 1.5])
    // La courbe FDA attendu par match reflète directement kda_expected par
    // match — null quand absent (point troué, pas de report).
    expect(opt.series[1].data).toEqual([1, null, 1])
    // Le point du match sans attendu figure quand même sur l'axe (3 catégories).
    expect(opt.xAxis.data).toHaveLength(3)
  })

  it('report D5 côté réel manquant également', () => {
    const opt = buildFdaGapCumulativeOption([row(undefined, 1.0), row(2.0, 1.0)], LABELS) as unknown as OptShape
    // #1 sans réel → gap null → cumul reste 0 ; #2 → +1.0.
    expect(opt.series[0].data).toEqual([0, 1])
  })

  it('ordre du service respecté (PAS de re-tri chronologique)', () => {
    // start_time volontairement DÉCROISSANT : le cumul suit l'ordre du tableau
    // (déjà trié côté service), jamais start_time (contrairement à Sessions).
    const opt = buildFdaGapCumulativeOption(
      [row(2.0, 1.0, 'Bazaar', '2025-01-02T00:00:00Z'), row(1.5, 1.0, 'Live Fire', '2025-01-01T00:00:00Z')],
      LABELS,
    ) as unknown as OptShape
    // Ordre du tableau : #1 gap 1.0 (cum 1.0), #2 gap 0.5 (cum 1.5).
    // Un re-tri chronologique donnerait [0.5, 1.5].
    expect(opt.series[0].data).toEqual([1, 1.5])
  })
})

describe('TimeseriesFdaGapTrend — masquage capability', () => {
  it('capability expected_stats présente → chart rendu', async () => {
    setTitleCaps(['expected_stats'])
    render(<TimeseriesFdaGapTrend rows={[row(1.5, 1.0), row(0.8, 1.2)]} labels={LABELS} locale="fr" title="Écart cumulé au FDA attendu" />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('capability expected_stats absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <TimeseriesFdaGapTrend rows={[row(1.5, 1.0), row(0.8, 1.2)]} labels={LABELS} locale="fr" title="Écart cumulé au FDA attendu" />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})

describe('TimeseriesFdaGapTrend — KPI écart moyen par match', () => {
  it('affiche l écart moyen signé formaté par locale (matchs avec attendu)', async () => {
    setTitleCaps(['expected_stats'])
    // gaps : +1.0 et 0.0 → moyenne +0.5 → « +0,5/match » (séparateur décimal fr).
    render(
      <TimeseriesFdaGapTrend
        rows={[row(2.0, 1.0), row(1.0, 1.0)]}
        labels={LABELS}
        locale="fr"
        title="Écart cumulé au FDA attendu"
      />,
    )
    const kpi = await screen.findByTestId('fda-gap-avg')
    expect(kpi).toHaveTextContent('Écart moyen par match')
    expect(kpi).toHaveTextContent('+0,5/match')
  })

  it('aucun match avec attendu → « — »', async () => {
    setTitleCaps(['expected_stats'])
    render(
      <TimeseriesFdaGapTrend
        rows={[row(1.5, undefined), row(0.8, undefined)]}
        labels={LABELS}
        locale="fr"
        title="Écart cumulé au FDA attendu"
      />,
    )
    const kpi = await screen.findByTestId('fda-gap-avg')
    expect(kpi).toHaveTextContent('—')
  })
})
