import { describe, it, expect, beforeEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { LabPage } from './LabPage'

const ENABLED_CAPABILITIES = {
  can_read_local_data: true,
  can_run_sync: true,
  can_use_live_halo: true,
  can_manage_settings: true,
  can_reset_media_index: true,
  can_view_media: true,
  can_self_provision: true,
  can_start_initial_sync: true,
  can_manage_instance: true,
}

beforeEach(() => {
  useAppShellStore.setState({
    currentPlayer: null,
    availablePlayers: [],
    currentTitleSlug: 'halo_infinite',
    availableTitles: [],
    isTitleSwitching: false,
    locale: 'fr',
    hintsVisible: true,
    capabilities: ENABLED_CAPABILITIES,
    setupRequired: false,
    authState: 'ready',
    setupState: 'ready',
    isBootstrapped: true,
    linkedHaloIdentity: null,
    activeSyncJobId: null,
  })
})

describe('LabPage', () => {
  it('affiche un message d’accès refusé sans capacité can_manage_instance', () => {
    useAppShellStore.setState({ capabilities: { ...ENABLED_CAPABILITIES, can_manage_instance: false } })

    renderWithProviders(<LabPage />)

    expect(screen.getByText(/Accès refusé/i)).toBeInTheDocument()
  })

  it('charge les trois panneaux du Lab en français', async () => {
    renderWithProviders(<LabPage />)

    expect(screen.getByText(/Comment utiliser ce Lab/i)).toBeInTheDocument()
    expect(screen.getByText(/Outil sélectionné/i)).toBeInTheDocument()
    expect(screen.getAllByText(/Explorateur interne/i)).toHaveLength(2)

    await waitFor(() => {
      expect(screen.getByText(/Ce que fait l'outil/i)).toBeInTheDocument()
      expect(screen.getByText(/Intérêt/i)).toBeInTheDocument()
      expect(screen.getByText(/Capacités/i)).toBeInTheDocument()
      expect(screen.getByText('season_calendar')).toBeInTheDocument()
      expect(screen.getByText('Aquarius')).toBeInTheDocument()
      expect(screen.getByText('DoubleKill')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('tab', { name: 'Contrats API' }))

    await waitFor(() => {
      expect(screen.getAllByText(/Diff de contrats API/i)).toHaveLength(2)
      expect(screen.getByText(/Routes supplémentaires côté Go/i)).toBeInTheDocument()
      expect(screen.getByText('/lab/resources')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('tab', { name: 'Diagnostics' }))

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Diagnostics d'instance/i })).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: /Rapport de parité/i })).toBeInTheDocument()
      expect(screen.getByText('health')).toBeInTheDocument()
    })
  })

  it('affiche le copy du Lab en anglais quand locale=en', async () => {
    useAppShellStore.setState({ locale: 'en' })

    renderWithProviders(<LabPage />)

    expect(screen.getByText(/How to use this Lab/i)).toBeInTheDocument()
    expect(screen.getByText(/Selected tool/i)).toBeInTheDocument()
    expect(screen.getAllByText(/Internal Explorer/i)).toHaveLength(2)

    await waitFor(() => {
      expect(screen.getByText(/What it does/i)).toBeInTheDocument()
      expect(screen.getByText(/Why it matters/i)).toBeInTheDocument()
      expect(screen.getByText(/Capabilities/i)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('tab', { name: 'API Contracts' }))

    await waitFor(() => {
      expect(screen.getAllByText(/API Contract Diff/i)).toHaveLength(2)
      expect(screen.getByText(/Extra Go routes/i)).toBeInTheDocument()
    })
  })
})
