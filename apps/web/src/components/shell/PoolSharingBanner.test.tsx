/**
 * Tests composant — PoolSharingBanner (consentement à la mise en commun du jeton).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { api } from '@/lib/api/client'
import { PoolSharingBanner } from './PoolSharingBanner'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => <a href={to}>{children}</a>,
  }
})
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

describe('PoolSharingBanner', () => {
  afterEach(() => {
    useAppShellStore.setState({ poolSharing: 'unavailable', demoMode: false, locale: 'fr' })
    vi.restoreAllMocks()
  })

  it.each(['unavailable', 'granted', 'declined'] as const)(
    'ne rend rien quand pool_sharing vaut %s',
    (state) => {
      useAppShellStore.setState({ poolSharing: state })
      const { container } = renderWithProviders(<PoolSharingBanner />)
      expect(container).toBeEmptyDOMElement()
    },
  )

  it('ne rend rien en démo même sans décision', () => {
    useAppShellStore.setState({ poolSharing: 'undecided', demoMode: true })
    const { container } = renderWithProviders(<PoolSharingBanner />)
    expect(container).toBeEmptyDOMElement()
  })

  it('pose la question avec deux choix et un lien vers la confidentialité', () => {
    useAppShellStore.setState({ poolSharing: 'undecided' })
    renderWithProviders(<PoolSharingBanner />)
    expect(screen.getByRole('region', { name: /Mise en commun des jetons/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Mettre mon jeton à disposition/i })).toBeEnabled()
    expect(screen.getByRole('button', { name: /Garder mon jeton pour moi/i })).toBeEnabled()
    expect(screen.getByRole('link', { name: /En savoir plus/i })).toHaveAttribute('href', '/privacy')
  })

  it('enregistre un accord et reflète la décision dans le shell', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({ pool_sharing: 'granted' })
    useAppShellStore.setState({ poolSharing: 'undecided' })
    renderWithProviders(<PoolSharingBanner />)

    fireEvent.click(screen.getByRole('button', { name: /Mettre mon jeton à disposition/i }))

    await waitFor(() => expect(useAppShellStore.getState().poolSharing).toBe('granted'))
    expect(put).toHaveBeenCalledWith('/me/pool-sharing', { granted: true })
  })

  it('enregistre un refus', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({ pool_sharing: 'declined' })
    useAppShellStore.setState({ poolSharing: 'undecided' })
    renderWithProviders(<PoolSharingBanner />)

    fireEvent.click(screen.getByRole('button', { name: /Garder mon jeton pour moi/i }))

    await waitFor(() => expect(useAppShellStore.getState().poolSharing).toBe('declined'))
    expect(put).toHaveBeenCalledWith('/me/pool-sharing', { granted: false })
  })

  it('reste sans décision quand l’enregistrement échoue', async () => {
    const put = vi.spyOn(api, 'put').mockRejectedValue(new Error('boom'))
    useAppShellStore.setState({ poolSharing: 'undecided' })
    renderWithProviders(<PoolSharingBanner />)

    fireEvent.click(screen.getByRole('button', { name: /Mettre mon jeton à disposition/i }))

    await waitFor(() => expect(put).toHaveBeenCalled())
    expect(useAppShellStore.getState().poolSharing).toBe('undecided')
  })

  it('rend le texte en anglais quand locale=en', () => {
    useAppShellStore.setState({ poolSharing: 'undecided', locale: 'en' })
    renderWithProviders(<PoolSharingBanner />)
    expect(screen.getByRole('button', { name: /Make my token available/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Keep my token to myself/i })).toBeInTheDocument()
  })
})
