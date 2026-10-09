/**
 * LE MONTAGE DES SECTIONS PAR ONGLET — le CÂBLAGE, pas le rendu de chaque section.
 *
 * Les tests de chaque carte lui passent son modèle à la main : ils resteraient tous verts si
 * l'onglet oubliait de brancher un bloc de la réponse ou si la capability masquait une section
 * pour de bon. Ces cas-ci pincent la chaîne réponse de page -> onglet -> section.
 *
 * ONGLET USAGES = L'EMPRISE DU PÉRIMÈTRE SOLO (plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05) :
 * l'ordre des blocs et leurs intertitres (maquette v4), le retrait d'un bloc sans donnée
 * (intertitre compris), l'état vide, l'anglais, et Halo 5 sans film (seuls les frags aux armes
 * spéciales de la feuille de match). Les autres onglets : les sections qui en sont parties n'y
 * sont plus.
 */import type { ReactNode } from 'react'
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
import { block2209 } from '@/features/squad/objectif/objectif.fixtures'
import { MATCH_ROWS, lives, soloEmprise, soloEmpriseSansFilm } from './usages/usages.fixtures'

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

function renderUsages(data: TimeseriesPageResponse, locale: 'fr' | 'en' = 'fr') {
  renderWithProviders(<TimeseriesUsagesTab data={data} locale={locale} t={(key) => key} />)
}

/** Le périmètre solo complet : Emprise, vies, objectif (escouade d'un seul joueur), portée. */
function fullPage(extra: Record<string, unknown> = {}): TimeseriesPageResponse {
  return page({
    match_rows: MATCH_ROWS,
    emprise: soloEmprise(),
    lives_near_teammate: lives(),
    formes_retenues: { ...block2209(), squad: [{ xuid: 'xj', gamertag: 'JGtm' }] },
    weapon_range: weaponRangeBlock,
    ...extra,
  })
}

/** Les blocs montés, dans l'ordre de la page. */
const blocks = () => Array.from(document.querySelectorAll('[data-testid^="usages-section-"]')).map((n) => n.getAttribute('data-testid')!.replace('usages-section-', ''))

describe('Onglet Usages — montage de « Portée »', () => {
  it('avec le bloc servi et la capability active, la section est montée', () => {
    setTitle(['weapon_range'])
    renderUsages(page({ weapon_range: weaponRangeBlock }))
    expect(screen.getByRole('region', { name: RANGE_REGION })).toBeInTheDocument()
  })

  it('sans bloc dans la réponse, la section ne s’affiche pas', () => {
    setTitle(['weapon_range'])
    renderUsages(page({ lives_near_teammate: lives() }))
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })

  it('sans la capability du titre, le bloc servi reste masqué', () => {
    setTitle(['matchmaking'])
    renderUsages(page({ weapon_range: weaponRangeBlock, lives_near_teammate: lives() }))
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
  })
})

describe('Onglet Usages — l’Emprise du périmètre solo', () => {
  it('les huit blocs dans l’ordre de la maquette, après « Portée »', () => {
    setTitle(['weapon_range'])
    renderUsages(fullPage())
    expect(blocks()).toEqual(['bilan', 'carte', 'mine', 'prendre', 'lives', 'objectif', 'equipment'])
    const range = screen.getByRole('region', { name: RANGE_REGION })
    const bilan = screen.getByTestId('usages-section-bilan')
    expect(range.compareDocumentPosition(bilan) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('intertitres concis, sans mention de couverture', () => {
    setTitle(['weapon_range'])
    renderUsages(fullPage())
    const titres = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent ?? '')
    for (const t of ['Portée', 'Par carte', 'Prises', 'Rendement des ressources', 'Isolement', 'Objectif', 'Équipement']) {
      expect(titres).toContain(t)
    }
    // Aucune mention de couverture à côté de l'intertitre (« N matchs filmés sur M »).
    expect(titres).toContain('Ressources')
    expect(titres.join(' ')).not.toMatch(/filmés sur/)
  })

  it('chaque bloc monte ses cartes', () => {
    setTitle(['weapon_range'])
    renderUsages(fullPage())
    for (const id of ['emprise-control', 'usages-map-grid', 'usages-mine', 'emprise-production', 'emprise-yield', 'usages-lives', 'objective-balance', 'objective-solo-sheet', 'usages-equipment']) {
      expect(screen.getByTestId(id)).toBeInTheDocument()
    }
    expect(screen.getByText('Contrôle des ressources, cumul par match')).toBeInTheDocument()
  })

  it('un bloc sans donnée se retire, intertitre compris', () => {
    setTitle(['weapon_range'])
    renderUsages(fullPage({ lives_near_teammate: undefined, formes_retenues: undefined }))
    expect(blocks()).toEqual(['bilan', 'carte', 'mine', 'prendre', 'equipment'])
    expect(screen.queryByText('Isolement')).not.toBeInTheDocument()
    expect(screen.queryByText('Objectif')).not.toBeInTheDocument()
  })

  it('l’équipement sans film se retire seul', () => {
    const e = soloEmprise()
    delete e.equipment
    renderUsages(fullPage({ emprise: e }))
    expect(blocks()).not.toContain('equipment')
  })

  it('Halo 5 sans film : seuls les frags aux armes spéciales de la feuille de match', () => {
    setTitle(['matchmaking'])
    renderUsages(page({ match_rows: MATCH_ROWS, emprise: soloEmpriseSansFilm(), weapon_range: weaponRangeBlock }))
    expect(blocks()).toEqual(['prendre'])
    expect(screen.getByTestId('piste-camps-row-power_weapon')).toBeInTheDocument()
    expect(screen.queryByTestId('emprise-yield')).not.toBeInTheDocument()
    expect(screen.queryByRole('region', { name: RANGE_REGION })).not.toBeInTheDocument()
    expect(screen.queryByText('timeseries.usages.empty_title')).not.toBeInTheDocument()
  })

  it('en anglais : intertitres et titres de carte', () => {
    setTitle(['weapon_range'])
    renderUsages(fullPage(), 'en')
    const titres = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent ?? '')
    for (const t of ['By map', 'Pickups', 'Resource efficiency', 'Isolation', 'Objective', 'Equipment']) {
      expect(titres).toContain(t)
    }
    // « Isolation » : l'intertitre ET le titre de sa carte (voulu, comme sur les autres pages).
    expect(screen.getAllByText('Isolation')).toHaveLength(2)
    expect(screen.getByText('Resource control, by map')).toBeInTheDocument()
  })
})

describe('Onglet Usages — état vide', () => {
  it('sans aucun bloc, l’onglet porte son état vide plutôt que rien', () => {
    setTitle(['weapon_range'])
    renderUsages(page())
    expect(screen.getByText('timeseries.usages.empty_title')).toBeInTheDocument()
    expect(screen.getByText('timeseries.usages.empty_description')).toBeInTheDocument()
    expect(blocks()).toEqual([])
  })

  it('un seul bloc servi retire l’état vide', () => {
    renderUsages(page({ lives_near_teammate: lives() }))
    expect(screen.queryByText('timeseries.usages.empty_title')).not.toBeInTheDocument()
    expect(blocks()).toEqual(['lives'])
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
  it('la Synthèse ne monte plus « Portée »', () => {
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

  it('la Progression garde son profil d’intensité, remonté avant la tendance de performance', () => {
    renderProgression(page())
    const titres = screen.getAllByTestId('chart-card').map((n) => n.textContent ?? '')
    const intensite = titres.findIndex((x) => x.includes('timeseries.progression.intensity_title'))
    const perf = titres.findIndex((x) => x.includes('timeseries.summary.perf_label'))
    expect(intensite).toBeGreaterThanOrEqual(0)
    expect(perf).toBeGreaterThan(intensite)
  })
})
