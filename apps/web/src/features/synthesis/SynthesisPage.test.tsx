/**
 * Tests composant — SynthesisPage (Slice 7 — Synthèse).
 *
 * Smoke : monte, spinner, puis tableau de comparaison depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { SynthesisPage } from './SynthesisPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('SynthesisPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SynthesisPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner pendant le chargement', () => {
    const { container } = renderWithProviders(<SynthesisPage />)
    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('affiche le titre Synthese après chargement', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByText('Synthèse')).toBeInTheDocument()
    })
  })

  it('affiche les selecteurs de période', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByText('Tout')).toBeInTheDocument()
    })
  })

  it('affiche des placeholders explicites pour les sections vides', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByText(/Activité indisponible/i)).toBeInTheDocument()
      expect(screen.getByText(/Aucune semaine remarquable/i)).toBeInTheDocument()
    })
  })
})
