/**
 * La mini-tuile « Rejeu » et la couche « autre joueur » : la distance à l'autre joueur s'écrit dans
 * la tuile (absente quand un point manque), la tuile du point survolé ressort, et survoler une tuile
 * prévient la vue de son index.
 */
import type React from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import type { CelluleTactique, TacticalContribution } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalZoneCard } from './TacticalZoneCard'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, Link: ({ children }: { children?: React.ReactNode }) => <a href="#rejeu">{children}</a> }
})
vi.mock('@/lib/i18n/fieldMappings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/i18n/fieldMappings')>()
  return { ...actual, useOutcomeMapping: () => undefined }
})
vi.mock('@/lib/title-routing', () => ({ useTitleSlug: () => 'halo_infinite' }))

const CELLULE: CelluleTactique = {
  col: -7,
  lig: 2,
  valeur: 3,
  brut: 3,
  matchs: 4,
  matchs_victoire: 2,
  matchs_defaite: 2,
  frags: 3,
  morts: 1,
  centre_x: -13,
  centre_y: 5,
}

function contribution(extra: Partial<TacticalContribution>): TacticalContribution {
  return {
    match_id: 'm1',
    instant_ms: 83_400,
    clock: 'match',
    xuid: '1',
    match_started_at: '2026-09-01T22:30:00Z',
    replay_available: false,
    face: 'mort',
    autre_gamertag: 'Rival',
    ...extra,
  }
}

const AVEC_POINTS = contribution({ point: { x: 0, y: 0 }, autre_point: { x: 23.4, y: 0 } })
const SANS_POINT = contribution({ match_id: 'm2', point: { x: 0, y: 0 } })

function rendre(locale: 'fr' | 'en', survol: number | null, onSurvol = vi.fn(), choisie: number | null = null, onChoisir = vi.fn()) {
  renderWithProviders(
    <TacticalZoneCard
      t={getTacticalText(locale)}
      locale={locale}
      playerSlug="JGtm"
      question="morts"
      signee={false}
      pasM={2}
      cellule={CELLULE}
      detail={{ data: { contributions: [AVEC_POINTS, SANS_POINT] }, isPending: false, isError: false }}
      survol={survol}
      onSurvol={onSurvol}
      choisie={choisie}
      onChoisir={onChoisir}
    />,
  )
  return onSurvol
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr', userTimezone: 'UTC', currentTitleSlug: 'halo_infinite' } as never)
})

describe('TacticalRejeuTile — la couche « autre joueur »', () => {
  it('la distance s’écrit « à 23 m » (FR) ou « 23 m away » (EN), seulement avec les deux points', () => {
    rendre('fr', null)
    const distances = screen.getAllByTestId('tactical-rejeu-distance')
    expect(distances).toHaveLength(1)
    expect(distances[0]).toHaveTextContent('à 23 m')
  })

  it('en anglais', () => {
    rendre('en', null)
    expect(screen.getByTestId('tactical-rejeu-distance')).toHaveTextContent('23 m away')
  })

  it('la tuile du point survolé ressort, les autres non', () => {
    rendre('fr', 1)
    const tuiles = screen.getAllByTestId('tactical-rejeu-tile')
    expect(tuiles[0]).not.toHaveAttribute('data-actif')
    expect(tuiles[1]).toHaveAttribute('data-actif', 'true')
  })

  it('survoler une tuile prévient la vue de son index, la quitter la remet à zéro', () => {
    const onSurvol = rendre('fr', null)
    const tuiles = screen.getAllByTestId('tactical-rejeu-tile')
    fireEvent.pointerEnter(tuiles[1]!)
    expect(onSurvol).toHaveBeenLastCalledWith(1)
    fireEvent.pointerLeave(tuiles[1]!)
    expect(onSurvol).toHaveBeenLastCalledWith(null)
  })
})

describe('TacticalRejeuTile — la sélection', () => {
  it('une tuile est une option focalisable, non choisie par défaut', () => {
    rendre('fr', null)
    expect(screen.getByRole('listbox')).toHaveAccessibleName('Événements de la zone, à sélectionner')
    const tuiles = screen.getAllByRole('option')
    expect(tuiles).toHaveLength(2)
    expect(tuiles[0]).toHaveAttribute('tabindex', '0')
    expect(tuiles[0]).toHaveAttribute('aria-selected', 'false')
  })

  it('chaque tuile porte au repos un chevron et l’apparence d’une carte (bordure, fond)', () => {
    rendre('fr', null)
    const tuiles = screen.getAllByRole('option')
    expect(screen.getAllByTestId('tactical-rejeu-chevron')).toHaveLength(2)
    expect(tuiles[0]?.className).toContain('border-border')
    expect(tuiles[0]?.className).toContain('bg-card')
    expect(tuiles[0]?.className).toContain('rounded-md')
    expect(tuiles[0]?.className).toContain('cursor-pointer')
  })

  it('la tuile choisie est marquée aria-selected et reste en relief sans survol', () => {
    rendre('fr', null, vi.fn(), 1)
    const tuiles = screen.getAllByRole('option')
    expect(tuiles[1]).toHaveAttribute('aria-selected', 'true')
    expect(tuiles[1]).toHaveAttribute('data-actif', 'true')
    expect(tuiles[0]).toHaveAttribute('aria-selected', 'false')
  })

  it('clic, puis Entrée et Espace : la vue est prévenue de l’index', () => {
    const onChoisir = vi.fn()
    rendre('fr', null, vi.fn(), null, onChoisir)
    const tuiles = screen.getAllByRole('option')
    fireEvent.click(tuiles[1]!)
    expect(onChoisir).toHaveBeenLastCalledWith(1)
    expect(onChoisir).toHaveBeenCalledTimes(1)
    fireEvent.keyDown(tuiles[0]!, { key: 'Enter' })
    fireEvent.keyDown(tuiles[0]!, { key: ' ' })
    expect(onChoisir).toHaveBeenCalledTimes(3)
    fireEvent.keyDown(tuiles[0]!, { key: 'a' })
    expect(onChoisir).toHaveBeenCalledTimes(3)
  })

  it('le lien de rejeu garde son propre clic : il ne choisit pas la tuile', () => {
    const onChoisir = vi.fn()
    useAppShellStore.setState({ locale: 'fr', userTimezone: 'UTC', currentTitleSlug: 'halo_infinite' } as never)
    renderWithProviders(
      <TacticalZoneCard
        t={getTacticalText('fr')}
        locale="fr"
        playerSlug="JGtm"
        question="morts"
        signee={false}
        pasM={2}
        cellule={CELLULE}
        detail={{ data: { contributions: [{ ...AVEC_POINTS, replay_available: true }] }, isPending: false, isError: false }}
        onChoisir={onChoisir}
      />,
    )
    fireEvent.click(screen.getByRole('link'))
    expect(onChoisir).not.toHaveBeenCalled()
  })
})
