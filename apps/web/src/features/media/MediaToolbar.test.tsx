/**
 * Tests composant — MediaToolbar : bouton « Fichiers introuvables » (masqué à zéro, nombre
 * affiché, clic, libellé FR et EN).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { getMediaText } from './i18n'
import { MediaToolbar } from './MediaToolbar'

vi.mock('./queries', () => ({
  useMediaAudioConfig: () => ({ data: { mode: 'auto' as const } }),
  useUpdateMediaAudioConfig: () => ({ mutate: vi.fn(), isPending: false, isError: false, error: null }),
}))

function renderToolbar(missingCount: number, onOpenMissing = vi.fn()) {
  const text = getMediaText(useAppShellStore.getState().locale)
  renderWithProviders(
    <MediaToolbar
      text={text}
      playerSlug="p"
      kindFilter=""
      authorSlugs={[]}
      authors={[]}
      playlistFilter=""
      mapFilter=""
      modeFilter=""
      groupBy=""
      sortKey="date_desc"
      likedOnly={false}
      unassignedOnly={false}
      totalUnassigned={0}
      missingCount={missingCount}
      onOpenMissing={onOpenMissing}
      playlistOptions={[]}
      mapOptions={[]}
      modeOptions={[]}
      onKindChange={vi.fn()}
      onAuthorSlugsChange={vi.fn()}
      onPlaylistChange={vi.fn()}
      onMapChange={vi.fn()}
      onModeChange={vi.fn()}
      onSortChange={vi.fn()}
      onGroupByChange={vi.fn()}
      onLikedOnlyChange={vi.fn()}
      onUnassignedOnlyChange={vi.fn()}
    />,
  )
  return onOpenMissing
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('MediaToolbar — fichiers introuvables', () => {
  it('bouton masqué quand aucun fichier n’est introuvable', () => {
    renderToolbar(0)
    expect(screen.queryByRole('button', { name: 'Afficher les fichiers introuvables' })).not.toBeInTheDocument()
  })

  it('bouton visible avec le nombre, et un clic ouvre la fenêtre', () => {
    const onOpen = renderToolbar(3)
    const button = screen.getByRole('button', { name: 'Afficher les fichiers introuvables' })
    expect(button).toHaveTextContent('!')
    expect(button).toHaveTextContent('3')
    fireEvent.click(button)
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('libellé en anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderToolbar(2)
    expect(screen.getByRole('button', { name: 'Show missing files' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Afficher les fichiers introuvables' })).not.toBeInTheDocument()
  })
})
