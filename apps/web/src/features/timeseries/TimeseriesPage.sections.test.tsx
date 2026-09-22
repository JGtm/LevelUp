/**
 * LE MONTAGE DES SECTIONS PAR ONGLET — le CÂBLAGE, pas le rendu de chaque section.
 *
 * Les tests de chaque section lui passent son bloc à la main : ils resteraient tous verts si
 * un onglet oubliait de le brancher (`data.weapon_range`, `data.equipment_usage`,
 * `data.formes_retenues`) ou si la capability produit masquait la section pour de bon. Ces
 * cas-ci pincent la chaîne réponse de page -> onglet -> section, et la porte de capability.
 *
 * REGROUPEMENT DU 2026-09-22 : les trois sections nourries par le film décodé (portée des
 * engagements, usages d'équipement, formes retenues) ont quitté la Synthèse et la
 * Progression pour l'onglet « Usages ». Ce fichier vérifie donc aussi qu'elles ne sont PLUS
 * sur leurs onglets d'origine, et que l'onglet Usages sans film porte son état vide plutôt
 * que de rester muet.
 */
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

// Le titre est RENDU par le double : c'est la seule marque qui distingue une carte de
// graphe d'une autre, et l'ordre des blocs d'un onglet se vérifie sur ces titres.
vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: ({ title }: { title?: ReactNode }) => <div data-testid="chart-card">{title}</div>,
}))

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import { TimeseriesSummaryTab } from './TimeseriesPage.summary'
import { TimeseriesProgressionTab } from './TimeseriesPage.progression'
import { TimeseriesUsagesTab } from './TimeseriesPage.usages'

const RANGE_REGION = 'Portée par arme'
const EQUIPMENT_REGION = "Usages d'équipement"
const PAD_CONTROL_REGION = 'Contrôle des armes spéciales'

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

function renderUsages(data: TimeseriesPageResponse) {
  renderWithProviders(<TimeseriesUsagesTab data={data} locale="fr" t={(key) => key} />)
}

describe('Onglet Usages — montage de « Portée des engagements »', () => {
  it('avec le bloc servi et la capability active, la section est montée', () => {
    setTitle(['weapon_range'])
    renderUsages(page({ weapon_range: weaponRangeBlock }))
    expect(screen.getByRole('region', { name: RANGE_REGION })).toBeInTheDocument()
  })

  it('sans bloc dans la réponse, la section ne s’affiche pas', () => {
    setTitle(['weapon_range'])
    renderUsages(page({ equipment_usage: equipmentUsageBlock }))
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })

  it('sans la capability du titre, le bloc servi reste masqué', () => {
    setTitle(['matchmaking'])
    renderUsages(page({ weapon_range: weaponRangeBlock, equipment_usage: equipmentUsageBlock }))
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })
})

describe('Onglet Usages — montage de « Usages d’équipement »', () => {
  it('avec le bloc servi, le titre de section et les cartes d’usage sont montés', () => {
    renderUsages(page({ equipment_usage: equipmentUsageBlock }))
    expect(
      screen.getByRole('heading', { name: 'timeseries.usages.equipment_title' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('region', { name: EQUIPMENT_REGION })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: PAD_CONTROL_REGION })).toBeInTheDocument()
  })
})

describe('Onglet Usages — état vide', () => {
  it('sans aucun bloc de film, l’onglet porte son état vide plutôt que rien', () => {
    setTitle(['weapon_range'])
    renderUsages(page())
    expect(screen.getByText('timeseries.usages.empty_title')).toBeInTheDocument()
    expect(screen.getByText('timeseries.usages.empty_description')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: EQUIPMENT_REGION })).not.toBeInTheDocument()
  })

  it('un bloc d’usage servi retire l’état vide', () => {
    renderUsages(page({ equipment_usage: equipmentUsageBlock }))
    expect(screen.queryByText('timeseries.usages.empty_title')).not.toBeInTheDocument()
  })
})

describe('« Balance des dégâts cumulée » — déplacée vers le Résumé (2026-09-22)', () => {
  it('le Résumé la monte, juste après « Assistances »', () => {
    renderSummary(page())
    const titres = screen.getAllByTestId('chart-card').map((n) => n.textContent ?? '')
    const assistances = titres.findIndex((x) => x.includes('Assistances'))
    const balance = titres.findIndex((x) => x.includes('timeseries.progression.net_lives_title'))
    expect(assistances).toBeGreaterThanOrEqual(0)
    expect(balance).toBe(assistances + 1)
  })

  it('la Progression ne la monte plus', () => {
    renderProgression(page())
    const titres = screen.getAllByTestId('chart-card').map((n) => n.textContent ?? '')
    expect(titres.some((x) => x.includes('timeseries.progression.net_lives_title'))).toBe(false)
  })
})

describe('Onglets d’origine — les sections du film n’y sont plus', () => {
  it('la Synthèse ne monte plus « Portée des engagements »', () => {
    setTitle(['weapon_range'])
    renderSummary(page({ weapon_range: weaponRangeBlock }))
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })

  it('la Progression garde ses blocs, dans l’ordre attendu', () => {
    renderProgression(page())
    const titres = screen.getAllByTestId('chart-card').map((n) => n.textContent ?? '')
    const attendus = [
      'timeseries.progression.per_minute_title',
      'timeseries.progression.intensity_title',
      'timeseries.summary.perf_label',
      'timeseries.progression.spree_headshots_title',
      'timeseries.progression.rank_score_title',
      'timeseries.progression.efficiency_title',
    ]
    const rangs = attendus.map((cle) => titres.findIndex((x) => x.includes(cle)))
    expect(rangs.every((r) => r >= 0)).toBe(true)
    expect(rangs).toEqual([...rangs].sort((a, b) => a - b))
  })

  it('la Progression ne monte plus « Usages d’équipement »', () => {
    renderProgression(page({ equipment_usage: equipmentUsageBlock }))
    expect(screen.queryByRole('region', { name: EQUIPMENT_REGION })).not.toBeInTheDocument()
  })

  it('la Progression garde son profil d’intensité, remonté avant la tendance de performance', () => {
    renderProgression(page({ equipment_usage: equipmentUsageBlock }))
    const titres = screen.getAllByTestId('chart-card').map((n) => n.textContent ?? '')
    const intensite = titres.findIndex((x) => x.includes('timeseries.progression.intensity_title'))
    const perf = titres.findIndex((x) => x.includes('timeseries.summary.perf_label'))
    expect(intensite).toBeGreaterThanOrEqual(0)
    expect(perf).toBeGreaterThan(intensite)
  })
})
