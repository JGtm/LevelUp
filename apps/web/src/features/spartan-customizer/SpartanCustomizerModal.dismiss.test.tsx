/**
 * SpartanCustomizerModal — fermeture : Échap ferme la modale, un clic sur le voile aussi, un clic
 * dans la carte non.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { SpartanCustomizerModal } from './SpartanCustomizerModal'

function ouvre() {
  server.use(http.get('*/spartan-catalog.json', () => HttpResponse.json({ emblem_ids: [], nameplate_ids: [] })))
  const onClose = vi.fn()
  renderWithProviders(<SpartanCustomizerModal onClose={onClose} playerSlug="p1" />)
  const voile = screen.getByTestId('spartan-customizer-modal')
  return { onClose, voile, carte: voile.firstElementChild as HTMLElement }
}

describe('SpartanCustomizerModal — fermeture', () => {
  it('Échap ferme la modale, une seule fois', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic sur le voile ferme la modale', () => {
    const { onClose, voile } = ouvre()
    fireEvent.mouseDown(voile)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic dans la carte ne ferme pas la modale', () => {
    const { onClose, carte } = ouvre()
    fireEvent.mouseDown(carte)
    fireEvent.click(carte)
    expect(onClose).not.toHaveBeenCalled()
  })
})
