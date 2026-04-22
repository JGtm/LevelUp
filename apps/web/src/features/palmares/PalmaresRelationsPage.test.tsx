import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import { PalmaresRelationsPage } from './PalmaresRelationsPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children, to }: { children?: ReactNode; to?: string }) => <a href={to}>{children}</a>,
  }
})

describe('PalmaresRelationsPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<PalmaresRelationsPage />)
    expect(container).toBeTruthy()
  })

  it('affiche les sections relationnelles après chargement', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-overview')).toBeInTheDocument()
    })

    expect(screen.getAllByText(/Alliés fréquents/i).length).toBeGreaterThan(0)
    expect(screen.getByRole('heading', { name: /Meilleures synergies/i })).toBeInTheDocument()
    expect(screen.getAllByText('DuoAlpha').length).toBeGreaterThan(0)
    expect(screen.getAllByText('NemesisBravo').length).toBeGreaterThan(0)
    expect(screen.getAllByText('QueueGhost').length).toBeGreaterThan(0)
  })
})
