/**
 * « XP de carrière (estimée) » de l'onglet Progression : la capability du titre
 * (`analytics.career_xp_estimate`) garde la carte même sans match estimé sur le périmètre (Firefight seul) —
 * titre et état vide, pas un trou ; un titre sans la capability n'a pas la carte, sauf si un match porte
 * déjà une estimation.
 */
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

const capability = vi.hoisted(() => ({ supported: false }))
vi.mock('@/lib/capabilities/dataCapabilities', () => ({ useDataCapability: () => capability.supported }))
vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: ({ title }: { title?: ReactNode }) => <div data-testid="chart-card">{title}</div>,
}))

import type { TimeseriesPageResponse } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { TimeseriesProgressionTab } from './TimeseriesPage.progression'
import { MATCH_ROWS } from './usages/usages.fixtures'

function render(rows: unknown[]) {
  const data = { total_matches: rows.length, match_rows: rows, distributions_tab: {}, cumul_tab: {}, intensity_tab: {}, top_weapons: [], outcomes_over_time: [], map_breakdown: [] } as unknown as TimeseriesPageResponse
  renderWithProviders(
    <TimeseriesProgressionTab
      data={data}
      playerSlug="test-player"
      locale="fr"
      t={(key) => key}
      fieldMappings={undefined}
      soloFilterContext={{ filter_mode: 'period' }}
      filterContextHash="h"
      explorerMatchRows={[]}
    />,
  )
}

const XP_TITLE = 'timeseries.progression.career_xp_title'

describe('Progression — XP de carrière estimée', () => {
  beforeEach(() => {
    capability.supported = false
  })

  it('titre sans la capability et aucun match estimé : pas de carte', () => {
    render(MATCH_ROWS)
    expect(screen.queryByText(XP_TITLE)).toBeNull()
  })

  it('titre avec la capability, aucun match estimé : la carte reste (le graphe dit son état vide)', () => {
    capability.supported = true
    render(MATCH_ROWS)
    expect(screen.getByText(XP_TITLE)).toBeInTheDocument()
  })

  it('un match estimé monte la carte même avant que la capability soit connue', () => {
    render(MATCH_ROWS.map((r, i) => (i === 0 ? { ...r, career_xp_estimated: 1200 } : r)))
    expect(screen.getByText(XP_TITLE)).toBeInTheDocument()
  })
})
