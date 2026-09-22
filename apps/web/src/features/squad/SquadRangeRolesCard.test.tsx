/**
 * La carte « Rôles de portée » — ce qu'elle montre et ce qu'elle tait.
 *
 * Ce que ces tests cadenassent : un bloc sans profil rend un ÉTAT VIDE NOMMÉ (film requis),
 * jamais un nuage à zéro point ; l'axe X porte les étiquettes « #N · carte » du plus ancien
 * au plus récent ; un point creux se dessine creux ; la couverture est en pied, en une
 * ligne ; et le repli de la bande des rôles laisse les quatre premiers matchs sans rôle.
 */
import { describe, expect, it, beforeEach, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchRangeBlock, MatchRangeProfile } from '@/lib/api/types'

import { SquadRangeRolesCard } from './SquadRangeRolesCard'
import { PLANCHER_MESURE } from './squadRangeRoles.logic'

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
    players: [
      { xuid: 'x1', gamertag: 'JGtm', median_m: 22, lobby_delta_m: 2, measured },
      { xuid: 'x2', gamertag: 'Kaya', median_m: 28, lobby_delta_m: 8, measured },
    ],
  }
}

function bloc(profiles: MatchRangeProfile[]): MatchRangeBlock {
  return { profiles, kills_measured: 412, kills_total: 544 }
}

const roster = ['JGtm', 'Kaya']

/** Le graphe est chargé en `lazy` : attendre son stub avant de lire l'option. */
async function option(): Promise<Record<string, unknown>> {
  await screen.findByTestId('portee-nuage-stub')
  return captured[captured.length - 1].option as Record<string, unknown>
}

describe('SquadRangeRolesCard', () => {
  it('rend un ÉTAT VIDE NOMMÉ quand aucun profil n’est servi', () => {
    renderWithProviders(<SquadRangeRolesCard bloc={bloc([])} roster={roster} />)
    expect(screen.getByText('Aucune portée mesurée')).toBeTruthy()
    expect(screen.queryByTestId('portee-nuage-stub')).toBeNull()
    expect(screen.queryByTestId('squad-portee-fold-bande')).toBeNull()
  })

  it('porte les étiquettes « #N · carte » du plus ancien au plus récent', async () => {
    renderWithProviders(
      <SquadRangeRolesCard
        bloc={bloc([profil(1, 12, 'Streets'), profil(0, 12, 'Fragmentation')])}
        roster={roster}
      />,
    )
    const x = (await option()).xAxis as { data: string[] }
    expect(x.data).toEqual(['#1 · Fragment…', '#2 · Streets'])
  })

  it('dessine CREUX un point sous le plancher de frags mesurés, plein au-dessus', async () => {
    renderWithProviders(
      <SquadRangeRolesCard
        bloc={bloc([profil(0, PLANCHER_MESURE), profil(1, PLANCHER_MESURE - 1)])}
        roster={roster}
      />,
    )
    const series = (await option()).series as Array<{
      type: string
      data: Array<{ itemStyle: Record<string, unknown> }>
    }>
    const nuage = series.filter((s) => s.type === 'scatter')
    expect(nuage).toHaveLength(2)
    expect(nuage[0].data[0].itemStyle.borderType).toBeUndefined()
    expect(nuage[0].data[1].itemStyle.borderType).toBe('dashed')
    expect(nuage[0].data[1].itemStyle.color).toBe('transparent')
  })

  it('trace une TENDANCE par joueur et trois bandes de rôle, la ligne du lobby à zéro', async () => {
    const profils = Array.from({ length: 6 }, (_, i) => profil(i))
    renderWithProviders(<SquadRangeRolesCard bloc={bloc(profils)} roster={roster} />)
    const series = (await option()).series as Array<Record<string, unknown>>
    expect(series.filter((s) => s.type === 'line')).toHaveLength(2)
    const markArea = series[0].markArea as { data: unknown[] }
    expect(markArea.data).toHaveLength(3)
    const markLine = series[0].markLine as { data: Array<{ yAxis: number }> }
    expect(markLine.data[0].yAxis).toBe(0)
  })

  it('écrit la couverture en UNE LIGNE de pied, et rien d’autre sous le graphe', () => {
    renderWithProviders(<SquadRangeRolesCard bloc={bloc([profil(0)])} roster={roster} />)
    expect(screen.getByTestId('squad-portee-couverture').textContent).toContain('412')
    expect(screen.getByTestId('squad-portee-couverture').textContent).toContain('544')
  })

  it('la bande des rôles laisse les quatre premiers matchs SANS RÔLE', () => {
    const profils = Array.from({ length: 6 }, (_, i) => profil(i))
    renderWithProviders(<SquadRangeRolesCard bloc={bloc(profils)} roster={roster} />)
    fireEvent.click(screen.getByText('Bande des rôles'))
    const jetons = screen.getAllByTestId('squad-portee-jeton')
    // 2 joueurs × 6 matchs, et les 4 premiers de chaque ligne sans rôle.
    expect(jetons).toHaveLength(12)
    expect(jetons.slice(0, 4).map((j) => j.getAttribute('data-role'))).toEqual([
      'aucun',
      'aucun',
      'aucun',
      'aucun',
    ])
    expect(jetons[4].getAttribute('data-role')).not.toBe('aucun')
  })

  it('rend deux textes distincts en FR et en EN', () => {
    const { unmount } = renderWithProviders(
      <SquadRangeRolesCard bloc={bloc([])} roster={roster} />,
    )
    expect(screen.getByText('Aucune portée mesurée')).toBeTruthy()
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadRangeRolesCard bloc={bloc([])} roster={roster} />)
    expect(screen.getByText('No range measured')).toBeTruthy()
  })
})

/**
 * La même carte en grandeur HAUTEUR (« Rôles de hauteur », E1 / D24 du 2026-09-22).
 *
 * Ce que ces tests cadenassent : c'est bien le dénivelé qui est projeté (pas la distance) ;
 * un bloc servi SANS dénivelé rend l'état vide nommé de la hauteur, jamais celui de la
 * portée ni un nuage à zéro point ; les bandes se nomment Contrebas / À niveau / Hauteurs ;
 * la ligne du lobby reste à zéro ; et chaque courbe de tendance porte le gamertag à son
 * bout (à quatre joueurs, la légende seule ne suffit plus).
 */
describe('SquadRangeRolesCard — grandeur hauteur', () => {
  function profilDz(i: number, dz: [number, number]): MatchRangeProfile {
    const p = profil(i)
    return {
      ...p,
      lobby_elevation_median_m: 0.2,
      players: [
        { ...p.players![0], elevation_median_m: dz[0] + 0.2, elevation_lobby_delta_m: dz[0] },
        { ...p.players![1], elevation_median_m: dz[1] + 0.2, elevation_lobby_delta_m: dz[1] },
      ],
    }
  }

  it('rend l’ÉTAT VIDE NOMMÉ de la hauteur quand aucun dénivelé n’est servi', () => {
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc([profil(0)])} roster={roster} grandeur="hauteur" />,
    )
    expect(screen.getByText('Aucune hauteur mesurée')).toBeTruthy()
    expect(screen.queryByText('Aucune portée mesurée')).toBeNull()
  })

  it('projette le DÉNIVELÉ en ordonnée, pas la distance', async () => {
    const profils = Array.from({ length: 5 }, (_, i) => profilDz(i, [2.4, -1.6]))
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc(profils)} roster={roster} grandeur="hauteur" />,
    )
    const series = (await option()).series as Array<Record<string, unknown>>
    const nuage = series.filter((s) => s.type === 'scatter')
    const premier = (nuage[0].data as Array<{ value: [number, number] }>)[0]
    expect(premier.value[1]).toBeCloseTo(2.4, 6)
    const second = (nuage[1].data as Array<{ value: [number, number] }>)[0]
    expect(second.value[1]).toBeCloseTo(-1.6, 6)
  })

  it('nomme les bandes Contrebas / À niveau / Hauteurs et garde le lobby à zéro', async () => {
    const profils = [0, 1, 2, 3, 4].map((i) => profilDz(i, [i - 2, 2 - i]))
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc(profils)} roster={roster} grandeur="hauteur" />,
    )
    const series = (await option()).series as Array<Record<string, unknown>>
    const markArea = series[0].markArea as { data: Array<Array<{ name?: string }>> }
    expect(markArea.data.map((b) => b[0].name)).toEqual(['Contrebas', 'À niveau', 'Hauteurs'])
    const markLine = series[0].markLine as { data: Array<{ yAxis: number }> }
    expect(markLine.data[0].yAxis).toBe(0)
  })

  it('écrit le gamertag AU BOUT de chaque courbe de tendance', async () => {
    const profils = Array.from({ length: 5 }, (_, i) => profilDz(i, [1, -1]))
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc(profils)} roster={roster} grandeur="hauteur" />,
    )
    const series = (await option()).series as Array<Record<string, unknown>>
    const lignes = series.filter((s) => s.type === 'line')
    expect(lignes.map((s) => (s.endLabel as { formatter: string }).formatter)).toEqual([
      'JGtm',
      'Kaya',
    ])
  })

  it('laisse la carte de PORTÉE intacte : ni étiquette de bout, ni point effacé', async () => {
    const profils = Array.from({ length: 5 }, (_, i) => profilDz(i, [1, -1]))
    renderWithProviders(<SquadRangeRolesCard bloc={bloc(profils)} roster={roster} />)
    const series = (await option()).series as Array<Record<string, unknown>>
    expect(series.filter((s) => s.type === 'line').every((s) => s.endLabel === undefined)).toBe(
      true,
    )
    const points = series[0].data as Array<{ itemStyle: { opacity?: number } }>
    expect(points[0].itemStyle.opacity).toBe(1)
  })
})
