/**
 * SquadEmpriseVehicles.test.tsx — la ressource « véhicules » dans l'onglet Emprise (lot L7.4 du plan
 * PLAN_EMPRISE_VEHICULES_2026-09-28) : une ligne dans chaque carte qui porte des ressources, masquée
 * quand le bloc n'en publie pas, pastilles pleines (D3), case vide sans texte pour un match non lu
 * (D8), frags de tout le lobby contre rendement sur les frags appariés (D9), anglais (S12).
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
    expect(piste.textContent).toContain('Prises de véhicules')
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
    // La tourelle fixe, prise par le seul reste du camp (joueurs inconnus), n'a pas de ligne.
    expect(screen.queryByTestId(`emprise-sheet-line-${XUID.jgtm}-tourelle_fixe`)).toBeNull()
    expect(within(screen.getByTestId('emprise-sheets')).queryByText('Tourelle fixe')).toBeNull()
    for (const dot of screen.getAllByTestId('emprise-sheets')[0].querySelectorAll('[data-dot="lost"]')) {
      // Seuls les bonus perdus (Madina97294, Camouflage) ont une pastille vide.
      expect(dot.closest('[data-testid^="emprise-sheet-line-"]')?.getAttribute('data-testid')).toContain('powerup')
    }
  })
})

describe('Véhicules — match par match (D8)', () => {
  it('case vide sans texte à Shogun (non lu), « — » à Curfew (zéro) ; 3–1 à Starboard ; valeur à Detachment sans film', () => {
    mount()
    const grid = within(screen.getByTestId('emprise-grid'))
    expect(grid.queryByText(/non mesuré|sans film/)).toBeNull()
    const cells = Array.from(screen.getByTestId('emprise-grid-table').querySelectorAll('[data-cell="blank"]'))
    expect(cells.length).toBeGreaterThanOrEqual(5)
    expect(cells.every((c) => c.textContent === '')).toBe(true)
    expect(grid.getAllByText('3–1').length).toBeGreaterThan(0)
    expect(grid.getByText('Warthog')).toBeInTheDocument()
    expect(grid.getByText('Tourelle fixe')).toBeInTheDocument()
    // Detachment : 2–2 sur la ligne des véhicules alors que les lignes du film y restent vides.
    expect(grid.getAllByText('2–2').length).toBeGreaterThan(0)
  })
})

describe('Véhicules — frags et rendement (D5, D9)', () => {
  it('Frags obtenus : tous les frags (14 · 60,9 %), temps à bord 3 min 30 · 67,7 %', () => {
    mount()
    const row = within(screen.getByTestId('emprise-production')).getByTestId('piste-camps-row-vehicle')
    expect(row.textContent).toContain('Frags depuis un véhicule')
    expect(row.textContent).toContain('14 · 60,9 %')
    expect(row.textContent).toContain('39,1 % · 9')
    expect(text('emprise-production-exposure-vehicle')).toBe('temps à bord : 3 min 30 · 67,7 %1 min 40')
  })

  it('Rendement : +27 % (2,3 contre 1,8) sur les frags appariés, sans note de couverture', () => {
    mount()
    expect(text('emprise-yield-gap-vehicle')).toBe('+27 %')
    expect(text('emprise-yield-raw-vehicle')).toBe('2,3 contre 1,8')
    expect(text('emprise-yield')).toContain('frags par minute à bord')
    expect(text('emprise-yield')).not.toMatch(/sur 23|robots/)
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
    const failed: SquadEmpriseBlock = { ...EMPRISE_2209 }
    mount(failed)
    expect(screen.queryByTestId('piste-camps-row-vehicle')).toBeNull()
  })
})

describe('Véhicules — anglais (S12)', () => {
  it('libellés en anglais : Vehicles, time aboard, kills per minute aboard', () => {
    useAppShellStore.setState({ locale: 'en' })
    mount()
    const piste = within(screen.getByTestId('emprise-control')).getByTestId('piste-camps-row-vehicle')
    expect(piste.textContent).toContain('Vehicle takes')
    expect(text('emprise-production-exposure-vehicle')).toBe('time aboard: 3 min 30 · 67.7%1 min 40')
    expect(within(screen.getByTestId('emprise-grid')).queryByText(/not measured|no film/)).toBeNull()
    expect(text('emprise-yield')).toContain('kills per minute aboard')
  })
})
