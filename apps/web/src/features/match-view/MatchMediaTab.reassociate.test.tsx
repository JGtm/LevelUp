import { afterEach, describe, expect, it } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchMediaTab } from './MatchMediaTab'
import { MATCH_VIEW_TEXT } from './i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchAssociatedMedia, PlayerSummary } from '@/lib/api/types'

// Règle serveur (domain.CanModifyMedia) : réassocier un média est réservé à son propriétaire
// (media_files.player_slug) ou à un admin. L'onglet Médias d'un match montre aussi les clips
// des coéquipiers : « Réassocier » n'y apparaît que sur les siens.

const ME = 'Moi'

function clip(owner: string): MatchAssociatedMedia {
  return {
    file_id: `f-${owner}`,
    file_name: `${owner}.mp4`,
    file_path: `/media/${owner}/clip.mp4`,
    kind: 'video',
    liked: false,
    owner_gamertag: owner,
  }
}

function openLightbox(owner: string) {
  const { container } = renderWithProviders(
    <MatchMediaTab items={[clip(owner)]} playerSlug={ME} matchId="m-1" t={MATCH_VIEW_TEXT.fr} />,
  )
  const card = container.querySelector('article')
  if (!card) throw new Error('vignette absente')
  fireEvent.click(card)
}

describe('MatchMediaTab - réassociation', () => {
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

  it("masque « Réassocier » sur le clip d'un coéquipier", () => {
    asOwnerOfPage()
    openLightbox('Coequipier')
    expect(screen.queryByRole('button', { name: 'Réassocier' })).not.toBeInTheDocument()
  })

  it('montre « Réassocier » sur son propre clip', () => {
    asOwnerOfPage()
    openLightbox(ME)
    expect(screen.getByRole('button', { name: 'Réassocier' })).toBeInTheDocument()
  })

  it("montre « Réassocier » à un admin, même sur le clip d'un autre", () => {
    asOwnerOfPage(true)
    openLightbox('Coequipier')
    expect(screen.getByRole('button', { name: 'Réassocier' })).toBeInTheDocument()
  })
})
