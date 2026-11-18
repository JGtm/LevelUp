/**
 * AlertDialog — fermeture dans la pile d'Échap : occupée, la boîte garde la frappe (elle ne se ferme pas et ne
 * laisse pas Échap fermer la couche du dessous) ; une fois libre, Échap la ferme.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render } from '@testing-library/react'

import { useDismissable } from '@/lib/hooks/useDismissable'

import { AlertDialog } from './alert-dialog'

function Dessous({ onClose }: { onClose: () => void }) {
  useDismissable(true, onClose, { outside: null })
  return null
}

function rendu(busy: boolean, onOpenChange: (open: boolean) => void, onCloseDessous: () => void) {
  return (
    <>
      <Dessous onClose={onCloseDessous} />
      <AlertDialog
        open
        onOpenChange={onOpenChange}
        title="Titre"
        description="Description"
        confirmLabel="Oui"
        cancelLabel="Non"
        busy={busy}
        onConfirm={() => {}}
      />
    </>
  )
}

describe('AlertDialog — Échap dans une pile de couches', () => {
  it('occupée : Échap ne ferme ni la boîte ni la couche du dessous', () => {
    const onOpenChange = vi.fn()
    const onCloseDessous = vi.fn()
    render(rendu(true, onOpenChange, onCloseDessous))
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onOpenChange).not.toHaveBeenCalled()
    expect(onCloseDessous).not.toHaveBeenCalled()
  })

  it('libre : la première frappe ferme la boîte seule, la suivante la couche du dessous', () => {
    const onOpenChange = vi.fn()
    const onCloseDessous = vi.fn()
    const { rerender } = render(rendu(false, onOpenChange, onCloseDessous))
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onOpenChange).toHaveBeenCalledExactlyOnceWith(false)
    expect(onCloseDessous).not.toHaveBeenCalled()

    rerender(
      <>
        <Dessous onClose={onCloseDessous} />
      </>,
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onCloseDessous).toHaveBeenCalledTimes(1)
  })
})
