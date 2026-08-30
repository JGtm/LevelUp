/**
 * MatchAssistChart — le graphe des assistances et, surtout, SES TROIS ÉTATS VIDES.
 *
 * Ce que ces tests cadenassent tient en une phrase : « on ne sait pas » ne doit jamais
 * s'écrire « aucune ». Les deux messages sont distincts, ils dépendent d'un DÉNOMINATEUR
 * (measured_deaths) et pas de la longueur de la liste, et l'absence totale de bloc ne
 * rend rien du tout.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchAssistChart } from './MatchAssistChart'
import { MATCH_VIEW_TEXT } from './i18n'
import { assistAvgPctLookup, assistStackedSeries, assistStolenKey, assistStolenLookup } from './_chartSeries'
import type { MatchAssistPair, MatchAssistPairs, MatchScoreboardRow } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(${token})`,
  resolveToken: (token: string) => `var(${token})`,
}))

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

const pair = (over: Partial<MatchAssistPair> = {}): MatchAssistPair => ({
  assist_xuid: 'X_A',
  assist_gamertag: 'Alice',
  killer_xuid: 'X_B',
  killer_gamertag: 'Bob',
  assist_count: 2,
  stolen_count: 0,
  ...over,
})

const block = (over: Partial<MatchAssistPairs> = {}): MatchAssistPairs => ({
  measured_deaths: 40,
  pairs: [pair()],
  ...over,
})

const emptyScoreboard: MatchScoreboardRow[] = []

describe('MatchAssistChart — états vides', () => {
  it('ne rend RIEN quand le bloc est absent (aucune ligne de film)', () => {
    const { container } = render(
      <MatchAssistChart block={undefined} scoreboard={emptyScoreboard} meXUID={null} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('dit « non disponibles » quand measured_deaths vaut 0 — jamais « aucune »', () => {
    render(
      <MatchAssistChart
        block={block({ measured_deaths: 0, pairs: [] })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistNotUsable)).toBeTruthy()
    expect(screen.queryByText(MATCH_VIEW_TEXT.fr.assistNoData)).toBeNull()
  })

  it('dit « aucune assistance » quand c\'est MESURÉ et que la liste est vide', () => {
    render(
      <MatchAssistChart
        block={block({ measured_deaths: 38, pairs: [] })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistNoData)).toBeTruthy()
    expect(screen.queryByText(MATCH_VIEW_TEXT.fr.assistNotUsable)).toBeNull()
  })

  it('traite `pairs: null` (tableau nullable du contrat) comme une liste vide', () => {
    render(
      <MatchAssistChart
        block={{ measured_deaths: 12, pairs: null } as unknown as MatchAssistPairs}
        scoreboard={emptyScoreboard}
        meXUID={null}
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistNoData)).toBeTruthy()
  })
})

describe('MatchAssistChart — rendu', () => {
  // ChartCard charge ECharts en différé (React.lazy) : le stub n'apparaît qu'après
  // résolution — d'où le `findByTestId` plutôt qu'un `getByTestId` synchrone.
  it('rend le graphe avec son titre et une barre par assistant', async () => {
    render(
      <MatchAssistChart
        block={block({
          pairs: [
            pair({ assist_count: 3 }),
            pair({ assist_xuid: 'X_C', assist_gamertag: 'Carol', assist_count: 1 }),
          ],
        })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistTitle)).toBeTruthy()
    const option = (await screen.findByTestId('echarts-stub')).textContent ?? ''
    expect(option).toContain('Alice')
    expect(option).toContain('Carol')
  })
})

describe('assistStackedSeries', () => {
  it('retourne [] sans paire', () => {
    expect(assistStackedSeries([])).toEqual([])
  })

  it('agrège par ASSISTANT (et non par tueur), total décroissant', () => {
    const series = assistStackedSeries([
      pair({ assist_xuid: 'X_A', assist_gamertag: 'Alice', killer_xuid: 'X_B', killer_gamertag: 'Bob', assist_count: 1 }),
      pair({ assist_xuid: 'X_C', assist_gamertag: 'Carol', killer_xuid: 'X_B', killer_gamertag: 'Bob', assist_count: 4 }),
      pair({ assist_xuid: 'X_A', assist_gamertag: 'Alice', killer_xuid: 'X_D', killer_gamertag: 'Dave', assist_count: 2 }),
    ])
    expect(series).toHaveLength(1)
    const dps = series[0].datapoints
    // Carol (4) devant Alice (1+2=3).
    expect(dps.map((d) => d.category)).toEqual(['Carol', 'Alice'])
    expect(dps[1].components).toEqual({ Bob: 1, Dave: 2 })
  })

  it('met les ENNEMIS en tête, comme le graphe des antagonistes', () => {
    const scoreboard = [
      { xuid: 'X_ME', team_side: 'ally' },
      { xuid: 'X_ALLY', team_side: 'ally' },
      { xuid: 'X_FOE', team_side: 'enemy' },
    ] as unknown as MatchScoreboardRow[]
    const series = assistStackedSeries(
      [
        pair({ assist_xuid: 'X_ALLY', assist_gamertag: 'Ally', assist_count: 9 }),
        pair({ assist_xuid: 'X_FOE', assist_gamertag: 'Foe', assist_count: 1 }),
      ],
      scoreboard,
      'X_ME',
    )
    expect(series[0].datapoints.map((d) => d.category)).toEqual(['Foe', 'Ally'])
  })

  it('replie un tueur sans gamertag sur le nom masqué, jamais sur le xuid brut', () => {
    const series = assistStackedSeries([
      pair({ killer_gamertag: undefined, killer_xuid: 'X_SECRET1234' }),
    ])
    const keys = Object.keys(series[0].datapoints[0].components)
    expect(keys[0]).not.toBe('X_SECRET1234')
    expect(keys[0]).toContain('1234')
  })
})

describe('assistStolenLookup', () => {
  it('n\'indexe QUE les couples ayant au moins une élimination volée', () => {
    const lookup = assistStolenLookup([
      pair({ stolen_count: 0 }),
      pair({ assist_gamertag: 'Carol', killer_gamertag: 'Dave', stolen_count: 2 }),
    ])
    expect(lookup.size).toBe(1)
    expect(lookup.get(assistStolenKey('Carol', 'Dave'))).toBe(2)
    expect(lookup.get(assistStolenKey('Alice', 'Bob'))).toBeUndefined()
  })

  it('ne confond pas deux couples dont les noms contiennent des espaces', () => {
    const lookup = assistStolenLookup([
      pair({ assist_gamertag: 'A B', killer_gamertag: 'C', stolen_count: 1 }),
      pair({ assist_gamertag: 'A', killer_gamertag: 'B C', stolen_count: 5 }),
    ])
    expect(lookup.get(assistStolenKey('A B', 'C'))).toBe(1)
    expect(lookup.get(assistStolenKey('A', 'B C'))).toBe(5)
  })
})

describe('assistAvgPctLookup', () => {
  it("n'indexe QUE les couples portant une part moyenne mesurée — jamais de 0 % fabriqué", () => {
    const lookup = assistAvgPctLookup([
      pair(), // avg_assist_pct absent du contrat : la paire n'entre pas dans la map
      pair({ assist_gamertag: 'Carol', killer_gamertag: 'Dave', avg_assist_pct: 45 }),
    ])
    expect(lookup.size).toBe(1)
    expect(lookup.get(assistStolenKey('Carol', 'Dave'))).toBe(45)
    expect(lookup.get(assistStolenKey('Alice', 'Bob'))).toBeUndefined()
  })

  it('transmet la valeur SANS la plafonner (mesures réelles au-delà de 100)', () => {
    const lookup = assistAvgPctLookup([
      pair({ assist_gamertag: 'Carol', killer_gamertag: 'Dave', avg_assist_pct: 228 }),
    ])
    expect(lookup.get(assistStolenKey('Carol', 'Dave'))).toBe(228)
  })
})
