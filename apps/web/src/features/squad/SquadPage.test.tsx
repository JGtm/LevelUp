/**
 * Tests composant — SquadPage (Slice 6 — Escouade).
 *
 * Smoke : monte, spinner, puis vue coéquipiers depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { SquadPage } from './SquadPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('SquadPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SquadPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner pendant le chargement', () => {
    const { container } = renderWithProviders(<SquadPage />)
    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('affiche le titre Escouade après chargement', async () => {
    renderWithProviders(<SquadPage />)
    await waitFor(() => {
      expect(screen.getByText('Escouade')).toBeInTheDocument()
    })
  })

  it('affiche 0 coéquipiers pour le fixture vide', async () => {
    renderWithProviders(<SquadPage />)
    await waitFor(() => {
      expect(screen.getByText(/Aucun coequipier/i)).toBeInTheDocument()
    })
  })
})
