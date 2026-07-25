/**
 * AdminInitialSyncCard — non-régression MULTI-TITRE (contre-revue V72).
 *
 * Constat corrigé : `handleRun` n'envoyait jamais `title_slug` → le job partait
 * TOUJOURS sur le titre par défaut du serveur, même quand l'admin était positionné
 * sur Halo 5 (re-import complet lancé sur le mauvais jeu, silencieusement).
 *
 * Ce test fige les DEUX moitiés du correctif :
 *  1. le titre ACTIF du shell est envoyé dans le corps de la mutation ;
 *  2. il est AFFICHÉ sur la carte (l'admin voit le jeu ciblé avant de confirmer).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TitleSummary } from '@/lib/api/types'

import type { InitialSyncVars } from '../monitoring/mutations'

/** Corps effectivement envoyé à POST /admin/actions/initial-sync/run. */
const submitted: InitialSyncVars[] = []

vi.mock('../monitoring/mutations', () => ({
  useRunInitialSync: () => ({
    mutateAsync: async (vars: InitialSyncVars) => {
      submitted.push(vars)
      return { job_id: 'job-1', job_type: 'initial_sync', status: 'running' }
    },
  }),
  conflictJobId: () => null,
}))

// Le suivi de job n'est pas le sujet : on coupe la query de progression.
vi.mock('../components/JobProgressInline', () => ({
  JobProgressInline: () => null,
}))

// Combobox réduit à un bouton de sélection : le test cible handleRun, pas la
// mécanique de suggestion (couverte par GamertagCombobox.test.tsx).
vi.mock('@/components/ui/GamertagCombobox', () => ({
  GamertagCombobox: ({ onChange }: { onChange: (v: string[]) => void }) => (
    <button type="button" onClick={() => onChange(['choco'])}>
      choisir-joueur
    </button>
  ),
}))

import { AdminInitialSyncCard } from './AdminInitialSyncCard'

const TITLES: TitleSummary[] = [
  { slug: 'halo_infinite', name: 'Halo Infinite' } as TitleSummary,
  { slug: 'halo_5', name: 'Halo 5' } as TitleSummary,
]

describe('AdminInitialSyncCard — titre ciblé (multi-titre)', () => {
  beforeEach(() => {
    submitted.length = 0
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    useAppShellStore.setState({ currentTitleSlug: 'halo_5', availableTitles: TITLES })
  })

  it('envoie le titre ACTIF (halo_5) en title_slug, pas le titre par défaut', async () => {
    renderWithProviders(<AdminInitialSyncCard onJobSettled={() => {}} />)

    fireEvent.click(screen.getByText('choisir-joueur'))
    fireEvent.click(screen.getByText(/Lancer la synchronisation initiale/i))

    await waitFor(() => expect(submitted).toHaveLength(1))
    expect(submitted[0]).toMatchObject({ player_slug: 'choco', title_slug: 'halo_5' })
  })

  it('suit la bascule de titre du shell (halo_infinite)', async () => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
    renderWithProviders(<AdminInitialSyncCard onJobSettled={() => {}} />)

    fireEvent.click(screen.getByText('choisir-joueur'))
    fireEvent.click(screen.getByText(/Lancer la synchronisation initiale/i))

    await waitFor(() => expect(submitted).toHaveLength(1))
    expect(submitted[0].title_slug).toBe('halo_infinite')
  })

  it('AFFICHE le jeu ciblé sur la carte (nom lisible du titre actif)', () => {
    renderWithProviders(<AdminInitialSyncCard onJobSettled={() => {}} />)
    expect(screen.getByText('Jeu ciblé :')).toBeInTheDocument()
    expect(screen.getByText('Halo 5')).toBeInTheDocument()
  })

  it('replie sur le slug quand availableTitles n’est pas encore hydraté', () => {
    useAppShellStore.setState({ availableTitles: [] })
    renderWithProviders(<AdminInitialSyncCard onJobSettled={() => {}} />)
    expect(screen.getByText('halo_5')).toBeInTheDocument()
  })
})
