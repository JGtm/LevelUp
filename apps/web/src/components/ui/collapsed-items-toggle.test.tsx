/**
 * CollapsedItemsToggle — comportement du bouton de repli partagé (lot H, cf. en-tête du
 * composant). Les deux invariants de la grammaire du repli : N=0 -> pas de bouton, et le
 * libellé bascule avec l'état déplié/replié sans jamais perdre le compte.
 */
import { fireEvent, render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { CollapsedItemsToggle } from './collapsed-items-toggle'

const showLabelFmt = (count: number) => `Voir plus (${count})`

describe('CollapsedItemsToggle', () => {
  it('count <= 0 : ne rend rien (aucun repli vide affiché comme un repli)', () => {
    const { container } = render(
      <CollapsedItemsToggle
        count={0}
        expanded={false}
        onToggle={vi.fn()}
        showLabelFmt={showLabelFmt}
        hideLabel="Replier"
        hint="Rien n'est supprimé"
      />,
    )
    expect(container.querySelector('button')).toBeNull()
  })

  it('count négatif : même garde que count = 0', () => {
    const { container } = render(
      <CollapsedItemsToggle
        count={-1}
        expanded={false}
        onToggle={vi.fn()}
        showLabelFmt={showLabelFmt}
        hideLabel="Replier"
        hint="Rien n'est supprimé"
      />,
    )
    expect(container.querySelector('button')).toBeNull()
  })

  it('replié : affiche le libellé formaté avec le compte', () => {
    const vue = render(
      <CollapsedItemsToggle
        count={3}
        expanded={false}
        onToggle={vi.fn()}
        showLabelFmt={showLabelFmt}
        hideLabel="Replier"
        hint="Rien n'est supprimé"
      />,
    )
    expect(vue.getByRole('button', { name: 'Voir plus (3)' })).toBeTruthy()
  })

  it('déplié : affiche le libellé de repli, pas le compte', () => {
    const vue = render(
      <CollapsedItemsToggle
        count={3}
        expanded={true}
        onToggle={vi.fn()}
        showLabelFmt={showLabelFmt}
        hideLabel="Replier"
        hint="Rien n'est supprimé"
      />,
    )
    expect(vue.getByRole('button', { name: 'Replier' })).toBeTruthy()
    expect(vue.queryByRole('button', { name: /Voir plus/ })).toBeNull()
  })

  it('le clic déclenche onToggle', () => {
    const onToggle = vi.fn()
    const vue = render(
      <CollapsedItemsToggle
        count={3}
        expanded={false}
        onToggle={onToggle}
        showLabelFmt={showLabelFmt}
        hideLabel="Replier"
        hint="Rien n'est supprimé"
      />,
    )
    fireEvent.click(vue.getByRole('button', { name: 'Voir plus (3)' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it("porte l'infobulle (title) transmise par l'appelant", () => {
    const vue = render(
      <CollapsedItemsToggle
        count={1}
        expanded={false}
        onToggle={vi.fn()}
        showLabelFmt={showLabelFmt}
        hideLabel="Replier"
        hint="La promesse du repli"
      />,
    )
    expect(vue.getByRole('button').getAttribute('title')).toBe('La promesse du repli')
  })
})
