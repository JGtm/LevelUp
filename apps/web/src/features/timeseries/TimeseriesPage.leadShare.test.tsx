/**
 * LE MONTAGE DE LA MAINMISE sur l'onglet Résumé des Séries temporelles : « par mois » juste après
 * « Performance solo », « par carte » sous la paire Taux de victoire / Performance par carte, et
 * rien sur un titre sans décodeur de film. Le titre de chaque carte est rendu par le double de
 * `ChartCard` : c'est lui qui donne l'ordre.
 */
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: ({ title, series }: { title?: ReactNode; series?: unknown[] }) => (
    <div data-testid="chart-card" data-empty={series && series.length === 0 ? 'oui' : undefined}>
      {title}
    </div>
  ),
}))

import { renderWithProviders } from '@/test/render-utils'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TimeseriesPageResponse } from '@/lib/api/types'

import { TimeseriesSummaryTab } from './TimeseriesPage.summary'

function setTitle(capabilities: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'sonde',
    availableTitles: [
      { slug: 'sonde', name: 'Sonde', status: 'active', capabilities, is_default: false },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
  })
}

afterEach(() => useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [], locale: 'fr' }))

function page(extra: Record<string, unknown> = {}): TimeseriesPageResponse {
  return {
    total_matches: 0,
    match_rows: [],
    distributions_tab: {},
    cumul_tab: {},
    intensity_tab: {},
    top_weapons: [],
    outcomes_over_time: [],
    map_breakdown: [
      { map_ui: 'Bazaar', match_count: 5, win_rate: 0.5, performance_avg: 50, historical_performance_avg: 52, lead_share: 0.6, lead_share_matches: 4 },
    ],
    solo_session_perf: { granularity: 'session', points: [{ session_label: 's', started_at_utc: '2026-11-01', match_count: 3, win_rate: 0.5 }] },
    lead_share_by_month: [{ month: '2026-10', start_date: '2026-10-01', lead_share: 0.5, measured_matches: 3, matches: 4 }],
    ...extra,
  } as unknown as TimeseriesPageResponse
}

function renderSummary(data: TimeseriesPageResponse) {
  renderWithProviders(
    <TimeseriesSummaryTab
      data={data}
      t={(key) => key}
      fieldMappings={undefined}
      outcomeLabels={{ win: 'V', loss: 'D', tie: 'N', dnf: 'X', unknown: '?' }}
      mapLabelOf={(m) => m}
    />,
  )
}

const titres = () => Array.from(document.querySelectorAll('[data-testid="chart-card"]')).map((n) => n.textContent ?? '')

describe('Séries temporelles — la Mainmise', () => {
  it('titre qui décode le film : par mois après « Performance solo », par carte sous la paire de cartes', () => {
    setTitle(['replay'])
    renderSummary(page())
    const t = titres()
    const solo = t.findIndex((x) => x.includes('solo_perf_session'))
    const mois = t.findIndex((x) => x.includes(LEAD_SHARE_TEXT.fr.perMonthTitle))
    const perfCarte = t.findIndex((x) => x.includes('mapperf_vs_history_title'))
    const carte = t.findIndex((x) => x.includes(LEAD_SHARE_TEXT.fr.perMapTitle))
    expect(solo).toBeGreaterThanOrEqual(0)
    expect(mois).toBe(solo + 1)
    expect(perfCarte).toBeGreaterThan(mois)
    expect(carte).toBe(t.length - 1)
    expect(carte).toBeGreaterThan(perfCarte)
  })

  it('titre sans décodeur de film : aucune des deux cartes', () => {
    setTitle(['analytics.timeseries'])
    renderSummary(page())
    const t = titres().join('|')
    expect(t).not.toContain(LEAD_SHARE_TEXT.fr.perMonthTitle)
    expect(t).not.toContain(LEAD_SHARE_TEXT.fr.perMapTitle)
  })

  it('aucun mois mesuré : la carte reste, en état vide (jamais zéro)', () => {
    setTitle(['replay'])
    renderSummary(
      page({ lead_share_by_month: [{ month: '2026-10', start_date: '2026-10-01', measured_matches: 0, matches: 4 }] }),
    )
    const carte = Array.from(document.querySelectorAll('[data-testid="chart-card"]')).find((n) =>
      n.textContent?.includes(LEAD_SHARE_TEXT.fr.perMonthTitle),
    )
    expect(carte?.getAttribute('data-empty')).toBe('oui')
  })
})
