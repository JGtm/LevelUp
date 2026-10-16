/**
 * Tests composant — PoolSharingCard (réglage de mise en commun du jeton).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { api } from '@/lib/api/client'
import { PoolSharingCard } from './PoolSharingCard'

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

describe('PoolSharingCard', () => {
  afterEach(() => {
    useAppShellStore.setState({ poolSharing: 'unavailable', locale: 'fr' })
    vi.restoreAllMocks()
  })

  it('ne rend rien sans jeton à mettre en commun', () => {
    useAppShellStore.setState({ poolSharing: 'unavailable' })
    const { container } = renderWithProviders(<PoolSharingCard />)
    expect(container).toBeEmptyDOMElement()
  })

  it.each([
    ['granted', /sert à synchroniser les parties des autres joueurs/i],
    ['declined', /plus lente/i],
    ['undecided', /pas encore choisi/i],
  ] as const)('décrit l’état %s', (state, text) => {
    useAppShellStore.setState({ poolSharing: state })
    renderWithProviders(<PoolSharingCard />)
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('un clic sur le réglage accorde puis retire la mise en commun', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValueOnce({ pool_sharing: 'granted' })
      .mockResolvedValueOnce({ pool_sharing: 'declined' })
    useAppShellStore.setState({ poolSharing: 'declined' })
    renderWithProviders(<PoolSharingCard />)

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(useAppShellStore.getState().poolSharing).toBe('granted'))
    expect(put).toHaveBeenLastCalledWith('/me/pool-sharing', { granted: true })

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(useAppShellStore.getState().poolSharing).toBe('declined'))
    expect(put).toHaveBeenLastCalledWith('/me/pool-sharing', { granted: false })
  })
})
