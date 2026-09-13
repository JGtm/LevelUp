/**
 * LE MONTAGE DES DEUX SECTIONS MIGRÉES DEPUIS LA SYNTHÈSE (2026-09-13).
 *
 * « Portée des engagements » vit désormais sur l'onglet Résumé, « Usages d'équipement » sur
 * l'onglet Progression. Les tests de chaque section lui passent son bloc à la main : ils
 * resteraient tous verts si l'onglet oubliait de le brancher (`data.weapon_range`,
 * `data.equipment_usage`) ou si la capability produit masquait la section pour de bon. Ces
 * cas-ci pincent le CÂBLAGE : réponse de page -> onglet -> section, et la porte de capability.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: () => <div data-testid="chart-card" />,
}))

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import { TimeseriesSummaryTab } from './TimeseriesPage.summary'
import { TimeseriesProgressionTab } from './TimeseriesPage.progression'

const RANGE_REGION = 'Portée par arme'

const weaponRangeBlock = {
  weapons: [
    {
      weapon_key: 'hinf_br75',
      label: 'Fusil de combat BR75',
      label_en: 'BR75 Battle Rifle',
      kills: { measured: 281, p10: 7.1, median: 13.6, p90: 24.9, above_pct: 37, level_pct: 49, below_pct: 14 },
      deaths: { measured: 402, p10: 8.9, median: 16.4, p90: 29.7, above_pct: 10, level_pct: 52, below_pct: 38 },
    },
  ],
  median_kills_m: 7.4,
  median_deaths_m: 11.8,
  measured_kills: 1214,
  total_kills: 1602,
  measured_deaths: 1087,
  total_deaths: 1455,
  below_threshold_kills: [],
  below_threshold_deaths: [],
}

const equipmentUsageBlock = {
  available: true,
  matches_measured: 5,
  matches_total: 5,
  families: [{ family_key: 'wall', taken: 12, used: 8, kept: 2, dropped: 2 }],
  players: [{ xuid: 'x1', taken: 12, used: 8, kept: 2, dropped: 2, pad_pickups: 3 }],
}

/** Une réponse de page réduite à ce que l'onglet lit vraiment ici. */
function page(extra: Record<string, unknown> = {}): TimeseriesPageResponse {
  return {
    total_matches: 0,
    match_rows: [],
    distributions_tab: {},
    cumul_tab: {},
    intensity_tab: {},
    top_weapons: [],
    outcomes_over_time: [],
    map_breakdown: [],
    ...extra,
  } as unknown as TimeseriesPageResponse
}

/** Un titre qui déclare (ou non) une capability — `useCapability` est fail-open sans titre. */
function setTitle(capabilities: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'sonde',
    availableTitles: [
      { slug: 'sonde', name: 'Sonde', status: 'active', capabilities, is_default: false },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

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

function renderProgression(data: TimeseriesPageResponse) {
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

describe('Onglet Résumé — montage de « Portée des engagements »', () => {
  it('avec le bloc servi et la capability active, la section est montée', () => {
    setTitle(['weapon_range'])
    renderSummary(page({ weapon_range: weaponRangeBlock }))
    expect(screen.getByRole('region', { name: RANGE_REGION })).toBeInTheDocument()
  })

  it('sans bloc dans la réponse, la section ne s’affiche pas', () => {
    setTitle(['weapon_range'])
    renderSummary(page())
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })

  it('sans la capability du titre, le bloc servi reste masqué', () => {
    setTitle(['matchmaking'])
    renderSummary(page({ weapon_range: weaponRangeBlock }))
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })
})

describe('Onglet Progression — montage de « Usages d’équipement »', () => {
  it('avec le bloc servi, les cartes d’usage sont montées', () => {
    renderProgression(page({ equipment_usage: equipmentUsageBlock }))
    expect(screen.getByRole('region', { name: "Usages d'équipement" })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Contrôle des armes spéciales' })).toBeInTheDocument()
  })

  it('sans bloc dans la réponse, aucune carte d’usage', () => {
    renderProgression(page())
    expect(screen.queryByRole('region', { name: "Usages d'équipement" })).not.toBeInTheDocument()
  })
})
