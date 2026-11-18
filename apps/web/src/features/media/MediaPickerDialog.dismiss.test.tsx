/**
 * MediaPickerDialog — fermeture : Échap ferme la fenêtre, un clic sur la superposition aussi, un clic dans la
 * carte non ; une couche ouverte au-dessus se ferme SEULE à la première frappe d'Échap, la fenêtre à la suivante.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { useDismissable } from '@/lib/hooks/useDismissable'

import { MediaPickerDialog } from './MediaPickerDialog'

function ouvre() {
  const onClose = vi.fn()
  render(
    <MediaPickerDialog title="Choisir un match" closeLabel="Fermer" onClose={onClose}>
      <p>contenu</p>
    </MediaPickerDialog>,
  )
  // La carte est le parent du titre (l'en-tête est entre les deux) ; la superposition est au-dessus.
  const carte = screen.getByRole('heading', { name: 'Choisir un match' }).closest('div.max-w-3xl') as HTMLElement
  return { onClose, carte, voile: carte.parentElement as HTMLElement }
}

describe('MediaPickerDialog — fermeture', () => {
  it('Échap ferme la fenêtre, une seule fois', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('une autre touche ne ferme pas', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('un clic sur la superposition ferme, un clic dans la carte ne ferme pas', () => {
    const { onClose, carte, voile } = ouvre()
    fireEvent.mouseDown(carte)
    fireEvent.click(carte)
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.click(voile)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un appui sur la superposition ne ferme pas : seul le clic ferme', () => {
    const { onClose, voile } = ouvre()
    fireEvent.mouseDown(voile)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('une couche ouverte au-dessus : la première frappe d’Échap la ferme seule, la suivante ferme la fenêtre', () => {
    const onClose = vi.fn()
    const onCloseDessus = vi.fn()
    function Dessus() {
      useDismissable(true, onCloseDessus, { outside: null })
      return null
    }
    const fenetre = (
      <MediaPickerDialog title="Choisir un match" closeLabel="Fermer" onClose={onClose}>
        <p>contenu</p>
      </MediaPickerDialog>
    )
    const { rerender } = render(
      <>
        {fenetre}
        <Dessus />
      </>,
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onCloseDessus).toHaveBeenCalledTimes(1)
    expect(onClose).not.toHaveBeenCalled()

    rerender(<>{fenetre}</>)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
