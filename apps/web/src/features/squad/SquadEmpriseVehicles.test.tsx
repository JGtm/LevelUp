/**
 * SquadEmpriseVehicles.test.tsx — la ressource « véhicules » dans l'onglet Emprise (lot L7.4 du plan
 * PLAN_EMPRISE_VEHICULES_2026-09-28) : une ligne dans chaque carte qui porte des ressources, masquée
 * quand le bloc n'en publie pas, pastilles pleines (D3), « non mesuré » (D8), frags de tout le lobby
 * contre rendement sur les frags appariés (D9) avec sa note de couverture, anglais (S12).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import type { SquadEmpriseBlock, TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { EMPRISE_2209, HISTORY_2209, XUID } from './emprise/emprise.fixtures'
import { VEHICLES_2209 } from './emprise/vehicles.fixtures'
import * as squadContextModule from './SquadContext'
import { SquadEmprisePage } from './SquadEmprisePage'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

vi.mock('@/lib/i18n/fieldMappings', async (importOriginal) => {
  const { useAppShellStore: store } = await import('@/stores/appShellStore')
  const labels: Record<string, Record<string, string>> = {
    fr: { win: 'Victoire', loss: 'Défaite', tie: 'Égalité', dnf: 'Abandon' },
    en: { win: 'Win', loss: 'Loss', tie: 'Tie', dnf: 'DNF' },
  }
  return {
    ...(await importOriginal<typeof import('@/lib/i18n/fieldMappings')>()),
    useOutcomeLabel: (key: string) => labels[store.getState().locale]?.[key] ?? key,
  }
})

const ROW = (gamertag: string): TeammateRow =>
  ({ gamertag, xuid: 'x', encounter_count: 7, with_kpis: { match_count: 7, wins: 3 } }) as unknown as TeammateRow

function mount(block: SquadEmpriseBlock = VEHICLES_2209) {
  const gamertags = ['Chocoboflor', 'Madina97294']
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows: gamertags.map(ROW),
    confirmedGamertags: gamertags,
    pageData: {
      options: [],
      teammates: [],
      total_matches: 7,
      session_labels: { solo: [], squad: [] },
      friends_count: 2,
      main_player: 'JGtm',
      match_history: HISTORY_2209,
      squad_emprise: block,
    } as TeammatesPageResponse,
    playerSlug: 'jgtm',
    currentPlayerXuid: XUID.jgtm,
  })
  return renderWithProviders(<SquadEmprisePage />)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})
afterEach(() => {
  vi.restoreAllMocks()
})

const text = (id: string) => screen.getByTestId(id).textContent ?? ''

describe('Véhicules — bilan et fil', () => {
  it('une piste « Véhicules » : 5 · 62,5 % contre 37,5 % · 3, pastille de la couleur de la ressource', () => {
    mount()
    const piste = within(screen.getByTestId('emprise-control')).getByTestId('piste-camps-row-vehicle')
    expect(piste.textContent).toContain('Véhicules')
    expect(piste.textContent).toContain('prises')
    expect(piste.textContent).toContain('5 · 62,5 %')
    expect(piste.textContent).toContain('37,5 % · 3')
    const dot = piste.querySelector('span[style*="background"]') as HTMLElement
    expect(dot.getAttribute('style')).toContain('resource-vehicle')
  })

  it('le fil nomme la ressource dans sa légende', () => {
    mount()
    expect(within(screen.getByTestId('emprise-fil')).getByTestId('chart-card-legend').textContent).toContain('Véhicules')
  })
})

describe('Véhicules — fiches (D3 : pastilles pleines)', () => {
  it('JGtm : Warthog 2 prises, Véhicule inconnu 1 ; toutes les pastilles pleines ; pied « 3 véhicules »', () => {
    mount()
    const warthog = screen.getByTestId(`emprise-sheet-line-${XUID.jgtm}-warthog`)
    expect(warthog.textContent).toContain('Warthog')
    expect(warthog.querySelectorAll('[data-dot="taken"]')).toHaveLength(2)
    expect(warthog.querySelectorAll('[data-dot="lost"]')).toHaveLength(0)
    expect(screen.getByTestId(`emprise-sheet-line-${XUID.jgtm}-unknown`).textContent).toContain('Véhicule inconnu')
    expect(text(`emprise-sheet-foot-${XUID.jgtm}-vehicle`)).toBe('3 véhicules')
    // Le libellé du titre pour la tourelle fixe, chez le reste du camp.
    expect(screen.getByTestId('emprise-sheet-line-rest-tourelle_fixe').textContent).toContain('Tourelle fixe')
    for (const dot of screen.getAllByTestId('emprise-sheets')[0].querySelectorAll('[data-dot="lost"]')) {
      // Seuls les bonus perdus (Madina97294, Camouflage) ont une pastille vide.
      expect(dot.closest('[data-testid^="emprise-sheet-line-"]')?.getAttribute('data-testid')).toContain('powerup')
    }
  })
})

describe('Véhicules — match par match (D8)', () => {
  it('« non mesuré » hachuré à Shogun, pas à Curfew (zéro mesuré : « — ») ; 3–1 à Starboard ; valeur à Detachment sans film', () => {
    mount()
    const grid = within(screen.getByTestId('emprise-grid'))
    // Shogun : la synthèse et les quatre familles de la soirée, une seule colonne.
    expect(grid.getAllByText('non mesuré')).toHaveLength(5)
    const cells = Array.from(screen.getByTestId('emprise-grid-table').querySelectorAll('[data-cell]'))
    expect(cells.filter((c) => c.getAttribute('data-cell') === 'unmeasured')).toHaveLength(5)
    expect(grid.getAllByText('3–1').length).toBeGreaterThan(0)
    expect(grid.getByText('Warthog')).toBeInTheDocument()
    expect(grid.getByText('Tourelle fixe')).toBeInTheDocument()
    // Detachment : 2–2 sur la ligne des véhicules alors que les lignes du film y disent « sans film ».
    expect(grid.getAllByText('2–2').length).toBeGreaterThan(0)
    expect(grid.getAllByText('sans film').length).toBeGreaterThan(0)
  })
})

describe('Véhicules — frags et rendement (D5, D9)', () => {
  it('Frags obtenus : tous les frags (14 · 60,9 %), temps à bord 3 min 30 · 67,7 %', () => {
    mount()
    const row = within(screen.getByTestId('emprise-production')).getByTestId('piste-camps-row-vehicle')
    expect(row.textContent).toContain('frags depuis un véhicule')
    expect(row.textContent).toContain('14 · 60,9 %')
    expect(row.textContent).toContain('39,1 % · 9')
    expect(text('emprise-production-exposure-vehicle')).toBe('temps à bord : 3 min 30 · 67,7 %1 min 40')
  })

  it('Rendement : +27 % (2,3 contre 1,8) sur les frags appariés, avec la note de couverture', () => {
    mount()
    expect(text('emprise-yield-gap-vehicle')).toBe('+27 %')
    expect(text('emprise-yield-raw-vehicle')).toBe('2,3 contre 1,8')
    expect(text('emprise-yield')).toContain('frags par minute à bord')
    const note = text('emprise-yield-vehicle-note')
    expect(note).toContain('11 frags sur 23 (47,8 %)')
    expect(note).toContain('2 passages sans joueur nommé (robots) ne comptent pas')
  })

  it('zéro passage sans joueur nommé : la note de couverture ne parle pas de robots', () => {
    mount({ ...VEHICLES_2209, vehicles: { ...VEHICLES_2209.vehicles!, episodes_unnamed: 0 } })
    const note = text('emprise-yield-vehicle-note')
    expect(note).toContain('11 frags sur 23 (47,8 %)')
    expect(note).not.toContain('robots')
  })

  it('sans véhicules, pas de note', () => {
    mount(EMPRISE_2209)
    expect(screen.queryByTestId('emprise-yield-vehicle-note')).toBeNull()
  })
})

describe('Véhicules — ligne masquée quand le bloc n’en publie pas', () => {
  it('bloc sans la ressource : aucune ligne « Véhicules » nulle part', () => {
    mount(EMPRISE_2209)
    expect(screen.queryByText('Véhicules')).toBeNull()
    expect(screen.queryByTestId('piste-camps-row-vehicle')).toBeNull()
    expect(screen.queryByTestId('emprise-yield-row-vehicle')).toBeNull()
  })

  it('lecture en échec : la ressource est absente du bloc, aucune ligne ni zéro', () => {
    const failed: SquadEmpriseBlock = {
      ...EMPRISE_2209,
      vehicles: { unavailable: 'load_failed', matches_measured: 0, matches_not_measured: 0, episodes_read: 0, episodes_unnamed: 0, episodes_no_camp: 0, proximity_episodes: 0, frags_matches: 0, frags_total: 0, frags_paired: 0 },
    }
    mount(failed)
    expect(screen.queryByTestId('piste-camps-row-vehicle')).toBeNull()
    expect(screen.queryByTestId('emprise-yield-vehicle-note')).toBeNull()
  })
})

describe('Véhicules — anglais (S12)', () => {
  it('libellés en anglais : Vehicles, time aboard, not measured, note de couverture', () => {
    useAppShellStore.setState({ locale: 'en' })
    mount()
    const piste = within(screen.getByTestId('emprise-control')).getByTestId('piste-camps-row-vehicle')
    expect(piste.textContent).toContain('Vehicles')
    expect(piste.textContent).toContain('takes')
    expect(text('emprise-production-exposure-vehicle')).toBe('time aboard: 3 min 30 · 67.7%1 min 40')
    expect(within(screen.getByTestId('emprise-grid')).getAllByText('not measured')).toHaveLength(5)
    expect(text('emprise-yield-vehicle-note')).toContain('11 of 23 kills (47.8%)')
    expect(text('emprise-yield')).toContain('kills per minute aboard')
  })
})
