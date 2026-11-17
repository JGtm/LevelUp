/**
 * CoverFlowModal — fermeture : Échap ferme le visualiseur, un clic sur le voile aussi, un clic dedans
 * non ; la confirmation de suppression, couche du dessus, se ferme SEULE à la première frappe d'Échap,
 * le visualiseur à la suivante.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { CoverFlowModal } from './CoverFlowModal'
import type { NormalizedMediaItem } from '@/lib/api/types'

const ITEM: NormalizedMediaItem = {
  basename: 'capture.png',
  file_path: '/media/capture.png',
  kind: 'image',
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
}

function ouvre(onDelete?: () => void) {
  const onClose = vi.fn()
  renderWithProviders(
    <CoverFlowModal
      items={[ITEM]}
      startIndex={0}
      onClose={onClose}
      onToggleLike={vi.fn()}
      onDelete={onDelete}
      currentPlayerGamertag="JGtm"
    />,
  )
  // Le cadre du visualiseur : le parent direct du libellé de l'image est dans le cadre, le voile est au-dessus.
  const cadre = screen.getByRole('button', { name: /Fermer/ }).closest('div.max-w-5xl') as HTMLElement
  return { onClose, cadre, voile: cadre.parentElement as HTMLElement }
}

describe('CoverFlowModal — fermeture', () => {
  it('Échap ferme le visualiseur, une seule fois', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic sur le voile ferme, un clic dans le cadre ne ferme pas', () => {
    const { onClose, cadre, voile } = ouvre()
    fireEvent.mouseDown(cadre)
    fireEvent.click(cadre)
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.click(voile)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('les flèches gardent leur écouteur : une flèche ne ferme pas', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('confirmation de suppression ouverte : la première frappe d’Échap la ferme seule, la suivante ferme le visualiseur', () => {
    const { onClose } = ouvre(vi.fn())
    fireEvent.click(screen.getByRole('button', { name: /Supprimer définitivement ce média/ }))
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('alertdialog')).toBeNull()
    expect(onClose).not.toHaveBeenCalled()

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
