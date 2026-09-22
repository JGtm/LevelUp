/**
 * La carte « Rôles de portée » du Résumé — ce qu'elle montre et ce qu'elle tait.
 *
 * Ce que ces tests cadenassent : un bloc absent ou sans profil rend un ÉTAT VIDE NOMMÉ (film
 * requis) et jamais un nuage à zéro point ; le nuage est SOLO (une seule série de points,
 * aucune légende de séries — il n'y a personne à nommer) ; l'axe X porte les étiquettes
 * « #N · carte » du plus ancien au plus récent ; un point sous le plancher se dessine creux ;
 * et la couverture est en pied, en une ligne.
 */
import { describe, expect, it, beforeEach, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchRangeBlock, MatchRangeProfile } from '@/lib/api/types'
import { PLANCHER_MESURE } from '@/features/squad/squadRangeRoles.logic'

import { TimeseriesRangeRolesCard } from './TimeseriesRangeRolesCard'

const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="portee-nuage-stub" />
  },
}))

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  captured.length = 0
})

function profil(i: number, measured = 12, map?: string): MatchRangeProfile {
  return {
    match_id: `m${i}`,
    played_at: `2026-09-${String(10 + i).padStart(2, '0')}T20:00:00Z`,
    map_name: map,
    lobby_median_m: 20,
    lobby_measured: 80,
    players: [{ xuid: 'x1', gamertag: 'JGtm', median_m: 22, lobby_delta_m: 2, measured }],
  }
}

function bloc(profiles: MatchRangeProfile[]): MatchRangeBlock {
  return { profiles, kills_measured: 412, kills_total: 544 }
}

async function option(): Promise<Record<string, unknown>> {
  await screen.findByTestId('portee-nuage-stub')
  return captured[captured.length - 1].option as Record<string, unknown>
}

describe('TimeseriesRangeRolesCard', () => {
  it('rend un ÉTAT VIDE NOMMÉ quand le bloc est absent', () => {
    renderWithProviders(<TimeseriesRangeRolesCard bloc={undefined} />)
    expect(screen.getByText('Aucune portée mesurée')).toBeInTheDocument()
    expect(screen.queryByTestId('timeseries-portee-couverture')).not.toBeInTheDocument()
  })

  it('rend un ÉTAT VIDE NOMMÉ quand aucun profil n’est servi', () => {
    renderWithProviders(<TimeseriesRangeRolesCard bloc={bloc([])} />)
    expect(screen.getByText(/aucun match de la sélection/i)).toBeInTheDocument()
  })

  it('trace UNE seule série de points et aucune légende de séries', async () => {
    renderWithProviders(
      <TimeseriesRangeRolesCard bloc={bloc([1, 2, 3, 4, 5, 6].map((i) => profil(i)))} />,
    )
    const opt = await option()
    const series = opt.series as Array<{ type: string }>
    expect(series.filter((s) => s.type === 'scatter')).toHaveLength(1)
    expect(series.filter((s) => s.type === 'line')).toHaveLength(1)
    expect(opt.legend).toEqual({ show: false })
  })

  it('étiquette l’axe X « #N · carte », du plus ancien au plus récent', async () => {
    renderWithProviders(
      <TimeseriesRangeRolesCard
        bloc={bloc([profil(1, 12, 'Live Fire'), profil(2, 12, 'Aquarius')])}
      />,
    )
    const opt = await option()
    expect((opt.xAxis as { data: string[] }).data).toEqual(['#1 · Live Fire', '#2 · Aquarius'])
  })

  it('dessine CREUX un match sous le plancher de mesure', async () => {
    renderWithProviders(
      <TimeseriesRangeRolesCard
        bloc={bloc([profil(1, PLANCHER_MESURE - 1), profil(2, PLANCHER_MESURE + 3)])}
      />,
    )
    const opt = await option()
    const nuage = (opt.series as Array<{ type: string; data: Array<{ itemStyle: { color: string } }> }>)
      .filter((s) => s.type === 'scatter')[0]
    expect(nuage.data[0].itemStyle.color).toBe('transparent')
    expect(nuage.data[1].itemStyle.color).not.toBe('transparent')
  })

  it('porte la couverture en pied, en une ligne', () => {
    renderWithProviders(<TimeseriesRangeRolesCard bloc={bloc([profil(1)])} />)
    expect(screen.getByTestId('timeseries-portee-couverture')).toHaveTextContent(
      /412\s*frags mesurés sur\s*544/,
    )
  })
})
