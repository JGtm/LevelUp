/**
 * Tests du lot L — « Balance des dégâts cumulée » (Timeseries / Progression).
 *
 * - `computeTimeseriesNetLives` : cumul signé correct sur 3 matchs, report D5 d'un
 *   match sans dégâts (le point reste sur l'axe), en-tête d'infobulle daté.
 * - `buildTimeseriesNetLivesOption` : aire divergente ancrée à 0 + markLine 0,
 *   étiquette du DERNIER point, ordre du service respecté (pas de re-tri), option
 *   `null` quand aucun match ne porte les dégâts.
 * - Composant : carte rendue avec son titre ; masquage par capability `damage_taken`.
 *   echarts-for-react mocké (canvas jsdom instable), comme TimeseriesFdaGapTrend.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { TimeseriesMatchRow } from '@/lib/api/types'

import {
  TimeseriesNetLivesTrend,
  buildTimeseriesNetLivesOption,
  computeTimeseriesNetLives,
  type TimeseriesNetLivesLabels,
} from './TimeseriesNetLivesTrend'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const HP = 225
const LABELS: TimeseriesNetLivesLabels = { series: 'Balance cumulée', match: 'Balance du match' }

function row(
  damageDealt: number | null,
  damageTaken: number | null,
  map = 'Bazaar',
  start = '2026-01-01T00:00:00Z',
): TimeseriesMatchRow {
  return {
    accuracy: null,
    assists: 0,
    damage_dealt: damageDealt,
    damage_taken: damageTaken,
    deaths: 0,
    index: 0,
    kills: 0,
    map_name: map,
    match_id: 'm',
    outcome: null,
    perf_score: null,
    personal_score: null,
    playlist_name: 'pl',
    rank: null,
    start_time: start,
  } as unknown as TimeseriesMatchRow
}

/** Série de 3 matchs : +2 vies, −1 vie, puis un match sans dégâts (report D5). */
const ROWS = [
  row(900, 450, 'Bazaar', '2026-01-01T00:00:00Z'),
  row(450, 675, 'Live Fire', '2026-01-02T00:00:00Z'),
  row(null, null, 'Recharge', '2026-01-03T00:00:00Z'),
]

interface SeriesLike {
  data: number[]
  areaStyle?: { origin?: number; opacity?: number }
  endLabel?: { show?: boolean; formatter?: (p: { value?: unknown }) => string }
  markLine?: { data?: Array<{ yAxis?: number }> }
}

function seriesOf(option: ReturnType<typeof buildTimeseriesNetLivesOption>): SeriesLike {
  return (option as unknown as { series: SeriesLike[] }).series[0]
}

describe('computeTimeseriesNetLives', () => {
  it('cumule la balance signée match après match, dans l ordre du service', () => {
    const points = computeTimeseriesNetLives(ROWS, HP, 'fr')
    expect(points.map((p) => p.value)).toEqual([2, -1, null])
    expect(points.map((p) => p.cumulative)).toEqual([2, 1, 1])
  })

  it('garde sur l axe le match sans dégâts et reporte le cumul (D5)', () => {
    const points = computeTimeseriesNetLives(ROWS, HP, 'fr')
    expect(points).toHaveLength(3)
    expect(points[2].value).toBeNull()
    expect(points[2].cumulative).toBe(points[1].cumulative)
  })

  it('date l en-tête d infobulle et porte la carte', () => {
    const points = computeTimeseriesNetLives(ROWS, HP, 'fr')
    expect(points[0].header).toContain('Bazaar')
    expect(points[0].header).toMatch(/2026/)
    expect(points[0].label).toContain('#1')
  })
})

describe('buildTimeseriesNetLivesOption', () => {
  it('trace une aire ancrée à 0 avec sa ligne de référence', () => {
    const option = buildTimeseriesNetLivesOption(computeTimeseriesNetLives(ROWS, HP, 'fr'), LABELS)
    const s = seriesOf(option)
    expect(s.data).toEqual([2, 1, 1])
    expect(s.areaStyle?.origin).toBe(0)
    expect(s.markLine?.data).toEqual([{ yAxis: 0 }])
  })

  it('étiquette le DERNIER point avec la valeur d arrivée du cumul', () => {
    const option = buildTimeseriesNetLivesOption(computeTimeseriesNetLives(ROWS, HP, 'fr'), LABELS)
    const s = seriesOf(option)
    expect(s.endLabel?.show).toBe(true)
    expect(s.endLabel?.formatter?.({ value: s.data[s.data.length - 1] })).toBe('+1')
  })

  it('rend l état vide quand aucun match ne porte les deux dégâts', () => {
    const points = computeTimeseriesNetLives([row(null, null), row(300, null)], HP, 'fr')
    expect(buildTimeseriesNetLivesOption(points, LABELS)).toBeNull()
    expect(buildTimeseriesNetLivesOption([], LABELS)).toBeNull()
  })
})

/** Titre de test porteur (ou non) de la capability `damage_taken`. */
function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      {
        slug: 'test_title',
        name: 'Test',
        status: 'active',
        capabilities: caps,
        is_default: true,
        effective_hp_to_kill: HP,
      } as unknown as never,
    ],
  })
}

describe('TimeseriesNetLivesTrend', () => {
  afterEach(() => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  function renderCard(rows: TimeseriesMatchRow[]) {
    return render(
      <TimeseriesNetLivesTrend
        rows={rows}
        locale="fr"
        title="Balance des dégâts cumulée"
        tooltip="Balance = (infligés − subis) ÷ {{HP}} PV."
        labels={LABELS}
        avgCaption="Avantage net par match"
        avgUnit="vies/match"
        emptyMessage="Aucun match ne porte les dégâts."
      />,
    )
  }

  it('rend la carte avec son titre et sa pastille de moyenne', async () => {
    setTitleCaps(['damage_taken'])
    renderCard(ROWS)
    expect(screen.getByText('Balance des dégâts cumulée')).toBeInTheDocument()
    expect(screen.getByTestId('timeseries-net-lives-kpi')).toBeInTheDocument()
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('affiche l état vide sans masquer le titre quand les dégâts manquent', () => {
    setTitleCaps(['damage_taken'])
    renderCard([row(null, null)])
    expect(screen.getByText('Balance des dégâts cumulée')).toBeInTheDocument()
    expect(screen.getByText('Aucun match ne porte les dégâts.')).toBeInTheDocument()
  })

  it('se retire pour un titre sans dégâts subis (capability damage_taken absente)', () => {
    setTitleCaps(['ranked'])
    const { container } = renderCard(ROWS)
    expect(container).toBeEmptyDOMElement()
  })
})
