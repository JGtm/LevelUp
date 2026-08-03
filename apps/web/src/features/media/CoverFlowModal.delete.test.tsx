/**
 * Tests de la suppression définitive d'un média depuis le visualiseur
 * (v7.3 lot 2, item 3.1).
 *
 * Ce que ces tests verrouillent :
 *  - l'action n'existe QUE pour le propriétaire (ou un administrateur) ;
 *  - elle passe TOUJOURS par une confirmation — un clic isolé ne détruit rien ;
 *  - annuler n'appelle pas la suppression ;
 *  - les libellés sont fournis en FR et en EN.
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { CoverFlowModal } from './CoverFlowModal'
import type { MediaItemRow } from '@/lib/api/types'

function makeItem(overrides: Partial<MediaItemRow> = {}): MediaItemRow {
  return {
    basename: 'clip.mp4',
    file_path: '/media/clip.mp4',
    kind: 'clip',
    thumbnail_path: null,
    match_id: 'match-1',
    capture_end_utc: '2026-04-26T14:30:00Z',
    match_start_time: null,
    section: 'mine',
    owner_gamertag: 'JGtm',
    map_name: 'Aquarius',
    mode_name: 'Slayer',
    liked: false,
    like_count: 0,
    likers: undefined,
    total_likers: undefined,
    ...overrides,
  }
}

const OWN_ITEM = makeItem()
const FOREIGN_ITEM = makeItem({ owner_gamertag: 'Chocoboflor', file_path: '/media/autre.mp4' })

function setStore(state: { locale?: 'fr' | 'en'; isAdmin?: boolean }) {
  act(() => {
    useAppShellStore.setState({
      locale: state.locale ?? 'fr',
      isAdmin: state.isAdmin ?? false,
    })
  })
}

describe('CoverFlowModal — suppression définitive', () => {
  afterEach(() => {
    setStore({ locale: 'fr', isAdmin: false })
  })

  it('affiche l\'action de suppression au propriétaire du média', () => {
    setStore({})
    renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={vi.fn()}
        currentPlayerGamertag="JGtm"
      />,
    )
    expect(screen.getByRole('button', { name: /Supprimer définitivement ce média/ })).toBeInTheDocument()
  })

  it('masque l\'action sur le média d\'un autre joueur (non-admin)', () => {
    setStore({})
    renderWithProviders(
      <CoverFlowModal
        items={[FOREIGN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={vi.fn()}
        currentPlayerGamertag="JGtm"
      />,
    )
    expect(screen.queryByRole('button', { name: /Supprimer définitivement ce média/ })).not.toBeInTheDocument()
  })

  it('affiche l\'action à un administrateur sur le média d\'un autre joueur', () => {
    setStore({ isAdmin: true })
    renderWithProviders(
      <CoverFlowModal
        items={[FOREIGN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={vi.fn()}
        currentPlayerGamertag="JGtm"
      />,
    )
    expect(screen.getByRole('button', { name: /Supprimer définitivement ce média/ })).toBeInTheDocument()
  })

  it('n\'affiche aucune action quand onDelete n\'est pas fourni', () => {
    setStore({})
    renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        currentPlayerGamertag="JGtm"
      />,
    )
    expect(screen.queryByRole('button', { name: /Supprimer définitivement ce média/ })).not.toBeInTheDocument()
  })

  it('ne supprime RIEN au clic : une confirmation est exigée', () => {
    setStore({})
    const onDelete = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={onDelete}
        currentPlayerGamertag="JGtm"
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Supprimer définitivement ce média/ }))

    expect(onDelete).not.toHaveBeenCalled()
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()
    expect(screen.getByText(/Supprimer ce média \?/)).toBeInTheDocument()
    // Le corps doit annoncer l'irréversibilité — c'est ce qui rend le
    // consentement éclairé.
    expect(screen.getByText(/ne pourra pas être récupéré/)).toBeInTheDocument()
  })

  it('confirme : appelle onDelete avec le média affiché', () => {
    setStore({})
    const onDelete = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={onDelete}
        currentPlayerGamertag="JGtm"
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Supprimer définitivement ce média/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Supprimer définitivement' }))

    expect(onDelete).toHaveBeenCalledTimes(1)
    expect(onDelete).toHaveBeenCalledWith(expect.objectContaining({ file_path: '/media/clip.mp4' }))
  })

  it('annule : n\'appelle pas onDelete et ferme la confirmation', () => {
    setStore({})
    const onDelete = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={onDelete}
        currentPlayerGamertag="JGtm"
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Supprimer définitivement ce média/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))

    expect(onDelete).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('referme le visualiseur quand le dernier média a été supprimé', () => {
    setStore({})
    const onClose = vi.fn()
    const { rerender } = renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={onClose}
        onToggleLike={vi.fn()}
        onDelete={vi.fn()}
        currentPlayerGamertag="JGtm"
      />,
    )
    expect(onClose).not.toHaveBeenCalled()

    // Le refetch qui suit la suppression renvoie une galerie vide.
    act(() => {
      rerender(
        <CoverFlowModal
          items={[]}
          startIndex={0}
          onClose={onClose}
          onToggleLike={vi.fn()}
          onDelete={vi.fn()}
          currentPlayerGamertag="JGtm"
        />,
      )
    })
    expect(onClose).toHaveBeenCalled()
  })

  it('rend les libellés en anglais quand la locale est EN', () => {
    setStore({ locale: 'en' })
    renderWithProviders(
      <CoverFlowModal
        items={[OWN_ITEM]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onDelete={vi.fn()}
        currentPlayerGamertag="JGtm"
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Permanently delete this media/ }))

    expect(screen.getByText(/Delete this media\?/)).toBeInTheDocument()
    expect(screen.getByText(/cannot be recovered/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete permanently' })).toBeInTheDocument()
  })
})
