/**
 * Tests composant — SettingsPage (Slice 1).
 *
 * Vérifie le chargement, le titre et l'affichage des toggles.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { SettingsPage } from './SettingsPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('SettingsPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SettingsPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner en phase de chargement', () => {
    renderWithProviders(<SettingsPage />)
    expect(screen.getByText(/Chargement des paramètres/i)).toBeInTheDocument()
  })

  it("affiche le titre 'Paramètres' une fois les données chargées", async () => {
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('Paramètres')).toBeInTheDocument()
    })
  })

  it('affiche la section Langue et affichage', async () => {
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/Langue/i)).toBeInTheDocument()
    })
  })
})
