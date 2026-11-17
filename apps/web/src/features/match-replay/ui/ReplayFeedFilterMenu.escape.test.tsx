/**
 * ReplayFeedFilterMenu — ÉCHAP NE FERME QUE LA COUCHE DU DESSUS : menu de filtre ouvert, focus dans
 * le menu, une autre couche du rejeu déjà ouverte (le panneau de la vue, par exemple) reste ouverte ;
 * la frappe suivante ferme cette autre couche.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useDismissable } from '@/lib/hooks/useDismissable'

import type { FeedFilterOption } from '../model/feedFilterLogic'
import { ReplayFeedFilterMenu } from './ReplayFeedFilterMenu'

const OPTIONS: readonly FeedFilterOption[] = [
  { id: 'tout', count: 12 },
  { id: 'kills', count: 5 },
]

/** Une couche du rejeu ouverte avant le menu : celle du dessous. */
function CoucheDuDessous({ onClose }: { onClose: () => void }) {
  const ref = useDismissable(true, onClose, { outside: 'pointerdown' })
  return <div ref={ref} data-testid="dessous" data-replay-escape-layer="" />
}

function renderPile() {
  const onCloseDessous = vi.fn()
  render(
    <>
      <CoucheDuDessous onClose={onCloseDessous} />
      <ReplayFeedFilterMenu options={OPTIONS} active="tout" onChange={vi.fn()} locale="fr" />
    </>,
  )
  const declencheur = screen.getByRole('button', { name: /Filtrer le fil/ })
  fireEvent.click(declencheur)
  return { onCloseDessous, declencheur }
}

describe('ReplayFeedFilterMenu — Échap avec une autre couche ouverte', () => {
  it('ferme le menu seul, la couche du dessous reste ouverte, le focus revient au déclencheur', () => {
    const { onCloseDessous, declencheur } = renderPile()
    expect(screen.getByRole('menu')).toBeTruthy()

    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })

    expect(screen.queryByRole('menu')).toBeNull()
    expect(onCloseDessous).not.toHaveBeenCalled()
    expect(document.activeElement).toBe(declencheur)
  })

  it('la frappe suivante ferme la couche du dessous', () => {
    const { onCloseDessous } = renderPile()
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onCloseDessous).toHaveBeenCalledTimes(1)
  })
})
