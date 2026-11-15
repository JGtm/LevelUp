/**
 * La carte « Rôles de portée » (ruban) — ce qu'elle montre et ce qu'elle tait.
 *
 * Ce que ces tests cadenassent : un bloc sans profil rend un ÉTAT VIDE NOMMÉ (film requis),
 * jamais un ruban à zéro ligne ; l'axe X porte les étiquettes « #N · carte » du plus ancien au
 * plus récent ; une ligne par joueur dans l'ordre du roster, ses noms à sa couleur ; trois
 * séries de rôle (une par teinte) et une série de cases vides ; les quatre premiers matchs sont
 * sans rôle ; rien sous le graphe (pas de légende ajoutée, pas de bande repliable).
 */
import { describe, expect, it, beforeEach, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchRangeBlock, MatchRangeProfile } from '@/lib/api/types'

import { SquadRangeRolesCard } from './SquadRangeRolesCard'

const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="portee-ruban-stub" />
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
const COULEURS = { JGtm: '#111111', Kaya: '#222222' }

type Option = {
  xAxis: { data: string[] }
  yAxis: { data: string[]; inverse: boolean }
  series: Array<{ type: string; name: string; data: Array<{ value: [number, number, number]; tip: string }> }>
}

/** Le graphe est chargé en `lazy` : attendre son stub avant de lire l'option. */
async function option(): Promise<Option> {
  await screen.findByTestId('portee-ruban-stub')
  return captured[captured.length - 1].option as Option
}

describe('SquadRangeRolesCard', () => {
  it('rend un ÉTAT VIDE NOMMÉ quand aucun profil n’est servi', () => {
    renderWithProviders(<SquadRangeRolesCard bloc={bloc([])} roster={roster} colorByPlayer={COULEURS} />)
    expect(screen.getByText('Aucune distance de frag')).toBeTruthy()
    expect(screen.queryByTestId('portee-ruban-stub')).toBeNull()
  })

  it('porte les étiquettes « #N · carte » du plus ancien au plus récent', async () => {
    renderWithProviders(
      <SquadRangeRolesCard
        bloc={bloc([profil(1, 12, 'Streets'), profil(0, 12, 'Fragmentation'), ...[2, 3, 4, 5].map((i) => profil(i))])}
        roster={roster}
        colorByPlayer={COULEURS}
      />,
    )
    expect((await option()).xAxis.data).toEqual(['#1 · Fragment…', '#2 · Streets', '#3', '#4', '#5', '#6'])
  })

  it('une ligne par joueur dans l’ordre du roster, le premier en haut', async () => {
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc(Array.from({ length: 6 }, (_, i) => profil(i)))} roster={roster} colorByPlayer={COULEURS} />,
    )
    const o = await option()
    expect(o.yAxis.data).toEqual(['JGtm', 'Kaya'])
    expect(o.yAxis.inverse).toBe(true)
  })

  it('trois séries de rôle (une teinte chacune) et une série de cases sans rôle', async () => {
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc(Array.from({ length: 6 }, (_, i) => profil(i)))} roster={roster} colorByPlayer={COULEURS} />,
    )
    const o = await option()
    expect(o.series.map((s) => s.name)).toEqual(['Ligne de front', 'Polyvalent', "Tireur d'élite", ''])
    expect(o.series.every((s) => s.type === 'custom')).toBe(true)
  })

  it('les quatre premiers matchs sont SANS RÔLE (cases vides), le suivant porte un rôle', async () => {
    const profils = Array.from({ length: 6 }, (_, i) => profil(i))
    renderWithProviders(<SquadRangeRolesCard bloc={bloc(profils)} roster={roster} colorByPlayer={COULEURS} />)
    const o = await option()
    const vides = o.series[3].data
    const avecRole = o.series.slice(0, 3).flatMap((s) => s.data)
    // 2 joueurs x 6 matchs : 8 cases vides (matchs 0 à 3), 4 cases avec rôle (matchs 4 et 5).
    expect(vides).toHaveLength(8)
    expect(avecRole).toHaveLength(4)
    expect(vides.every((d) => d.value[0] <= 3)).toBe(true)
    expect(avecRole.every((d) => d.value[0] >= 4)).toBe(true)
  })

  it('un match sous le plancher : case vide, le petit échantillon se dit dans l’infobulle seulement', async () => {
    const profils = Array.from({ length: 6 }, (_, i) => profil(i, i === 5 ? 3 : 12))
    renderWithProviders(<SquadRangeRolesCard bloc={bloc(profils)} roster={roster} colorByPlayer={COULEURS} />)
    const o = await option()
    const derniere = o.series[3].data.filter((d) => d.value[0] === 5)
    expect(derniere).toHaveLength(2)
    expect(derniere[0].tip).toContain('3 frags, données insuffisantes')
    // Aucune case avec rôle ne porte la mention : seul un match sous le plancher la porte.
    expect(o.series.slice(0, 3).flatMap((s) => s.data).some((d) => d.tip.includes('données insuffisantes'))).toBe(false)
  })

  it('quatre matchs ou moins : aucun rôle, donc l’état vide du rôle, jamais une grille de cases transparentes', () => {
    const profils = Array.from({ length: 4 }, (_, i) => profil(i))
    renderWithProviders(<SquadRangeRolesCard bloc={bloc(profils)} roster={roster} colorByPlayer={COULEURS} />)
    expect(screen.getByText('Pas encore de rôle')).toBeTruthy()
    expect(screen.getByText(/fenêtre de 5 matchs, le dernier comptant au moins 5 frags/)).toBeTruthy()
    expect(screen.queryByTestId('portee-ruban-stub')).toBeNull()
  })

  it('cinq matchs, le dernier assez fourni : le ruban s’affiche', async () => {
    renderWithProviders(
      <SquadRangeRolesCard bloc={bloc(Array.from({ length: 5 }, (_, i) => profil(i)))} roster={roster} colorByPlayer={COULEURS} />,
    )
    expect((await option()).series).toHaveLength(4)
    expect(screen.queryByText('Pas encore de rôle')).toBeNull()
  })

  it('rien sous le graphe : ni légende ajoutée, ni bande repliable, ni compte de frags mesurés', () => {
    renderWithProviders(<SquadRangeRolesCard bloc={bloc([profil(0)])} roster={roster} colorByPlayer={COULEURS} />)
    expect(screen.queryByTestId('squad-portee-couverture')).toBeNull()
    expect(screen.queryByTestId('squad-portee-legende')).toBeNull()
    expect(screen.queryByTestId('squad-portee-fold-bande')).toBeNull()
    expect(screen.queryByText(/frags mesurés/)).toBeNull()
  })

  it('rend deux textes distincts en FR et en EN', () => {
    const { unmount } = renderWithProviders(
      <SquadRangeRolesCard bloc={bloc([])} roster={roster} colorByPlayer={COULEURS} />,
    )
    expect(screen.getByText('Aucune distance de frag')).toBeTruthy()
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadRangeRolesCard bloc={bloc([])} roster={roster} colorByPlayer={COULEURS} />)
    expect(screen.getByText('No kill distance')).toBeTruthy()
  })
})
