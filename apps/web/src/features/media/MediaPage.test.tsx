/**
 * Tests composant — MediaPage (Slice 8 — Médias).
 *
 * Smoke : monte, spinner, puis galerie vide depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MediaPage } from './MediaPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('MediaPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<MediaPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner pendant le chargement', () => {
    const { container } = renderWithProviders(<MediaPage />)
    // MediaPage utilise <Spinner size="lg" /> sans label — vérifier la présence du SVG
    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('affiche le titre Médias après chargement', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Médias')).toBeInTheDocument()
    })
  })

  it('affiche les filtres de type de média', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Tous types')).toBeInTheDocument()
      expect(screen.getByText('Screenshots')).toBeInTheDocument()
    })
  })
})
