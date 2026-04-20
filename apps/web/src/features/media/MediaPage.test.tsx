/**
 * Tests composant — MediaPage (Slice 8 — Médias).
 *
 * Smoke : monte, spinner, puis galerie vide depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
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

  it('affiche le filtre Clips', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Clips')).toBeInTheDocument()
    })
  })

  it('affiche le sélecteur de tri', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      // Le select de tri doit être présent
      const sortSelect = document.querySelector('select')
      expect(sortSelect).toBeInTheDocument()
    })
  })

  it('affiche la zone upload', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      // La zone d'upload contient une mention "parcourir" ou "glisser"
      const matches = screen.getAllByText(/parcourir|glisser|déposer/i)
      expect(matches.length).toBeGreaterThan(0)
    })
  })

  it('affiche la checkbox Likés seulement', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      // Checkbox likedOnly
      const checkbox = document.querySelector('input[type="checkbox"]')
      expect(checkbox).toBeInTheDocument()
    })
  })

  it('checkbox likedOnly est décochée par défaut', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      const checkbox = document.querySelector('input[type="checkbox"]') as HTMLInputElement
      expect(checkbox).not.toBeNull()
      expect(checkbox.checked).toBe(false)
    })
  })

  it('cliquer sur un filtre de type ne lève pas d\'erreur', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Screenshots')).toBeInTheDocument()
    })
    // Cliquer ne doit pas lever d'exception
    expect(() => fireEvent.click(screen.getByText('Screenshots'))).not.toThrow()
  })
})
