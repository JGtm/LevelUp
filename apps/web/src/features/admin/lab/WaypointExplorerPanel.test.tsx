/**
 * Tests composant — WaypointExplorerPanel (Lab · Explorateur d'API).
 *
 * Vérifie le rendu du formulaire et l'affichage du résultat résolu après une
 * requête (MSW mocke GET /lab/waypoint → asset_name "Live Fire").
 */
import { beforeEach, describe, it, expect } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { WaypointExplorerPanel } from './WaypointExplorerPanel'

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('WaypointExplorerPanel', () => {
  it('rend le formulaire et le bouton Interroger', () => {
    renderWithProviders(<WaypointExplorerPanel />)
    // Le select segment (combobox) + le bouton d'action.
    expect(screen.getByRole('combobox')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Interroger/i })).toBeInTheDocument()
  })

  it('désactive le bouton tant que asset_id / version_id sont vides', () => {
    renderWithProviders(<WaypointExplorerPanel />)
    expect(screen.getByRole('button', { name: /Interroger/i })).toBeDisabled()
  })

  it('affiche le résultat résolu après une requête', async () => {
    renderWithProviders(<WaypointExplorerPanel />)
    // textbox[0] = asset_id, [1] = version_id, [2] = langue (préremplie).
    const inputs = screen.getAllByRole('textbox')
    fireEvent.change(inputs[0], { target: { value: 'demo-asset' } })
    fireEvent.change(inputs[1], { target: { value: '1' } })
    fireEvent.click(screen.getByRole('button', { name: /Interroger/i }))
    await waitFor(() => {
      expect(screen.getByText('Live Fire')).toBeInTheDocument()
    })
  })
})
