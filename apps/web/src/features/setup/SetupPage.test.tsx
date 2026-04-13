/**
 * Tests composant — SetupPage (Slice 1).
 *
 * Smoke test : le composant monte et affiche l'étape initiale.
 * Les mutations API sont mockées via MSW (setup.ts).
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { SetupPage } from './SetupPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('SetupPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SetupPage />)
    expect(container).toBeTruthy()
  })

  it("affiche l'étape de choix du mode auth au départ", async () => {
    renderWithProviders(<SetupPage />)
    await waitFor(() => {
      expect(screen.getByText(/Mode d'authentification/i)).toBeInTheDocument()
    })
  })

  it('affiche le bouton Device Code Flow', async () => {
    renderWithProviders(<SetupPage />)
    await waitFor(() => {
      expect(screen.getByText(/Device Code Flow/i)).toBeInTheDocument()
    })
  })

  it('affiche le bouton Refresh Token', async () => {
    renderWithProviders(<SetupPage />)
    await waitFor(() => {
      // Regex ancrée pour éviter le match sur la description "...refresh token..."
      expect(screen.getByText(/^Refresh Token$/i)).toBeInTheDocument()
    })
  })
})
