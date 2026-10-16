import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { RecentMediaRail } from './RecentMediaRail'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MediaItemRow, PlayerSummary } from '@/lib/api/types'

// Règle serveur (domain.CanModifyMedia) : réassocier un média est réservé à son propriétaire
// ou à un admin. Le rail de l'accueil mêle les médias de plusieurs auteurs.

const ME = 'Moi'

function row(owner: string): MediaItemRow {
  return {
    basename: `${owner}.mp4`,
    file_path: `/media/${owner}/clip.mp4`,
    kind: 'clip',
    thumbnail_path: null,
    match_id: 'm-1',
    capture_end_utc: null,
    match_start_time: null,
    section: 'teammates',
    owner_gamertag: owner,
    map_name: null,
    mode_name: null,
    liked: false,
    like_count: 0,
  }
}

const rail = vi.hoisted(() => ({ items: [] as MediaItemRow[] }))

vi.mock('@/features/media/queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/features/media/queries')>()
  return {
    ...actual,
    useRecentMediaRail: () => ({
      data: { items: { items: rail.items, pagination: { total: rail.items.length } } },
      isLoading: false,
      isError: false,
    }),
    useToggleMediaLike: () => ({ mutate: vi.fn(), isPending: false }),
  }
})

function openLightbox(owner: string) {
  rail.items = [row(owner)]
  const { container } = renderWithProviders(<RecentMediaRail playerSlug={ME} />)
  const card = container.querySelector('article')
  if (!card) throw new Error('vignette absente')
  fireEvent.click(card)
}

describe("RecentMediaRail - réassociation", () => {
  afterEach(() => {
    useAppShellStore.setState({ writablePlayerSlugs: null, availablePlayers: [], currentPlayer: null, isAdmin: false })
  })

  function asOwnerOfPage(isAdmin = false) {
    useAppShellStore.setState({
      writablePlayerSlugs: [ME],
      availablePlayers: [{ player_slug: ME, gamertag: ME } as PlayerSummary],
      currentPlayer: null,
      isAdmin,
    })
  }

  it("masque « Réassocier » sur le clip d'un autre joueur", () => {
    asOwnerOfPage()
    openLightbox('Ami')
    expect(screen.queryByRole('button', { name: 'Réassocier' })).not.toBeInTheDocument()
  })

  it('montre « Réassocier » sur son propre clip', () => {
    asOwnerOfPage()
    openLightbox(ME)
    expect(screen.getByRole('button', { name: 'Réassocier' })).toBeInTheDocument()
  })

  it("montre « Réassocier » à un admin, même sur le clip d'un autre", () => {
    asOwnerOfPage(true)
    openLightbox('Ami')
    expect(screen.getByRole('button', { name: 'Réassocier' })).toBeInTheDocument()
  })
})
