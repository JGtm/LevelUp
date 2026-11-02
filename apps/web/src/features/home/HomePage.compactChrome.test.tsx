/**
 * Tests — accueil sous la capacité `can_use_compact_chrome` (application de bureau).
 *
 * L'image de couverture disparaît ; la bannière d'identité et les indicateurs restent EXACTEMENT
 * comme sur le site. Le site (capacité fausse) garde l'image de couverture.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { HomePage } from './HomePage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

function setCapabilities(desktop: boolean) {
  act(() => {
    useAppShellStore.setState({
      capabilities: testCapabilities(desktop ? desktopCapabilities() : {}),
    })
  })
}

describe('HomePage — chrome de bureau', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ capabilities: null })
    })
  })

  it("sans l'image de couverture, avec la bannière d'identité et les indicateurs du site", async () => {
    setCapabilities(true)
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByTestId('home-spartan-identity-banner')).toBeInTheDocument()
    })
    expect(screen.queryByTestId('home-hero-banner')).not.toBeInTheDocument()
    expect(screen.queryByTestId('home-hero-banner-sticky')).not.toBeInTheDocument()
    expect(screen.getByText('Taux de victoire')).toBeInTheDocument()
  })

  it("le site garde l'image de couverture", async () => {
    setCapabilities(false)
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByTestId('home-hero-banner')).toBeInTheDocument()
    })
    expect(screen.getByTestId('home-hero-banner-sticky')).toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-identity-banner')).toBeInTheDocument()
  })
})
