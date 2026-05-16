import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { AlertDialog } from './alert-dialog'

describe('AlertDialog', () => {
  it('open=false : ne rend rien', () => {
    render(
      <AlertDialog
        open={false}
        onOpenChange={() => {}}
        title="t"
        description="d"
        confirmLabel="OK"
        cancelLabel="Annuler"
        onConfirm={() => {}}
      />,
    )
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })

  it('open=true : rend titre, description, et 2 boutons', () => {
    render(
      <AlertDialog
        open
        onOpenChange={() => {}}
        title="Confirmer l'action ?"
        description="Cette action est irréversible."
        confirmLabel="Confirmer"
        cancelLabel="Annuler"
        onConfirm={() => {}}
      />,
    )
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()
    expect(screen.getByText("Confirmer l'action ?")).toBeInTheDocument()
    expect(screen.getByText('Cette action est irréversible.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmer' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Annuler' })).toBeInTheDocument()
  })

  it('clic Confirmer : appelle onConfirm', () => {
    const onConfirm = vi.fn()
    render(
      <AlertDialog
        open
        onOpenChange={() => {}}
        title="t"
        description="d"
        confirmLabel="Confirmer"
        cancelLabel="Annuler"
        onConfirm={onConfirm}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Confirmer' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('clic Annuler : appelle onOpenChange(false), pas onConfirm', () => {
    const onConfirm = vi.fn()
    const onOpenChange = vi.fn()
    render(
      <AlertDialog
        open
        onOpenChange={onOpenChange}
        title="t"
        description="d"
        confirmLabel="Confirmer"
        cancelLabel="Annuler"
        onConfirm={onConfirm}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('touche Escape : appelle onOpenChange(false)', () => {
    const onOpenChange = vi.fn()
    render(
      <AlertDialog
        open
        onOpenChange={onOpenChange}
        title="t"
        description="d"
        confirmLabel="OK"
        cancelLabel="Annuler"
        onConfirm={() => {}}
      />,
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('busy=true : boutons désactivés, Escape ignoré', () => {
    const onOpenChange = vi.fn()
    const onConfirm = vi.fn()
    render(
      <AlertDialog
        open
        onOpenChange={onOpenChange}
        title="t"
        description="d"
        confirmLabel="Confirmer"
        cancelLabel="Annuler"
        busy
        onConfirm={onConfirm}
      />,
    )
    expect(screen.getByRole('button', { name: 'Confirmer' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Annuler' })).toBeDisabled()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onOpenChange).not.toHaveBeenCalled()
  })

  it('attributs ARIA : role=alertdialog + aria-modal + labelling', () => {
    render(
      <AlertDialog
        open
        onOpenChange={() => {}}
        title="Titre"
        description="Description"
        confirmLabel="OK"
        cancelLabel="Annuler"
        onConfirm={() => {}}
      />,
    )
    const dialog = screen.getByRole('alertdialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby', 'alert-dialog-title')
    expect(dialog).toHaveAttribute('aria-describedby', 'alert-dialog-description')
  })
})
